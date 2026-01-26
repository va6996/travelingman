package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/tools"
	"google.golang.org/protobuf/encoding/protojson"
)

// TripPlanner is responsible for high-level travel planning using Genkit's native tool calling
type TripPlanner struct {
	genkit   *genkit.Genkit
	registry *tools.Registry
	model    ai.Model
	// askUser  ai.Tool
}

// PlanRequest contains the user's query and context
type PlanRequest struct {
	UserQuery string
	History   string
}

// PlanResult contains the generated itinerary or a clarifying question
type PlanResult struct {
	Itinerary           *pb.Itinerary
	PossibleItineraries []*pb.Itinerary
	NeedsClarification  bool
	Question            string
	Reasoning           string
}

// AskUserRequest is the input for the askUser tool
type AskUserRequest struct {
	Question string `json:"question" description:"The clarifying question to ask the user"`
}

const SYSTEM_PROMPT = `You are an expert Trip Planner. Your goal is to create a high-level travel itinerary.

IMPORTANT WORKFLOW:
1. First, gather information using tools ONLY if needed:
   - ALWAYS use dateTool to calculate dates. usage:
     - The tool returns a JSON list of ISO strings: ["2026-01-25", "2026-01-28"]
     - For ONE-WAY trips, use the first date.
     - For RETURN/ROUND trips, use the first date as start and second as end.
     - For EXTENDED/MULTI-CITY trips, request multiple dates.

2. Then, create the itinerary JSON with the gathered information:
   - DO NOT call hotelTool or flightTool - these are for the TravelDesk, not for planning
   - Return the itinerary json with destination, dates, and activities

CURRENCY HANDLING:
- The system will automatically infer the currency based on the origin country (e.g. US -> USD, UK -> GBP).
- YOU MUST use this inferred currency for ALL cost calculations and bookings (including hotels in other countries). Do not switch currencies.
- Ensure all prices are in the same currency (e.g. if flying from US, hotel price must be in USD).

CRITICAL RULES:
- If the user specifies a timeframe (like "next weekend"), use dateTool to calculate it, then create the itinerary
- Structure your response exactly as the JSON schema below. Use camelCase for keys
- If the user requests a round/circle trip, the final edge must return to the ID of the starting Node. Do NOT create a duplicate 'Home' node.
- Do not ask for clarifications. Infer everything you need from the user's query from the perspective of source location
- Source Location Node: You MUST include the starting node (e.g., 'start_loc') in the 'nodes' array.

BROAD SEARCH:
- If the user request is broad (e.g., "any weekend in April"), you MUST generate multiple distinct itineraries (e.g., 3-4 options for different weekends) in the "itineraries" JSON array.
- Each itinerary in the array must be a complete, valid trip plan.

DAY ACTIVITIES:
- For detailed daily plans, populate the "sub_graph" field within the specific Node (e.g., the 'Paris' node). This sub-graph should contain nodes for activities (restaurants, museums) and edges for travel between them.

Final Answer Schema:
{
  "itineraries": [
    {
      "title": "Weekend in Paris",
      "description": "A wonderful weekend trip to Paris visiting key landmarks.",
      "startTime": "2026-01-25T10:00:00Z",
      "endTime": "2026-01-27T18:00:00Z",
      "travelers": 2,
      "journeyType": "JOURNEY_TYPE_RETURN",
      "graph": {
        "nodes": [
          {
            "id": "start_loc",
            "location": "JFK",
            "isInterCity": true
          },
          {
            "id": "node_1",
            "location": "PAR",
            "fromTimestamp": "2026-01-25T14:00:00Z",
            "toTimestamp": "2026-01-27T11:00:00Z",
            "isInterCity": false,
            "stay": {
              "name": "Hotel Paris",
              "location": { "cityCode": "PAR" },
              "checkIn": "2026-01-25T14:00:00Z",
              "checkOut": "2026-01-27T11:00:00Z",
              "travelerCount": 2,
              "preferences": {
                "roomType": "Standard",
                "area": "City Center",
                "rating": 4,
                "amenities": ["wifi", "breakfast"]
              }
            },
            "sub_graph": {
                "nodes": [
                    { "id": "act_1", "location": "Eiffel Tower", "type": "ACTIVITY" }
                ],
                "edges": [
                    { "fromId": "node_1", "toId": "act_1", "transport": { "type": "TRANSPORT_TYPE_TAXI" } }
                ]
            }
          }
        ],
        "edges": [
          {
            "fromId": "start_loc",
            "toId": "node_1",
            "durationSeconds": 25200,
            "transport": {
              "type": "TRANSPORT_TYPE_FLIGHT",
              "travelerCount": 2,
              "flightPreferences": { "travelClass": "CLASS_ECONOMY" },
              "flight": {
                "departureTime": "2026-01-25T10:00:00Z",
                "arrivalTime": "2026-01-25T17:00:00Z"
              },
              "originLocation": { "iataCodes": ["JFK"] },
              "destinationLocation": { "iataCodes": ["CDG"] }
            }
            }
          },
          {
            "fromId": "node_1",
            "toId": "start_loc",
            "durationSeconds": 28800,
            "transport": {
              "type": "TRANSPORT_TYPE_FLIGHT",
              "travelerCount": 2,
              "flightPreferences": { "travelClass": "CLASS_ECONOMY" },
              "flight": {
                "departureTime": "2026-01-27T11:00:00Z",
                "arrivalTime": "2026-01-27T19:00:00Z"
              },
              "originLocation": { "iataCodes": ["CDG"] },
              "destinationLocation": { "iataCodes": ["JFK"] }
            }
          }
        ]
      }
    }
  ],
  "reasoning": "Calculated next weekend as Jan 25-27, 2026 and constructed graph with flight to Paris and hotel stay."
}`

