// Package engine implements the TMS simulation loop.
//
// The simulation advances in fixed timesteps. Each step has two passes:
//
//  1. Safety pass - every service computes its minimal Movement Authority (MA),
//     which is the track ahead it physically needs to stop (braking distance).
//
//  2. Motion pass - every service proposes its desired movement, has that
//     proposal trimmed by the MA record from pass 1 and any edge speed limits,
//     then updates its position, velocity, and state accordingly.
package engine

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/cxd309/tms-engine/internal/graph"
	"github.com/cxd309/tms-engine/internal/service"
)

// NewTMS constructs a TMS from a SimulationInput, building the graph and
// placing each service at its initial position.
func NewTMS(input SimulationInput) (*TMS, error) {
	g, err := graph.NewGraph(input.GraphData)
	if err != nil {
		return nil, fmt.Errorf("building graph: %w", err)
	}

	services := make([]*service.SimService, 0, len(input.ServiceList))
	for _, svc := range input.ServiceList {
		firstStop, _, err := service.GetFirstStop(svc)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", svc.ServiceID, err)
		}
		initialPos, err := g.GetPathStartPosition(svc.InitialPosition, firstStop)
		if err != nil {
			return nil, fmt.Errorf("service %q initial position: %w", svc.ServiceID, err)
		}
		simSvc, err := service.NewSimService(svc, initialPos)
		if err != nil {
			return nil, fmt.Errorf("creating service %q: %w", svc.ServiceID, err)
		}
		services = append(services, simSvc)
	}

	return &TMS{
		meta:     input.Meta,
		graph:    g,
		services: services,
		curTime:  0,
	}, nil
}

// Run executes the full simulation and returns the log.
func (t *TMS) Run() (SimulationLog, error) {
	log := SimulationLog{Meta: t.meta}
	for t.curTime <= t.meta.RunTime {
		row, err := t.step()
		if err != nil {
			return SimulationLog{}, fmt.Errorf("at t=%.2f: %w", t.curTime, err)
		}
		log.Output = append(log.Output, row)
		t.curTime += t.meta.TimeStep
	}
	return log, nil
}

// step advances the simulation by one timestep and returns the resulting log row.
func (t *TMS) step() (SimulationLogRow, error) {
	dt := t.meta.TimeStep

	// Pass 1: compute the minimal MA (braking-distance safety envelope) for each service.
	minMAs := make(map[string]movementAuthority, len(t.services))
	for _, svc := range t.services {
		minMAs[svc.ServiceID] = svc.BrakingDistance()
	}

	// Pass 2: propose, grant, and apply movement for each service.
	for _, svc := range t.services {
		switch svc.State {
		case service.StateStationary:
			// Hold until the departure delay has elapsed, then start moving.
			if t.curTime < svc.DepartureDelay {
				continue
			}
			svc.State = service.StateAccelerating
			continue
		case service.StateDwelling:
			svc.AdvanceDwell(dt)
			continue
		}

		distToStop, err := t.distanceToNextStop(svc)
		if err != nil {
			return SimulationLogRow{}, fmt.Errorf("service %q distance to stop: %w", svc.ServiceID, err)
		}

		sl, err := t.getSpeedLimitInfo(svc)
		if err != nil {
			return SimulationLogRow{}, fmt.Errorf("service %q speed limit info: %w", svc.ServiceID, err)
		}

		// Kinematic proposal: how far would this service travel in dt with no MA constraints?
		proposedDist, newVelocity, newState := proposeMovement(svc, dt, distToStop, sl)

		// MA check: how far is the service allowed to travel given other services' safety envelopes?
		maxAllowed, err := t.computeMaxAllowedDistance(svc, minMAs)
		if err != nil {
			return SimulationLogRow{}, fmt.Errorf("service %q MA check: %w", svc.ServiceID, err)
		}

		grantedDist := math.Min(proposedDist, maxAllowed)

		// If MA trims the movement, recompute velocity from the shorter granted distance.
		if grantedDist < proposedDist {
			newVelocity, newState = constrainedKinematics(svc, grantedDist)
		}

		// Advance position and detect stop arrival.
		arrived, err := t.advancePosition(svc, grantedDist)
		if err != nil {
			return SimulationLogRow{}, fmt.Errorf("service %q advance: %w", svc.ServiceID, err)
		}

		if arrived {
			svc.ArriveAtStop()
		} else {
			svc.Velocity = newVelocity
			svc.State = newState
		}
	}

	// Snapshot all services for the log.
	logs := make([]service.ServiceLog, len(t.services))
	for i, svc := range t.services {
		logs[i] = svc.GetLog()
	}
	return SimulationLogRow{Timestamp: t.curTime, ServiceLogs: logs}, nil
}

