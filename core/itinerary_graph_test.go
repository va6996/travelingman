package core

import (
	"fmt"
	"testing"
	"time"

	"example.com/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConstructGraph_Empty(t *testing.T) {
	g := ConstructGraph(nil)
	if g == nil {
		t.Fatal("Expected graph, got nil")
	}
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Error("Expected empty graph")
	}

	it := &pb.Itinerary{}
	g = ConstructGraph(it)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Error("Expected empty graph for empty itinerary")
	}
}

func TestConstructGraph_Accommodation(t *testing.T) {
	now := time.Now()
	checkIn := timestamppb.New(now)
	checkOut := timestamppb.New(now.Add(24 * time.Hour))

	acc := &pb.Accommodation{
		Id:       123,
		Name:     "Test Hotel",
		Address:  "123 Test St",
		CheckIn:  checkIn,
		CheckOut: checkOut,
	}

	it := &pb.Itinerary{
		StartTime:     checkIn,
		Accommodation: []*pb.Accommodation{acc},
	}

	g := ConstructGraph(it)

	if len(g.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("Expected 0 edges, got %d", len(g.Edges))
	}

	node := g.Nodes[0]
	expectedID := fmt.Sprintf("acc_%d", acc.Id)
	if node.Id != expectedID {
		t.Errorf("Expected node ID %s, got %s", expectedID, node.Id)
	}
	if node.Location != acc.Address {
		t.Errorf("Expected location %s, got %s", acc.Address, node.Location)
	}
	if node.Stay != acc {
		t.Error("Expected valid Stay reference")
	}
}

func TestConstructGraph_Transport(t *testing.T) {
	now := time.Now()
	depTime := timestamppb.New(now)
	arrTime := timestamppb.New(now.Add(2 * time.Hour))

	flight := &pb.Flight{
		DepartureAirport: "JFK",
		ArrivalAirport:   "LHR",
		DepartureTime:    depTime,
		ArrivalTime:      arrTime,
	}

	transport := &pb.Transport{
		Id:   456,
		Type: pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		Details: &pb.Transport_Flight{
			Flight: flight,
		},
	}

	it := &pb.Itinerary{
		StartTime: depTime,
		Transport: []*pb.Transport{transport},
	}

	g := ConstructGraph(it)

	// Transport creates 2 nodes (src, dst) and 1 edge
	if len(g.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(g.Edges))
	}

	edge := g.Edges[0]
	srcID := fmt.Sprintf("transport_%d_dep", transport.Id)
	dstID := fmt.Sprintf("transport_%d_arr", transport.Id)

	if edge.FromId != srcID {
		t.Errorf("Expected edge FromId %s, got %s", srcID, edge.FromId)
	}
	if edge.ToId != dstID {
		t.Errorf("Expected edge ToId %s, got %s", dstID, edge.ToId)
	}
	if edge.DurationSeconds != 7200 { // 2 hours
		t.Errorf("Expected duration 7200, got %d", edge.DurationSeconds)
	}
}

func TestConstructGraph_Group(t *testing.T) {
	// Group containing one transport and one accommodation (via recursive itineraries)
	acc := &pb.Accommodation{
		Id:   100,
		Name: "Hotel A",
	}

	transport := &pb.Transport{
		Id: 200,
		Details: &pb.Transport_Train{
			Train: &pb.Train{
				DepartureStation: "Station A",
				ArrivalStation:   "Station B",
			},
		},
	}

	// Itinerary acting as a group with recursive itineraries
	itGroup := &pb.Itinerary{
		Itineraries: []*pb.Itinerary{
			{
				Accommodation: []*pb.Accommodation{acc},
			},
			{
				Transport: []*pb.Transport{transport},
			},
		},
	}

	g := ConstructGraph(itGroup)

	// Expect: 1 node (acc) + 2 nodes (transport) = 3 nodes
	// Expect: 1 edge (transport)
	if len(g.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}
}

