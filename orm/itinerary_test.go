package orm

import (
	"testing"

	"example.com/travelingman/pb"
	"github.com/stretchr/testify/assert"
)

func TestItineraryHierarchy(t *testing.T) {
	db := SetupTestDB(t)

	// Create User and Group
	user := pb.User{
		Email:    "vinayak@example.com",
		FullName: "Vinayak",
	}
	// Use helper
	err := CreateUser(db, &user)
	assert.NoError(t, err)

	group := pb.TravelGroup{
		Name:        "EuroTrip 2025",
		OrganizerId: 1, // dummy
	}
	err = CreateTravelGroup(db, &group)
	assert.NoError(t, err)

	err = AddMember(db, uint(group.GroupId), uint(user.Id))
	assert.NoError(t, err)

	// Create Parent Itinerary linked to Group
	parent := pb.Itinerary{
		GroupId: group.GroupId,
		Title:   "Europe Trip",
		Type:    pb.ItineraryType_ITINERARY_TYPE_TRANSPORT, // Just picking a type
	}
	err = CreateItinerary(db, &parent)
	assert.NoError(t, err)

	// Create Child Itinerary
	// Note: In PB, we don't have direct ParentID field for recursive insert in CreateItinerary unless we use the nested structure.
	// But our GORM model has ParentItineraryID.
	// The CreateItinerary(pb) converts PB to ORM. ORM has ParentItineraryID.
	// PB Itinerary doesn't have ParentItineraryID field explicitly in the definition I saw?
	// Wait, models/itinerary.proto doesn't have parent_id!
	// It relies on nesting in `itineraries` repeated field.
	// So to create a child, I should add it to the parent's list and update the parent?
	// OR, I can Create just the child, but how do I link it?
	// If I cannot link it via ID, I must use the nesting feature of Create.

	// Let's create a NEW parent PB that contains the child, and Create that?
	// But `CreateItinerary` inserts a single root.

	// Actually, for the test, let's create the child and verify it's linked if I add it to parent's list?
	// But `CreateItinerary` takes `*pb.Itinerary`.
	// If I pass a parent with children, `ItineraryFromPB` converts it to ORM `Itinerary` with populated `Itineraries` slice.
	// GORM `Create` on the parent will create the children too.

	// So, let's create Parent AND Child in one go.

	complexItin := pb.Itinerary{
		GroupId: group.GroupId,
		Title:   "Europe Trip Complex",
		Type:    1,
		Itineraries: []*pb.Itinerary{
			{
				Title: "Day 1 in Paris",
				Type:  2,
				Transport: []*pb.Transport{
					{
						Provider: "AirFrance",
						Status:   "Confirmed",
						Details: &pb.Transport_Flight{
							Flight: &pb.Flight{
								CarrierCode:  "AF",
								FlightNumber: "123",
							},
						},
					},
				},
			},
		},
	}
	err = CreateItinerary(db, &complexItin)
	assert.NoError(t, err)

	// Complex Parent ID
	parentID := uint(complexItin.Id)

	// Verify Hierarchy using Get helper
	fetchedGroup, err := GetTravelGroup(db, uint(group.GroupId))
	assert.NoError(t, err)

	// fetchedGroup have itineraries?
	// We created "parent" (first one) and "complexItin". Both linked to group.
	assert.GreaterOrEqual(t, len(fetchedGroup.Itinerary), 2)

	// Verify members
	assert.Len(t, fetchedGroup.Members, 1)
	assert.Equal(t, "Vinayak", fetchedGroup.Members[0].FullName)

	// Verify Complex Itinerary
	fetchedParent, err := GetItinerary(db, parentID)
	assert.NoError(t, err)

	assert.Equal(t, group.GroupId, fetchedParent.GroupId)
	assert.Len(t, fetchedParent.Itineraries, 1)
	assert.Equal(t, "Day 1 in Paris", fetchedParent.Itineraries[0].Title)
	assert.Len(t, fetchedParent.Itineraries[0].Transport, 1)
	assert.Equal(t, "AirFrance", fetchedParent.Itineraries[0].Transport[0].Provider)
	assert.NotNil(t, fetchedParent.Itineraries[0].Transport[0].GetFlight())
	assert.Equal(t, "AF", fetchedParent.Itineraries[0].Transport[0].GetFlight().CarrierCode)
}
