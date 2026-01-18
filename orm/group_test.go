package orm

import (
	"testing"

	"github.com/va6996/travelingman/pb"
	"github.com/stretchr/testify/assert"
)

func TestGroupCRUD(t *testing.T) {
	db := SetupTestDB(t)

	// Setup Organizer
	org := &pb.User{Email: "org@example.com", FullName: "Organizer"}
	CreateUser(db, org)

	// Create Group
	group := &pb.TravelGroup{
		Name:        "Test Group",
		OrganizerId: org.Id,
		Destination: "Berlin",
	}
	err := CreateTravelGroup(db, group)
	assert.NoError(t, err)
	assert.NotZero(t, group.GroupId)

	// Add Member
	member := &pb.User{Email: "member@example.com", FullName: "Member"}
	CreateUser(db, member)

	err = AddMember(db, uint(group.GroupId), uint(member.Id))
	assert.NoError(t, err)

	// Verify
	fetched, err := GetTravelGroup(db, uint(group.GroupId))
	assert.NoError(t, err)
	assert.Equal(t, "Test Group", fetched.Name)
	assert.Equal(t, org.Id, fetched.OrganizerId)

	assert.Len(t, fetched.Members, 1)
	assert.Equal(t, "Member", fetched.Members[0].FullName)
}
