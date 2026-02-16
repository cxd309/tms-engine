package graph

import (
	"fmt"
	"math"
)

// computeShortestPaths runs Floyd-Warshall over all nodes and edges.
func (g *Graph) computeShortestPaths() {
	nodeIDs := make([]NodeID, len(g.nodes))
	for i, n := range g.nodes {
		nodeIDs[i] = n.ID
	}

	dist := make(map[NodeID]map[NodeID]float64, len(nodeIDs))
	next := make(map[NodeID]map[NodeID]NodeID, len(nodeIDs))
	for _, i := range nodeIDs {
		dist[i] = make(map[NodeID]float64, len(nodeIDs))
		next[i] = make(map[NodeID]NodeID, len(nodeIDs))
		for _, j := range nodeIDs {
			dist[i][j] = math.Inf(1)
		}
		dist[i][i] = 0
	}
	for _, e := range g.edges {
		dist[e.U][e.V] = e.Length
		next[e.U][e.V] = e.V
	}
	for _, k := range nodeIDs {
		for _, i := range nodeIDs {
			for _, j := range nodeIDs {
				if d := dist[i][k] + dist[k][j]; d < dist[i][j] {
					dist[i][j] = d
					next[i][j] = next[i][k]
				}
			}
		}
	}

	g.dist = dist
	g.nextNode = next
	g.pathCache = make(map[PathID]PathInfo) // clear stale cache
}

func (g *Graph) ensureShortestPaths() {
	if g.dist == nil {
		g.computeShortestPaths()
	}
}

func (g *Graph) reconstructPath(u, v NodeID) []NodeID {
	route := []NodeID{u}
	for u != v {
		n, ok := g.nextNode[u][v]
		if !ok || n == "" {
			return nil // no path
		}
		u = n
		route = append(route, u)
	}
	return route
}

// GetShortestPath returns the shortest path between start and end, using a cache.
// Returns an error if no path exists.
func (g *Graph) GetShortestPath(start, end NodeID) (PathInfo, error) {
	if start == end {
		return PathInfo{ID: pathKey(start, end), Route: []NodeID{start}, Length: 0}, nil
	}
	key := pathKey(start, end)
	if p, ok := g.pathCache[key]; ok {
		return p, nil
	}
	g.ensureShortestPaths()
	d, ok := g.dist[start][end]
	if !ok || math.IsInf(d, 1) {
		return PathInfo{}, fmt.Errorf("no path from %q to %q", start, end)
	}
	route := g.reconstructPath(start, end)
	p := PathInfo{ID: key, Route: route, Length: d}
	g.pathCache[key] = p
	return p, nil
}