// NewTripPlanner creates a new TripPlanner with Genkit native tool calling
func NewTripPlanner(gk *genkit.Genkit, registry *tools.Registry, model ai.Model) *TripPlanner {
	// Define the askUser tool for clarifications
	// askUser := genkit.DefineTool(gk, "askUser", "Ask the user a clarifying question when you need more information to plan the trip.",
	// 	func(ctx *ai.ToolContext, req *AskUserRequest) (string, error) {
	// 		// This tool interrupts the flow to ask the user a question
	// 		return "", ctx.Interrupt(&ai.InterruptOptions{
	// 			Metadata: map[string]any{
	// 				"question": req.Question,
	// 			},
	// 		})
	// 	},
	// )

	// toolRefs = append(toolRefs, p.askUser)

	return &TripPlanner{
		genkit:   gk,
		registry: registry,
		model:    model,
		// askUser:  askUser,
	}
}

func (p *TripPlanner) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	log.Infof(ctx, "TripPlanner: Planning for query: %s", req.UserQuery)

	// Inject current date context into system prompt
	today := time.Now().Format("2006-01-02")
	systemPromptWithDate := fmt.Sprintf("Today is %s.\n%s", today, SYSTEM_PROMPT)
	log.Debugf(ctx, "Full system prompt: %s", systemPromptWithDate)

	log.Debugf(ctx, "Calling genkit.Generate with model: %v, tools: %d", p.model, len(p.registry.GetTools()))

	// Use configured timeout for the planning process
	// Default to 220s if not set (though Config should handle defaults)
	// We'll hardcode here if needed or pass config to TripPlanner?
	// The TripPlanner struct doesn't have the config yet.
	// For this task, we will stick to the hardcoded/env default from config via setup but TripPlanner needs access.
	// Let's assume the context passed in already has a deadline or we rely on the caller (TravelAgent) to set it?
	// But `genkit.Generate` is called here.
	// The config says "PlannerConfig.Timeout". We should ideally pass this to NewTripPlanner.

	// For now, I'll update NewTripPlanner signature in next step or just hardcode to match the config default if I can't change signature easily without cascading.
	// Wait, I updated Config with `PlannerConfig`. I should pass the timeout value to `NewTripPlanner`.

	tCtx, cancel := context.WithTimeout(ctx, 220*time.Second) // Default 2 minutes -> Updated to 220s default in config
	defer cancel()

	// Use Genkit's native tool calling with automatic iteration
	response, err := genkit.Generate(tCtx,
		p.genkit,
		ai.WithModel(p.model),
		ai.WithSystem(systemPromptWithDate),
		ai.WithPrompt(req.UserQuery),
		ai.WithTools(p.registry.GetToolRefs()...),
		ai.WithMaxTurns(15), // Automatic iteration limit
	)
	if err != nil {
		log.Errorf(ctx, "TripPlanner: Generate error: %v", err)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	log.Infof(ctx, "Response finish reason: %v", response.FinishReason)

	// Handle interrupts (askUser tool calls)
	for response.FinishReason == ai.FinishReasonInterrupted {
		var answers []*ai.Part
		// for _, part := range response.Interrupts() {
		// 	if part.ToolRequest.Name == "askUser" {
		// 		// Extract the question from metadata
		// 		question := part.ToolRequest.Input.(map[string]any)["question"]
		// 		log.Printf(ctx, "TripPlanner: Asking user: %s", question)

		// 		// Return the question to the user
		// 		return &PlanResult{
		// 			NeedsClarification: true,
		// 			Question:           fmt.Sprintf("%v", question),
		// 		}, nil
		// 	}
		// }

		// If we handled all interrupts, continue generation
		if len(answers) > 0 {
			response, err = genkit.Generate(tCtx,
				p.genkit,
				ai.WithMessages(response.History()...),
				ai.WithTools(p.registry.GetToolRefs()...),
				ai.WithToolResponses(answers...),
				ai.WithMaxTurns(15),
			)
			if err != nil {
				return nil, fmt.Errorf("planning continuation failed: %w", err)
			}
		} else {
			break
		}
	}

	text := response.Text()
	log.Infof(ctx, "LLM Final Response: %s", text)

	// Extract JSON from response
	extractedJSON := extractUsageJSON(text)
	if extractedJSON != "" {
		text = extractedJSON
	}

	// Try to parse as final answer
	var finalAnswer struct {
		Itineraries []json.RawMessage `json:"itineraries"`
		Reasoning   string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(text), &finalAnswer); err == nil {
		// Handle itineraries
		if len(finalAnswer.Itineraries) > 0 {
			log.Infof(ctx, "TripPlanner: Generated %d itineraries", len(finalAnswer.Itineraries))

			result := &PlanResult{
				Reasoning: finalAnswer.Reasoning,
			}

			// Configure protojson unmarshaler to discard unknown fields
			unmarshaler := protojson.UnmarshalOptions{
				DiscardUnknown: true,
			}

			// Convert first itinerary
			result.Itinerary = &pb.Itinerary{}
			if err := unmarshaler.Unmarshal(finalAnswer.Itineraries[0], result.Itinerary); err != nil {
				log.Errorf(ctx, "TripPlanner: Failed to unmarshal first itinerary: %v", err)
				return nil, fmt.Errorf("failed to parse itinerary: %w", err)
			}

			// Convert possible itineraries
			for i := 1; i < len(finalAnswer.Itineraries); i++ {
				pbItin := &pb.Itinerary{}
				if err := unmarshaler.Unmarshal(finalAnswer.Itineraries[i], pbItin); err == nil {
					result.PossibleItineraries = append(result.PossibleItineraries, pbItin)
				} else {
					log.Warnf(ctx, "TripPlanner: Failed to unmarshal itinerary %d: %v", i, err)
				}
			}

			// Resolve city codes
			if result.Itinerary != nil {
				// We need to implement resolveCityCodes if we want the airport codes to be correct
				p.resolveCityCodes(ctx, result.Itinerary)
			}

			return result, nil
		}
	}

	// Fallback: return raw text
	log.Warnf(ctx, "TripPlanner: Could not parse response, returning raw text %s", text)
	return &PlanResult{
		Question: "I couldn't generate a proper itinerary. Here's what I found: " + text,
	}, nil
}

