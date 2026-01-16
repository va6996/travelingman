package orm

import (
	"testing"
	"time"

	"example.com/travelingman/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAccommodationCRUD(t *testing.T) {
	db := SetupTestDB(t)

	// Create Group to link
	group := &pb.TravelGroup{Name: "Acc Group"}
	CreateTravelGroup(db, group)

	itinerary := &pb.Itinerary{Title: "Acc Itinerary", GroupId: group.GroupId}
	CreateItinerary(db, itinerary)

	acc := &pb.Accommodation{
		GroupId: group.GroupId,
		Name:    "Grand Hotel",
		Address: "123 Main St",
		Status:  "Booked",
		CheckIn: timestamppb.New(time.Now()),
	}

	// Link via itinerary list for creation or direct creation?
	// helper `CreateAccommodation` exists in `orm/accommodation.go`
	// but currently it doesn't take ItineraryID as argument in PB?
	// The ORM struct has ItineraryID. The PB struct doesn't have ItineraryID field.
	// `CreateAccommodation` helper converts PB to ORM.
	// PB has GroupID.
	// If we create it directly, it won't be linked to an Itinerary unless we set it manually or use nested creation.
	// Let's test direct creation linked to Group.

	err := CreateAccommodation(db, acc)
	assert.NoError(t, err)
	assert.NotZero(t, acc.Id)

	// Read back
	fetched, err := GetAccommodation(db, uint(acc.Id))
	assert.NoError(t, err)
	assert.Equal(t, "Grand Hotel", fetched.Name)
	assert.Equal(t, group.GroupId, fetched.GroupId)
}
