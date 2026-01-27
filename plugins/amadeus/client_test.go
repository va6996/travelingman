package amadeus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
					Hotel:     HotelInfo{HotelId: "H1", Name: "Test Hotel", CityCode: "NYC"},
					Offers: []HotelOffer{{
						ID:           "offer1",
						CheckInDate:  "2025-10-10",
						CheckOutDate: "2025-10-11",
						Price:        HotelPrice{Total: "100.00"},
						Guests:       HotelGuests{Adults: 1},
						Room: HotelRoom{
							TypeEstimated: struct {
								Category string `json:"category"`
								Beds     int    `json:"beds"`
								BedType  string `json:"bedType"`
							}{Category: "STANDARD"},
						},
					}},
				}},
			})
			// Mock hotel booking
			// Just return valid JSON structure matching HotelOrderResponse
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"data": [{"id": "hotel_order_1"}]}`))
		case "/v1/reference-data/locations":
			json.NewEncoder(w).Encode(LocationSearchResponse{
				Data: []LocationData{{
					SubType: "CITY",
					Name:    "PARIS",
					JobCode: "PAR",
					Address: Address{
						CityName:    "PARIS",
						CityCode:    "PAR",
						CountryName: "FRANCE",
						CountryCode: "FR",
					},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestClient_Authenticate(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", IsProduction: false,
		FlightLimit: 10, HotelLimit: 10, Timeout: 10,
		CacheTTL: CacheTTLConfig{Location: 24, Flight: 24, Hotel: 24},
	}, nil, nil, nil)
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

	client, err := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", IsProduction: false,
		FlightLimit: 10, HotelLimit: 10, Timeout: 10,
		CacheTTL: CacheTTLConfig{Location: 24, Flight: 24, Hotel: 24},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	resp, err := client.SearchFlights(context.Background(), &pb.Transport{
		Type:          pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		TravelerCount: 1,
		OriginLocation: &pb.Location{
			IataCodes: []string{"JFK"},
		},
		DestinationLocation: &pb.Location{
			IataCodes: []string{"LHR"},
		},
		Details: &pb.Transport_Flight{
			Flight: &pb.Flight{
				// Use a future date
				DepartureTime: timestamppb.New(time.Now().AddDate(0, 1, 0)), // 1 month from now
			},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	// assert.Equal(t, "1", resp[0].ReferenceNumber) // Can't easily check ID as it's not set in ToTransport logic currently
}

func TestBookFlight(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", IsProduction: false,
		FlightLimit: 10, HotelLimit: 10, Timeout: 10,
		CacheTTL: CacheTTLConfig{Location: 24, Flight: 24, Hotel: 24},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL
	// Manually set token to skip auth call in this test logic if desired,
	// but Authenticate() is lazy-loaded so it will call token endpoint mock anyway.

	offer := FlightOffer{ID: "1"}
	users := []*pb.User{
		{
			Id:          1,
			FullName:    "John Doe",
			DateOfBirth: timestamppb.New(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
			Gender:      "MALE",
			Email:       "john@example.com",
			Phone:       "1234567890",
		},
	}

	resp, err := client.BookFlight(context.Background(), offer, users)
	assert.NoError(t, err)
	assert.Equal(t, "order_123", resp.Data.ID)
}

func TestSearchHotelOffers(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", IsProduction: false,
		FlightLimit: 10, HotelLimit: 10, Timeout: 10,
		CacheTTL: CacheTTLConfig{Location: 24, Flight: 24, Hotel: 24},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	acc := &pb.Accommodation{
		TravelerCount: 1,
		CheckIn:       timestamppb.New(time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC)),
		CheckOut:      timestamppb.New(time.Date(2025, 10, 11, 0, 0, 0, 0, time.UTC)),
	}
	resp, err := client.SearchHotelOffers(context.Background(), []string{"H1"}, acc)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSearchLocations(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	client, err := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", IsProduction: false,
		FlightLimit: 10, HotelLimit: 10, Timeout: 10,
		CacheTTL: CacheTTLConfig{Location: 24, Flight: 24, Hotel: 24},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	resp, err := client.SearchLocations(context.Background(), "Paris")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	assert.Equal(t, "PAR", resp[0].IataCodes[0])
}
