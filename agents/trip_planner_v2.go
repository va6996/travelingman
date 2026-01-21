package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/tools"
)

// TripPlannerV2 is the new implementation using Genkit's native tool calling
type TripPlannerV2 struct {
	genkit   *genkit.Genkit
	registry *tools.Registry
	model    ai.Model
	askUser  ai.Tool
}

// AskUserRequest is the input for the askUser tool
type AskUserRequest struct {
	Question string `json:"question" description:"The clarifying question to ask the user"`
}

// NewTripPlannerV2 creates a new TripPlannerV2 with Genkit native tool calling
func NewTripPlannerV2(gk *genkit.Genkit, registry *tools.Registry, model ai.Model) *TripPlannerV2 {
	// Define the askUser tool for clarifications
	askUser := genkit.DefineTool(gk, "askUser", "Ask the user a clarifying question when you need more information to plan the trip.",
		func(ctx *ai.ToolContext, req *AskUserRequest) (string, error) {
			// This tool interrupts the flow to ask the user a question
			return "", ctx.Interrupt(&ai.InterruptOptions{
				Metadata: map[string]any{
					"question": req.Question,
				},
			})
		},
	)

	return &TripPlannerV2{
		genkit:   gk,
		registry: registry,
		model:    model,
		askUser:  askUser,
	}
}

