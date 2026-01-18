package core

import (
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
