package agents

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/tools"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/*
// MockLLMClient
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Name() string {
	return "mock-model"
}

func (m *MockLLMClient) Generate(ctx context.Context, req *ai.ModelRequest, cb ai.ModelStreamCallback) (*ai.ModelResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ai.ModelResponse), args.Error(1)
}

func TestTripPlanner_Plan_WithToolCall(t *testing.T) {
	// ... (content commented out)
}

func TestTripPlanner_Plan_MalformedJSON(t *testing.T) {
	// ... (content commented out)
}

func TestTripPlanner_Plan_MultipleItineraries(t *testing.T) {
	// ... (content commented out)
}
*/

func TestTripPlanner_PopulateOptions(t *testing.T) {
	// Setup
	registry := tools.NewRegistry()
	gk := genkit.Init(context.Background())

	// Register mock flightTool that returns *amadeus.FlightSearchResponse
	registry.Register(genkit.DefineTool[map[string]interface{}, *amadeus.FlightSearchResponse](
		gk,
		"amadeus_flight_tool",
		"mock flight tool",
		func(ctx *ai.ToolContext, args map[string]interface{}) (*amadeus.FlightSearchResponse, error) {
			return &amadeus.FlightSearchResponse{
				Data: []amadeus.FlightOffer{
					{
						Type: "flight-offer",
						ID:   "1",
						Price: amadeus.Price{
							Total: "100.00",
						},
						Itineraries: []amadeus.Itinerary{
							{
								Segments: []amadeus.Segment{
									{
										CarrierCode: "BA",
										Number:      "123",
										Departure:   amadeus.FlightEndPoint{IataCode: "LHR", At: "2026-06-01T10:00:00"},
										Arrival:     amadeus.FlightEndPoint{IataCode: "JFK", At: "2026-06-01T14:00:00"},
									},
								},
							},
						},
					},
				},
			}, nil
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Native handler for direct execution
		return &amadeus.FlightSearchResponse{
			Data: []amadeus.FlightOffer{
				{
					Type: "flight-offer",
					ID:   "mock-opt-1",
					Price: amadeus.Price{
						Total: "150.50",
					},
					Itineraries: []amadeus.Itinerary{
						{
							Segments: []amadeus.Segment{
								{
									CarrierCode: "AF",
									Number:      "999",
									Departure:   amadeus.FlightEndPoint{IataCode: "LHR", At: "2026-06-01T10:00:00"},
									Arrival:     amadeus.FlightEndPoint{IataCode: "CDG", At: "2026-06-01T12:00:00"},
								},
							},
						},
					},
				},
			},
		}, nil
	})

	// Pass nil for model as populateOptions doesn't use it
	planner := NewTripPlanner(gk, registry, nil)

	// Create dummy itinerary with one flight edge
	itin := &pb.Itinerary{
		Graph: &pb.Graph{
			Edges: []*pb.Edge{
				{
					Transport: &pb.Transport{
						Type: pb.TransportType_TRANSPORT_TYPE_FLIGHT,
						OriginLocation: &pb.Location{
							IataCodes: []string{"LHR"},
						},
						DestinationLocation: &pb.Location{
							IataCodes: []string{"CDG"},
						},
						TravelerCount: 1,
						// Flight preference to trigger date logic
						FlightPreferences: &pb.FlightPreferences{}, // Date logic looks at Details-Flight usually
					},
				},
			},
		},
	}
	// Add Details to trigger date logic
	itin.Graph.Edges[0].Transport.Details = &pb.Transport_Flight{
		Flight: &pb.Flight{
			DepartureTime: timestamppb.New(timestamppb.Now().AsTime()), // Just need a valid time
		},
	}

	// EXECUTE
	// populateOptions is private, but we are in the same package (agents)
	// However, the test file package is 'agents', so it can access it.
	planner.populateOptions(context.Background(), itin)

	// VERIFY
	// Check if TransportOptions were populated
	edge := itin.Graph.Edges[0]
	assert.NotEmpty(t, edge.TransportOptions, "Expected transport options to be populated")
	if len(edge.TransportOptions) > 0 {
		opt := edge.TransportOptions[0]
		assert.Equal(t, pb.TransportType_TRANSPORT_TYPE_FLIGHT, opt.Type)
		// Check details populated from mock
		// Mock returned price 150.50 (from direct handler)
		assert.Equal(t, float32(150.50), opt.PriceTotal)
		assert.Equal(t, "LHR", opt.OriginLocation.IataCodes[0])
		assert.Equal(t, "CDG", opt.DestinationLocation.IataCodes[0])
	}
}
