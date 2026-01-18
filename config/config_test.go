package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		// Save original env vars
		origAIPlugin := os.Getenv("AI_PLUGIN")
		origGeminiKey := os.Getenv("GEMINI_API_KEY")
		origAmadeusID := os.Getenv("AMADEUS_CLIENT_ID")
		origAmadeusSecret := os.Getenv("AMADEUS_CLIENT_SECRET")

		// Clear env vars for this test
		os.Unsetenv("AI_PLUGIN")
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("AMADEUS_CLIENT_ID")
		os.Unsetenv("AMADEUS_CLIENT_SECRET")

		defer func() {
			// Restore original env vars
			if origAIPlugin != "" {
				os.Setenv("AI_PLUGIN", origAIPlugin)
			}
			if origGeminiKey != "" {
				os.Setenv("GEMINI_API_KEY", origGeminiKey)
			}
			if origAmadeusID != "" {
				os.Setenv("AMADEUS_CLIENT_ID", origAmadeusID)
			}
			if origAmadeusSecret != "" {
				os.Setenv("AMADEUS_CLIENT_SECRET", origAmadeusSecret)
			}
		}()

		cfg, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Test default values
		assert.Equal(t, "gemini", cfg.AI.Plugin)
		assert.Equal(t, "qwen3:4b", cfg.AI.Ollama.Model)
		assert.Equal(t, "http://localhost:11434", cfg.AI.Ollama.BaseURL)
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		// Save original env vars
		origAIPlugin := os.Getenv("AI_PLUGIN")
		origGeminiKey := os.Getenv("GEMINI_API_KEY")

		// Set test env vars
		os.Setenv("AI_PLUGIN", "ollama")
		os.Setenv("GEMINI_API_KEY", "test-key")

		defer func() {
			// Restore original env vars
			if origAIPlugin != "" {
				os.Setenv("AI_PLUGIN", origAIPlugin)
			} else {
				os.Unsetenv("AI_PLUGIN")
			}
			if origGeminiKey != "" {
				os.Setenv("GEMINI_API_KEY", origGeminiKey)
			} else {
				os.Unsetenv("GEMINI_API_KEY")
			}
		}()

		cfg, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "ollama", cfg.AI.Plugin)
		assert.Equal(t, "test-key", cfg.AI.Gemini.APIKey)
	})
}
