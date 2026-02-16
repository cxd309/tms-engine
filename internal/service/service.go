// Package service defines the vehicle, route, and service types used in the TMS
// simulation, along with the SimService state machine.
package service

import (
	"encoding/json"
	"fmt"

	"github.com/cxd309/tms-engine/internal/graph"
	"github.com/cxd309/tms-engine/internal/kinematics"
)

// ServiceID is a unique string identifier for a service.
type ServiceID = string

// ServiceState describes the current motion state of a service.
type ServiceState string

const (
	StateStationary   ServiceState = "stationary"
	StateDwelling     ServiceState = "dwelling"
	StateAccelerating ServiceState = "accelerating"
	StateDecelerating ServiceState = "decelerating"
	StateCruising     ServiceState = "cruising"
)

// RouteStop is a node on a service's route with a required dwell time.
type RouteStop struct {
	NodeID graph.NodeID `json:"node_id"`
	TDwell float64      `json:"t_dwell"` // seconds
}

// Vehicle holds the static parameters of a vehicle type.
// The physics of acceleration and braking are encapsulated by the Kinem field;
// adding a new model only requires implementing kinematics.MotionModel and registering
// it in UnmarshalJSON below — no engine code changes needed.
type Vehicle struct {
	Name   string            `json:"name"`
	Length float64           `json:"length"` // vehicle length, metres
	Kinem  kinematics.MotionModel `json:"-"` // set by UnmarshalJSON
}

// kinematicsDisc is the minimum JSON structure needed to read the model discriminator.
type kinematicsDisc struct {
	Model string `json:"model"`
}

// vehicleJSON is the raw JSON shape of a Vehicle, before the kinematics model is resolved.
type vehicleJSON struct {
	Name   string          `json:"name"`
	Length float64         `json:"length"`
	Kinem  json.RawMessage `json:"kinematics"`
}

// UnmarshalJSON implements json.Unmarshaler for Vehicle.
// The "kinematics" field must contain a "model" discriminator key that selects
// the concrete implementation; the rest of the kinematics object is forwarded to
// that implementation's own unmarshaler.
//
// Supported models:
//   - "constant": fixed a_acc / a_dcc rates.
func (v *Vehicle) UnmarshalJSON(data []byte) error {
	var aux vehicleJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	v.Name = aux.Name
	v.Length = aux.Length

	if len(aux.Kinem) == 0 {
		return fmt.Errorf("vehicle %q: missing \"kinematics\" field", v.Name)
	}

	var disc kinematicsDisc
	if err := json.Unmarshal(aux.Kinem, &disc); err != nil {
		return fmt.Errorf("vehicle %q: reading kinematics model discriminator: %w", v.Name, err)
	}

	switch disc.Model {
	case kinematics.ConstantModelName:
		var k kinematics.ConstantAcceleration
		if err := json.Unmarshal(aux.Kinem, &k); err != nil {
			return fmt.Errorf("vehicle %q: parsing constant kinematics: %w", v.Name, err)
		}
		v.Kinem = k
	default:
		return fmt.Errorf("vehicle %q: unknown kinematics model %q", v.Name, disc.Model)
	}
	return nil
}

// Service is the static definition of a scheduled service.
type Service struct {
	ServiceID       ServiceID    `json:"service_id"`
	InitialPosition graph.NodeID `json:"initial_position"`
	Route           []RouteStop  `json:"route"`
	Vehicle         Vehicle      `json:"vehicle"`
	// DepartureDelay is the number of simulation-seconds the service waits
	// stationary before beginning to move. Use this to model staggered timetabled
	// departures (e.g. service B departs 120 s after service A). Zero = immediate.
	DepartureDelay float64 `json:"departure_delay,omitempty"` // seconds
}

// SimService is a Service enriched with live simulation state.
type SimService struct {
	Service
	CurrentPosition graph.Position `json:"current_position"`
	State           ServiceState   `json:"state"`
	Velocity        float64        `json:"velocity"`        // m/s
	RemainingDwell  float64        `json:"remaining_dwell"` // seconds
	NextStop        graph.NodeID   `json:"next_stop"`
	nextStopIndex   int
}

// GetFirstStop returns the first target stop node ID and its index in svc.Route.
// If the service starts at the first route stop, the second stop is returned instead.
func GetFirstStop(svc Service) (graph.NodeID, int, error) {
	if len(svc.Route) == 0 {
		return "", 0, fmt.Errorf("service %q has no route stops", svc.ServiceID)
	}
	if svc.InitialPosition == svc.Route[0].NodeID {
		if len(svc.Route) < 2 {
			return "", 0, fmt.Errorf("service %q: initial position is the only stop", svc.ServiceID)
		}
		return svc.Route[1].NodeID, 1, nil
	}
	return svc.Route[0].NodeID, 0, nil
}

// NewSimService creates a SimService from a static Service definition and a pre-computed
// initial graph position.
func NewSimService(svc Service, initialPos graph.Position) (*SimService, error) {
	nextStop, nextStopIdx, err := GetFirstStop(svc)
	if err != nil {
		return nil, err
	}
	return &SimService{
		Service:         svc,
		CurrentPosition: initialPos,
		State:           StateStationary,
		Velocity:        0,
		RemainingDwell:  0,
		NextStop:        nextStop,
		nextStopIndex:   nextStopIdx,
	}, nil
}

// BrakingDistance returns the minimum stopping distance from the service's current velocity.
func (s *SimService) BrakingDistance() float64 {
	return s.Vehicle.Kinem.BrakingDistance(s.Velocity)
}

// AdvanceDwell decrements the remaining dwell time by dt seconds.
// If the service is not yet dwelling it is transitioned into the dwelling state first.
func (s *SimService) AdvanceDwell(dt float64) {
	if s.State != StateDwelling {
		s.startDwell()
	}
	s.RemainingDwell -= dt
	if s.RemainingDwell <= 0 {
		s.endDwell()
	}
}

// ArriveAtStop transitions the service into the dwelling state upon reaching a stop.
func (s *SimService) ArriveAtStop() {
	s.startDwell()
}

func (s *SimService) startDwell() {
	s.State = StateDwelling
	s.Velocity = 0
	s.RemainingDwell = s.Route[s.nextStopIndex].TDwell
	s.advanceNextStop()
}

func (s *SimService) endDwell() {
	s.State = StateAccelerating
	s.Velocity = 0
	s.RemainingDwell = 0
}

func (s *SimService) advanceNextStop() {
	s.nextStopIndex = (s.nextStopIndex + 1) % len(s.Route)
	s.NextStop = s.Route[s.nextStopIndex].NodeID
}

// ServiceLog is a point-in-time snapshot of a SimService's state.
type ServiceLog struct {
	ServiceID       ServiceID      `json:"service_id"`
	CurrentPosition graph.Position `json:"current_position"`
	State           ServiceState   `json:"state"`
	Velocity        float64        `json:"velocity"`
	RemainingDwell  float64        `json:"remaining_dwell"`
	NextStop        graph.NodeID   `json:"next_stop"`
}

// GetLog returns a point-in-time snapshot of the service state.
func (s *SimService) GetLog() ServiceLog {
	return ServiceLog{
		ServiceID:       s.ServiceID,
		CurrentPosition: s.CurrentPosition,
		State:           s.State,
		Velocity:        s.Velocity,
		RemainingDwell:  s.RemainingDwell,
		NextStop:        s.NextStop,
	}
}
