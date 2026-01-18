package orm

import (
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
		Type:        pb.ItineraryType(i.Type),
		Title:       i.Title,
		Description: i.Description,
	}
	// Map Nested Itineraries
	for _, sub := range i.Itineraries {
		pbItin.Itineraries = append(pbItin.Itineraries, sub.ToPB())
	}
	// Map Transports
	for _, t := range i.Transports {
		pbItin.Transport = append(pbItin.Transport, t.ToPB())
	}
	// Map Accommodations
	for _, a := range i.Accommodations {
		pbItin.Accommodation = append(pbItin.Accommodation, a.ToPB())
	}
	return pbItin
}

func ItineraryFromPB(p *pb.Itinerary) *Itinerary {
	if p == nil {
		return nil
	}
	i := &Itinerary{
		ID:          uint(p.Id),
		GroupID:     uint(p.GroupId),
		DayNumber:   p.DayNumber,
		StartTime:   p.StartTime.AsTime(),
		EndTime:     p.EndTime.AsTime(),
		Type:        int32(p.Type),
		Title:       p.Title,
		Description: p.Description,
	}
	// Map Nested Itineraries
	for _, sub := range p.Itineraries {
		if subItin := ItineraryFromPB(sub); subItin != nil {
			i.Itineraries = append(i.Itineraries, *subItin)
		}
	}
	// Map Transports
	for _, t := range p.Transport {
		if tr := TransportFromPB(t); tr != nil {
			i.Transports = append(i.Transports, *tr)
		}
	}
	// Map Accommodations
	for _, a := range p.Accommodation {
		if ac := AccommodationFromPB(a); ac != nil {
			i.Accommodations = append(i.Accommodations, *ac)
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
