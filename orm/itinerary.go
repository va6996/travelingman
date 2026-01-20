package orm

import (
	"fmt"
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Itinerary struct {
	ID                uint  `gorm:"primaryKey"`
	GroupID           uint  // Linked to TravelGroup.ID
	ParentItineraryID *uint // Pointer to handle null root
	DayNumber         int32
	StartTime         time.Time
	EndTime           time.Time
	Type              int32 // Enum
	Title             string
	Description       string

	// Relationships
	Transports     []Transport     `gorm:"foreignKey:ItineraryID"`
	Accommodations []Accommodation `gorm:"foreignKey:ItineraryID"`
	Itineraries    []Itinerary     `gorm:"foreignKey:ParentItineraryID"`
}

func (i *Itinerary) ToPB() *pb.Itinerary {
	if i == nil {
		return nil
	}
	pbItin := &pb.Itinerary{
		Id:          int64(i.ID),
		GroupId:     int64(i.GroupID),
		DayNumber:   i.DayNumber,
		StartTime:   timestamppb.New(i.StartTime),
		EndTime:     timestamppb.New(i.EndTime),
		Title:       i.Title,
		Description: i.Description,
		Graph:       &pb.Graph{}, // Initialize Graph
	}

	// Map Accommodations to Nodes
	for idx, a := range i.Accommodations {
		pbAccommodation := a.ToPB()
		node := &pb.Node{
			Id:            fmt.Sprintf("node_acc_%d", a.ID), // Use ID if available or index
			Location:      pbAccommodation.Address,
			FromTimestamp: pbAccommodation.CheckIn,
			ToTimestamp:   pbAccommodation.CheckOut,
			Stay:          pbAccommodation,
			IsInterCity:   false,
		}
		// Fallback for ID if 0 (new)
		if a.ID == 0 {
			node.Id = fmt.Sprintf("node_acc_idx_%d", idx)
		}
		pbItin.Graph.Nodes = append(pbItin.Graph.Nodes, node)
	}

	// Map Transports to Edges
	// Note: Mapping flat transports to edges requires knowing From/To nodes.
	// If the DB schema doesn't store graph connectivity, we might just list them as disjoint edges or infer.
	// For now, let's create edges with placeholder IDs or try to link if logic allows.
	// Since ORM is flat, we map them as edges with empty connectivity or standalone.
	for _, t := range i.Transports {
		pbTransport := t.ToPB()
		edge := &pb.Edge{
			Transport: pbTransport,
			// FromId: ?, ToId: ? - Missing in flat DB model
		}

		// Attempt to guess duration
		// ... (omitted complexity for basic compile fix)

		pbItin.Graph.Edges = append(pbItin.Graph.Edges, edge)
	}

	// WARNING: Nested Itineraries `i.Itineraries` are not directly supported in the new flat Graph
	// unless we flatten them recursively. The new Proto seems to favor a single Graph.
	// If we must support them, we should recursively collect nodes/edges.
	for _, sub := range i.Itineraries {
		subPB := sub.ToPB()
		if subPB != nil && subPB.Graph != nil {
			pbItin.Graph.Nodes = append(pbItin.Graph.Nodes, subPB.Graph.Nodes...)
			pbItin.Graph.Edges = append(pbItin.Graph.Edges, subPB.Graph.Edges...)
		}
	}

	return pbItin
}

func ItineraryFromPB(p *pb.Itinerary) *Itinerary {
	if p == nil {
		return nil
	}
	i := &Itinerary{
		ID:        uint(p.Id),
		GroupID:   uint(p.GroupId),
		DayNumber: p.DayNumber,
		StartTime: p.StartTime.AsTime(),
		EndTime:   p.EndTime.AsTime(),
		// Type field removed from Proto
		Title:       p.Title,
		Description: p.Description,
	}

	if p.Graph != nil {
		// Map Nodes -> Accommodations
		for _, node := range p.Graph.Nodes {
			if node.Stay != nil {
				if ac := AccommodationFromPB(node.Stay); ac != nil {
					i.Accommodations = append(i.Accommodations, *ac)
				}
			}
		}
		// Map Edges -> Transports
		for _, edge := range p.Graph.Edges {
			if edge.Transport != nil {
				if tr := TransportFromPB(edge.Transport); tr != nil {
					i.Transports = append(i.Transports, *tr)
				}
			}
		}
	}

	return i
}

func CreateItinerary(db *gorm.DB, pbItin *pb.Itinerary) error {
	itinerary := ItineraryFromPB(pbItin)
	if err := db.Create(itinerary).Error; err != nil {
		return err
	}
	// Write back ID
	pbItin.Id = int64(itinerary.ID)
	return nil
}

func GetItinerary(db *gorm.DB, id uint) (*pb.Itinerary, error) {
	var itinerary Itinerary
	err := db.Preload("Transports").
		Preload("Transports.Flight").
		Preload("Transports.Train").
		Preload("Transports.CarRental").
		Preload("Accommodations").
		Preload("Itineraries"). // Recursion layer 1
		Preload("Itineraries.Transports").
		Preload("Itineraries.Transports.Flight").
		Preload("Itineraries.Transports.Train").
		Preload("Itineraries.Transports.CarRental").
		First(&itinerary, id).Error
	if err != nil {
		return nil, err
	}
	return itinerary.ToPB(), nil
}
