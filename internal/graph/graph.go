// Package graph provides graph data structures and shortest-path algorithms
// for the TMS simulation network.
package graph

import (
	"fmt"
)

// NodeID, EdgeID, PathID are string aliases used as identifiers.
type (
	NodeID = string
	EdgeID = string
	PathID = string
)

// NodeType classifies a node in the network.
type NodeType string

const (
	NodeTypeMain    NodeType = "main"
	NodeTypeStation NodeType = "station"
	NodeTypeSide    NodeType = "side"
)

// Coordinate is a 2D position in metres.
type Coordinate struct {
	X float64 `json:"x"` // metres
	Y float64 `json:"y"` // metres
}

// Node is a point in the network graph.
type Node struct {
	ID   NodeID     `json:"node_id"`
	Loc  Coordinate `json:"loc"`
	Type NodeType   `json:"type"`
}

// Edge is a directed connection between two nodes with a length in metres.
// SpeedLimit is optional: if nil the edge imposes no limit and the vehicle's
// own VMax applies. Set it (in m/s) to restrict speed on a particular section.
type Edge struct {
	ID         EdgeID   `json:"edge_id"`
	U          NodeID   `json:"u"`
	V          NodeID   `json:"v"`
	Length     float64  `json:"length"`                // metres
	SpeedLimit *float64 `json:"speed_limit,omitempty"` // m/s; nil = no restriction
}

// GraphData is the serialisable input representation of a network graph.
type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Position is a point along a directed edge in the graph.
type Position struct {
	Edge              EdgeID  `json:"edge"`
	DistanceAlongEdge float64 `json:"distance_along_edge"` // metres
}

// PathInfo holds the result of a shortest-path computation.
type PathInfo struct {
	ID     PathID
	Route  []NodeID // ordered node IDs from start to end
	Length float64  // total path length in metres
}

// Segment is a contiguous stretch of a single edge, defined by start and end distances.
type Segment struct {
	Edge  EdgeID  `json:"edge"`
	Start float64 `json:"start"` // metres along edge
	End   float64 `json:"end"`   // metres along edge
}

// Length returns the length of the segment in metres.
func (s Segment) Length() float64 { return s.End - s.Start }

// Graph is a directed weighted graph with cached shortest-path computation.
type Graph struct {
	nodes       []Node
	edges       []Edge
	nodeMap     map[NodeID]Node
	edgeMap     map[EdgeID]Edge
	edgeByNodes map[NodeID]map[NodeID]Edge // u → v → edge
	// Floyd-Warshall tables; nil until first needed.
	dist     map[NodeID]map[NodeID]float64
	nextNode map[NodeID]map[NodeID]NodeID
	// Path cache; cleared whenever the graph topology changes.
	pathCache map[PathID]PathInfo
}

// NewGraph builds a Graph from GraphData, returning an error if any node or edge
// references are invalid.
func NewGraph(data GraphData) (*Graph, error) {
	g := &Graph{
		nodeMap:     make(map[NodeID]Node),
		edgeMap:     make(map[EdgeID]Edge),
		edgeByNodes: make(map[NodeID]map[NodeID]Edge),
		pathCache:   make(map[PathID]PathInfo),
	}
	for _, n := range data.Nodes {
		if err := g.AddNode(n); err != nil {
			return nil, err
		}
	}
	for _, e := range data.Edges {
		if err := g.AddEdge(e); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// AddNode adds a node to the graph. Returns an error if the node ID already exists.
func (g *Graph) AddNode(n Node) error {
	if _, exists := g.nodeMap[n.ID]; exists {
		return fmt.Errorf("node %q already exists", n.ID)
	}
	g.nodes = append(g.nodes, n)
	g.nodeMap[n.ID] = n
	g.dist = nil // invalidate cached paths
	return nil
}

// AddEdge adds a directed edge to the graph. Returns an error if the edge ID already
// exists or either endpoint node is missing.
func (g *Graph) AddEdge(e Edge) error {
	if _, exists := g.edgeMap[e.ID]; exists {
		return fmt.Errorf("edge %q already exists", e.ID)
	}
	if _, ok := g.nodeMap[e.U]; !ok {
		return fmt.Errorf("edge %q: source node %q not found", e.ID, e.U)
	}
	if _, ok := g.nodeMap[e.V]; !ok {
		return fmt.Errorf("edge %q: target node %q not found", e.ID, e.V)
	}
	g.edges = append(g.edges, e)
	g.edgeMap[e.ID] = e
	if g.edgeByNodes[e.U] == nil {
		g.edgeByNodes[e.U] = make(map[NodeID]Edge)
	}
	g.edgeByNodes[e.U][e.V] = e
	g.dist = nil // invalidate cached paths
	return nil
}

// pathKey returns a canonical string key for a start→end pair.
func pathKey(start, end NodeID) PathID { return start + "->" + end }

// GetEdgeByID looks up an edge by its ID.
func (g *Graph) GetEdgeByID(id EdgeID) (Edge, error) {
	e, ok := g.edgeMap[id]
	if !ok {
		return Edge{}, fmt.Errorf("edge %q not found", id)
	}
	return e, nil
}

// GetEdge returns the directed edge from u to v.
func (g *Graph) GetEdge(u, v NodeID) (Edge, error) {
	if m, ok := g.edgeByNodes[u]; ok {
		if e, ok := m[v]; ok {
			return e, nil
		}
	}
	return Edge{}, fmt.Errorf("no edge from %q to %q", u, v)
}

// GetNextEdge returns the first edge on the shortest path from u toward dest.
func (g *Graph) GetNextEdge(u, dest NodeID) (Edge, error) {
	path, err := g.GetShortestPath(u, dest)
	if err != nil {
		return Edge{}, err
	}
	if len(path.Route) < 2 {
		return Edge{}, fmt.Errorf("already at destination %q", dest)
	}
	return g.GetEdge(path.Route[0], path.Route[1])
}

// GetPathStartPosition returns the Position at the start of the first edge on the
// shortest path from u to v.
func (g *Graph) GetPathStartPosition(u, v NodeID) (Position, error) {
	path, err := g.GetShortestPath(u, v)
	if err != nil {
		return Position{}, err
	}
	if len(path.Route) < 2 {
		return Position{}, fmt.Errorf("no edges on path from %q to %q", u, v)
	}
	edge, err := g.GetEdge(path.Route[0], path.Route[1])
	if err != nil {
		return Position{}, err
	}
	return Position{Edge: edge.ID, DistanceAlongEdge: 0}, nil
}
