package bootstrap

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/sirupsen/logrus"
	"github.com/va6996/travelingman/agents"
	zaiconfig "github.com/va6996/travelingman/bootstrap/zai"
	"github.com/va6996/travelingman/config"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/plugins/core"
	"github.com/va6996/travelingman/plugins/nager"
	"github.com/va6996/travelingman/plugins/tavily"
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
	// 0. Setup Logging
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
	log.Infof(ctx, "Log level set to: %s", level)

	// 1. Setup Genkit with AI Plugin
	var gk *genkit.Genkit
	var model ai.Model

	if cfg.AI.Plugin == "ollama" {
		log.Infof(ctx, "Using Ollama Plugin (Model: %s)...", cfg.AI.Ollama.Model)
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
	} else if cfg.AI.Plugin == "zai" {
		log.Infof(ctx, "Using Z.ai Plugin (Model: %s)...", cfg.AI.Zai.Model)
		if cfg.AI.Zai.APIKey == "" {
			return nil, fmt.Errorf("ZAI_API_KEY must be set (or set AI_PLUGIN=gemini or ollama)")
		}

		// Z.ai is OpenAI-compatible with base URL https://api.z.ai/api/paas/v4/
		zaiPlugin := &zaiconfig.Zai{
			APIKey:  cfg.AI.Zai.APIKey,
			BaseURL: "https://api.z.ai/api/paas/v4/",
		}
		gk = genkit.Init(ctx, genkit.WithPlugins(zaiPlugin))
		model = zaiPlugin.Model(gk, cfg.AI.Zai.Model)
	} else {
		log.Info(context.Background(), "Using Gemini Plugin...")
		if cfg.AI.Gemini.APIKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY must be set (or set AI_PLUGIN=ollama or zai)")
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

	// Tavily Search API (optional - if API key is provided)
	if cfg.Tavily.APIKey != "" {
		log.Info(context.Background(), "Initializing Tavily client...")
		tavily.NewClient(cfg.Tavily.APIKey, gk, registry)
	} else {
		log.Info(ctx, "Tavily API key not provided, Tavily tools will not be available")
	}

	// 3. Init New Agents
	log.Info(context.Background(), "Initializing New Agents...")
	tripPlanner := agents.NewTripPlanner(gk, registry, model)
	travelDesk := agents.NewTravelDesk(amadeusClient)
	travelAgent := agents.NewTravelAgent(tripPlanner, travelDesk)

	return &App{
		TravelAgent: travelAgent,
		Genkit:      gk,
		Registry:    registry,
		Model:       model,
	}, nil
}
