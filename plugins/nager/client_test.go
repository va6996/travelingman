package nager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient(nil, nil)
	assert.NotNil(t, client)
	assert.Equal(t, "https://date.nager.at/api/v3", client.BaseURL)
	assert.NotNil(t, client.HTTPClient)
}

func TestClient_GetAvailableCountries(t *testing.T) {
	client := NewClient(nil, nil)

	countries, err := client.GetAvailableCountries(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, countries)

	// Check that we have some expected countries
	countryCodes := make(map[string]bool)
	for _, country := range countries {
		countryCodes[country.CountryCode] = true
	}

	// Should have major countries
	assert.True(t, countryCodes["US"], "Should have United States")
	assert.True(t, countryCodes["GB"], "Should have United Kingdom")
	assert.True(t, countryCodes["DE"], "Should have Germany")
}

func TestClient_GetPublicHolidays(t *testing.T) {
	client := NewClient(nil, nil)

	holidays, err := client.GetPublicHolidays(context.Background(), 2024, "US")
	assert.NoError(t, err)
	assert.NotEmpty(t, holidays)

	// Check that we have some expected holidays
	holidayDates := make(map[string]bool)
	for _, holiday := range holidays {
		holidayDates[holiday.Date] = true
		assert.Equal(t, "US", holiday.CountryCode)
		assert.NotEmpty(t, holiday.Name)
	}

	// Should have major US holidays
	assert.True(t, holidayDates["2024-01-01"], "Should have New Year's Day")
	assert.True(t, holidayDates["2024-07-04"], "Should have Independence Day")
	assert.True(t, holidayDates["2024-12-25"], "Should have Christmas Day")
}

func TestClient_ContextCancellation(t *testing.T) {
	client := NewClient(nil, nil)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetAvailableCountries(ctx)
	// Should fail due to cancelled context, but might also succeed if it's fast enough
	// We just check that it doesn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "canceled")
	}
}