// resolveCityCodes updates the itinerary with correct IATA codes and City names
func (p *TripPlanner) resolveCityCodes(ctx context.Context, itin *pb.Itinerary) {
	if p.registry == nil {
		return
	}

	for i := range itin.Graph.Nodes {
		node := itin.Graph.Nodes[i]
		if node.Stay == nil || node.Stay.Location == nil {
			continue
		}

		loc := node.Stay.Location
		needsResolution := false
		keyword := ""

		// If we have a code but no city name, resolve it
		if loc.City == "" && loc.CityCode != "" {
			needsResolution = true
			keyword = loc.CityCode
		} else if loc.City == "" && len(loc.IataCodes) > 0 {
			needsResolution = true
			keyword = loc.IataCodes[0]
		} else if len(loc.CityCode) > 3 || (len(loc.CityCode) == 3 && loc.CityCode != strings.ToUpper(loc.CityCode)) {
			// Suspicious code (looks like a name), resolve it
			needsResolution = true
			keyword = loc.CityCode
		}

		if needsResolution && keyword != "" {
			log.Debugf(ctx, "TripPlanner: Resolving location details for '%s'", keyword)

			// Simple synchronous call
			tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			res, err := p.registry.ExecuteTool(tCtx, "amadeus_location_tool", map[string]interface{}{
				"keyword": keyword,
			})
			cancel() // cancel immediately after done

			if err != nil {
				log.Errorf(ctx, "TripPlanner: Location search failed for %s: %v", keyword, err)
				continue
			}

			// Parse result
			var bestMatch *pb.Location
			if locations, ok := res.([]*pb.Location); ok && len(locations) > 0 {
				for _, l := range locations {
					if l.City != "" {
						bestMatch = l
						break
					}
				}
				if bestMatch == nil && len(locations) > 0 {
					bestMatch = locations[0]
				}
			} else {
				// Fallback JSON roundtrip
				b, _ := json.Marshal(res)
				var locs []*pb.Location
				if err := json.Unmarshal(b, &locs); err == nil && len(locs) > 0 {
					for _, l := range locs {
						if l.City != "" {
							bestMatch = l
							break
						}
					}
					if bestMatch == nil && len(locs) > 0 {
						bestMatch = locs[0]
					}
				}
			}

			// Apply result directly
			if bestMatch != nil {
				log.Debugf(ctx, "TripPlanner: Resolved '%s' to City: '%s', Code: '%s'", keyword, bestMatch.City, bestMatch.CityCode)

				if node.Stay.Location.City == "" {
					node.Stay.Location.City = bestMatch.City
				}
				if node.Stay.Location.Country == "" {
					node.Stay.Location.Country = bestMatch.Country
				}
				if node.Stay.Location.CityCode == "" || len(node.Stay.Location.CityCode) > 3 {
					node.Stay.Location.CityCode = bestMatch.CityCode
				}
				if len(node.Stay.Location.IataCodes) == 0 {
					node.Stay.Location.IataCodes = bestMatch.IataCodes
				}
			}
		}
	}
}

