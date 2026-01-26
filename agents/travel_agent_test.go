package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockPlanner
type MockPlanner struct {
	mock.Mock
}

func (m *MockPlanner) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PlanResult), args.Error(1)
}

func TestTravelAgent_OrchestrateRequest(t *testing.T) {
	// Setup Mock Planner
	mockPlanner := new(MockPlanner)

	// Setup TravelDesk with Mock Amadeus
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Mock responses to avoid errors
		switch r.URL.Path {
		case "/v1/security/oauth2/token":
			json.NewEncoder(w).Encode(amadeus.AuthToken{AccessToken: "token"})
		case "/v2/shopping/flight-offers":
			json.NewEncoder(w).Encode(amadeus.FlightSearchResponse{
				Data: []amadeus.FlightOffer{{
					ID:    "flight1",
					Price: amadeus.Price{Total: "200.00"},
					Itineraries: []amadeus.Itinerary{{Segments: []amadeus.Segment{{
						CarrierCode: "BA", Number: "123",
						Departure: amadeus.FlightEndPoint{IataCode: "LHR", At: "2026-06-01T10:00:00"},
						Arrival:   amadeus.FlightEndPoint{IataCode: "JFK", At: "2026-06-01T14:00:00"},
					}}}},
				}},
			})
		case "/v1/reference-data/locations/hotels/by-city":
			json.NewEncoder(w).Encode(amadeus.HotelListResponse{Data: []amadeus.HotelData{{HotelId: "H1", Name: "Hotel A"}}})
		case "/v3/shopping/hotel-offers":
			json.NewEncoder(w).Encode(amadeus.HotelSearchResponse{Data: []amadeus.HotelOfferData{{
				Available: true,
				Hotel:     amadeus.HotelInfo{HotelId: "H1", Name: "Hotel A"},
				Offers: []amadeus.HotelOffer{{
					ID: "offer1", Price: amadeus.HotelPrice{Total: "150.00"}, Guests: amadeus.HotelGuests{Adults: 1},
				}},
			}}})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	client, _ := amadeus.NewClient("id", "secret", false, nil, nil, 10, 10, 30, nil)
	client.BaseURL = ts.URL
	desk := NewTravelDesk(client)

	agent := NewTravelAgent(mockPlanner, desk)

	// User Query
	query := "Plan a trip"

	// Mock Plan Result
	itin := &pb.Itinerary{
		Title: "Test Itinerary",
		Graph: &pb.Graph{
			Edges: []*pb.Edge{{
				Transport: &pb.Transport{
					Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
					OriginLocation:      &pb.Location{IataCodes: []string{"LHR"}},
					DestinationLocation: &pb.Location{IataCodes: []string{"JFK"}},
					Details:             &pb.Transport_Flight{Flight: &pb.Flight{DepartureTime: timestamppb.Now()}},
				},
			}},
		},
	}

	planRes := &PlanResult{
		Itinerary: itin,
		Reasoning: "Good plan",
	}

	mockPlanner.On("Plan", mock.Anything, mock.MatchedBy(func(req PlanRequest) bool {
		return req.UserQuery == query
	})).Return(planRes, nil).Once()

	// Execute
	response, err := agent.OrchestrateRequest(context.Background(), query, "")

	// Verify
	assert.NoError(t, err)
	assert.Contains(t, response, "Test Itinerary")
	assert.Contains(t, response, "Flight")
	// The flight price from mock (200.00) should NOT necessarily be in the final text response unless the formatter includes details from options.
	// The current formatter uses the transport details, not options.
	// But CheckAvailability updates the itinerary with errors if any. Since we mocked success, no errors.

	mockPlanner.AssertExpectations(t)
}

func TestTravelAgent_OrchestrateRequest_Clarification(t *testing.T) {
	mockPlanner := new(MockPlanner)
	agent := NewTravelAgent(mockPlanner, nil)

	mockPlanner.On("Plan", mock.Anything, mock.Anything).Return(&PlanResult{
		NeedsClarification: true,
		Question:           "Where to?",
	}, nil).Once()

	response, err := agent.OrchestrateRequest(context.Background(), "Trip", "")

	assert.NoError(t, err)
	assert.Equal(t, "Where to?", response)
}

func TestTravelAgent_OrchestrateRequest_RetryOnFailure(t *testing.T) {
	// Simulate Planner returning a plan that fails verification (e.g. no flights), then a revised plan that works
	mockPlanner := new(MockPlanner)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Token Endpoint
		if r.URL.Path == "/v1/security/oauth2/token" {
			json.NewEncoder(w).Encode(amadeus.AuthToken{AccessToken: "token", ExpiresIn: 3600})
			return
		}

		// Fail first flight search
		if strings.Contains(r.URL.RawQuery, "originLocationCode=FAIL") {
			json.NewEncoder(w).Encode(amadeus.FlightSearchResponse{Data: []amadeus.FlightOffer{}})
			return
		}
		// Succeed others
		if strings.Contains(r.URL.Path, "flight-offers") {
			json.NewEncoder(w).Encode(amadeus.FlightSearchResponse{
				Data: []amadeus.FlightOffer{{ID: "1", Price: amadeus.Price{Total: "100.00"}}},
			})
			return
		}
		// Default success for others
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, _ := amadeus.NewClient("id", "secret", false, nil, nil, 10, 10, 30, nil)
	client.BaseURL = ts.URL
	desk := NewTravelDesk(client)
	agent := NewTravelAgent(mockPlanner, desk)

	// 1. Bad Plan
	badItin := &pb.Itinerary{
		Title: "Bad Plan",
		Graph: &pb.Graph{
			Edges: []*pb.Edge{{
				Transport: &pb.Transport{
					Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
					OriginLocation:      &pb.Location{IataCodes: []string{"FAIL"}},
					DestinationLocation: &pb.Location{IataCodes: []string{"JFK"}},
					Details:             &pb.Transport_Flight{Flight: &pb.Flight{DepartureTime: timestamppb.Now()}},
				},
			}},
		},
	}

	// 2. Good Plan
	goodItin := &pb.Itinerary{
		Title: "Good Plan",
		Graph: &pb.Graph{
			Edges: []*pb.Edge{{
				Transport: &pb.Transport{
					Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
					OriginLocation:      &pb.Location{IataCodes: []string{"LHR"}},
					DestinationLocation: &pb.Location{IataCodes: []string{"JFK"}},
					Details:             &pb.Transport_Flight{Flight: &pb.Flight{DepartureTime: timestamppb.Now()}},
				},
			}},
		},
	}

	// Sequence of returns
	// Call 1: Returns Bad Plan
	mockPlanner.On("Plan", mock.Anything, mock.MatchedBy(func(req PlanRequest) bool {
		return !strings.Contains(req.History, "The proposed plans had issues")
	})).Return(&PlanResult{Itinerary: badItin, Reasoning: "Attempt 1"}, nil).Once()

	// Call 2: Receives feedback and returns Good Plan
	mockPlanner.On("Plan", mock.Anything, mock.MatchedBy(func(req PlanRequest) bool {
		return strings.Contains(req.History, "The proposed plans had issues")
	})).Return(&PlanResult{Itinerary: goodItin, Reasoning: "Attempt 2"}, nil).Once()

	response, err := agent.OrchestrateRequest(context.Background(), "Plan trip", "")

	assert.NoError(t, err)
	assert.Contains(t, response, "Good Plan")
	mockPlanner.AssertExpectations(t)
}
