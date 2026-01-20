package core

import (
	"fmt"
	"sort"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type graphItem struct {
	startTime *timestamppb.Timestamp
	acc       *pb.Accommodation
	trans     *pb.Transport
}

// ConstructGraph converts an Itinerary (which may differ in structure) into a linear Graph representation
func ConstructGraph(itinerary *pb.Itinerary) *pb.Graph {
	g := NewGraph()

	if itinerary == nil {
		return g
	}

	// 1. Flatten the itinerary to get a linear list of items
	items := collectItems(itinerary)

	// 2. Sort items by start time to ensure logical flow
	sort.Slice(items, func(i, j int) bool {
		t1 := items[i].startTime
		t2 := items[j].startTime
		if t1 == nil || t2 == nil {
			return false
		}
		return t1.GetSeconds() < t2.GetSeconds()
	})

	// 3. Convert each item to nodes/edges
	for _, item := range items {
		if item.acc != nil {
			addAccommodationNode(g, item.acc)
		} else if item.trans != nil {
			addTransportEdge(g, item.trans)
		}
	}

	return g
}

func collectItems(it *pb.Itinerary) []graphItem {
	var items []graphItem

	if it == nil || it.Graph == nil {
		return items
	}

	// Add Accommodations from Nodes
	for _, node := range it.Graph.Nodes {
		if node.Stay != nil {
			items = append(items, graphItem{
				startTime: node.Stay.CheckIn, // Or node.FromTimestamp
				acc:       node.Stay,
			})
		}
	}

	// Add Transports from Edges
	for _, edge := range it.Graph.Edges {
		if edge.Transport != nil {
			trans := edge.Transport
			// Determine start time based on transport type (same logic as before)
			var startTime *timestamppb.Timestamp
			if trans.Details != nil {
				switch d := trans.Details.(type) {
				case *pb.Transport_Flight:
					if d.Flight != nil {
						startTime = d.Flight.DepartureTime
					}
				case *pb.Transport_Train:
					if d.Train != nil {
						startTime = d.Train.DepartureTime
					}
				case *pb.Transport_CarRental:
					if d.CarRental != nil {
						startTime = d.CarRental.PickupTime
					}
				}
			}
			items = append(items, graphItem{
				startTime: startTime,
				trans:     trans,
			})
		}
	}

	// Note: Nested Itineraries recursion is removed as Graph structure is flat or handled differently now.
	// If we need recursion, we assume the input Itinerary has already flattened its graph.

	return items
}

func addAccommodationNode(g *pb.Graph, acc *pb.Accommodation) {
	if acc == nil {
		return
	}

	node := &pb.Node{
		Id:            fmt.Sprintf("acc_%d", acc.Id),
		Location:      acc.Address,
		FromTimestamp: acc.CheckIn,
		ToTimestamp:   acc.CheckOut,
		Stay:          acc,
		IsInterCity:   false,
	}
	// Fallback if address is empty, use name
	if node.Location == "" {
		node.Location = acc.Name
	}

	AddNode(g, node)
}

func addTransportEdge(g *pb.Graph, t *pb.Transport) {
	if t == nil {
		return
	}

	var depLoc, arrLoc string
	var depTime, arrTime *timestamppb.Timestamp

	// Extract generic locations from transport
	if t.OriginLocation != nil {
		if len(t.OriginLocation.IataCodes) > 0 {
			depLoc = t.OriginLocation.IataCodes[0]
		} else {
			depLoc = t.OriginLocation.CityCode
		}
	}
	if t.DestinationLocation != nil {
		if len(t.DestinationLocation.IataCodes) > 0 {
			arrLoc = t.DestinationLocation.IataCodes[0]
		} else {
			arrLoc = t.DestinationLocation.CityCode
		}
	}

	switch d := t.Details.(type) {
	case *pb.Transport_Flight:
		if d.Flight != nil {
			depTime = d.Flight.DepartureTime
			arrTime = d.Flight.ArrivalTime
		}
	case *pb.Transport_Train:
		if d.Train != nil {
			depTime = d.Train.DepartureTime
			arrTime = d.Train.ArrivalTime
		}
	case *pb.Transport_CarRental:
		if d.CarRental != nil {
			depTime = d.CarRental.PickupTime
			arrTime = d.CarRental.DropoffTime
		}
	}

	// Create source node
	srcNodeID := fmt.Sprintf("transport_%d_dep", t.Id)
	srcNode := &pb.Node{
		Id:            srcNodeID,
		Location:      depLoc,
		FromTimestamp: depTime, // User arrives at departure point at departure time
		ToTimestamp:   depTime, // User departs departure point
		IsInterCity:   true,
	}
	AddNode(g, srcNode)

	// Create destination node
	destNodeID := fmt.Sprintf("transport_%d_arr", t.Id)
	destNode := &pb.Node{
		Id:            destNodeID,
		Location:      arrLoc,
		FromTimestamp: arrTime, // User arrives at destination
		ToTimestamp:   arrTime,
		IsInterCity:   true,
	}
	AddNode(g, destNode)

	// Create edge
	duration := int64(0)
	if depTime != nil && arrTime != nil {
		duration = arrTime.GetSeconds() - depTime.GetSeconds()
	}

	edge := &pb.Edge{
		FromId:          srcNodeID,
		ToId:            destNodeID,
		DurationSeconds: duration,
		Transport:       t,
	}
	AddEdge(g, edge)
}
