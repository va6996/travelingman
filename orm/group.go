package orm

import (
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type TravelGroup struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	OrganizerID int64
	Destination string
	TravelDate  time.Time
	// Relationships
	Members     []User      `gorm:"many2many:user_groups;"`
	Itineraries []Itinerary `gorm:"foreignKey:GroupID"`
}

func (g *TravelGroup) ToPB() *pb.TravelGroup {
	if g == nil {
		return nil
	}
	pbGroup := &pb.TravelGroup{
		GroupId:     int64(g.ID),
		Name:        g.Name,
		OrganizerId: g.OrganizerID,
		Destination: g.Destination,
		TravelDate:  timestamppb.New(g.TravelDate),
	}
	// Map Members
	for _, m := range g.Members {
		pbGroup.Members = append(pbGroup.Members, m.ToPB())
	}
	// Map Itineraries
	for _, i := range g.Itineraries {
		pbGroup.Itinerary = append(pbGroup.Itinerary, i.ToPB())
	}
	return pbGroup
}

func TravelGroupFromPB(p *pb.TravelGroup) *TravelGroup {
	if p == nil {
		return nil
	}
	g := &TravelGroup{
		ID:          uint(p.GroupId),
		Name:        p.Name,
		OrganizerID: p.OrganizerId,
		Destination: p.Destination,
		TravelDate:  p.TravelDate.AsTime(),
	}

	// Map Members
	for _, m := range p.Members {
		g.Members = append(g.Members, *UserFromPB(m))
	}
	// Map Itineraries
	for _, i := range p.Itinerary {
		// We need ItineraryFromPB here, defined in itinerary.go
		if it := ItineraryFromPB(i); it != nil {
			g.Itineraries = append(g.Itineraries, *it)
		}
	}
	return g
}

func CreateTravelGroup(db *gorm.DB, pbGroup *pb.TravelGroup) error {
	group := TravelGroupFromPB(pbGroup)
	if err := db.Create(group).Error; err != nil {
		return err
	}
	// Write back ID
	pbGroup.GroupId = int64(group.ID)
	return nil
}

func GetTravelGroup(db *gorm.DB, id uint) (*pb.TravelGroup, error) {
	var group TravelGroup
	// Preload all relationships
	err := db.Preload("Members").
		Preload("Itineraries").
		Preload("Itineraries.Transports").
		Preload("Itineraries.Transports.Flight").
		Preload("Itineraries.Itineraries").
		Preload("Itineraries.Itineraries.Transports").
		Preload("Itineraries.Itineraries.Transports.Flight").
		First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return group.ToPB(), nil
}

func AddMember(db *gorm.DB, groupID uint, userID uint) error {
	return db.Model(&TravelGroup{ID: groupID}).Association("Members").Append(&User{ID: userID})
}
