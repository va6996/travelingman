package core

import (
	"fmt"
	"strings"

	"github.com/va6996/travelingman/pb"
)

// NewGraph creates a new empty graph
func NewGraph() *pb.Graph {
	return &pb.Graph{
		Nodes: make([]*pb.Node, 0),
		Edges: make([]*pb.Edge, 0),
	}
}

// AddNode adds a node to the graph
func AddNode(g *pb.Graph, node *pb.Node) {
	g.Nodes = append(g.Nodes, node)
}

// AddEdge adds an edge to the graph
func AddEdge(g *pb.Graph, edge *pb.Edge) {
	g.Edges = append(g.Edges, edge)
}

// GetNodeByID returns a node by its ID
func GetNodeByID(g *pb.Graph, id string) *pb.Node {
	for _, node := range g.Nodes {
		if node.Id == id {
			return node
		}
	}
	return nil
}

// GetEdgesFromNode returns all edges originating from a given node
func GetEdgesFromNode(g *pb.Graph, nodeID string) []*pb.Edge {
	var edges []*pb.Edge
	for _, edge := range g.Edges {
		if edge.FromId == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

// GetEdgesToNode returns all edges leading to a given node
func GetEdgesToNode(g *pb.Graph, nodeID string) []*pb.Edge {
	var edges []*pb.Edge
	for _, edge := range g.Edges {
		if edge.ToId == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

// ValidateNodes checks if all nodes have valid IDs and no duplicates.
func ValidateNodes(g *pb.Graph) error {
	if g == nil {
		return nil
	}
	nodeIDs := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Id == "" {
			return fmt.Errorf("found node with missing ID")
		}
		if nodeIDs[n.Id] {
			return fmt.Errorf("duplicate Node ID found: %s", n.Id)
		}
		nodeIDs[n.Id] = true
	}
	return nil
}

// ValidateGraph performs comprehensive validation of the graph structure.
func ValidateGraph(g *pb.Graph) error {
	if g == nil {
		return fmt.Errorf("graph is nil")
	}

	// Validate nodes first
	if err := ValidateNodes(g); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	// Create node ID map for edge validation
	nodeIDs := make(map[string]bool)
	for _, n := range g.Nodes {
		nodeIDs[n.Id] = true
	}

	// Validate edges and collect all errors
	var errors []string
	for i, edge := range g.Edges {
		if edge.FromId == "" {
			errors = append(errors, fmt.Sprintf("edge %d: FromId is empty", i))
		}
		if edge.ToId == "" {
			errors = append(errors, fmt.Sprintf("edge %d: ToId is empty", i))
		}
		if !nodeIDs[edge.FromId] {
			errors = append(errors, fmt.Sprintf("edge %d: FromId '%s' not found in nodes", i, edge.FromId))
		}
		if !nodeIDs[edge.ToId] {
			errors = append(errors, fmt.Sprintf("edge %d: ToId '%s' not found in nodes", i, edge.ToId))
		}
		if edge.DurationSeconds < 0 {
			errors = append(errors, fmt.Sprintf("edge %d: negative duration %d", i, edge.DurationSeconds))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("graph validation failed with %d errors:\n- %s", len(errors), strings.Join(errors, "\n- "))
	}

	return nil
}

// HasCycle detects if there is a cycle in the directed graph.
func HasCycle(g *pb.Graph) bool {
	if g == nil || len(g.Edges) == 0 {
		return false
	}

	// Adjacency list
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.FromId] = append(adj[e.FromId], e.ToId)
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var isCyclic func(string) bool
	isCyclic = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if isCyclic(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check all starting points (nodes and any IDs used in edges)
	allIDs := make(map[string]bool)
	for _, n := range g.Nodes {
		allIDs[n.Id] = true
	}
	for _, e := range g.Edges {
		allIDs[e.FromId] = true
		allIDs[e.ToId] = true
	}

	for id := range allIDs {
		if !visited[id] {
			if isCyclic(id) {
				return true
			}
		}
	}

	return false
}
