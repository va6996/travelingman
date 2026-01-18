package gemini

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("EmptyAPIKey", func(t *testing.T) {
		client, err := NewClient("")
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("ValidAPIKey", func(t *testing.T) {
		// Test with a valid-looking API key
		apiKey := "test-api-key-12345"
		client, err := NewClient(apiKey)
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, apiKey, client.APIKey)

		// Clean up
		client.Close()
	})
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient("test-api-key")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Close should not panic
	client.Close()

	// Double close should not panic
	client.Close()
}

func TestClient_GenerateContent_InvalidClient(t *testing.T) {
	client := &Client{
		APIKey: "test",
		client: nil, // Invalid client
		ctx:    context.Background(),
	}

	_, err := client.GenerateContent(context.Background(), "test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not initialized")
}
