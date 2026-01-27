package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
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
	Currency    string               `json:"currency,omitempty"`
}

type HotelListInput struct {
	Location  *ToolLocation `json:"location"`
	Rating    int           `json:"rating,omitempty" description:"Hotel rating (1-5)"`
	Amenities []string      `json:"amenities,omitempty" description:"List of amenities"`
}

type HotelOffersInput struct {
	HotelIDs []string `json:"hotel_ids"`
	Adults   int      `json:"adults"`
	CheckIn  string   `json:"check_in"`
	CheckOut string   `json:"check_out"`
	Currency string   `json:"currency,omitempty"`
}

type LocationInput struct {
	Keyword string `json:"keyword"`
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

func (t *FlightTool) Execute(ctx context.Context, input *FlightInput) ([]*pb.Transport, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "FlightTool executing with input: %s", string(inputJSON))

	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	if input == nil {
		return nil, fmt.Errorf("input required")
	}

	// Default adults if invalid
	adults := input.Adults
	if adults <= 0 {
		adults = 1
	}

	if input.Origin == nil || input.Destination == nil || input.Date == "" {
		return nil, fmt.Errorf("origin, destination (Location objects), and date are required")
	}

	// Use input currency if provided
	currency := input.Currency

	transport := &pb.Transport{
		Type:                pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		TravelerCount:       int32(adults),
		OriginLocation:      toPBOrigin(input.Origin),
		DestinationLocation: toPBDestination(input.Destination),
		Details: &pb.Transport_Flight{
			Flight: &pb.Flight{
				DepartureTime: timestamppb.New(parseDate(input.Date)),
			},
		},
	}

	if currency != "" {
		transport.Cost = &pb.Cost{Currency: currency}
	}

	resp, err := t.Client.SearchFlights(ctx, transport)

	if err != nil {
		log.Errorf(ctx, "FlightTool failed: %v", err)
		return nil, fmt.Errorf("flight search failed: %w", err)
	}

	log.Debugf(ctx, "FlightTool completed successfully. Found %d offers.", len(resp))
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
	registry.Register(genkit.DefineTool[*FlightInput, []*pb.Transport](
		gk,
		"amadeus_flight_tool",
		t.Description(),
		func(ctx *ai.ToolContext, input *FlightInput) ([]*pb.Transport, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		in := &FlightInput{}
		b, _ := json.Marshal(args)
		if err := json.Unmarshal(b, in); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, in)
	})
	return t
}

// HotelListTool implementation
type HotelListTool struct {
	Client *Client
}

func NewHotelListTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *HotelListTool {
	t := &HotelListTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*HotelListInput, *HotelListResponse](
		gk,
		"amadeus_hotel_list",
		"Searches for hotels in a specific city. Returns a list of hotels with IDs.",
		func(ctx *ai.ToolContext, input *HotelListInput) (*HotelListResponse, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		in := &HotelListInput{}
		b, _ := json.Marshal(args)
		if err := json.Unmarshal(b, in); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, in)
	})
	return t
}

func (t *HotelListTool) Execute(ctx context.Context, input *HotelListInput) (*HotelListResponse, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "HotelListTool executing with input: %s", string(inputJSON))

	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}
	if input == nil || input.Location == nil {
		return nil, fmt.Errorf("location is required")
	}

	acc := &pb.Accommodation{
		Location: toPBLocation(input.Location),
		Preferences: &pb.AccommodationPreferences{
			Rating:    int32(input.Rating),
			Amenities: input.Amenities,
		},
	}

	resp, err := t.Client.SearchHotelsByCity(ctx, acc)
	if err != nil {
		log.Errorf(ctx, "HotelListTool failed: %v", err)
		return nil, err
	}
	log.Debugf(ctx, "HotelListTool completed successfully. Found %d hotels.", len(resp.Data))
	return resp, nil
}

// HotelOffersTool implementation
type HotelOffersTool struct {
	Client *Client
}

func NewHotelOffersTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *HotelOffersTool {
	t := &HotelOffersTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*HotelOffersInput, []*pb.Accommodation](
		gk,
		"amadeus_hotel_offers",
		"Searches for offers for specific hotels. Requires hotel IDs (from hotel_list tool), check-in/out dates, and number of adults.",
		func(ctx *ai.ToolContext, input *HotelOffersInput) ([]*pb.Accommodation, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		in := &HotelOffersInput{}
		b, _ := json.Marshal(args)
		if err := json.Unmarshal(b, in); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, in)
	})
	return t
}

func (t *HotelOffersTool) Execute(ctx context.Context, input *HotelOffersInput) ([]*pb.Accommodation, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "HotelOffersTool executing with input: %s", string(inputJSON))

	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}
	if input == nil {
		return nil, fmt.Errorf("input required")
	}
	if len(input.HotelIDs) == 0 {
		return nil, fmt.Errorf("hotel_ids are required")
	}
	if input.CheckIn == "" || input.CheckOut == "" {
		return nil, fmt.Errorf("check_in and check_out dates are required")
	}

	adults := input.Adults
	if adults <= 0 {
		adults = 1
	}

	// Construct temporary accommodation object for the search
	acc := &pb.Accommodation{
		TravelerCount: int32(adults),
		CheckIn:       timestamppb.New(parseDate(input.CheckIn)),
		CheckOut:      timestamppb.New(parseDate(input.CheckOut)),
		Cost: &pb.Cost{
			Currency: input.Currency,
		},
		// Location info missing in this tool input context, so enrichment won't happen here
		// unless we change the tool input as well, but for now we match the signature.
	}

	resp, err := t.Client.SearchHotelOffers(ctx, input.HotelIDs, acc)
	if err != nil {
		log.Errorf(ctx, "HotelOffersTool failed: %v", err)
		return nil, err
	}
	log.Debugf(ctx, "HotelOffersTool completed successfully. Found %d offers.", len(resp))
	return resp, nil
}

// LocationTool implementation
type LocationTool struct {
	Client *Client
}

func (t *LocationTool) Name() string {
	return "location_tool"
}

func (t *LocationTool) Description() string {
	return "Searches for cities and airports. Arguments: keyword (string, e.g. 'Paris'). Returns a list of Location objects. Use full city/location name, instead of abbreviations."
}

func (t *LocationTool) Execute(ctx context.Context, input *LocationInput) ([]*pb.Location, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "LocationTool executing with input: %s", string(inputJSON))

	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	if input == nil || input.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	resp, err := t.Client.SearchLocations(ctx, input.Keyword)
	if err != nil {
		log.Errorf(ctx, "LocationTool failed: %v", err)
		return nil, err // Returning error as is
	}
	log.Debugf(ctx, "LocationTool completed successfully. Found %d locations.", len(resp))
	return resp, nil
}

// NewLocationTool initializes and registers the LocationTool
func NewLocationTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *LocationTool {
	t := &LocationTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[*LocationInput, []*pb.Location](
		gk,
		"amadeus_location_tool",
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
