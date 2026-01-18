package amadeus

import (
	"context"
	"fmt"

	"github.com/va6996/travelingman/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Input definitions for Amadeus tools
type FlightInput struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Date        string `json:"date"`
	Adults      int    `json:"adults"`
}

type HotelInput struct {
	CityCode string `json:"city_code"`
	CheckIn  string `json:"check_in,omitempty"`
	CheckOut string `json:"check_out,omitempty"`
	Adults   int    `json:"adults,omitempty"`
}

// FlightTool implementation
type FlightTool struct {
	Client *Client
}

func (t *FlightTool) Name() string {
	return "flight_tool"
}

func (t *FlightTool) Description() string {
	return "Searches for flights. Arguments: origin (IATA code), destination (IATA code), date (YYYY-MM-DD), adults (int)."
}

func (t *FlightTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	origin, _ := args["origin"].(string)
	destination, _ := args["destination"].(string)
	date, _ := args["date"].(string)
	var adults int
	if val, ok := args["adults"].(int); ok {
		adults = val
	} else if val, ok := args["adults"].(float64); ok {
		adults = int(val)
	} else {
		adults = 1 // default
	}

	if origin == "" || destination == "" || date == "" {
		return nil, fmt.Errorf("origin, destination, and date are required")
	}

	resp, err := t.Client.SearchFlights(ctx, origin, destination, date, "", "", adults)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// NewFlightTool initializes and registers the FlightTool
func NewFlightTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *FlightTool {
	t := &FlightTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[FlightInput, string](
		gk,
		"flightTool",
		t.Description(),
		func(ctx *ai.ToolContext, input FlightInput) (string, error) {
			args := map[string]interface{}{
				"origin":      input.Origin,
				"destination": input.Destination,
				"date":        input.Date,
				"adults":      input.Adults,
			}
			res, err := t.Execute(ctx, args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%+v", res), nil
		},
	), t.Execute)
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
	return "Searches for hotels. Arguments: city_code (IATA code, e.g. NYC, PAR), check_in (YYYY-MM-DD), check_out (YYYY-MM-DD), adults (int)."
}

func (t *HotelTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	cityCode, _ := args["city_code"].(string)
	if cityCode == "" {
		return nil, fmt.Errorf("city_code is required")
	}

	// Step 1: Search hotels by city to get IDs
	listResp, err := t.Client.SearchHotelsByCity(ctx, cityCode)
	if err != nil {
		return nil, fmt.Errorf("search hotels by city failed: %v", err)
	}

	if len(listResp.Data) == 0 {
		return nil, fmt.Errorf("no hotels found for city code: %s", cityCode)
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

	checkIn, _ := args["check_in"].(string)
	checkOut, _ := args["check_out"].(string)
	var adults int
	if val, ok := args["adults"].(int); ok {
		adults = val
	} else if val, ok := args["adults"].(float64); ok {
		adults = int(val)
	} else {
		adults = 1
	}

	if checkIn == "" || checkOut == "" {
		return listResp, nil
	}

	offersResp, err := t.Client.SearchHotelOffers(ctx, hotelIds, adults, checkIn, checkOut)
	if err != nil {
		return map[string]interface{}{
			"warning": "offer search failed",
			"error":   err.Error(),
			"hotels":  listResp,
		}, nil
	}

	return offersResp, nil
}

// NewHotelTool initializes and registers the HotelTool
func NewHotelTool(c *Client, gk *genkit.Genkit, registry *tools.Registry) *HotelTool {
	t := &HotelTool{Client: c}
	if gk == nil || registry == nil {
		return t
	}
	registry.Register(genkit.DefineTool[HotelInput, string](
		gk,
		"hotelTool",
		t.Description(),
		func(ctx *ai.ToolContext, input HotelInput) (string, error) {
			args := map[string]interface{}{
				"city_code": input.CityCode,
				"check_in":  input.CheckIn,
				"check_out": input.CheckOut,
				"adults":    input.Adults,
			}
			res, err := t.Execute(ctx, args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%+v", res), nil
		},
	), t.Execute)
	return t
}
