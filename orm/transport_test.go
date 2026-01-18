package orm

import (
	"testing"

	"github.com/va6996/travelingman/pb"
	"github.com/stretchr/testify/assert"
)

func TestTransport_Train(t *testing.T) {
	db := SetupTestDB(t)

	// Create Transport with Train details
	trans := &pb.Transport{
		Plugin: "Eurostar",
		Status: "Confirmed",
		Type:   pb.TransportType_TRANSPORT_TYPE_TRAIN,
		Details: &pb.Transport_Train{
			Train: &pb.Train{
				DepartureStation: "London St Pancras",
				ArrivalStation:   "Paris Gare du Nord",
				TrainNumber:      "9001",
			},
		},
	}

	err := CreateTransport(db, trans)
	assert.NoError(t, err)
	assert.NotZero(t, trans.Id)

	// Read back
	fetched, err := GetTransport(db, uint(trans.Id))
	assert.NoError(t, err)
	assert.Equal(t, "Eurostar", fetched.Plugin)
	assert.Equal(t, pb.TransportType_TRANSPORT_TYPE_TRAIN, fetched.Type)

	// Check Train Details
	assert.NotNil(t, fetched.GetTrain())
	assert.Equal(t, "9001", fetched.GetTrain().TrainNumber)
	assert.Equal(t, "London St Pancras", fetched.GetTrain().DepartureStation)
}

func TestTransport_CarRental(t *testing.T) {
	db := SetupTestDB(t)

	// Create Transport with CarRental details
	trans := &pb.Transport{
		Plugin: "Hertz",
		Status: "Reserved",
		Type:   pb.TransportType_TRANSPORT_TYPE_CAR,
		Details: &pb.Transport_CarRental{
			CarRental: &pb.CarRental{
				Company:        "Hertz",
				PickupLocation: "LAX",
				CarType:        "SUV",
				PriceTotal:     "150.00",
			},
		},
	}

	err := CreateTransport(db, trans)
	assert.NoError(t, err)

	// Read back
	fetched, err := GetTransport(db, uint(trans.Id))
	assert.NoError(t, err)
	assert.NotNil(t, fetched.GetCarRental())
	assert.Equal(t, "LAX", fetched.GetCarRental().PickupLocation)
	assert.Equal(t, "SUV", fetched.GetCarRental().CarType)
}
