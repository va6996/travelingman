package orm

import (
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Transport struct {
	ID              uint `gorm:"primaryKey"`
	ItineraryID     uint // FK
	BookingID       int64
	Plugin          string
	ReferenceNumber string
	Status          string
	Type            int32 // Enum

	// Preferences (stored as JSON or separate columns, simplifed here as embedded for GORM references if needed, or just fields)
	// For this refactor, we are adding them as pointer references similar to details, or we could store as JSON.
	// Let's assume we map them to the PB directly for now in memory.
	// To actually store them in DB, we'd need new tables or JSON columns.
	// Given the scope, I will update the ToPB/FromPB mapping.
	// But first, I need update the struct. I will add them as ignored fields (-) if not adhering to DB schema yet, or just assume DB migration happens elsewhere.
	// Let's stick to update logic mapping first.
	// Wait, if I don't add them to the struct, I can't map them.

	FlightPreferences    *FlightPreferences    `gorm:"embedded;embeddedPrefix:flight_pref_"`
	TrainPreferences     *TrainPreferences     `gorm:"embedded;embeddedPrefix:train_pref_"`
	CarRentalPreferences *CarRentalPreferences `gorm:"embedded;embeddedPrefix:car_pref_"`

	// One-to-One relationships (Polymorphic-like via exclusive fields)
	Flight    *Flight    `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Train     *Train     `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CarRental *CarRental `gorm:"foreignKey:TransportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type FlightPreferences struct {
	TravelClass                  int32
	MaxStops                     int32
	PreferredOriginAirports      []string `gorm:"type:text"` // Check GORM string array support or use serializer
	PreferredDestinationAirports []string `gorm:"type:text"`
}

type TrainPreferences struct {
	TravelClass int32
	SeatType    string
}

type CarRentalPreferences struct {
	Transmission int32
	CarClass     string
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
		Plugin:          t.Plugin,
		ReferenceNumber: t.ReferenceNumber,
		Status:          t.Status,
		Type:            pb.TransportType(t.Type),
	}
	if t.Flight != nil {
		pbTrans.Details = &pb.Transport_Flight{Flight: t.Flight.ToPB()}
		pbTrans.OriginLocation = &pb.Location{IataCodes: []string{t.Flight.DepartureAirport}}
		pbTrans.DestinationLocation = &pb.Location{IataCodes: []string{t.Flight.ArrivalAirport}}
	} else if t.Train != nil {
		pbTrans.Details = &pb.Transport_Train{Train: t.Train.ToPB()}
		pbTrans.OriginLocation = &pb.Location{IataCodes: []string{t.Train.DepartureStation}}
		pbTrans.DestinationLocation = &pb.Location{IataCodes: []string{t.Train.ArrivalStation}}
	} else if t.CarRental != nil {
		pbTrans.Details = &pb.Transport_CarRental{CarRental: t.CarRental.ToPB()}
		pbTrans.OriginLocation = &pb.Location{IataCodes: []string{t.CarRental.PickupLocation}}
		pbTrans.DestinationLocation = &pb.Location{IataCodes: []string{t.CarRental.DropoffLocation}}
	}
	if t.FlightPreferences != nil {
		pbTrans.FlightPreferences = &pb.FlightPreferences{
			TravelClass:                  pb.Class(t.FlightPreferences.TravelClass),
			MaxStops:                     t.FlightPreferences.MaxStops,
			PreferredOriginAirports:      t.FlightPreferences.PreferredOriginAirports,
			PreferredDestinationAirports: t.FlightPreferences.PreferredDestinationAirports,
		}
	}
	if t.TrainPreferences != nil {
		pbTrans.TrainPreferences = &pb.TrainPreferences{
			TravelClass: pb.Class(t.TrainPreferences.TravelClass),
			SeatType:    t.TrainPreferences.SeatType,
		}
	}
	if t.CarRentalPreferences != nil {
		pbTrans.CarRentalPreferences = &pb.CarRentalPreferences{
			Transmission: pb.Transmission(t.CarRentalPreferences.Transmission),
			CarClass:     t.CarRentalPreferences.CarClass,
		}
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
		Plugin:          p.Plugin,
		ReferenceNumber: p.ReferenceNumber,
		Status:          p.Status,
		Type:            int32(p.Type),
	}

	if p.FlightPreferences != nil {
		t.FlightPreferences = &FlightPreferences{
			TravelClass:                  int32(p.FlightPreferences.TravelClass),
			MaxStops:                     p.FlightPreferences.MaxStops,
			PreferredOriginAirports:      p.FlightPreferences.PreferredOriginAirports,
			PreferredDestinationAirports: p.FlightPreferences.PreferredDestinationAirports,
		}
	}
	if p.TrainPreferences != nil {
		t.TrainPreferences = &TrainPreferences{
			TravelClass: int32(p.TrainPreferences.TravelClass),
			SeatType:    p.TrainPreferences.SeatType,
		}
	}
	if p.CarRentalPreferences != nil {
		t.CarRentalPreferences = &CarRentalPreferences{
			Transmission: int32(p.CarRentalPreferences.Transmission),
			CarClass:     p.CarRentalPreferences.CarClass,
		}
	}

	origin := ""
	if p.OriginLocation != nil {
		if len(p.OriginLocation.IataCodes) > 0 {
			origin = p.OriginLocation.IataCodes[0]
		} else {
			origin = p.OriginLocation.CityCode
		}
	}
	dest := ""
	if p.DestinationLocation != nil {
		if len(p.DestinationLocation.IataCodes) > 0 {
			dest = p.DestinationLocation.IataCodes[0]
		} else {
			dest = p.DestinationLocation.CityCode
		}
	}

	// Details OneOf
	if flight := p.GetFlight(); flight != nil {
		t.Flight = FlightFromPB(flight)
		t.Flight.DepartureAirport = origin
		t.Flight.ArrivalAirport = dest
	} else if train := p.GetTrain(); train != nil {
		t.Train = TrainFromPB(train)
		t.Train.DepartureStation = origin
		t.Train.ArrivalStation = dest
	} else if car := p.GetCarRental(); car != nil {
		t.CarRental = CarRentalFromPB(car)
		t.CarRental.PickupLocation = origin
		t.CarRental.DropoffLocation = dest
	}
	return t
}

func (f *Flight) ToPB() *pb.Flight {
	if f == nil {
		return nil
	}
	return &pb.Flight{
		CarrierCode:   f.CarrierCode,
		FlightNumber:  f.FlightNumber,
		DepartureTime: timestamppb.New(f.DepartureTime),
		ArrivalTime:   timestamppb.New(f.ArrivalTime),
	}
}

func FlightFromPB(p *pb.Flight) *Flight {
	if p == nil {
		return nil
	}
	return &Flight{
		CarrierCode:   p.CarrierCode,
		FlightNumber:  p.FlightNumber,
		DepartureTime: p.DepartureTime.AsTime(),
		ArrivalTime:   p.ArrivalTime.AsTime(),
	}
}

func (t *Train) ToPB() *pb.Train {
	if t == nil {
		return nil
	}
	return &pb.Train{
		DepartureTime: timestamppb.New(t.DepartureTime),
		ArrivalTime:   timestamppb.New(t.ArrivalTime),
		TrainNumber:   t.TrainNumber,
	}
}

func TrainFromPB(p *pb.Train) *Train {
	if p == nil {
		return nil
	}
	return &Train{
		DepartureTime: p.DepartureTime.AsTime(),
		ArrivalTime:   p.ArrivalTime.AsTime(),
		TrainNumber:   p.TrainNumber,
	}
}

func (c *CarRental) ToPB() *pb.CarRental {
	if c == nil {
		return nil
	}
	return &pb.CarRental{
		Company:     c.Company,
		PickupTime:  timestamppb.New(c.PickupTime),
		DropoffTime: timestamppb.New(c.DropoffTime),
		CarType:     c.CarType,
	}
}

func CarRentalFromPB(p *pb.CarRental) *CarRental {
	if p == nil {
		return nil
	}
	return &CarRental{
		Company:     p.Company,
		PickupTime:  p.PickupTime.AsTime(),
		DropoffTime: p.DropoffTime.AsTime(),
		CarType:     p.CarType,
		// PriceTotal not in pb.CarRental anymore, handled at Transport level if needed
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
