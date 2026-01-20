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
			}
		})
	}
}
