package amadeus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockAmadeusServer creates a test server that mocks Amadeus endpoints
func mockAmadeusServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/security/oauth2/token":
			json.NewEncoder(w).Encode(AuthToken{
				AccessToken: "test_token",
				ExpiresIn:   1800,
				TokenType:   "Bearer",
			})
		case "/v2/shopping/flight-offers":
			// Mock flight search response
			json.NewEncoder(w).Encode(FlightSearchResponse{
				Data: []FlightOffer{{ID: "1"}},
			})
		case "/v1/booking/flight-orders":
			// Mock flight booking response
			json.NewEncoder(w).Encode(FlightOrderResponse{
				Data: struct {
					Type              string             `json:"type"`
					ID                string             `json:"id"`
					QueuingOfficeId   string             `json:"queuingOfficeId"`
					AssociatedRecords []AssociatedRecord `json:"associatedRecords"`
					FlightOffers      []FlightOffer      `json:"flightOffers"`
					Travelers         []TravelerInfo     `json:"travelers"`
				}{ID: "order_123"},
			})
		case "/v3/shopping/hotel-offers":
			json.NewEncoder(w).Encode(HotelSearchResponse{
				Data: []HotelOfferData{{
					Available: true,
					Hotel:     HotelInfo{HotelId: "H1"},
				}},
			})
		case "/v2/booking/hotel-orders":
			// Mock hotel booking
			// Just return valid JSON structure matching HotelOrderResponse
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"data": [{"id": "hotel_order_1"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestClient_Authenticate(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient("id", "secret", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	err = client.Authenticate()
	assert.NoError(t, err)
	assert.Equal(t, "test_token", client.Token.AccessToken)
}

func TestSearchFlights(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient("id", "secret", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	resp, err := client.SearchFlights("JFK", "LHR", "2025-10-10", "", "", 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
	assert.Equal(t, "1", resp.Data[0].ID)
}

func TestBookFlight(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient("id", "secret", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL
	// Manually set token to skip auth call in this test logic if desired,
	// but Authenticate() is lazy-loaded so it will call token endpoint mock anyway.

	offer := FlightOffer{ID: "1"}
	travelers := []TravelerInfo{{ID: "1", Name: Name{FirstName: "John", LastName: "Doe"}}}

	resp, err := client.BookFlight(offer, travelers)
	assert.NoError(t, err)
	assert.Equal(t, "order_123", resp.Data.ID)
}

func TestSearchHotelOffers(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient("id", "secret", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	resp, err := client.SearchHotelOffers([]string{"H1"}, 1, "2025-10-10", "2025-10-11")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}
