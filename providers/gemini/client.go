package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Provider defines the interface for generative AI models
type Provider interface {
	GenerateContent(prompt string) (string, error)
}

// Client handles Gemini API requests using the official SDK
type Client struct {
	APIKey string
	client *genai.Client
	ctx    context.Context
}

// Ensure Client satisfies Provider
var _ Provider = (*Client)(nil)

// NewClient creates a new Gemini API client
// Returns an error if the client cannot be initialized
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		APIKey: apiKey,
		client: client,
		ctx:    ctx,
	}, nil
}

// GenerateContent sends a prompt to Gemini and returns the response
func (c *Client) GenerateContent(prompt string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("client not initialized")
	}

	// Use gemini-2.5-flash (fast) or gemini-2.5-pro (better quality)
	// Available models: gemini-2.5-flash, gemini-2.5-pro, gemini-pro-latest
	modelName := "gemini-pro-latest"
	model := c.client.GenerativeModel(modelName)

	// Generate content
	resp, err := model.GenerateContent(c.ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract text from response
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no content in candidate")
	}

	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%v", part)
	}

	return result, nil
}

// Close closes the Gemini client
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
