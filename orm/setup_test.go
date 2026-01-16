package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&Itinerary{}, &Transport{}, &Accommodation{}, &Flight{}, &Train{}, &CarRental{}, &User{}, &TravelGroup{})
	assert.NoError(t, err)

	return db
}
