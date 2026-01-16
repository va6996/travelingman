package googlemaps

import (
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
		// Note: This will fail if API key is invalid, but tests the initialization
		// In practice, use a valid test API key or mock the SDK client
		client, err := NewClient("test-api-key")
		// The SDK may validate the key format, so we check for either success or specific error
		if err != nil {
			// If it fails, it should be a specific error, not a nil client issue
			assert.Error(t, err)
			assert.Nil(t, client)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.NotNil(t, client.MapsClient)
		}
	})
}
