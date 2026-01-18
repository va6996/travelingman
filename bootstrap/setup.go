package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/va6996/travelingman/config"
	"github.com/va6996/travelingman/plugins"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/plugins/core"
	"github.com/va6996/travelingman/plugins/gemini"
	"github.com/va6996/travelingman/plugins/ollama"
	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/genkit"
)

// App holds the initialized components of the application
type App struct {
	Agent    *tools.Agent
	Genkit   *genkit.Genkit
	Registry *tools.Registry
}

// Setup initializes the application components based on the configuration
func Setup(ctx context.Context, cfg *config.Config) (*App, error) {
	// 1. Setup AI Plugin
	var aiClient plugins.LLMClient
	var err error

	if cfg.AI.Plugin == "ollama" {
		log.Printf("Using Ollama Plugin (Model: %s)...", cfg.AI.Ollama.Model)
		aiClient = ollama.NewClient(cfg.AI.Ollama.BaseURL, cfg.AI.Ollama.Model)
	} else {
		log.Println("Using Gemini Plugin...")
		if cfg.AI.Gemini.APIKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY must be set (or set AI_PLUGIN=ollama)")
		}

		client, err := gemini.NewClient(cfg.AI.Gemini.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		// Note: We can't easily defer close here as we return the client.
		// The caller might need to handle cleanup if we exposed the client directly,
		// but currently the Agent wraps it and doesn't expose Close.
		// For a long-running service, reliance on GC or process exit is typical for this client type unless explicit lifecycle management is added.
		aiClient = client
	}

	// 2. Init Genkit & Tools
	gk := genkit.Init(ctx)
	registry := tools.NewRegistry()

	// Amadeus
	if cfg.Amadeus.ClientID == "" || cfg.Amadeus.ClientSecret == "" {
		return nil, fmt.Errorf("AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}

	// Initializing Amadeus client registers its tools automatically
	_, err = amadeus.NewClient(cfg.Amadeus.ClientID, cfg.Amadeus.ClientSecret, false, gk, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Amadeus client: %w", err)
	}

	// Core Tools
	core.NewClient(gk, registry)

	// 3. Init Agent
	log.Println("Initializing Agent...")
	agent, err := tools.NewAgent(gk, registry, aiClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Agent: %w", err)
	}

	return &App{
		Agent:    agent,
		Genkit:   gk,
		Registry: registry,
	}, nil
}
