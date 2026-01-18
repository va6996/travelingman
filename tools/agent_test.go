package tools_test

import (
	"context"
	"testing"
	"time"

	"github.com/va6996/travelingman/plugins/gemini"
	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
)

func TestNewAgent(t *testing.T) {
	geminiClient := &gemini.Client{}

	t.Run("ValidClients", func(t *testing.T) {
		gk := genkit.Init(context.Background())
		registry := tools.NewRegistry()
		agent, err := tools.NewAgent(gk, registry, geminiClient)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
		// TripData is initially nil
		assert.Nil(t, agent.GetTripData())
	})

	t.Run("NilGeminiClient", func(t *testing.T) {
		gk := genkit.Init(context.Background())
		registry := tools.NewRegistry()
		// Register dummy?
		agent, err := tools.NewAgent(gk, registry, nil)
		// NewAgent stores it? yes.
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})
}

func TestAgent_GetTripData(t *testing.T) {
	geminiClient := &gemini.Client{}
	gk := genkit.Init(context.Background())
	registry := tools.NewRegistry()

	agent, err := tools.NewAgent(gk, registry, geminiClient)
	assert.NoError(t, err)

	// Initially should be nil
	tripData := agent.GetTripData()
	assert.Nil(t, tripData)
}

func TestAgent_PlanTrip_ContextCancellation(t *testing.T) {
	geminiClient := &gemini.Client{}
	gk := genkit.Init(context.Background())
	registry := tools.NewRegistry()

	agent, err := tools.NewAgent(gk, registry, geminiClient)
	assert.NoError(t, err)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = agent.PlanTrip(ctx, "test query")
	// Should return error due to context cancellation
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestTripData(t *testing.T) {
	now := time.Now()
	tripData := &tools.TripData{
		Query:     "test query",
		CreatedAt: now,
	}

	assert.Equal(t, "test query", tripData.Query)
	assert.Equal(t, now, tripData.CreatedAt)
}
