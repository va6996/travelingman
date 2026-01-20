package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/tools"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TripPlanner is responsible for high-level planning
type TripPlanner struct {
	genkit   *genkit.Genkit
	registry *tools.Registry
	model    ai.Model
}

// NewTripPlanner creates a new TripPlanner
func NewTripPlanner(gk *genkit.Genkit, registry *tools.Registry, model ai.Model) *TripPlanner {
	return &TripPlanner{
		genkit:   gk,
		registry: registry,
		model:    model,
	}
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

// Internal structs for JSON unmarshalling
type JSONItinerary struct {
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	StartTime     string          `json:"start_time"`
	EndTime       string          `json:"end_time"`
	Travelers     int32           `json:"travelers"`
	Accommodation []Accommodation `json:"accommodation"`
	Transport     []Transport     `json:"transport"`
}

type JSONPlanResponse struct {
	Itinerary   *JSONItinerary  `json:"itinerary"`
	Itineraries []JSONItinerary `json:"itineraries"`
	Reasoning   string          `json:"reasoning"`
	Question    string          `json:"question"`
	NeedClarify bool            `json:"need_clarification"`
}

type Accommodation struct {
	Name          string   `json:"name"`
	CityCode      string   `json:"city_code"`
	CheckIn       string   `json:"check_in"`
	CheckOut      string   `json:"check_out"`
	TravelerCount int32    `json:"traveler_count"`
	RoomType      string   `json:"room_type"`
	Area          string   `json:"area"`
	Rating        int32    `json:"rating"`
	Amenities     []string `json:"amenities"`
}

type Transport struct {
	Type          string         `json:"type"`
	TravelerCount int32          `json:"traveler_count"`
	Class         string         `json:"class"` // ECONOMY, BUSINESS, etc.
	Flight        *FlightDetails `json:"flight"`
}

type FlightDetails struct {
	DepAirport string `json:"departure_airport"`
	ArrAirport string `json:"arrival_airport"`
	DepTime    string `json:"departure_time"`
	ArrTime    string `json:"arrival_time"`
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

// Plan generates a high-level itinerary based on the user query
func (p *TripPlanner) Plan(ctx context.Context, req PlanRequest) (*PlanResult, error) {
	// ReAct Loop
	maxSteps := 15
	chatHistory := req.History

	// Define tools available
	var toolsDefs []string
	if p.registry != nil {
		for _, tool := range p.registry.GetTools() {
			def := tool.Definition()
			// Format tool with its input schema
			toolDef := fmt.Sprintf("- %s: %s", def.Name, def.Description)

			// Add input schema if available
			if def.InputSchema != nil {
				schemaJSON, err := json.MarshalIndent(def.InputSchema, "  ", "  ")
				if err == nil {
					toolDef += fmt.Sprintf("\n  Input Schema: %s", string(schemaJSON))
				}
			}
			toolsDefs = append(toolsDefs, toolDef)
		}
	}
	toolsDef := strings.Join(toolsDefs, "\n")

	basePrompt := fmt.Sprintf(`You are an expert Trip Planner. Your goal is to create a high-level travel itinerary.
You have access to tools. You must use them to get accurate information (dates, etc) before generating the final plan.

User Query: %s
Current Time: %s

Available Tools:
%s

Instructions:
1. Analyze the request. 
2. If you need to calculate dates (e.g. "next weekend", "next friday"), USE the dateTool. Do NOT guess.
3. You can call MULTIPLE tools in parallel by returning a list of JSON objects.
   Example: [{"tool": "dateTool", "args": {"expression": "..."}}, {"tool": "flightTool", "args": {"origin": "NYC", ...}}]
4. If you have enough info, return the FINAL PLAN as a JSON object with "itinerary" (or "itineraries" for multiple options) or "question".

Final Answer Schema:
{
  "itineraries": [ { ... }, { ... } ], // For multiple options (e.g. different weekends)
  "reasoning": "..."
}
OR
{ "question": "...", "reasoning": "..." }

CRITICAL INSTRUCTIONS:
- Return ONLY valid JSON.
- DO NOT generate Python, JavaScript, or any other code. 
- DO NOT define functions like 'def hotelTool(...)'.
- DO NOT use variables. Use actual values in args.
- Just output the JSON object or list of objects.`, req.UserQuery, time.Now().Format(time.RFC3339), toolsDef)

	var lastItin *JSONItinerary
	var lastItineraries []JSONItinerary
	var finalRes JSONPlanResponse

	for step := 0; step < maxSteps; step++ {
		log.Printf("TripPlanner: Step %d/%d", step+1, maxSteps)

		// Construct dynamic prompt with history
		stepPrompt := basePrompt
		if chatHistory != "" {
			stepPrompt += fmt.Sprintf("\n\nHistory:\n%s", chatHistory)
		}

		log.Println("TripPlanner: Sending prompt to LLM...")

		// Use Genkit's Generate API
		resp, err := genkit.Generate(ctx,
			p.genkit,
			ai.WithModel(p.model),
			ai.WithPrompt(stepPrompt),
		)
		if err != nil {
			log.Printf("TripPlanner: LLM error: %v", err)
			return nil, fmt.Errorf("planning step failed: %w", err)
		}

		text := resp.Text()
		log.Printf("[DEBUG] LLM Raw Response: %s", text)
		extractedJSON := extractUsageJSON(text)
		if extractedJSON != "" {
			text = extractedJSON
		}

		// Try to parse as Tool Call (List or Single)
		// We define a struct to hold tool call data
		type ToolCall struct {
			Tool string                 `json:"tool"`
			Args map[string]interface{} `json:"args"`
		}

		var toolCalls []ToolCall
		var singleToolCall ToolCall
		isToolCall := false

		// Try parsing as list first
		if err := json.Unmarshal([]byte(text), &toolCalls); err == nil && len(toolCalls) > 0 {
			isToolCall = true
		} else if err := json.Unmarshal([]byte(text), &singleToolCall); err == nil && singleToolCall.Tool != "" {
			// If single object, wrap in list
			toolCalls = []ToolCall{singleToolCall}
			isToolCall = true
		}

		if isToolCall {
			log.Printf("TripPlanner: Found %d tool calls", len(toolCalls))

			// Execute tools sequentially
			type toolResult struct {
				idx    int
				output string
				param  ToolCall
			}
			results := make([]toolResult, len(toolCalls))

			for i, tc := range toolCalls {
				log.Printf("TripPlanner: Executing tool %s args: %v", tc.Tool, tc.Args)
				res, err := p.registry.ExecuteTool(ctx, tc.Tool, tc.Args)
				var outStr string
				if err != nil {
					outStr = fmt.Sprintf("Error: %v", err)
					log.Printf("TripPlanner: Tool %s execution failed: %v", tc.Tool, err)
				} else {
					b, err := json.Marshal(res)
					if err != nil {
						outStr = fmt.Sprintf("Tool call returned %v", res)
					} else {
						outStr = string(b)
					}
				}
				results[i] = toolResult{idx: i, output: outStr, param: tc}
			}

			// Append to history
			for _, r := range results {
				log.Printf("TripPlanner: Tool Output [%d]: %s", r.idx, r.output)
				chatHistory += fmt.Sprintf("\nAssistant: Call %s(%v)\nSystem: Output: %s", r.param.Tool, r.param.Args, r.output)
			}
			continue
		}

		// Try to parse as Final Response
		if err := json.Unmarshal([]byte(text), &finalRes); err == nil && (finalRes.Itinerary != nil || len(finalRes.Itineraries) > 0 || finalRes.Question != "") {
			log.Println("TripPlanner: Parsed Final Response.")
			lastItin = finalRes.Itinerary
			lastItineraries = finalRes.Itineraries
			break
		}

		// If neither, invalid or reasoning
		log.Printf("TripPlanner: Could not parse response, assuming reasoning or failure: %s", text)
		chatHistory += fmt.Sprintf("\nAssistant: Output: %s", text)
	}

	result := &PlanResult{
		Reasoning: finalRes.Reasoning,
	}

	if finalRes.NeedClarify || finalRes.Question != "" {
		result.NeedsClarification = true
		result.Question = finalRes.Question
		return result, nil
	}

	output := finalRes
	output.Itinerary = lastItin // Ensure we have the itinerary if parsed

	// Handle single itinerary
	if output.Itinerary != nil {
		p.resolveCityCodes(ctx, output.Itinerary)
		result.Itinerary = p.convertItinerary(output.Itinerary)
	}

	// Handle multiple itineraries
	if len(lastItineraries) > 0 {
		for i := range lastItineraries {
			p.resolveCityCodes(ctx, &lastItineraries[i])
			if pbItin := p.convertItinerary(&lastItineraries[i]); pbItin != nil {
				result.PossibleItineraries = append(result.PossibleItineraries, pbItin)
			}
		}
		// If both single and multiple are present, usually prefer one or merge.
		// For now if multiple exist, let's treat the first one as "main" if not set,
		// or just rely on PossibleItineraries in the caller.
		if result.Itinerary == nil && len(result.PossibleItineraries) > 0 {
			result.Itinerary = result.PossibleItineraries[0]
		}
	}

	return result, nil
}

// convertItinerary maps the internal JSON itinerary to packet buffer format
func (p *TripPlanner) convertItinerary(in *JSONItinerary) *pb.Itinerary {
	if in == nil {
		return nil
	}
	itin := &pb.Itinerary{
		Title:       in.Title,
		Description: in.Description,
		Travelers:   in.Travelers,
		Graph:       &pb.Graph{},
	}

	if t, err := time.Parse(time.RFC3339, in.StartTime); err == nil {
		itin.StartTime = timestamppb.New(t)
	}
	if t, err := time.Parse(time.RFC3339, in.EndTime); err == nil {
		itin.EndTime = timestamppb.New(t)
	}

	// Map Accommodation to Nodes
	for i, acc := range in.Accommodation {
		node := &pb.Node{
			Id: fmt.Sprintf("node_acc_%d", i),
			// Node.Location is a string, keep it for general identification if needed, or remove if redundant.
			// For now, setting it to CityCode.
			Location: acc.CityCode,
			Stay: &pb.Accommodation{
				Name: acc.Name,
				Location: &pb.Location{
					CityCode: acc.CityCode,
				},
			},
		}
		// Parse check-in date
		if acc.CheckIn != "" {
			if t, err := parseFlexibleTime(acc.CheckIn); err == nil {
				node.Stay.CheckIn = timestamppb.New(t)
				node.FromTimestamp = timestamppb.New(t)
			} else {
				log.Printf("TripPlanner: Warning - failed to parse check-in date '%s': %v", acc.CheckIn, err)
			}
		}
		// Parse check-out date
		if acc.CheckOut != "" {
			if t, err := parseFlexibleTime(acc.CheckOut); err == nil {
				node.Stay.CheckOut = timestamppb.New(t)
				node.ToTimestamp = timestamppb.New(t)
			} else {
				log.Printf("TripPlanner: Warning - failed to parse check-out date '%s': %v", acc.CheckOut, err)
			}
		}

		// Preferences
		node.Stay.Preferences = &pb.AccommodationPreferences{
			RoomType:  acc.RoomType,
			Area:      acc.Area,
			Rating:    acc.Rating,
			Amenities: acc.Amenities,
		}
		// Use item-level traveler_count if provided, otherwise use itinerary-level
		if acc.TravelerCount > 0 {
			node.Stay.TravelerCount = acc.TravelerCount
		} else {
			node.Stay.TravelerCount = in.Travelers
		}

		itin.Graph.Nodes = append(itin.Graph.Nodes, node)
	}

	// Map Transport to Edges
	for i, tr := range in.Transport {
		edge := &pb.Edge{
			Transport: &pb.Transport{},
		}

		// Use item-level traveler_count if provided, otherwise use itinerary-level
		if tr.TravelerCount > 0 {
			edge.Transport.TravelerCount = tr.TravelerCount
		} else {
			edge.Transport.TravelerCount = in.Travelers
		}

		if tr.Type == "TRANSPORT_TYPE_FLIGHT" {
			edge.Transport.Type = pb.TransportType_TRANSPORT_TYPE_FLIGHT
			if tr.Flight != nil {
				edge.Transport.OriginLocation = &pb.Location{
					IataCodes: []string{tr.Flight.DepAirport},
				}
				edge.Transport.DestinationLocation = &pb.Location{
					IataCodes: []string{tr.Flight.ArrAirport},
				}

				edge.Transport.FlightPreferences = &pb.FlightPreferences{
					TravelClass: mapClass(tr.Class),
				}
				f := &pb.Flight{}
				// Parse departure time with multiple format support
				if tr.Flight.DepTime != "" {
					if tm, err := parseFlexibleTime(tr.Flight.DepTime); err == nil {
						f.DepartureTime = timestamppb.New(tm)
					} else {
						log.Printf("TripPlanner: Warning - failed to parse departure time '%s': %v", tr.Flight.DepTime, err)
					}
				}
				// Parse arrival time with multiple format support
				if tr.Flight.ArrTime != "" {
					if tm, err := parseFlexibleTime(tr.Flight.ArrTime); err == nil {
						f.ArrivalTime = timestamppb.New(tm)
					} else {
						log.Printf("TripPlanner: Warning - failed to parse arrival time '%s': %v", tr.Flight.ArrTime, err)
					}
				}
				edge.Transport.Details = &pb.Transport_Flight{Flight: f}
			}
		} else {
			edge.Transport.Type = pb.TransportType_TRANSPORT_TYPE_UNSPECIFIED
		}

		// Simple linking for now
		if i > 0 && i <= len(itin.Graph.Nodes) {
			edge.FromId = itin.Graph.Nodes[i-1].Id
		}
		if i < len(itin.Graph.Nodes) {
			edge.ToId = itin.Graph.Nodes[i].Id
		}

		itin.Graph.Edges = append(itin.Graph.Edges, edge)
	}

	// Populate options by re-running search (post-processing)
	p.populateOptions(context.Background(), itin)

	return itin
}

// parseFlexibleTime attempts to parse time strings in multiple formats
func parseFlexibleTime(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}

// resolveCityCodes updates the itinerary with correct IATA codes
func (p *TripPlanner) resolveCityCodes(ctx context.Context, itin *JSONItinerary) {
	if p.registry == nil {
		return
	}

	type resolutionResult struct {
		index int
		code  string
	}

	// Use a channel to collect results from goroutines
	results := make(chan resolutionResult, len(itin.Accommodation))
	var wg sync.WaitGroup

	// Heuristic: if CityCode looks like a name (> 3 chars or lowercase), try to resolve it
	for i := range itin.Accommodation {
		acc := &itin.Accommodation[i]
		if len(acc.CityCode) > 3 || (len(acc.CityCode) == 3 && acc.CityCode != "NYC" && acc.CityCode != "LON") { // Basic check
			wg.Add(1)
			go func(idx int, cityCode string) {
				defer wg.Done()
				log.Printf("TripPlanner: Resolving city code for '%s'", cityCode)

				// Create a derived context with timeout for individual lookups to avoid hanging
				tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				res, err := p.registry.ExecuteTool(tCtx, "locationTool", map[string]interface{}{
					"keyword": cityCode,
				})
				if err != nil {
					log.Printf("TripPlanner: Location search failed for %s: %v", cityCode, err)
					return
				}

				// Map result
				resBytes, _ := json.Marshal(res)
				var searchResp struct {
					Data []struct {
						JobCode string `json:"iataCode"`
					} `json:"data"`
				}
				if err := json.Unmarshal(resBytes, &searchResp); err == nil && len(searchResp.Data) > 0 {
					log.Printf("TripPlanner: Resolved '%s' to '%s'", cityCode, searchResp.Data[0].JobCode)
					results <- resolutionResult{index: idx, code: searchResp.Data[0].JobCode}
				}
			}(i, acc.CityCode)
		}
	}

	// Closer goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Apply results
	for res := range results {
		itin.Accommodation[res.index].CityCode = res.code
	}
}

// extractUsageJSON attempts to find the first valid JSON object or list in a string.
func extractUsageJSON(text string) string {
	// Look for first '{' or '['
	startObj := strings.Index(text, "{")
	startArr := strings.Index(text, "[")

	start := -1
	if startObj != -1 && startArr != -1 {
		if startObj < startArr {
			start = startObj
		} else {
			start = startArr
		}
	} else if startObj != -1 {
		start = startObj
	} else if startArr != -1 {
		start = startArr
	}

	if start == -1 {
		return ""
	}

	trimmed := strings.TrimSpace(text[start:])
	// remove trailing ; if present
	if strings.HasSuffix(trimmed, ";") {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed
	}

	// Heuristic: matching brackets
	balance := 0
	foundStart := false
	openChar := '{'
	closeChar := '}'
	if text[start] == '[' {
		openChar = '['
		closeChar = ']'
	}

	for i, r := range text {
		if i < start {
			continue
		}
		if r == openChar {
			balance++
			foundStart = true
		} else if r == closeChar {
			balance--
		}

		if foundStart && balance == 0 {
			// Potential end
			candidate := text[start : i+1]
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return ""
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

					log.Printf("TripPlanner: Fetching flight options for %s -> %s on %s", o, d, dt)
					res, err := p.registry.ExecuteTool(tCtx, "flightTool", map[string]interface{}{
						"origin":        o,
						"destination":   d,
						"departureDate": dt,
						"adults":        int(e.Transport.TravelerCount),
					})
					if err != nil {
						log.Printf("TripPlanner: Flight search failed: %v", err)
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
						log.Printf("TripPlanner: Added %d flight options", len(options))
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

				log.Printf("TripPlanner: Fetching hotel options for %s (%s to %s)", c, ci, co)
				res, err := p.registry.ExecuteTool(tCtx, "hotelTool", map[string]interface{}{
					"cityCode":     c,
					"checkInDate":  ci,
					"checkOutDate": co,
					"adults":       int(n.Stay.TravelerCount),
				})
				if err != nil {
					log.Printf("TripPlanner: Hotel search failed: %v", err)
					return
				}

				if resp, ok := res.(*amadeus.HotelSearchResponse); ok {
					var options []*pb.Accommodation
					for _, data := range resp.Data {
						options = append(options, data.ToAccommodations()...)
					}
					if len(options) > 10 {
						options = options[:10]
					}
					n.StayOptions = options
					log.Printf("TripPlanner: Added %d stay options", len(options))
				} else if _, ok := res.(*amadeus.HotelListResponse); ok {
					log.Printf("TripPlanner: Received HotelListResponse, skipping options population for now as offers are required.")
				}
			}(node, city, checkIn, checkOut)
		}
	}

	wg.Wait()
}
