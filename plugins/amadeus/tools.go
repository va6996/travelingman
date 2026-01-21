package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/tools"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToolLocation is a simplified location struct for tool inputs to ensure valid schema generation
type ToolLocation struct {
	City      string   `json:"city,omitempty"`
	Country   string   `json:"country,omitempty"`
	IataCodes []string `json:"iata_codes,omitempty"`
	CityCode  string   `json:"city_code,omitempty"`
}

// Distinct types to avoid schema generation issues with repeated types
type OriginLocation ToolLocation
type DestinationLocation ToolLocation

// Input definitions for Amadeus tools
type FlightInput struct {
	Origin      *OriginLocation      `json:"origin"`
	Destination *DestinationLocation `json:"destination"`
	Date        string               `json:"date"`
	Adults      int                  `json:"adults"`
}

type HotelInput struct {
	Location *ToolLocation `json:"location"`
	CheckIn  string        `json:"check_in,omitempty"`
	CheckOut string        `json:"check_out,omitempty"`
	Adults   int           `json:"adults,omitempty"`
}

type LocationInput struct {
	Keyword string `json:"keyword"`
}

type HotelToolOutput struct {
	Hotels  *HotelListResponse   `json:"hotels,omitempty"`
	Offers  *HotelSearchResponse `json:"offers,omitempty"`
	Warning string               `json:"warning,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// Helper to convert ToolLocation to pb.Location
func toPBLocation(l *ToolLocation) *pb.Location {
	if l == nil {
		return nil
	}
	return &pb.Location{
		City:      l.City,
		Country:   l.Country,
		IataCodes: l.IataCodes,
		CityCode:  l.CityCode,
	}
}

func toPBOrigin(l *OriginLocation) *pb.Location {
	if l == nil {
		return nil
	}
	tl := ToolLocation(*l)
	return toPBLocation(&tl)
}

func toPBDestination(l *DestinationLocation) *pb.Location {
	if l == nil {
		return nil
	}
	tl := ToolLocation(*l)
	return toPBLocation(&tl)
}

// FlightTool implementation
type FlightTool struct {
	Client *Client
}

func (t *FlightTool) Name() string {
	return "flight_tool"
}

func (t *FlightTool) Description() string {
	return "Searches for flights. Arguments: origin (Location object), destination (Location object), date (YYYY-MM-DD), adults (int). Use the full Location objects returned by locationTool."
}

func (t *FlightTool) Execute(ctx context.Context, input *FlightInput) (*FlightSearchResponse, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	if input == nil {
		return nil, fmt.Errorf("input required")
	}

	inputJSON, _ := json.Marshal(input)
	fmt.Printf("[DEBUG] FlightTool Executing with input: %s\n", string(inputJSON))

	if input.Origin == nil || input.Destination == nil || input.Date == "" {
		return nil, fmt.Errorf("origin, destination (Location objects), and date are required")
	}

	adults := input.Adults
	if adults <= 0 {
		adults = 1
	}

	resp, err := t.Client.SearchFlights(ctx, &pb.Transport{
		Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		TravelerCount:       int32(adults),
		OriginLocation:      toPBOrigin(input.Origin),
		DestinationLocation: toPBDestination(input.Destination),
		Details: &pb.Transport_Flight{
			Flight: &pb.Flight{
				DepartureTime: timestamppb.New(parseDate(input.Date)),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("flight search failed: %w", err)
	}

	return resp, nil
}

func parseDate(d string) time.Time {
	t, _ := time.Parse("2006-01-02", d)
	return t
}

// NewFlightTool initializes and registers the FlightTool
func NewFlightTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *FlightTool {
	t := &FlightTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*FlightInput, *FlightSearchResponse](
		gk,
		"flightTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *FlightInput) (*FlightSearchResponse, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		in := &FlightInput{}
		// Use JSON roundtrip for robust mapping of complex objects
		b, _ := json.Marshal(args)
		if err := json.Unmarshal(b, in); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, in)
	})
	return t
}

// HotelTool implementation
type HotelTool struct {
	Client *Client
}

func (t *HotelTool) Name() string {
	return "hotel_tool"
}

func (t *HotelTool) Description() string {
	return "Searches for hotels. Arguments: location (Location object from locationTool), check_in (YYYY-MM-DD), check_out (YYYY-MM-DD), adults (int)."
}

func (t *HotelTool) Execute(ctx context.Context, input *HotelInput) (*HotelToolOutput, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}
	if input == nil {
		return nil, fmt.Errorf("input required")
	}

	inputJSON, _ := json.Marshal(input)
	fmt.Printf("[DEBUG] HotelTool Executing with input: %s\n", string(inputJSON))

	if input.Location == nil {
		return nil, fmt.Errorf("location object is required")
	}

	// Extract city code from Location object
	cityCode := ""
	if len(input.Location.IataCodes) > 0 {
		cityCode = input.Location.IataCodes[0]
	} else {
		cityCode = input.Location.CityCode
	}

	if cityCode == "" {
		return nil, fmt.Errorf("location has no valid city code or IATA code")
	}

	// Update input with simple city code if needed internally, but we use the Location object now.
	// Actually, searchHotelsByCity needs simple code or we pass Location?
	// Client.SearchHotelsByCity signature likely needs Update too, or we construct the proto accurately.

	adults := input.Adults
	if adults <= 0 {
		adults = 1
	}

	// Passing Location object to client helper
	return t.executeOnce(ctx, input, adults, cityCode)
}

func (t *HotelTool) executeOnce(ctx context.Context, input *HotelInput, adults int, cityCode string) (*HotelToolOutput, error) {
	// Step 1: Search hotels by city
	listResp, err := t.Client.SearchHotelsByCity(ctx, &pb.Accommodation{
		Location:      toPBLocation(input.Location), // Pass full location
		TravelerCount: int32(adults),
	})
	if err != nil {
		return nil, fmt.Errorf("search hotels by city failed: %v", err)
	}

	if len(listResp.Data) == 0 {
		return nil, fmt.Errorf("no hotels found for city code: %s", cityCode)
	}

	if input.CheckIn == "" || input.CheckOut == "" {
		return &HotelToolOutput{Hotels: listResp}, nil
	}

	// Step 2: Pick top 15 hotels and search for offers
	limit := 15
	if len(listResp.Data) < limit {
		limit = len(listResp.Data)
	}

	var hotelIds []string
	for i := 0; i < limit; i++ {
		hotelIds = append(hotelIds, listResp.Data[i].HotelId)
	}

	offersResp, err := t.Client.SearchHotelOffers(ctx, hotelIds, adults, input.CheckIn, input.CheckOut)
	if err != nil {
		return &HotelToolOutput{
			Hotels:  listResp,
			Warning: "offer search failed",
			Error:   err.Error(),
		}, nil
	}

	return &HotelToolOutput{Offers: offersResp}, nil
}

// NewHotelTool initializes and registers the HotelTool
func NewHotelTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *HotelTool {
	t := &HotelTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*HotelInput, *HotelToolOutput](
		gk,
		"hotelTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *HotelInput) (*HotelToolOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		in := &HotelInput{}
		b, _ := json.Marshal(args)
		if err := json.Unmarshal(b, in); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, in)
	})
	return t
}

// LocationTool implementation
type LocationTool struct {
	Client *Client
}

func (t *LocationTool) Name() string {
	return "location_tool"
}

func (t *LocationTool) Description() string {
	return "Searches for cities and airports. Arguments: keyword (string, e.g. 'Paris'). Returns a list of Location objects."
}

func (t *LocationTool) Execute(ctx context.Context, input *LocationInput) ([]*pb.Location, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	if input == nil || input.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	inputJSON, _ := json.Marshal(input)
	fmt.Printf("[DEBUG] LocationTool Executing with input: %s\n", string(inputJSON))

	return t.Client.SearchLocations(ctx, input.Keyword)
}

// NewLocationTool initializes and registers the LocationTool
func NewLocationTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *LocationTool {
	t := &LocationTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*LocationInput, []*pb.Location](
		gk,
		"locationTool",
		t.Description(),
		func(ctx *ai.ToolContext, input *LocationInput) ([]*pb.Location, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		keyword, ok := args["keyword"].(string)
		if !ok {
			return nil, fmt.Errorf("keyword is required")
		}
		return t.Execute(ctx, &LocationInput{Keyword: keyword})
	})
	return t
}
