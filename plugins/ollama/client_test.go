package ollama

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		client := NewClient("", "")
		assert.NotNil(t, client)
		// NewClient doesn't set defaults, so empty strings remain
		assert.Equal(t, "", client.BaseURL)
		assert.Equal(t, "", client.Model)
		assert.NotNil(t, client.client)
	})

	t.Run("CustomValues", func(t *testing.T) {
		baseURL := "http://custom:8080"
		model := "custom-model"
		client := NewClient(baseURL, model)
		assert.NotNil(t, client)
		assert.Equal(t, baseURL, client.BaseURL)
		assert.Equal(t, model, client.Model)
		assert.NotNil(t, client.client)
	})
}

func TestGenerateRequest(t *testing.T) {
	req := GenerateRequest{
		Model:  "test-model",
		Prompt: "test prompt",
		Stream: false,
	}

	assert.Equal(t, "test-model", req.Model)
	assert.Equal(t, "test prompt", req.Prompt)
	assert.False(t, req.Stream)
}

func TestGenerateResponse(t *testing.T) {
	resp := GenerateResponse{
		Response: "test response",
		Done:     true,
	}

	assert.Equal(t, "test response", resp.Response)
	assert.True(t, resp.Done)
}

// Note: Full integration tests for GenerateContent would require a running Ollama server
// These are unit tests for the structure and basic functionality
func TestClient_GenerateContent_Context(t *testing.T) {
	client := NewClient("http://test", "test-model")

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GenerateContent(ctx, "test prompt")
	// Should fail due to cancelled context, but might also fail due to connection error
	// We just check that it doesn't panic and returns an error
	assert.Error(t, err)
}
