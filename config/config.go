package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config aggregates all application configuration
type Config struct {
	AI      AIConfig      `yaml:"ai"`
	Amadeus AmadeusConfig `yaml:"amadeus"`
}

type AIConfig struct {
	Plugin string       `yaml:"plugin" env:"AI_PLUGIN" env-default:"gemini"`
	Gemini GeminiConfig `yaml:"gemini"`
	Ollama OllamaConfig `yaml:"ollama"`
}

type GeminiConfig struct {
	APIKey string `yaml:"api_key" env:"GEMINI_API_KEY"`
	Model  string `yaml:"model" env:"GEMINI_MODEL" env-default:"gemini-1.5-flash"`
}

type OllamaConfig struct {
	Model   string `yaml:"model" env:"OLLAMA_MODEL" env-default:"qwen3:4b"`
	BaseURL string `yaml:"base_url" env:"OLLAMA_BASE_URL" env-default:"http://localhost:11434"`
}

type AmadeusConfig struct {
	ClientID     string `yaml:"client_id" env:"AMADEUS_CLIENT_ID"`
	ClientSecret string `yaml:"client_secret" env:"AMADEUS_CLIENT_SECRET"`
}

// Load reads configuration from config.yaml and environment variables
// Priority: Env Vars > Config File > Defaults
func Load() (*Config, error) {
	var cfg Config

	// 1. Try to load from config.yaml if it exists
	// We ignore error here because we fallback to env vars,
	// unless it's a specific parsing error which cleanenv handles well by just populating what it can.
	// But commonly one might want to enforce file existence if explicit.
	// Here we just say "read config.yaml if present, then override with envs".
	err := cleanenv.ReadConfig("config.yaml", &cfg)
	if err != nil {
		// If file doesn't exist, just read env vars
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to read env config: %w", err)
		}
	}

	return &cfg, nil
}
