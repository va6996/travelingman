package orm

import (
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Accommodation struct {
	ID               uint `gorm:"primaryKey"`
	ItineraryID      uint // FK
	GroupID          uint // FK to TravelGroup
	Name             string
	Address          string
	CheckIn          time.Time
	CheckOut         time.Time
	CostValue        float64
	CostCurrency     string
	BookingReference string
	Status           string
}

func (a *Accommodation) ToPB() *pb.Accommodation {
	if a == nil {
		return nil
	}
	return &pb.Accommodation{
		Id:       int64(a.ID),
		GroupId:  int64(a.GroupID),
		Name:     a.Name,
		CheckIn:  timestamppb.New(a.CheckIn),
		CheckOut: timestamppb.New(a.CheckOut),
		Cost: &pb.Cost{
			Value:    a.CostValue,
			Currency: a.CostCurrency,
		},
		BookingReference: a.BookingReference,
		Status:           a.Status,
	}
}

func AccommodationFromPB(p *pb.Accommodation) *Accommodation {
	if p == nil {
		return nil
	}
	return &Accommodation{
		ID:               uint(p.Id),
		GroupID:          uint(p.GroupId),
		Name:             p.Name,
		CheckIn:          p.CheckIn.AsTime(),
		CheckOut:         p.CheckOut.AsTime(),
		CostValue:        p.GetCost().GetValue(),
		CostCurrency:     p.GetCost().GetCurrency(),
		BookingReference: p.BookingReference,
		Status:           p.Status,
	}
}

func CreateAccommodation(db *gorm.DB, pbAccom *pb.Accommodation) error {
	accommodation := AccommodationFromPB(pbAccom)
	if err := db.Create(accommodation).Error; err != nil {
		return err
	}
	pbAccom.Id = int64(accommodation.ID)
	return nil
}

func GetAccommodation(db *gorm.DB, id uint) (*pb.Accommodation, error) {
	var accommodation Accommodation
	err := db.First(&accommodation, id).Error
	if err != nil {
		return nil, err
	}
	return accommodation.ToPB(), nil
}
