package core

import (
	"context"
	"testing"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/tools"
)

func TestDateTool_Execute_Validation(t *testing.T) {
	// Setup
	registry := tools.NewRegistry()
	gk := genkit.Init(context.Background())

	dt := NewDateTool(gk, registry)
	dt.Now = func() time.Time {
		// Mock time: 2026-01-01
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name      string
		code      string
		expectErr bool
	}{
		{
			name:      "Valid Date Object",
			code:      "new Date('2026-01-02T00:00:00Z')",
			expectErr: false,
		},
		{
			name:      "Valid ISO String",
			code:      "'2026-01-02T00:00:00Z'",
			expectErr: false,
		},
		{
			name:      "Invalid Return Type (Number)",
			code:      "12345",
			expectErr: true,
		},
		{
			name:      "Null Return",
			code:      "null",
			expectErr: true,
		},
		{
			name:      "Undefined Return (no return)",
			code:      "var x = 1;",
			expectErr: true,
		},
		{
			name:      "LLM Generated Code",
			code:      "var d = new Date(now); d.setDate(d.getDate() + (12 - d.getDay()) % 7); if(d.getDay() !== 5 || d <= now) d.setDate(d.getDate() + 7); d",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &DateInput{Expression: tt.code}
			res, err := dt.Execute(context.Background(), input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Len(t, res, 1, "single date expressions should return a slice with 1 element")
			}
		})
	}

	// Test array return values
	t.Run("Array of Dates", func(t *testing.T) {
		input := &DateInput{Expression: "[new Date(now + 86400000), new Date(now + 172800000)]"}
		res, err := dt.Execute(context.Background(), input)
		assert.NoError(t, err)
		assert.Len(t, res, 2)

		expected1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		expected2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

		assert.WithinDuration(t, expected1, res[0], time.Minute)
		assert.WithinDuration(t, expected2, res[1], time.Minute)
	})

	t.Run("Array of ISO Strings", func(t *testing.T) {
		input := &DateInput{Expression: "['2026-01-02T00:00:00Z', '2026-01-03T00:00:00Z']"}
		res, err := dt.Execute(context.Background(), input)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
		expected1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		expected2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

		assert.Equal(t, expected1.UTC(), res[0].UTC())
		assert.Equal(t, expected2.UTC(), res[1].UTC())
	})

	t.Run("Empty Array", func(t *testing.T) {
		input := &DateInput{Expression: "[]"}
		res, err := dt.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Array with Invalid Element", func(t *testing.T) {
		input := &DateInput{Expression: "[new Date(now), 12345]"}
		res, err := dt.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Nested Array (should flatten)", func(t *testing.T) {
		input := &DateInput{Expression: "[[new Date(now + 86400000), new Date(now + 172800000)]]"}
		res, err := dt.Execute(context.Background(), input)
		assert.NoError(t, err)
		assert.Len(t, res, 2)

		expected1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		expected2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

		assert.WithinDuration(t, expected1, res[0], time.Minute)
		assert.WithinDuration(t, expected2, res[1], time.Minute)
	})
}
