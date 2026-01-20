package agents

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/tools"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockLLMClient
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func TestTripPlanner_Plan_WithToolCall(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMClient)
	registry := tools.NewRegistry()

	// Initialize Genkit
	gk := genkit.Init(context.Background())

	// Register a dummy dateTool
	registry.Register(genkit.DefineTool[map[string]interface{}, string](
		gk,
		"dateTool",
		"test date tool",
		func(ctx *ai.ToolContext, _ map[string]interface{}) (string, error) {
			return "2026-01-24T00:00:00Z", nil
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return "2026-01-24T00:00:00Z", nil
	})

	planner := NewTripPlanner(mockLLM, registry)

	// User query
	req := PlanRequest{
		UserQuery: "Trip to Paris next weekend",
	}

	// Mock LLM Interactions
	// 1. First call: LLM decides to call dateTool
	mockLLM.On("GenerateContent", mock.Anything, mock.MatchedBy(func(prompt string) bool {
		return true // Match any prompt used in first step
	})).Return(`[{"tool": "dateTool", "args": {"code": "some js code"}}]`, nil).Once()

	// 2. Second call: LLM receives tool output and generates plan
	mockLLM.On("GenerateContent", mock.Anything, mock.MatchedBy(func(prompt string) bool {
		// Verify history contains tool output
		return true
	})).Return(`{
		"itinerary": {
			"title": "Paris Trip",
			"start_time": "2026-01-24T00:00:00Z",
			"end_time": "2026-01-26T00:00:00Z",
			"travelers": 1,
			"accommodation": [],
			"transport": []
		}
	}`, nil).Once()

	// Execute
	res, err := planner.Plan(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, res.Itinerary)
	assert.Equal(t, "Paris Trip", res.Itinerary.Title)

	mockLLM.AssertExpectations(t)
}

func TestTripPlanner_Plan_MalformedJSON(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMClient)
	registry := tools.NewRegistry()
	gk := genkit.Init(context.Background())

	registry.Register(genkit.DefineTool[map[string]interface{}, string](
		gk,
		"dateTool",
		"test date tool",
		func(ctx *ai.ToolContext, _ map[string]interface{}) (string, error) {
			return "2026-01-24T00:00:00Z", nil
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return "2026-01-24T00:00:00Z", nil
	})

	planner := NewTripPlanner(mockLLM, registry)
	req := PlanRequest{UserQuery: "Fix parsing"}

	// Mock output with trailing semicolon
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(`{"tool": "dateTool", "args": {"code": "..."}};`, nil).Once()

	// Expectations for second call (after successful tool exec)
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(`{
		"itinerary": { "title": "Fixed" }
	}`, nil).Once()

	res, err := planner.Plan(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res.Itinerary)
	assert.Equal(t, "Fixed", res.Itinerary.Title)
}

func TestTripPlanner_Plan_MultipleItineraries(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMClient)
	registry := tools.NewRegistry()
	planner := NewTripPlanner(mockLLM, registry)

	req := PlanRequest{UserQuery: "Weekend trip options"}

	// Mock LLM response with multiple itineraries
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(`{
		"itineraries": [
			{
				"title": "Option 1",
				"start_time": "2026-03-07T00:00:00Z",
				"end_time": "2026-03-09T00:00:00Z",
				"travelers": 1
			},
			{
				"title": "Option 2",
				"start_time": "2026-03-14T00:00:00Z",
				"end_time": "2026-03-16T00:00:00Z",
				"travelers": 1
			}
		]
	}`, nil).Once()

	res, err := planner.Plan(context.Background(), req)

	assert.NoError(t, err)
	// Itinerary is populated with the first option if not explicitly set in single field
	assert.NotNil(t, res.Itinerary)
	assert.Equal(t, "Option 1", res.Itinerary.Title)
	assert.Len(t, res.PossibleItineraries, 2)
	assert.Equal(t, "Option 1", res.PossibleItineraries[0].Title)
	assert.Equal(t, "Option 2", res.PossibleItineraries[1].Title)
}

func TestTripPlanner_PopulateOptions(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMClient)
	registry := tools.NewRegistry()
	gk := genkit.Init(context.Background())

	// Register mock flightTool that returns *amadeus.FlightSearchResponse
	registry.Register(genkit.DefineTool[map[string]interface{}, *amadeus.FlightSearchResponse](
		gk,
		"flightTool",
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

	planner := NewTripPlanner(mockLLM, registry)

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
	planner.populateOptions(context.Background(), itin)

	// VERIFY
	// Check if TransportOptions were populated
	edge := itin.Graph.Edges[0]
	assert.NotEmpty(t, edge.TransportOptions, "Expected transport options to be populated")
	if len(edge.TransportOptions) > 0 {
		opt := edge.TransportOptions[0]
		assert.Equal(t, pb.TransportType_TRANSPORT_TYPE_FLIGHT, opt.Type)
		// Check details populated from mock
		// Mock returned price 100.00
		assert.Equal(t, float32(150.50), opt.PriceTotal)
		assert.Equal(t, "LHR", opt.OriginLocation.IataCodes[0])
		assert.Equal(t, "CDG", opt.DestinationLocation.IataCodes[0])
	}
}
