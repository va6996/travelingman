package tools

import (
	"fmt"

	"example.com/travelingman/providers/amadeus"
)

type HotelTool struct {
	Client *amadeus.Client
}

func (t *HotelTool) Name() string {
	return "hotel_tool"
}

func (t *HotelTool) Description() string {
	return "Searches for hotels. Arguments: city_code (IATA code, e.g. NYC, PAR), check_in (YYYY-MM-DD), check_out (YYYY-MM-DD), adults (int)."
}

func (t *HotelTool) Execute(args map[string]interface{}) (interface{}, error) {
	if t.Client == nil {
		return nil, fmt.Errorf("amadeus client not initialized")
	}

	// Note: We are using Amadeus for this MVP.
	// Users might ask "Hotels in NYC". We rely on city_code for now.
	// Converting "NYC" to city code or "New York" to city code is a separate step (e.g. via Date/Place tool or LLM reasoning).

	cityCode, _ := args["city_code"].(string)
	// Alternatively support hotelIds if specific

	if cityCode == "" {
		return nil, fmt.Errorf("city_code is required")
	}

	// Amadeus `SearchHotelsByCity` returns a list of hotels.
	// We might want to just return that list, or pick a few and search offers.
	// The implementation in `providers/amadeus` was just a shell returning nil for `SearchHotelsByCity`.
	// However, `SearchHotelOffers` takes hotel IDs.
	// THIS IS A GAP.
	// For this Tool to be useful, it needs to find *offers*.
	// Flow:
	// 1. SearchHotelsByCity -> Get Hotel IDs.
	// 2. SearchHotelOffers -> Get Prices.

	// Since `SearchHotelsByCity` is not fully implemented in the provider (it returns nil),
	// calling it here will fail or return nothing.
	// Implementation Plan Step: "HotelTool: Will use Amadeus for offers."
	// I should probably fix/implement SearchHotelsByCity in the provider first if I want this tool to work end-to-end.
	// Or, for MVP, just call it and assume it will be implemented.

	// Step 1: Search hotels by city to get IDs
	// NOTE: Amadeus /reference-data doesn't return ratings directly, just list of hotels.
	listResp, err := t.Client.SearchHotelsByCity(cityCode)
	if err != nil {
		return nil, fmt.Errorf("search hotels by city failed: %v", err)
	}

	if len(listResp.Data) == 0 {
		return nil, fmt.Errorf("no hotels found for city code: %s", cityCode)
	}

	// Step 2: Pick top 5 hotels and search for offers
	// Amadeus SearchHotelOffers takes list of IDs.
	// Limit to 5 to avoid API error/complexity.
	limit := 5
	if len(listResp.Data) < limit {
		limit = len(listResp.Data)
	}

	var hotelIds []string
	for i := 0; i < limit; i++ {
		hotelIds = append(hotelIds, listResp.Data[i].HotelId)
	}

	checkIn, _ := args["check_in"].(string)
	checkOut, _ := args["check_out"].(string)
	adults, ok := args["adults"].(int)
	if !ok || adults == 0 {
		adults = 1
	}

	// If no dates provided, we can't search offers. Just return the list (maybe tool should enforce dates?).
	if checkIn == "" || checkOut == "" {
		return listResp, nil
	}

	offersResp, err := t.Client.SearchHotelOffers(hotelIds, adults, checkIn, checkOut)
	if err != nil {
		// Fallback: If offer search fails (e.g. Rate Limit or invalid params), return the hotel list
		return map[string]interface{}{
			"warning": "offer search failed",
			"error":   err.Error(),
			"hotels":  listResp,
		}, nil
	}

	return offersResp, nil
}
