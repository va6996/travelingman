package tools

import (
	"testing"
	"time"

	"example.com/travelingman/providers/amadeus"
	"github.com/stretchr/testify/assert"
)

func TestDateTool_JS(t *testing.T) {
	dt := &DateTool{}

	tests := []struct {
		name     string
		code     string
		expected time.Time
		isoCheck bool
	}{
		{
			"Fixed Date",
			"new Date('2025-01-05T00:00:00Z')",
			time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
			true,
		},
		{
			"Date Calculation",
			"new Date(new Date('2025-01-01T00:00:00Z').getTime() + 86400000)", // +1 day
			time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := dt.Execute(map[string]interface{}{"code": tt.code})
			assert.NoError(t, err)

			resTime, ok := res.(time.Time)
			assert.True(t, ok, "Expected time.Time result")

			if tt.isoCheck {
				// Compare strings to avoid timezone location pointer differences if any
				assert.Equal(t, tt.expected.UTC().Format(time.RFC3339), resTime.UTC().Format(time.RFC3339))
			} else {
				assert.Equal(t, tt.expected.Unix(), resTime.Unix())
			}
		})
	}
}

func TestDateTool_Invalid(t *testing.T) {
	dt := &DateTool{}
	_, err := dt.Execute(map[string]interface{}{"code": "invalid js"})
	assert.Error(t, err)
}

func TestFlightTool_Validation(t *testing.T) {
	ft := &FlightTool{Client: &amadeus.Client{}}

	_, err := ft.Execute(map[string]interface{}{})
	assert.Error(t, err)
}
