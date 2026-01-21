package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/va6996/travelingman/agents"
	"github.com/va6996/travelingman/config"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/plugins/core"
	"github.com/va6996/travelingman/plugins/nager"
	"github.com/va6996/travelingman/tools"
)

// App holds the initialized components of the application
type App struct {
	TravelAgent *agents.TravelAgent
	Genkit      *genkit.Genkit
	Registry    *tools.Registry
	Model       ai.Model
}

// Setup initializes the application components based on the configuration
func Setup(ctx context.Context, cfg *config.Config) (*App, error) {
	// 1. Setup Genkit with AI Plugin
	var gk *genkit.Genkit
	var model ai.Model

	if cfg.AI.Plugin == "ollama" {
		log.Printf("Using Ollama Plugin (Model: %s)...", cfg.AI.Ollama.Model)
		ollamaPlugin := &ollama.Ollama{
			ServerAddress: cfg.AI.Ollama.BaseURL,
		}
		gk = genkit.Init(ctx, genkit.WithPlugins(ollamaPlugin))

		// Define the model with capabilities - explicitly enable tool support
		model = ollamaPlugin.DefineModel(gk, ollama.ModelDefinition{
			Name: cfg.AI.Ollama.Model,
			Type: "chat",
		}, &ai.ModelOptions{
			Supports: &ai.ModelSupports{
				Multiturn:  true,
				SystemRole: true,
				Tools:      true, // Enable tool support
				Media:      false,
			},
		})
	} else {
		log.Println("Using Gemini Plugin...")
		if cfg.AI.Gemini.APIKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY must be set (or set AI_PLUGIN=ollama)")
		}

		gk = genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: cfg.AI.Gemini.APIKey,
		}))
		model = googlegenai.GoogleAIModel(gk, cfg.AI.Gemini.Model)
	}

	// 2. Init Tools Registry
	registry := tools.NewRegistry()

	// Amadeus
	if cfg.Amadeus.ClientID == "" || cfg.Amadeus.ClientSecret == "" {
		return nil, fmt.Errorf("AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}

	// Initializing Amadeus client registers its tools automatically
	amadeusClient, err := amadeus.NewClient(cfg.Amadeus.ClientID, cfg.Amadeus.ClientSecret, false, gk, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Amadeus client: %w", err)
	}

	// Core Tools
	core.NewClient(gk, registry)

	// Nager Holiday API
	nager.NewClient(gk, registry)

	// 3. Init New Agents
	log.Println("Initializing New Agents...")
	tripPlanner := agents.NewTripPlannerV2(gk, registry, model)
	travelDesk := agents.NewTravelDesk(amadeusClient)
	travelAgent := agents.NewTravelAgent(tripPlanner, travelDesk)

	return &App{
		TravelAgent: travelAgent,
		Genkit:      gk,
		Registry:    registry,
		Model:       model,
	}, nil
}
