package orm

import (
	"testing"
	"time"

	"example.com/travelingman/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUserCRUD(t *testing.T) {
	db := SetupTestDB(t)

	now := time.Now()
	user := &pb.User{
		Email:     "crud@example.com",
		FullName:  "CRUD User",
		CreatedAt: timestamppb.New(now),
	}

	// Create
	err := CreateUser(db, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.Id)

	// Read
	fetched, err := GetUser(db, uint(user.Id))
	assert.NoError(t, err)
	assert.Equal(t, "crud@example.com", fetched.Email)
	assert.Equal(t, "CRUD User", fetched.FullName)

	// Update
	// Note: UpdateUser helper exists
	user.FullName = "Updated Name"
	err = UpdateUser(db, user)
	assert.NoError(t, err)

	fetched2, err := GetUser(db, uint(user.Id))
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", fetched2.FullName)
}
