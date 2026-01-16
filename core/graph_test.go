package core

import (
	"fmt"
	"testing"
	"time"

	"example.com/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	if g == nil {
		t.Fatal("NewGraph() returned nil")
	}

	if g.Nodes == nil {
		t.Error("Graph.Nodes is nil")
	}

	if g.Edges == nil {
		t.Error("Graph.Edges is nil")
	}

	if len(g.Nodes) != 0 {
		t.Errorf("Expected empty nodes, got %d", len(g.Nodes))
	}

	if len(g.Edges) != 0 {
		t.Errorf("Expected empty edges, got %d", len(g.Edges))
	}
}

func TestAddNode(t *testing.T) {
	g := NewGraph()

	node := &pb.Node{
		Id:       "node1",
		Location: "New York",
	}

	AddNode(g, node)

	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}

	if g.Nodes[0].Id != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", g.Nodes[0].Id)
	}

	// Add another node
	node2 := &pb.Node{
		Id:       "node2",
		Location: "Los Angeles",
	}

	AddNode(g, node2)

	if len(g.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestAddEdge(t *testing.T) {
	g := NewGraph()

	edge := &pb.Edge{
		FromId:          "node1",
		ToId:            "node2",
		DurationSeconds: 3600, // 1 hour
	}

	AddEdge(g, edge)

	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}

	if g.Edges[0].FromId != "node1" {
		t.Errorf("Expected FromId 'node1', got '%s'", g.Edges[0].FromId)
	}

	if g.Edges[0].ToId != "node2" {
		t.Errorf("Expected ToId 'node2', got '%s'", g.Edges[0].ToId)
	}

	// Add another edge
	edge2 := &pb.Edge{
		FromId:          "node2",
		ToId:            "node3",
		DurationSeconds: 7200, // 2 hours
	}

	AddEdge(g, edge2)

	if len(g.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(g.Edges))
	}
}

func TestGetNodeByID(t *testing.T) {
	g := NewGraph()

	node1 := &pb.Node{
		Id:       "node1",
		Location: "New York",
	}
	node2 := &pb.Node{
		Id:       "node2",
		Location: "Los Angeles",
	}

	AddNode(g, node1)
	AddNode(g, node2)

	// Test finding existing node
	found := GetNodeByID(g, "node1")
	if found == nil {
		t.Fatal("Expected to find node1, got nil")
	}

	if found.Id != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", found.Id)
	}

	if found.Location != "New York" {
		t.Errorf("Expected location 'New York', got '%s'", found.Location)
	}

	// Test finding another node
	found2 := GetNodeByID(g, "node2")
	if found2 == nil {
		t.Fatal("Expected to find node2, got nil")
	}

	if found2.Id != "node2" {
		t.Errorf("Expected node ID 'node2', got '%s'", found2.Id)
	}

	// Test finding non-existent node
	notFound := GetNodeByID(g, "nonexistent")
	if notFound != nil {
		t.Errorf("Expected nil for non-existent node, got %v", notFound)
	}
}

func TestGetEdgesFromNode(t *testing.T) {
	g := NewGraph()

	// Create nodes
	node1 := &pb.Node{Id: "node1", Location: "New York"}
	node2 := &pb.Node{Id: "node2", Location: "Chicago"}
	node3 := &pb.Node{Id: "node3", Location: "Los Angeles"}

	AddNode(g, node1)
	AddNode(g, node2)
	AddNode(g, node3)

	// Create edges: node1 -> node2, node1 -> node3, node2 -> node3
	edge1 := &pb.Edge{FromId: "node1", ToId: "node2", DurationSeconds: 3600}
	edge2 := &pb.Edge{FromId: "node1", ToId: "node3", DurationSeconds: 7200}
	edge3 := &pb.Edge{FromId: "node2", ToId: "node3", DurationSeconds: 5400}

	AddEdge(g, edge1)
	AddEdge(g, edge2)
	AddEdge(g, edge3)

	// Get edges from node1
	edges := GetEdgesFromNode(g, "node1")

	if len(edges) != 2 {
		t.Errorf("Expected 2 edges from node1, got %d", len(edges))
	}

	// Verify edge destinations
	toIDs := make(map[string]bool)
	for _, edge := range edges {
		toIDs[edge.ToId] = true
		if edge.FromId != "node1" {
			t.Errorf("Expected FromId 'node1', got '%s'", edge.FromId)
		}
	}

	if !toIDs["node2"] || !toIDs["node3"] {
		t.Error("Expected edges to node2 and node3")
	}

	// Get edges from node2
	edges2 := GetEdgesFromNode(g, "node2")
	if len(edges2) != 1 {
		t.Errorf("Expected 1 edge from node2, got %d", len(edges2))
	}

	// Get edges from node3 (should be empty)
	edges3 := GetEdgesFromNode(g, "node3")
	if len(edges3) != 0 {
		t.Errorf("Expected 0 edges from node3, got %d", len(edges3))
	}
}