func TestConstructGraph_Mixed(t *testing.T) {
	now := time.Now()
	acc := &pb.Accommodation{
		Id:       301,
		Name:     "Hotel Mix",
		Address:  "Addr Mix",
		CheckIn:  timestamppb.New(now),
		CheckOut: timestamppb.New(now.Add(24 * time.Hour)),
	}
	trans := &pb.Transport{
		Id: 302,
		Details: &pb.Transport_Flight{
			Flight: &pb.Flight{
				DepartureAirport: "A",
				ArrivalAirport:   "B",
				DepartureTime:    timestamppb.New(now),
				ArrivalTime:      timestamppb.New(now.Add(time.Hour)),
			},
		},
	}

	it := &pb.Itinerary{
		Itineraries: []*pb.Itinerary{
			{
				Accommodation: []*pb.Accommodation{acc},
				Transport:     []*pb.Transport{trans},
			},
		},
	}

	g := ConstructGraph(it)
	// 1 Acc Node + 2 Transport Nodes = 3 Nodes
	if len(g.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(g.Nodes))
	}
	// 1 Transport Edge
	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}
}

func TestConstructGraph_NestedSorted(t *testing.T) {
	t0 := time.Now()
	t1 := t0.Add(1 * time.Hour)
	t2 := t0.Add(2 * time.Hour)

	// Item 1: Accommodation at T0
	acc1 := &pb.Accommodation{
		Id:       401,
		Name:     "Early Hotel",
		CheckIn:  timestamppb.New(t0),
		CheckOut: timestamppb.New(t0.Add(24 * time.Hour)),
	}
	item1 := &pb.Itinerary{
		StartTime:     timestamppb.New(t0),
		Accommodation: []*pb.Accommodation{acc1},
	}

	// Item 2: Transport at T2 (Late)
	trans2 := &pb.Transport{
		Id: 402,
		Details: &pb.Transport_Train{
			Train: &pb.Train{
				DepartureStation: "X",
				ArrivalStation:   "Y",
				DepartureTime:    timestamppb.New(t2),
				ArrivalTime:      timestamppb.New(t2.Add(time.Hour)),
			},
		},
	}
	item2 := &pb.Itinerary{
		StartTime: timestamppb.New(t2),
		Transport: []*pb.Transport{trans2},
	}

	// Item 3: Deeply nested Accommodation at T1 (Middle)
	acc3 := &pb.Accommodation{
		Id:       403,
		Name:     "Middle Hotel",
		CheckIn:  timestamppb.New(t1),
		CheckOut: timestamppb.New(t1.Add(24 * time.Hour)),
	}
	// The child itself must have start time if sorting relies on it?
	// ConstructGraph sorts the *flattened* items.
	// flattenItinerary returns `items`.
	// If `item3` has only nested items, `item3` itself is NOT in the list.
	// The child inside `item3` IS in the list.
	// So we need to ensure the CHILD has the timestamp.
	item3Child := &pb.Itinerary{
		StartTime:     timestamppb.New(t1),
		Accommodation: []*pb.Accommodation{acc3},
	}
	item3 := &pb.Itinerary{
		Itineraries: []*pb.Itinerary{item3Child},
	}

	// Root contains them out of order: Item 2 (Late), Item 1 (Early), Item 3 (Middle)
	root := &pb.Itinerary{
		Itineraries: []*pb.Itinerary{item2, item1, item3},
	}

	g := ConstructGraph(root)

	// Nodes should be sorted by time: Acc1 (T0) -> Acc3 (T1) -> Trans2 (T2)
	// Note: Transport creates 2 nodes. Accommodation creates 1.

	if len(g.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes, got %d", len(g.Nodes))
	}

	// Check Order
	// Node 0 should be Acc1 (Early)
	if g.Nodes[0].Id != "acc_401" {
		t.Errorf("Expected first node to be acc_401, got %s", g.Nodes[0].Id)
	}
	// Node 1 should be Acc3 (Middle)
	if g.Nodes[1].Id != "acc_403" {
		t.Errorf("Expected second node to be acc_403, got %s", g.Nodes[1].Id)
	}
	// Node 2/3 are Transport (Late)
	if g.Nodes[2].Id != "transport_402_dep" {
		t.Errorf("Expected third node to be transport_402_dep, got %s", g.Nodes[2].Id)
	}
}
