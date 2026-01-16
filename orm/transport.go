package orm

import (
	"time"

	"example.com/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Transport struct {
	ID              uint `gorm:"primaryKey"`
	ItineraryID     uint // FK
	BookingID       int64
	Provider        string
	ReferenceNumber string
	Status          string
	Type            int32 // Enum

	// One-to-One relationships (Polymorphic-like via exclusive fields)
	Flight    *Flight    `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Train     *Train     `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CarRental *CarRental `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Flight struct {
	ID               uint `gorm:"primaryKey"`
	TransportID      uint // FK
	CarrierCode      string
	FlightNumber     string
	DepartureAirport string
	ArrivalAirport   string
	DepartureTime    time.Time
	ArrivalTime      time.Time
}

type Train struct {
	ID               uint `gorm:"primaryKey"`
	TransportID      uint // FK
	DepartureStation string
	ArrivalStation   string
	DepartureTime    time.Time
	ArrivalTime      time.Time
	TrainNumber      string
}

type CarRental struct {
	ID              uint `gorm:"primaryKey"`
	TransportID     uint // FK
	Company         string
	PickupLocation  string
	DropoffLocation string
	PickupTime      time.Time
	DropoffTime     time.Time
	CarType         string
	PriceTotal      string
}

func (t *Transport) ToPB() *pb.Transport {
	if t == nil {
		return nil
	}
	pbTrans := &pb.Transport{
		Id:              int64(t.ID),
		BookingId:       t.BookingID,
		Provider:        t.Provider,
		ReferenceNumber: t.ReferenceNumber,
		Status:          t.Status,
		Type:            pb.TransportType(t.Type),
	}
	if t.Flight != nil {
		pbTrans.Details = &pb.Transport_Flight{Flight: t.Flight.ToPB()}
	} else if t.Train != nil {
		pbTrans.Details = &pb.Transport_Train{Train: t.Train.ToPB()}
	} else if t.CarRental != nil {
		pbTrans.Details = &pb.Transport_CarRental{CarRental: t.CarRental.ToPB()}
	}
	return pbTrans
}

func TransportFromPB(p *pb.Transport) *Transport {
	if p == nil {
		return nil
	}
	t := &Transport{
		ID:              uint(p.Id),
		BookingID:       p.BookingId,
		Provider:        p.Provider,
		ReferenceNumber: p.ReferenceNumber,
		Status:          p.Status,
		Type:            int32(p.Type),
	}
	// Details OneOf
	if flight := p.GetFlight(); flight != nil {
		t.Flight = FlightFromPB(flight)
	} else if train := p.GetTrain(); train != nil {
		t.Train = TrainFromPB(train)
	} else if car := p.GetCarRental(); car != nil {
		t.CarRental = CarRentalFromPB(car)
	}
	return t
}

func (f *Flight) ToPB() *pb.Flight {
	if f == nil {
		return nil
	}
	return &pb.Flight{
		CarrierCode:      f.CarrierCode,
		FlightNumber:     f.FlightNumber,
		DepartureAirport: f.DepartureAirport,
		ArrivalAirport:   f.ArrivalAirport,
		DepartureTime:    timestamppb.New(f.DepartureTime),
		ArrivalTime:      timestamppb.New(f.ArrivalTime),
	}
}

func FlightFromPB(p *pb.Flight) *Flight {
	if p == nil {
		return nil
	}
	return &Flight{
		CarrierCode:      p.CarrierCode,
		FlightNumber:     p.FlightNumber,
		DepartureAirport: p.DepartureAirport,
		ArrivalAirport:   p.ArrivalAirport,
		DepartureTime:    p.DepartureTime.AsTime(),
		ArrivalTime:      p.ArrivalTime.AsTime(),
	}
}

func (t *Train) ToPB() *pb.Train {
	if t == nil {
		return nil
	}
	return &pb.Train{
		DepartureStation: t.DepartureStation,
		ArrivalStation:   t.ArrivalStation,
		DepartureTime:    timestamppb.New(t.DepartureTime),
		ArrivalTime:      timestamppb.New(t.ArrivalTime),
		TrainNumber:      t.TrainNumber,
	}
}

func TrainFromPB(p *pb.Train) *Train {
	if p == nil {
		return nil
	}
	return &Train{
		DepartureStation: p.DepartureStation,
		ArrivalStation:   p.ArrivalStation,
		DepartureTime:    p.DepartureTime.AsTime(),
		ArrivalTime:      p.ArrivalTime.AsTime(),
		TrainNumber:      p.TrainNumber,
	}
}

func (c *CarRental) ToPB() *pb.CarRental {
	if c == nil {
		return nil
	}
	return &pb.CarRental{
		Company:         c.Company,
		PickupLocation:  c.PickupLocation,
		DropoffLocation: c.DropoffLocation,
		PickupTime:      timestamppb.New(c.PickupTime),
		DropoffTime:     timestamppb.New(c.DropoffTime),
		CarType:         c.CarType,
		PriceTotal:      c.PriceTotal,
	}
}

func CarRentalFromPB(p *pb.CarRental) *CarRental {
	if p == nil {
		return nil
	}
	return &CarRental{
		Company:         p.Company,
		PickupLocation:  p.PickupLocation,
		DropoffLocation: p.DropoffLocation,
		PickupTime:      p.PickupTime.AsTime(),
		DropoffTime:     p.DropoffTime.AsTime(),
		CarType:         p.CarType,
		PriceTotal:      p.PriceTotal,
	}
}

func CreateTransport(db *gorm.DB, pbTrans *pb.Transport) error {
	transport := TransportFromPB(pbTrans)
	if err := db.Create(transport).Error; err != nil {
		return err
	}
	pbTrans.Id = int64(transport.ID)
	return nil
}

func GetTransport(db *gorm.DB, id uint) (*pb.Transport, error) {
	var transport Transport
	err := db.Preload("Flight").
		Preload("Train").
		Preload("CarRental").
		First(&transport, id).Error
	if err != nil {
		return nil, err
	}
	return transport.ToPB(), nil
}
