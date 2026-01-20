package agents

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/plugins/core"
	"github.com/va6996/travelingman/tools"
)

func TestTripPlannerV2_Plan(t *testing.T) {
	ctx := context.Background()

	// Initialize Genkit with Ollama plugin
	ollamaPlugin := &ollama.Ollama{
		ServerAddress: "http://localhost:11434",
	}
	gk := genkit.Init(ctx, genkit.WithPlugins(ollamaPlugin))

	// Define the model
	model := ollamaPlugin.DefineModel(gk, ollama.ModelDefinition{
		Name: "llama3.2:3b",
		Type: "chat",
	}, nil)

	// Create registry and register core tools
	registry := tools.NewRegistry()
	core.NewClient(gk, registry)

	// Create TripPlannerV2
	planner := NewTripPlannerV2(gk, registry, model)

	// Test simple query
	t.Run("Simple trip query", func(t *testing.T) {
		result, err := planner.Plan(ctx, PlanRequest{
			UserQuery: "Plan a weekend trip to Paris",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result.NeedsClarification {
			t.Logf("Planner asked: %s", result.Question)
		} else {
			t.Logf("Generated %d itineraries", len(result.PossibleItineraries))
			// Note: PossibleItineraries might be empty if conversion isn't implemented yet
		}
	})

	// Test query requiring date calculation
	t.Run("Query with date calculation", func(t *testing.T) {
		result, err := planner.Plan(ctx, PlanRequest{
			UserQuery: "Find hotels in NYC for next weekend",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result.NeedsClarification {
			t.Logf("Planner asked: %s", result.Question)
		} else {
			t.Logf("Result: %+v", result)
		}
	})
}
