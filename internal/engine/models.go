package engine

import (
	"github.com/cxd309/tms-engine/internal/graph"
	"github.com/cxd309/tms-engine/internal/service"
)

// SimulationMeta holds the identity and timing parameters for a simulation run.
type SimulationMeta struct {
	SimulationID string  `json:"simulation_id"`
	RunTime      float64 `json:"run_time"`  // seconds
	TimeStep     float64 `json:"time_step"` // seconds
}

// SimulationInput is the JSON-serialisable input to the engine.
type SimulationInput struct {
	Meta        SimulationMeta    `json:"simulation_meta"`
	GraphData   graph.GraphData   `json:"graph_data"`
	ServiceList []service.Service `json:"service_list"`
}

// SimulationLogRow is the state of all services at a single simulation timestep.
type SimulationLogRow struct {
	Timestamp   float64              `json:"timestamp"` // seconds
	ServiceLogs []service.ServiceLog `json:"service_logs"`
}

// SimulationLog is the complete output of a simulation run.
type SimulationLog struct {
	Meta   SimulationMeta     `json:"simulation_meta"`
	Output []SimulationLogRow `json:"output"`
}

// movementAuthority is the distance ahead (metres) a service is authorised to travel.
type movementAuthority = float64

// speedLimitInfo carries effective speed limit context derived from the graph for
// a single service at a single timestep.
type speedLimitInfo struct {
	currentMax   float64 // effective speed limit on the current edge (min of vehicle VMax and edge limit)
	distToChange float64 // distance remaining on the current edge (where the limit may change)
	nextMax      float64 // effective speed limit on the next edge; 0 if the next stop ends the current edge
}

// TMS simulation engine state.
type TMS struct {
	meta     SimulationMeta
	graph    *graph.Graph
	services []*service.SimService
	curTime  float64
}