// distanceToNextStop returns the metres from svc's current position to its next stop node.
func (t *TMS) distanceToNextStop(svc *service.SimService) (float64, error) {
	edge, err := t.graph.GetEdgeByID(svc.CurrentPosition.Edge)
	if err != nil {
		return 0, err
	}
	remainingOnEdge := edge.Length - svc.CurrentPosition.DistanceAlongEdge

	if edge.V == svc.NextStop {
		return remainingOnEdge, nil
	}

	path, err := t.graph.GetShortestPath(edge.V, svc.NextStop)
	if err != nil {
		return 0, fmt.Errorf("no path to next stop %q: %w", svc.NextStop, err)
	}
	return remainingOnEdge + path.Length, nil
}

// getSpeedLimitInfo returns the effective speed limits relevant to svc's current position.
// currentMax is the effective VMax on the current edge (min of vehicle VMax and edge limit).
// distToChange is the remaining distance on the current edge.
// nextMax is the effective VMax on the next edge toward the next stop; it is 0 when the
// next stop is at the end of the current edge (stop braking handles that case instead).
func (t *TMS) getSpeedLimitInfo(svc *service.SimService) (speedLimitInfo, error) {
	edge, err := t.graph.GetEdgeByID(svc.CurrentPosition.Edge)
	if err != nil {
		return speedLimitInfo{}, err
	}

	currentMax := svc.Vehicle.Kinem.VMax()
	if edge.SpeedLimit != nil && *edge.SpeedLimit < currentMax {
		currentMax = *edge.SpeedLimit
	}

	distToChange := edge.Length - svc.CurrentPosition.DistanceAlongEdge

	// If the next stop is at the end of this edge, stop braking already handles the approach.
	if edge.V == svc.NextStop {
		return speedLimitInfo{currentMax: currentMax, distToChange: distToChange, nextMax: 0}, nil
	}

	// Look ahead one edge to anticipate an upcoming speed limit change.
	nextMax := svc.Vehicle.Kinem.VMax()
	nextEdge, err := t.graph.GetNextEdge(edge.V, svc.NextStop)
	if err == nil && nextEdge.SpeedLimit != nil && *nextEdge.SpeedLimit < nextMax {
		nextMax = *nextEdge.SpeedLimit
	}

	return speedLimitInfo{currentMax: currentMax, distToChange: distToChange, nextMax: nextMax}, nil
}

// computeMaxAllowedDistance returns the maximum distance svc may travel without
// entering any other service's safety envelope (minimal MA + vehicle length).
//
// TODO: extend to full segment-based MA comparison for branching networks.
// Currently only checks services on the same edge.
func (t *TMS) computeMaxAllowedDistance(svc *service.SimService, minMAs map[string]movementAuthority) (float64, error) {
	maxDist := math.Inf(1)

	for _, other := range t.services {
		if other.ServiceID == svc.ServiceID {
			continue
		}

		// Only check services ahead on the same edge.
		// TODO: resolve conflicts across edge boundaries for full network coverage.
		if other.CurrentPosition.Edge != svc.CurrentPosition.Edge {
			continue
		}

		otherPos := other.CurrentPosition.DistanceAlongEdge
		myPos := svc.CurrentPosition.DistanceAlongEdge

		if otherPos <= myPos {
			continue // other is behind or level
		}

		// Other's protected zone: from its rear (front − length) minus its braking distance.
		// We must not enter that zone.
		safetyZoneStart := otherPos - other.Vehicle.Length - minMAs[other.ServiceID]
		allowed := safetyZoneStart - myPos
		if allowed < maxDist {
			maxDist = allowed
		}
	}

	if math.IsInf(maxDist, 1) {
		return math.MaxFloat64, nil
	}
	return math.Max(0, maxDist), nil
}

