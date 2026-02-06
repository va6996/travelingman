package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config aggregates all application configuration
type Config struct {
	AI      AIConfig       `yaml:"ai"`
	Planner PlannerConfig  `yaml:"planner"`
	Amadeus AmadeusConfig  `yaml:"amadeus"`
	Tavily  TavilyConfig   `yaml:"tavily"`
	Log     LogConfig      `yaml:"log"`
	DB      DatabaseConfig `yaml:"database"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

type AIConfig struct {
	Plugin string       `yaml:"plugin" env:"AI_PLUGIN" env-default:"gemini"`
	Gemini GeminiConfig `yaml:"gemini"`
	Ollama OllamaConfig `yaml:"ollama"`
	Zai    ZaiConfig    `yaml:"zai"`
}

type GeminiConfig struct {
	APIKey string `yaml:"api_key" env:"GEMINI_API_KEY"`
	Model  string `yaml:"model" env:"GEMINI_MODEL" env-default:"gemini-1.5-flash"`
}

type OllamaConfig struct {
	Model   string `yaml:"model" env:"OLLAMA_MODEL" env-default:"qwen3:4b"`
	BaseURL string `yaml:"base_url" env:"OLLAMA_BASE_URL" env-default:"http://localhost:11434"`
}

type ZaiConfig struct {
	APIKey string `yaml:"api_key" env:"ZAI_API_KEY"`
	Model  string `yaml:"model" env:"ZAI_MODEL" env-default:"glm-4.7"`
}

type AmadeusConfig struct {
	ClientID     string `yaml:"client_id" env:"AMADEUS_CLIENT_ID"`
	ClientSecret string `yaml:"client_secret" env:"AMADEUS_CLIENT_SECRET"`
	Environment  string `yaml:"environment" env:"AMADEUS_ENV" env-default:"test"`
	Limit        struct {
		Flight int `yaml:"flight" env:"AMADEUS_LIMIT_FLIGHT" env-default:"10"`
		Hotel  int `yaml:"hotel" env:"AMADEUS_LIMIT_HOTEL" env-default:"10"`
	} `yaml:"limit"`
	Timeout  int `yaml:"timeout" env:"AMADEUS_TIMEOUT" env-default:"30"` // Seconds
	CacheTTL struct {
		Location int `yaml:"location" env:"AMADEUS_CACHE_TTL_LOCATION" env-default:"24"` // Hours
		Flight   int `yaml:"flight" env:"AMADEUS_CACHE_TTL_FLIGHT" env-default:"1"`      // Hours
		Hotel    int `yaml:"hotel" env:"AMADEUS_CACHE_TTL_HOTEL" env-default:"1"`        // Hours
	} `yaml:"cache_ttl"`
}

type TavilyConfig struct {
	APIKey  string `yaml:"api_key" env:"TAVILY_API_KEY"`
	Timeout int    `yaml:"timeout" env:"TAVILY_TIMEOUT" env-default:"30"` // Seconds
}

type PlannerConfig struct {
	Timeout int `yaml:"timeout" env:"PLANNER_TIMEOUT" env-default:"220"` // Seconds
}

type DatabaseConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"travelingman"`
	SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
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