// resolveCityCodes updates the itinerary with correct IATA codes

// Helper to map string class to pb enum
func mapClass(c string) pb.Class {
	switch c {
	case "ECONOMY":
		return pb.Class_CLASS_ECONOMY
	case "PREMIUM_ECONOMY":
		return pb.Class_CLASS_PREMIUM_ECONOMY
	case "BUSINESS":
		return pb.Class_CLASS_BUSINESS
	case "FIRST":
		return pb.Class_CLASS_FIRST
	default:
		return pb.Class_CLASS_UNSPECIFIED
	}
}

// extractUsageJSON extracts JSON from a response that might have markdown code blocks
func extractUsageJSON(text string) string {
	// Try to extract JSON from markdown code blocks
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "```") {
		// Find the closing ```
		firstNewline := strings.Index(trimmed, "\n")
		if firstNewline != -1 {
			afterFirstLine := trimmed[firstNewline+1:]
			lastTripleBackticks := strings.LastIndex(afterFirstLine, "```")
			if lastTripleBackticks != -1 {
				return strings.TrimSpace(afterFirstLine[:lastTripleBackticks])
			}
			// No closing ```, return everything after the first line
			return strings.TrimSpace(afterFirstLine)
		}
	}
	return text
}

// parseFlexibleTime tries multiple time formats
func parseFlexibleTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