// advancePosition moves svc along the graph by dist metres, following the shortest
// path toward its next stop. Returns true if the service arrived at the next stop.
func (t *TMS) advancePosition(svc *service.SimService, dist float64) (bool, error) {
	for dist > 0 {
		edge, err := t.graph.GetEdgeByID(svc.CurrentPosition.Edge)
		if err != nil {
			return false, err
		}
		remaining := edge.Length - svc.CurrentPosition.DistanceAlongEdge

		if dist < remaining {
			svc.CurrentPosition.DistanceAlongEdge += dist
			return false, nil
		}

		dist -= remaining

		if edge.V == svc.NextStop {
			svc.CurrentPosition.DistanceAlongEdge = edge.Length
			return true, nil
		}

		nextEdge, err := t.graph.GetNextEdge(edge.V, svc.NextStop)
		if err != nil {
			return false, fmt.Errorf("advancing past edge %q: %w", edge.ID, err)
		}
		svc.CurrentPosition = graph.Position{Edge: nextEdge.ID, DistanceAlongEdge: 0}
	}
	return false, nil
}

// proposeMovement returns the distance, resulting velocity, and resulting state for svc
// over timestep dt, applying speed limits from sl and braking for the next stop.
//
// Priority (highest first):
//  1. Braking to stop at next stop
//  2. Braking for an upcoming edge speed limit reduction (lookahead)
//  3. Decelerating to the current edge speed limit (if currently over it)
//  4. Normal state machine (accelerate / cruise / decelerate)
func proposeMovement(svc *service.SimService, dt, distToStop float64, sl speedLimitInfo) (float64, float64, service.ServiceState) {
	v := svc.Velocity
	m := svc.Vehicle.Kinem
	effectiveVMax := sl.currentMax

	// 1. Stop braking (highest priority).
	if distToStop <= m.BrakingDistance(v) {
		dist, newV := m.DecelerateStep(v, 0, dt)
		if newV <= 0 {
			return dist, 0, service.StateDwelling
		}
		return dist, newV, service.StateDecelerating
	}

	// 2. Lookahead braking for an upcoming lower speed limit on the next edge.
	if sl.nextMax > 0 && sl.nextMax < effectiveVMax && v > sl.nextMax {
		if sl.distToChange <= m.BrakingDistanceTo(v, sl.nextMax) {
			dist, newV := m.DecelerateStep(v, sl.nextMax, dt)
			if newV <= sl.nextMax {
				return dist, newV, service.StateCruising
			}
			return dist, newV, service.StateDecelerating
		}
	}

	// 3. Decelerate to current edge speed limit if currently over it.
	if v > effectiveVMax {
		dist, newV := m.DecelerateStep(v, effectiveVMax, dt)
		if newV <= effectiveVMax {
			return dist, newV, service.StateCruising
		}
		return dist, newV, service.StateDecelerating
	}

	// 4. Normal state machine.
	switch svc.State {
	case service.StateAccelerating:
		dist, newV := m.AccelerateStep(v, effectiveVMax, dt)
		if newV >= effectiveVMax {
			return dist, effectiveVMax, service.StateCruising
		}
		return dist, newV, service.StateAccelerating

	case service.StateCruising:
		return effectiveVMax * dt, effectiveVMax, service.StateCruising

	case service.StateDecelerating:
		dist, newV := m.DecelerateStep(v, 0, dt)
		if newV <= 0 {
			return dist, 0, service.StateDwelling
		}
		return dist, newV, service.StateDecelerating

	default:
		return 0, v, svc.State
	}
}

// constrainedKinematics derives the velocity after travelling grantedDist under
// maximum braking (used when the MA limits movement to less than proposed).
func constrainedKinematics(svc *service.SimService, grantedDist float64) (float64, service.ServiceState) {
	newV := svc.Vehicle.Kinem.VelocityAfterBraking(svc.Velocity, grantedDist)
	if newV <= 0 {
		return 0, service.StateDwelling
	}
	return newV, service.StateDecelerating
}

// RunJSON is the primary entry point for all three compilation targets (CLI, WASM, clib).
// It accepts a JSON-encoded SimulationInput, runs the simulation, and returns a
// JSON-encoded SimulationLog.
func RunJSON(jsonInput string) (string, error) {
	var input SimulationInput
	if err := json.Unmarshal([]byte(jsonInput), &input); err != nil {
		return "", fmt.Errorf("invalid input JSON: %w", err)
	}

	tms, err := NewTMS(input)
	if err != nil {
		return "", err
	}

	simLog, err := tms.Run()
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(simLog)
	if err != nil {
		return "", fmt.Errorf("marshaling output: %w", err)
	}
	return string(out), nil
}
