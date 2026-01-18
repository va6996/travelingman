package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/va6996/travelingman/plugins"
)

// Client handles Ollama API requests
type Client struct {
	BaseURL string
	Model   string
	client  *http.Client
}

// Ensure Client satisfies LLMClient
var _ plugins.LLMClient = (*Client)(nil)

// NewClient creates a new Ollama API client
func NewClient(baseURL, model string) *Client {

	return &Client{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{},
	}
}

// GenerateRequest represents the payload for Ollama generate API
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// GenerateResponse represents the response from Ollama generate API
type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// GenerateContent sends a prompt to Ollama and returns the response
// This satisfies the plugins.LLMClient interface used by the agent
func (c *Client) GenerateContent(ctx context.Context, prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}
