package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
- DO NOT ask for information that's already provided in the query
- Only use askUser if something is genuinely unclear or missing
- Return itinerary with ACTUAL dates in YYYY-MM-DD format

Final Answer Schema:
{
  "itineraries": [ { 
    "destination": "Paris", 
    "startDate": "2026-01-25",  // Actual date, not "next weekend"
    "endDate": "2026-01-26",
    "activities": []
  } ],
  "reasoning": "Calculated next weekend as Jan 25-26, 2026"
}`

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

	// Log if any tool calls were made in history
	history := response.History()
	log.Printf("[DEBUG] Response history has %d messages", len(history))

	// Log if any tool calls were made
	toolCallCount := 0
	for _, msg := range history {
		for _, part := range msg.Content {
			if part.ToolRequest != nil {
				toolCallCount++
				log.Printf("[DEBUG] Tool call detected: %s", part.ToolRequest.Name)
			}
		}
	}
	log.Printf("[DEBUG] Total tool calls in response: %d", toolCallCount)

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