func TestGetEdgesToNode(t *testing.T) {
	g := NewGraph()

	// Create nodes
	node1 := &pb.Node{Id: "node1", Location: "New York"}
	node2 := &pb.Node{Id: "node2", Location: "Chicago"}
	node3 := &pb.Node{Id: "node3", Location: "Los Angeles"}

	AddNode(g, node1)
	AddNode(g, node2)
	AddNode(g, node3)

	// Create edges: node1 -> node2, node1 -> node3, node2 -> node3
	edge1 := &pb.Edge{FromId: "node1", ToId: "node2", DurationSeconds: 3600}
	edge2 := &pb.Edge{FromId: "node1", ToId: "node3", DurationSeconds: 7200}
	edge3 := &pb.Edge{FromId: "node2", ToId: "node3", DurationSeconds: 5400}

	AddEdge(g, edge1)
	AddEdge(g, edge2)
	AddEdge(g, edge3)

	// Get edges to node3
	edges := GetEdgesToNode(g, "node3")

	if len(edges) != 2 {
		t.Errorf("Expected 2 edges to node3, got %d", len(edges))
	}

	// Verify edge sources
	fromIDs := make(map[string]bool)
	for _, edge := range edges {
		fromIDs[edge.FromId] = true
		if edge.ToId != "node3" {
			t.Errorf("Expected ToId 'node3', got '%s'", edge.ToId)
		}
	}

	if !fromIDs["node1"] || !fromIDs["node2"] {
		t.Error("Expected edges from node1 and node2")
	}

	// Get edges to node2
	edges2 := GetEdgesToNode(g, "node2")
	if len(edges2) != 1 {
		t.Errorf("Expected 1 edge to node2, got %d", len(edges2))
	}

	// Get edges to node1 (should be empty)
	edges3 := GetEdgesToNode(g, "node1")
	if len(edges3) != 0 {
		t.Errorf("Expected 0 edges to node1, got %d", len(edges3))
	}
}

func TestGraphWithTimestamps(t *testing.T) {
	g := NewGraph()

	now := time.Now()
	timestamp1 := timestamppb.New(now)
	timestamp2 := timestamppb.New(now.Add(2 * time.Hour))

	node := &pb.Node{
		Id:            "node1",
		Location:      "New York",
		FromTimestamp: timestamp1,
		ToTimestamp:   timestamp2,
		IsInterCity:   true,
	}

	AddNode(g, node)

	found := GetNodeByID(g, "node1")
	if found == nil {
		t.Fatal("Expected to find node1")
	}

	if found.FromTimestamp == nil {
		t.Error("Expected FromTimestamp to be set")
	}

	if found.ToTimestamp == nil {
		t.Error("Expected ToTimestamp to be set")
	}

	if !found.IsInterCity {
		t.Error("Expected IsInterCity to be true")
	}
}

func TestGraphWithTransport(t *testing.T) {
	g := NewGraph()

	// Create nodes
	node1 := &pb.Node{Id: "node1", Location: "New York"}
	node2 := &pb.Node{Id: "node2", Location: "Los Angeles"}

	AddNode(g, node1)
	AddNode(g, node2)

	// Create edge with transport
	transport := &pb.Transport{
		Id:     1,
		Type:   pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		Status: "confirmed",
	}

	edge := &pb.Edge{
		FromId:          "node1",
		ToId:            "node2",
		DurationSeconds: 18000, // 5 hours
		Transport:       transport,
	}

	AddEdge(g, edge)

	edges := GetEdgesFromNode(g, "node1")
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].Transport == nil {
		t.Error("Expected Transport to be set")
	}

	if edges[0].Transport.Id != 1 {
		t.Errorf("Expected Transport ID 1, got %d", edges[0].Transport.Id)
	}

	if edges[0].DurationSeconds != 18000 {
		t.Errorf("Expected DurationSeconds 18000, got %d", edges[0].DurationSeconds)
	}
}

func TestGraphComplexScenario(t *testing.T) {
	g := NewGraph()

	// Create a multi-city trip: NYC -> Chicago -> LA -> NYC
	cities := []string{"New York", "Chicago", "Los Angeles", "New York"}

	for i, city := range cities {
		node := &pb.Node{
			Id:       fmt.Sprintf("node%d", i+1),
			Location: city,
		}
		AddNode(g, node)
	}

	// Create edges
	edges := []struct {
		from, to string
		duration int64
	}{
		{"node1", "node2", 3600},  // NYC -> Chicago
		{"node2", "node3", 7200},  // Chicago -> LA
		{"node3", "node4", 10800}, // LA -> NYC
	}

	for _, e := range edges {
		edge := &pb.Edge{
			FromId:          e.from,
			ToId:            e.to,
			DurationSeconds: e.duration,
		}
		AddEdge(g, edge)
	}

	// Verify graph structure
	if len(g.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(g.Nodes))
	}

	if len(g.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(g.Edges))
	}

	// Verify each node has correct edges
	node1Edges := GetEdgesFromNode(g, "node1")
	if len(node1Edges) != 1 || node1Edges[0].ToId != "node2" {
		t.Error("Node1 should have one edge to node2")
	}

	node2Edges := GetEdgesFromNode(g, "node2")
	if len(node2Edges) != 1 || node2Edges[0].ToId != "node3" {
		t.Error("Node2 should have one edge to node3")
	}

	node3Edges := GetEdgesFromNode(g, "node3")
	if len(node3Edges) != 1 || node3Edges[0].ToId != "node4" {
		t.Error("Node3 should have one edge to node4")
	}

	// Verify incoming edges
	node2Incoming := GetEdgesToNode(g, "node2")
	if len(node2Incoming) != 1 || node2Incoming[0].FromId != "node1" {
		t.Error("Node2 should have one incoming edge from node1")
	}
}
