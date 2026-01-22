package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
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
   - Use dateTool to calculate dates (e.g., "next weekend" → actual dates like "2026-01-25")

2. Then, create the itinerary JSON with the gathered information:
   - DO NOT call hotelTool or flightTool - these are for the TravelDesk, not for planning
   - Just return the itinerary json with destination, dates, and activities
   - Use the actual dates you calculated, not relative terms

CRITICAL RULES:
- If the user specifies a timeframe (like "next weekend"), use dateTool to calculate it, then create the itinerary
- Return itinerary with ACTUAL dates in RFC3339 format (YYYY-MM-DDTHH:mm:ssZ)
- Structure your response exactly as the JSON schema below. Use camelCase for keys.

Final Answer Schema:
{
  "itineraries": [
    {
      "title": "Weekend in Paris",
      "description": "A wonderful weekend trip to Paris visiting key landmarks.",
      "startTime": "2026-01-25T10:00:00Z",
      "endTime": "2026-01-27T18:00:00Z",
      "travelers": 2,
      "graph": {
        "nodes": [
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

	// Increase timeout for the tool calls
	// We wrap the context with a longer timeout if it's not already long enough
	// But `genkit.Generate` uses the passed context.
	// The user reported "context canceled", which likely means the parent context or a default timeout was hit.
	// We'll use a generous timeout here.
	tCtx, cancel := context.WithTimeout(ctx, 220*time.Second) // 2 minutes
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
			for i := range finalAnswer.Itineraries {
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

// resolveCityCodes updates the itinerary with correct IATA codes
func (p *TripPlanner) resolveCityCodes(ctx context.Context, itin *pb.Itinerary) {
	if p.registry == nil {
		return
	}

	type resolutionResult struct {
		index int
		code  string
	}

	// Use a channel to collect results from goroutines
	results := make(chan resolutionResult, len(itin.Graph.Nodes))
	var wg sync.WaitGroup

	// Heuristic: if CityCode looks like a name (> 3 chars or lowercase), try to resolve it
	for i := range itin.Graph.Nodes {
		node := itin.Graph.Nodes[i]
		if node.Stay == nil || node.Stay.Location == nil {
			continue
		}

		cityCode := node.Stay.Location.CityCode
		// Basic check: if it looks like a city name rather than a code
		if len(cityCode) > 3 || (len(cityCode) == 3 && cityCode != strings.ToUpper(cityCode)) {
			wg.Add(1)
			go func(idx int, kw string) {
				defer wg.Done()
				log.Debugf(ctx, "TripPlanner: Resolving city code for '%s'", kw)

				// Create a derived context with timeout for individual lookups to avoid hanging
				tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				res, err := p.registry.ExecuteTool(tCtx, "amadeus_location_tool", map[string]interface{}{
					"keyword": kw,
				})
				if err != nil {
					log.Errorf(ctx, "TripPlanner: Location search failed for %s: %v", kw, err)
					return
				}

				// Map result - tool returns []*pb.Location directly now?
				// Let's check tool definition. Yes, returns []*pb.Location.
				// But ExecuteTool returns interface{}, so we need to cast or marshal/unmarshal.
				// Since we are inside the same process using Genkit local runner, it might return the struct.
				// However, to be safe and consistent with previous patterns:

				if locations, ok := res.([]*pb.Location); ok && len(locations) > 0 {
					if len(locations[0].IataCodes) > 0 {
						log.Debugf(ctx, "TripPlanner: Resolved '%s' to '%s'", kw, locations[0].IataCodes[0])
						results <- resolutionResult{index: idx, code: locations[0].IataCodes[0]}
					}
				} else {
					// Fallback using JSON roundtrip if it's map[string]interface{}
					b, _ := json.Marshal(res)
					var locs []*pb.Location
					if err := json.Unmarshal(b, &locs); err == nil && len(locs) > 0 && len(locs[0].IataCodes) > 0 {
						log.Debugf(ctx, "TripPlanner: Resolved '%s' to '%s'", kw, locs[0].IataCodes[0])
						results <- resolutionResult{index: idx, code: locs[0].IataCodes[0]}
					}
				}
			}(i, cityCode)
		}
	}

	// Closer goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Apply results
	for res := range results {
		if itin.Graph.Nodes[res.index].Stay != nil && itin.Graph.Nodes[res.index].Stay.Location != nil {
			itin.Graph.Nodes[res.index].Stay.Location.CityCode = res.code
			// Also update the IataCodes list if empty
			if len(itin.Graph.Nodes[res.index].Stay.Location.IataCodes) == 0 {
				itin.Graph.Nodes[res.index].Stay.Location.IataCodes = []string{res.code}
			}
		}
	}
}

// populateOptions fetches live options for transport and accommodation
func (p *TripPlanner) populateOptions(ctx context.Context, itin *pb.Itinerary) {
	if p.registry == nil || itin.Graph == nil {
		return
	}

	var wg sync.WaitGroup

	// Helper to extract first IATA code or CityCode
	getLoc := func(l *pb.Location) string {
		if l == nil {
			return ""
		}
		if len(l.IataCodes) > 0 {
			return l.IataCodes[0]
		}
		return l.CityCode
	}

	// 1. Transport Options
	for _, edge := range itin.Graph.Edges {
		if edge.Transport == nil {
			continue
		}
		if edge.Transport.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
			// Construct input for flightTool
			origin := getLoc(edge.Transport.OriginLocation)
			dest := getLoc(edge.Transport.DestinationLocation)
			date := ""
			if edge.Transport.FlightPreferences != nil {
				// use transport details if available (e.g. Flight.DepartureTime)
				if f := edge.Transport.GetFlight(); f != nil {
					date = f.DepartureTime.AsTime().Format("2006-01-02")
				}
			}

			if origin != "" && dest != "" && date != "" {
				wg.Add(1)
				go func(e *pb.Edge, o, d, dt string) {
					defer wg.Done()

					// Timeout for API call
					tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					defer cancel()

					log.Debugf(ctx, "TripPlanner: Fetching flight options for %s -> %s on %s", o, d, dt)
					res, err := p.registry.ExecuteTool(tCtx, "amadeus_flight_tool", map[string]interface{}{
						"origin": map[string]interface{}{
							"iata_codes": []string{o},
						},
						"destination": map[string]interface{}{
							"iata_codes": []string{d},
						},
						"date":   dt,
						"adults": int(e.Transport.TravelerCount),
					})
					if err != nil {
						log.Errorf(ctx, "TripPlanner: Flight search failed: %v", err)
						return
					}

					// Cast result
					if resp, ok := res.(*amadeus.FlightSearchResponse); ok {
						var options []*pb.Transport
						for _, offer := range resp.Data {
							options = append(options, offer.ToTransport())
						}
						// Limit options to avoid massive proto payload
						if len(options) > 10 {
							options = options[:10]
						}
						// Protected modification if needed, but since we modify unique edge per goroutine, it's generally safe
						// as long as we don't resize the slice header concurrently in a way that affects others.
						// Edges is a slice of pointers, so modifying the pointed-to struct is safe.
						e.TransportOptions = options
						log.Debugf(ctx, "TripPlanner: Added %d flight options", len(options))
					}
				}(edge, origin, dest, date)
			}
		}
	}

	// 2. Accommodation Options
	for _, node := range itin.Graph.Nodes {
		if node.Stay == nil {
			continue
		}
		// HotelTool args: cityCode, checkInDate, checkOutDate ...
		city := getLoc(node.Stay.Location)
		if city == "" {
			// Fallback to node location string if it looks like city code
			if len(node.Location) == 3 {
				city = node.Location
			}
		}
		if city == "" {
			continue
		}

		checkIn := ""
		checkOut := ""
		if node.Stay.CheckIn != nil {
			checkIn = node.Stay.CheckIn.AsTime().Format("2006-01-02")
		}
		if node.Stay.CheckOut != nil {
			checkOut = node.Stay.CheckOut.AsTime().Format("2006-01-02")
		}

		if city != "" && checkIn != "" && checkOut != "" {
			wg.Add(1)
			go func(n *pb.Node, c, ci, co string) {
				defer wg.Done()

				tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				log.Debugf(ctx, "TripPlanner: Fetching hotel options for %s (%s to %s)", c, ci, co)

				// Step 1: List hotels
				listRes, err := p.registry.ExecuteTool(tCtx, "amadeus_hotel_list", map[string]interface{}{
					"location": map[string]interface{}{
						"iata_codes": []string{c},
					},
				})
				if err != nil {
					log.Errorf(ctx, "TripPlanner: Hotel list search failed: %v", err)
					return
				}

				listResp, ok := listRes.(*amadeus.HotelListResponse)
				if !ok || len(listResp.Data) == 0 {
					log.Warnf(ctx, "TripPlanner: No hotels found for %s", c)
					return
				}

				// Step 2: Get Offers
				var hotelIds []string
				limit := 5
				if len(listResp.Data) < limit {
					limit = len(listResp.Data)
				}
				for i := 0; i < limit; i++ {
					hotelIds = append(hotelIds, listResp.Data[i].HotelId)
				}

				offersRes, err := p.registry.ExecuteTool(tCtx, "amadeus_hotel_offers", map[string]interface{}{
					"hotel_ids": hotelIds,
					"check_in":  ci,
					"check_out": co,
					"adults":    int(n.Stay.TravelerCount),
				})
				if err != nil {
					log.Errorf(ctx, "TripPlanner: Hotel offers search failed: %v", err)
					return
				}

				if resp, ok := offersRes.(*amadeus.HotelSearchResponse); ok {
					var options []*pb.Accommodation
					for _, data := range resp.Data {
						options = append(options, data.ToAccommodations()...)
					}
					if len(options) > 10 {
						options = options[:10]
					}
					n.StayOptions = options
					log.Debugf(ctx, "TripPlanner: Added %d stay options", len(options))
				}
			}(node, city, checkIn, checkOut)
		}
	}

	wg.Wait()
}

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
