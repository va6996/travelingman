// Copyright 2025
//
// Z.ai plugin for Firebase Genkit Go
// Provides integration with Z.ai's OpenAI-compatible API

package zai

import (
	"context"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

const provider = "zai"

// Zai is a plugin that provides integration with Z.ai's GLM models.
type Zai struct {
	// APIKey is the API key for the Z.ai API. If empty, the values of the environment variable "ZAI_API_KEY" will be consulted.
	// Request a key at https://docs.z.ai/api-reference/introduction
	APIKey string
	// BaseURL is the base URL for the Z.ai API. Defaults to https://api.z.ai/api/paas/v4/
	BaseURL string

	openAICompatible *compat_oai.OpenAICompatible
}

// Name implements genkit.Plugin.
func (z *Zai) Name() string {
	return provider
}

// Init implements genkit.Plugin.
func (z *Zai) Init(ctx context.Context) []api.Action {
	apiKey := z.APIKey
 baseURL := z.BaseURL

	// if api key is not set, get it from environment variable
	if apiKey == "" {
		apiKey = os.Getenv("ZAI_API_KEY")
	}

	if apiKey == "" {
		panic("zai plugin initialization failed: apiKey is required (set ZAI_API_KEY or pass APIKey)")
	}

	// Set default base URL if not provided
	if baseURL == "" {
		baseURL = "https://api.z.ai/api/paas/v4/"
	}

	if z.openAICompatible == nil {
		z.openAICompatible = &compat_oai.OpenAICompatible{}
	}

	// set the options
	z.openAICompatible.Opts = []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	z.openAICompatible.Provider = provider
	compatActions := z.openAICompatible.Init(ctx)

	var actions []api.Action
	actions = append(actions, compatActions...)

	// define default models
	supportedModels := map[string]ai.ModelOptions{
		"glm-4.7": {
			Label:    "Z.ai GLM-4.7",
			Supports: &compat_oai.Multimodal,
			Versions: []string{"glm-4.7"},
		},
		"glm-4.7-flash": {
			Label:    "Z.ai GLM-4.7 Flash",
			Supports: &compat_oai.Multimodal,
			Versions: []string{"glm-4.7-flash"},
		},
		"glm-4.6": {
			Label:    "Z.ai GLM-4.6",
			Supports: &compat_oai.Multimodal,
			Versions: []string{"glm-4.6"},
		},
		"glm-4.5": {
			Label:    "Z.ai GLM-4.5",
			Supports: &compat_oai.Multimodal,
			Versions: []string{"glm-4.5"},
		},
	}

	for model, opts := range supportedModels {
		actions = append(actions, z.DefineModel(model, opts).(api.Action))
	}

	return actions
}

// Model returns a model by name.
func (z *Zai) Model(g *genkit.Genkit, name string) ai.Model {
	return z.openAICompatible.Model(g, api.NewName(provider, name))
}

// DefineModel defines a model with the given ID and options.
func (z *Zai) DefineModel(id string, opts ai.ModelOptions) ai.Model {
	return z.openAICompatible.DefineModel(provider, id, opts)
}

// DefineEmbedder defines an embedder with the given ID and options.
func (z *Zai) DefineEmbedder(id string, opts *ai.EmbedderOptions) ai.Embedder {
	return z.openAICompatible.DefineEmbedder(provider, id, opts)
}

// Embedder returns an embedder by name.
func (z *Zai) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return z.openAICompatible.Embedder(g, api.NewName(provider, name))
}

// ListActions returns a list of actions provided by this plugin.
func (z *Zai) ListActions(ctx context.Context) []api.ActionDesc {
	return z.openAICompatible.ListActions(ctx)
}

// ResolveAction resolves an action by type and name.
func (z *Zai) ResolveAction(atype api.ActionType, name string) api.Action {
	return z.openAICompatible.ResolveAction(atype, name)
}

// Helper function to create a model with default options
func (z *Zai) DefineModelWithDefaults(id string) ai.Model {
	return z.DefineModel(id, ai.ModelOptions{
		Label:    "Z.ai " + id,
		Supports: &compat_oai.Multimodal,
	})
}