// Plan generates a high-level itinerary based on the user query
func (p *TripPlannerV2) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	log.Printf("TripPlannerV2: Planning for query: %s", req.UserQuery)

	// Get all registered tools and add askUser
	var toolRefs []ai.ToolRef
	if p.registry != nil {
		for _, tool := range p.registry.GetTools() {
			toolRefs = append(toolRefs, tool)
			log.Printf("[DEBUG] Registered tool: %s - %s", tool.Name(), tool.Definition().Description)
		}
	}
	toolRefs = append(toolRefs, p.askUser)
	log.Printf("[DEBUG] Total tools available: %d (including askUser)", len(toolRefs))

	// System prompt for the trip planner
	systemPrompt := `You are an expert Trip Planner. Your goal is to create a high-level travel itinerary.

IMPORTANT WORKFLOW:
1. First, gather information using tools ONLY if needed:
   - Use dateTool to calculate dates (e.g., "next weekend" → actual dates like "2026-01-25")
   - Use askUser ONLY if critical information is truly missing (e.g., user said "sometime" without specifying when)
   
2. Then, create the itinerary JSON with the gathered information:
   - DO NOT call hotelTool or flightTool - these are for the TravelDesk, not for planning
   - Just return the itinerary with destination, dates, and activities
   - Use the actual dates you calculated, not relative terms

CRITICAL RULES:
- If the user specifies a timeframe (like "next weekend"), use dateTool to calculate it, then create the itinerary
- Return itinerary with ACTUAL dates in YYYY-MM-DD format

Final Answer Schema:
{
  "itineraries": [
    {
      "title": "Weekend in Paris",
      "description": "A wonderful weekend trip to Paris visiting key landmarks.",
      "start_time": "2026-01-25T10:00:00Z",
      "end_time": "2026-01-27T18:00:00Z",
      "travelers": 2,
      "graph": {
        "nodes": [
          {
            "id": "node_1",
            "location": "PAR",
            "from_timestamp": "2026-01-25T14:00:00Z",
            "to_timestamp": "2026-01-27T11:00:00Z",
            "is_inter_city": false,
            "stay": {
              "name": "Hotel Paris",
              "city_code": "PAR",
              "check_in": "2026-01-25T14:00:00Z",
              "check_out": "2026-01-27T11:00:00Z",
              "traveler_count": 2,
              "room_type": "Standard",
              "area": "City Center",
              "rating": 4,
              "amenities": ["wifi", "breakfast"]
            }
          }
        ],
        "edges": [
          {
            "from_id": "start_loc", // Logical start if needed, or previous node
            "to_id": "node_1",
            "duration_seconds": 25200,
            "transport": {
              "type": "TRANSPORT_TYPE_FLIGHT",
              "traveler_count": 2,
              "class": "ECONOMY",
              "flight": {
                "departure_airport": "JFK",
                "arrival_airport": "CDG",
                "departure_time": "2026-01-25T10:00:00Z",
                "arrival_time": "2026-01-25T17:00:00Z"
              }
            }
          }
        ]
      }
    }
  ],
  "reasoning": "Calculated next weekend as Jan 25-27, 2026 and constructed graph with flight to Paris and hotel stay."
}`

	// Inject current date context into system prompt
	today := time.Now().Format("2006-01-02")
	systemPrompt = fmt.Sprintf("Today is %s.\n%s", today, systemPrompt)
	log.Printf("[DEBUG] Full system prompt: %s", systemPrompt)

	log.Printf("[DEBUG] Calling genkit.Generate with model: %v, tools: %d", p.model, len(toolRefs))

	// Use Genkit's native tool calling with automatic iteration
	response, err := genkit.Generate(ctx,
		p.genkit,
		ai.WithModel(p.model),
		ai.WithSystem(systemPrompt),
		ai.WithPrompt(req.UserQuery),
		ai.WithTools(toolRefs...),
		ai.WithMaxTurns(15), // Automatic iteration limit
	)
	if err != nil {
		log.Printf("TripPlannerV2: Generate error: %v", err)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	log.Printf("[DEBUG] Response finish reason: %v", response.FinishReason)

	// Handle interrupts (askUser tool calls)
	for response.FinishReason == ai.FinishReasonInterrupted {
		var answers []*ai.Part
		for _, part := range response.Interrupts() {
			if part.ToolRequest.Name == "askUser" {
				// Extract the question from metadata
				question := part.ToolRequest.Input.(map[string]any)["question"]
				log.Printf("TripPlannerV2: Asking user: %s", question)

				// Return the question to the user
				return &PlanResult{
					NeedsClarification: true,
					Question:           fmt.Sprintf("%v", question),
				}, nil
			}
		}

		// If we handled all interrupts, continue generation
		if len(answers) > 0 {
			response, err = genkit.Generate(ctx,
				p.genkit,
				ai.WithMessages(response.History()...),
				ai.WithTools(toolRefs...),
				ai.WithToolResponses(answers...),
			)
			if err != nil {
				return nil, fmt.Errorf("planning continuation failed: %w", err)
			}
		} else {
			break
		}
	}

	text := response.Text()
	log.Printf("[DEBUG] LLM Final Response: %s", text)

	// Extract JSON from response
	extractedJSON := extractUsageJSON(text)
	if extractedJSON != "" {
		text = extractedJSON
	}

	// Try to parse as final answer
	var finalAnswer struct {
		Itineraries []JSONItinerary `json:"itineraries"`
		Itinerary   *JSONItinerary  `json:"itinerary"`
		Reasoning   string          `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(text), &finalAnswer); err == nil {
		// Handle itineraries
		var jsonItineraries []JSONItinerary
		if len(finalAnswer.Itineraries) > 0 {
			jsonItineraries = finalAnswer.Itineraries
		} else if finalAnswer.Itinerary != nil {
			jsonItineraries = []JSONItinerary{*finalAnswer.Itinerary}
		}

		if len(jsonItineraries) > 0 {
			log.Printf("TripPlannerV2: Generated %d itineraries", len(jsonItineraries))
			// TODO: Convert JSONItineraries to pb.Itinerary
			// For now, just return empty PossibleItineraries
			return &PlanResult{
				Reasoning: finalAnswer.Reasoning,
			}, nil
		}
	}

	// Fallback: return raw text
	log.Printf("TripPlannerV2: Could not parse response, returning raw text")
	return &PlanResult{
		Question: "I couldn't generate a proper itinerary. Here's what I found: " + text,
	}, nil
}
