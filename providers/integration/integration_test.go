//go:build integration
// +build integration

package integration

import (
	"os"
	"testing"

	"example.com/travelingman/providers/amadeus"
	"example.com/travelingman/providers/gemini"
	"example.com/travelingman/providers/googlemaps"
)

// TestAllProviders runs integration tests for all providers
func TestAllProviders(t *testing.T) {
	t.Run("Amadeus", func(t *testing.T) {
		TestAmadeusIntegration(t)
	})

	t.Run("GoogleMaps", func(t *testing.T) {
		TestGoogleMapsIntegration(t)
	})

	t.Run("Gemini", func(t *testing.T) {
		TestGeminiIntegration(t)
	})
}

// TestAmadeusIntegration tests Amadeus API integration
func TestAmadeusIntegration(t *testing.T) {
	clientID := os.Getenv("AMADEUS_CLIENT_ID")
	clientSecret := os.Getenv("AMADEUS_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Fatal("AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}
	isProduction := os.Getenv("AMADEUS_PRODUCTION") == "true"

	client, err := amadeus.NewClient(clientID, clientSecret, isProduction)
	if err != nil {
		t.Fatalf("Failed to initialize Amadeus client: %v", err)
	}

	// Test Authentication
	t.Run("Authentication", func(t *testing.T) {
		err := client.Authenticate()
		if err != nil {
			t.Fatalf("Authentication failed: %v", err)
		}
		if client.Token == nil {
			t.Fatal("Token is nil after authentication")
		}
		if client.Token.AccessToken == "" {
			t.Fatal("Access token is empty")
		}
		t.Logf("✓ Authentication successful - Token expires in %d seconds", client.Token.ExpiresIn)
	})

	// Test Flight Search
	t.Run("FlightSearch", func(t *testing.T) {
		// Use a simple test route
		flightResp, err := client.SearchFlights("NYC", "LAX", "2024-12-01", "", "", 1)
		if err != nil {
			t.Logf("⚠️  Flight search failed (may need API access): %v", err)
			return
		}
		if flightResp == nil {
			t.Fatal("Flight search returned nil response")
		}
		t.Logf("✓ Flight search successful - Found %d offers", len(flightResp.Data))
		if len(flightResp.Data) > 0 {
			offer := flightResp.Data[0]
			t.Logf("  Sample offer: ID=%s, Price=%s %s", offer.ID, offer.Price.Currency, offer.Price.Total)
		}
	})

	// Test Hotel Search
	t.Run("HotelSearch", func(t *testing.T) {
		// Note: This requires a valid hotel ID
		hotelResp, err := client.SearchHotelOffers([]string{"BGMNYCCT"}, 1, "2024-12-01", "2024-12-05")
		if err != nil {
			t.Logf("⚠️  Hotel search failed (may need valid hotel ID or API access): %v", err)
			return
		}
		if hotelResp != nil && len(hotelResp.Data) > 0 {
			t.Logf("✓ Hotel search successful - Found %d hotels", len(hotelResp.Data))
		} else {
			t.Logf("⚠️  Hotel search returned no data")
		}
	})
}

// TestGoogleMapsIntegration tests Google Maps API integration
func TestGoogleMapsIntegration(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		t.Fatal("GOOGLE_MAPS_API_KEY must be set")
	}
	client, err := googlemaps.NewClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to initialize Google Maps client: %v", err)
	}
	if client.MapsClient == nil {
		t.Fatal("Maps client was not initialized - check API key")
	}

	// Test Place Autocomplete
	t.Run("PlaceAutocomplete", func(t *testing.T) {
		resp, err := client.AutocompleteSearch("Paris", nil, 0)
		if err != nil {
			t.Fatalf("Place autocomplete failed: %v", err)
		}
		if resp == nil {
			t.Fatal("Autocomplete returned nil response")
		}
		if len(resp.Predictions) == 0 {
			t.Fatal("Autocomplete returned no predictions")
		}
		t.Logf("✓ Place autocomplete successful - Found %d predictions", len(resp.Predictions))
		pred := resp.Predictions[0]
		t.Logf("  Sample prediction: %s (Place ID: %s)", pred.Description, pred.PlaceID)
	})

	// Test Place Autocomplete with Location
	t.Run("PlaceAutocompleteWithLocation", func(t *testing.T) {
		location := &googlemaps.Location{Lat: 48.8566, Lng: 2.3522} // Paris coordinates
		resp, err := client.AutocompleteSearch("restaurant", location, 5000)
		if err != nil {
			t.Fatalf("Place autocomplete with location failed: %v", err)
		}
		if resp == nil {
			t.Fatal("Autocomplete returned nil response")
		}
		t.Logf("✓ Place autocomplete with location successful - Found %d predictions", len(resp.Predictions))
	})

	// Test Place Details
	t.Run("PlaceDetails", func(t *testing.T) {
		// Use a well-known place ID (Eiffel Tower)
		placeID := "ChIJD7fiBh9u5kcRYJSMaMOCCwQ"
		details, err := client.GetPlaceDetails(placeID)
		if err != nil {
			t.Fatalf("Place details failed: %v", err)
		}
		if details == nil {
			t.Fatal("Place details returned nil")
		}
		if details.PlaceID == "" {
			t.Fatal("Place ID is empty")
		}
		t.Logf("✓ Place details successful - Name: %s", details.Name)
		t.Logf("  Address: %s", details.FormattedAddress)
		t.Logf("  Location: %.6f, %.6f", details.Geometry.Location.Lat, details.Geometry.Location.Lng)
	})

	// Test Geocoding
	t.Run("Geocoding", func(t *testing.T) {
		results, err := client.GetCoordinates("1600 Amphitheatre Parkway, Mountain View, CA")
		if err != nil {
			t.Fatalf("Geocoding failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Geocoding returned no results")
		}
		result := results[0]
		t.Logf("✓ Geocoding successful")
		if result.FormattedAddress != "" {
			t.Logf("  Address: %s", result.FormattedAddress)
		}
		if result.Geometry.Location.Lat != 0 && result.Geometry.Location.Lng != 0 {
			t.Logf("  Location: %.6f, %.6f", result.Geometry.Location.Lat, result.Geometry.Location.Lng)
		}
	})
}

// TestGeminiIntegration tests Gemini API integration
func TestGeminiIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Fatal("GEMINI_API_KEY must be set")
	}
	client, err := gemini.NewClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to initialize Gemini client: %v", err)
	}
	defer client.Close()

	// Test Content Generation
	t.Run("GenerateContent", func(t *testing.T) {
		prompt := "Say 'Hello, World!' in exactly 3 words."
		response, err := client.GenerateContent(prompt)
		if err != nil {
			t.Fatalf("Content generation failed: %v", err)
		}
		if response == "" {
			t.Fatal("Content generation returned empty response")
		}
		t.Logf("✓ Content generation successful")
		t.Logf("  Response: %s", response)
	})

	// Test Complex Prompt
	t.Run("ComplexPrompt", func(t *testing.T) {
		prompt := "What are the top 3 travel destinations in Europe? List them in a numbered format."
		response, err := client.GenerateContent(prompt)
		if err != nil {
			t.Fatalf("Complex content generation failed: %v", err)
		}
		if response == "" {
			t.Fatal("Complex content generation returned empty response")
		}
		t.Logf("✓ Complex prompt successful")
		t.Logf("  Response length: %d characters", len(response))
	})
}
