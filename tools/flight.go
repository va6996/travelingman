package tools

import (
	"fmt"

	"example.com/travelingman/providers/amadeus"
)

type FlightTool struct {
	Client *amadeus.Client
}

func (t *FlightTool) Name() string {
	return "flight_tool"
}

func (t *FlightTool) Description() string {
	return "Searches for flights. Arguments: origin (IATA code), destination (IATA code), date (YYYY-MM-DD), adults (int)."
}

func (t *FlightTool) Execute(args map[string]interface{}) (interface{}, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	origin, _ := args["origin"].(string)
	destination, _ := args["destination"].(string)
	date, _ := args["date"].(string)
	adults, ok := args["adults"].(int)
	if !ok || adults == 0 {
		adults = 1 // default
	}

	if origin == "" || destination == "" || date == "" {
		return nil, fmt.Errorf("origin, destination, and date are required")
	}

	// Wrapper call
	// Note: Return Date is optional, we pass empty string for one-way by default unless tool arg supports returnDate
	resp, err := t.Client.SearchFlights(origin, destination, date, "", "", adults)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
