package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockAmadeusServer creates a test server that mocks Amadeus endpoints
func mockAmadeusServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/security/oauth2/token":
			json.NewEncoder(w).Encode(amadeus.AuthToken{
				AccessToken: "test_token",
				ExpiresIn:   1800,
				TokenType:   "Bearer",
			})
		case "/v2/shopping/flight-offers":
			// Mock flight search response
			json.NewEncoder(w).Encode(amadeus.FlightSearchResponse{
				Data: []amadeus.FlightOffer{{
					ID: "flight_1",
					Price: amadeus.Price{
						Total: "100.00",
					},
					Itineraries: []amadeus.Itinerary{{
						Segments: []amadeus.Segment{{
							CarrierCode: "BA",
							Number:      "123",
							Departure:   amadeus.FlightEndPoint{IataCode: "LHR", At: "2026-06-01T10:00:00"},
							Arrival:     amadeus.FlightEndPoint{IataCode: "JFK", At: "2026-06-01T14:00:00"},
						}},
					}},
				}},
			})
		case "/v1/reference-data/locations/hotels/by-city":
			// Mock hotel list response
			json.NewEncoder(w).Encode(amadeus.HotelListResponse{
				Data: []amadeus.HotelData{{
					HotelId: "H1",
					Name:    "Test Hotel",
				}},
			})
		case "/v3/shopping/hotel-offers":
			// Mock hotel offers response
			json.NewEncoder(w).Encode(amadeus.HotelSearchResponse{
				Data: []amadeus.HotelOfferData{{
					Available: true,
					Hotel:     amadeus.HotelInfo{HotelId: "H1", Name: "Test Hotel", CityCode: "NYC"},
					Offers: []amadeus.HotelOffer{{
						ID:           "offer1",
						CheckInDate:  "2026-06-01",
						CheckOutDate: "2026-06-05",
						Price:        amadeus.HotelPrice{Total: "500.00"},
						Guests:       amadeus.HotelGuests{Adults: 1},
					}},
				}},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestTravelDesk_CheckAvailability(t *testing.T) {
	ts := mockAmadeusServer()
	defer ts.Close()

	// Initialize Amadeus Client pointing to mock server
	// Passing nil for genkit and registry as we're testing TravelDesk logic directly calling Client methods
	client, err := amadeus.NewClient("id", "secret", false, nil, nil)
	assert.NoError(t, err)
	client.BaseURL = ts.URL

	desk := NewTravelDesk(client)

	// Create Itinerary
	itin := &pb.Itinerary{
		Title: "Test Trip",
		Graph: &pb.Graph{
			Edges: []*pb.Edge{
				{
					Transport: &pb.Transport{
						Type: pb.TransportType_TRANSPORT_TYPE_FLIGHT,
						OriginLocation: &pb.Location{
							IataCodes: []string{"LHR"},
						},
						DestinationLocation: &pb.Location{
							IataCodes: []string{"JFK"},
						},
						TravelerCount: 1,
						Details: &pb.Transport_Flight{
							Flight: &pb.Flight{
								DepartureTime: timestamppb.New(time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)),
							},
						},
					},
				},
			},
			Nodes: []*pb.Node{
				{
					Location: "NYC",
					Stay: &pb.Accommodation{
						Address:       "NYC",
						TravelerCount: 1,
						CheckIn:       timestamppb.New(time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)),
						CheckOut:      timestamppb.New(time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC)),
					},
				},
			},
		},
	}

	// Execute
	updatedItin, err := desk.CheckAvailability(context.Background(), itin)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, updatedItin)

	// Verify Flights
	flightEdge := updatedItin.Graph.Edges[0]
	assert.NotEmpty(t, flightEdge.TransportOptions)
	assert.Equal(t, "100.00", fmt.Sprintf("%.2f", flightEdge.TransportOptions[0].PriceTotal))
	assert.Nil(t, flightEdge.Transport.Error)

	// Verify Hotels
	hotelNode := updatedItin.Graph.Nodes[0]
	assert.NotEmpty(t, hotelNode.StayOptions)
	assert.Equal(t, "500.00", hotelNode.StayOptions[0].PriceTotal)
	assert.Nil(t, hotelNode.Stay.Error)
}

func TestTravelDesk_CheckAvailability_NoAvailability(t *testing.T) {
	// Mock server that returns empty results
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/security/oauth2/token":
			json.NewEncoder(w).Encode(amadeus.AuthToken{AccessToken: "token"})
		case "/v2/shopping/flight-offers":
			json.NewEncoder(w).Encode(amadeus.FlightSearchResponse{Data: []amadeus.FlightOffer{}})
		case "/v1/reference-data/locations/hotels/by-city":
			json.NewEncoder(w).Encode(amadeus.HotelListResponse{Data: []amadeus.HotelData{}})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	client, _ := amadeus.NewClient("id", "secret", false, nil, nil)
	client.BaseURL = ts.URL
	desk := NewTravelDesk(client)

	itin := &pb.Itinerary{
		Graph: &pb.Graph{
			Edges: []*pb.Edge{{
				Transport: &pb.Transport{
					Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
					OriginLocation:      &pb.Location{IataCodes: []string{"LHR"}},
					DestinationLocation: &pb.Location{IataCodes: []string{"JFK"}},
					Details:             &pb.Transport_Flight{Flight: &pb.Flight{DepartureTime: timestamppb.Now()}},
				},
			}},
			Nodes: []*pb.Node{{
				Stay: &pb.Accommodation{Address: "NYC", CheckIn: timestamppb.Now(), CheckOut: timestamppb.Now()},
			}},
		},
	}

	updatedItin, _ := desk.CheckAvailability(context.Background(), itin)

	// Verify errors are populated
	assert.NotNil(t, updatedItin.Graph.Edges[0].Transport.Error)
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND, updatedItin.Graph.Edges[0].Transport.Error.Code)

	assert.NotNil(t, updatedItin.Graph.Nodes[0].Stay.Error)
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND, updatedItin.Graph.Nodes[0].Stay.Error.Code)
}
