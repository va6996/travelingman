package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Structs for Flight Search (Simplified) ---

type FlightSearchResponse struct {
	Data []FlightOffer `json:"data"`
}

type FlightOffer struct {
	Type                     string            `json:"type"`
	ID                       string            `json:"id"`
	Source                   string            `json:"source"`
	InstantTicketingRequired bool              `json:"instantTicketingRequired"`
	NonHomogeneous           bool              `json:"nonHomogeneous"`
	OneWay                   bool              `json:"oneWay"`
	LastTicketingDate        string            `json:"lastTicketingDate"`
	NumberOfBookableSeats    int               `json:"numberOfBookableSeats"`
	Itineraries              []Itinerary       `json:"itineraries"`
	Price                    Price             `json:"price"`
	PricingOptions           PricingOptions    `json:"pricingOptions"`
	ValidatingAirlineCodes   []string          `json:"validatingAirlineCodes"`
	TravelerPricings         []TravelerPricing `json:"travelerPricings"`
}

type Itinerary struct {
	Duration string    `json:"duration"`
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Departure   FlightEndPoint `json:"departure"`
	Arrival     FlightEndPoint `json:"arrival"`
	CarrierCode string         `json:"carrierCode"`
	Number      string         `json:"number"`
	Aircraft    struct {
		Code string `json:"code"`
	} `json:"aircraft"`
	Operating struct {
		CarrierCode string `json:"carrierCode"`
	} `json:"operating"`
	Duration        string `json:"duration"`
	ID              string `json:"id"`
	NumberOfStops   int    `json:"numberOfStops"`
	BlacklistedInEU bool   `json:"blacklistedInEU"`
}

type FlightEndPoint struct {
	IataCode string `json:"iataCode"`
	Terminal string `json:"terminal,omitempty"`
	At       string `json:"at"`
}

type Price struct {
	Currency   string `json:"currency"`
	Total      string `json:"total"`
	Base       string `json:"base"`
	Fees       []Fee  `json:"fees,omitempty"`
	GrandTotal string `json:"grandTotal,omitempty"`
}

type Fee struct {
	Amount string `json:"amount"`
	Type   string `json:"type"`
}

type PricingOptions struct {
	FareType                []string `json:"fareType"`
	IncludedCheckedBagsOnly bool     `json:"includedCheckedBagsOnly"`
}

type TravelerPricing struct {
	TravelerID   string `json:"travelerId"`
	FareOption   string `json:"fareOption"`
	TravelerType string `json:"travelerType"`
	Price        Price  `json:"price"`
	// FareDetailsBySegment would go here
}

// --- Structs for Flight Price Confirmation ---
// Uses FlightSearchResponse as response as well

type FlightPriceCheckRequest struct {
	Data struct {
		Type         string        `json:"type"`
		FlightOffers []FlightOffer `json:"flightOffers"`
	} `json:"data"`
}

// --- Structs for Flight Booking ---

type FlightOrderRequest struct {
	Data struct {
		Type               string              `json:"type"`
		FlightOffers       []FlightOffer       `json:"flightOffers"`
		Travelers          []TravelerInfo      `json:"travelers"`
		Remarks            *Remarks            `json:"remarks,omitempty"`
		TicketingAgreement *TicketingAgreement `json:"ticketingAgreement,omitempty"`
		Contacts           []Contact           `json:"contacts,omitempty"`
	} `json:"data"`
}

type TravelerInfo struct {
	ID          string     `json:"id"`
	DateOfBirth string     `json:"dateOfBirth"`
	Name        Name       `json:"name"`
	Gender      string     `json:"gender"`
	Contact     *Contact   `json:"contact,omitempty"`
	Documents   []Document `json:"documents,omitempty"`
}

type Name struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type Contact struct {
	EmailAddress string         `json:"emailAddress,omitempty"`
	Phones       []Phone        `json:"phones,omitempty"`
	Address      *PostalAddress `json:"address,omitempty"`
	Purpose      string         `json:"purpose,omitempty"` // STANDARD, INVOICE, etc.
}

type Phone struct {
	DeviceType         string `json:"deviceType"` // MOBILE, LANDLINE
	CountryCallingCode string `json:"countryCallingCode"`
	Number             string `json:"number"`
}

type PostalAddress struct {
	Lines       []string `json:"lines"`
	PostalCode  string   `json:"postalCode"`
	CountryCode string   `json:"countryCode"`
	CityName    string   `json:"cityName"`
}

type Document struct {
	DocumentType     string `json:"documentType"` // PASSPORT, etc.
	BirthPlace       string `json:"birthPlace"`
	IssuanceLocation string `json:"issuanceLocation"`
	IssuanceDate     string `json:"issuanceDate"`
	Number           string `json:"number"`
	ExpiryDate       string `json:"expiryDate"`
	IssuanceCountry  string `json:"issuanceCountry"`
	ValidityCountry  string `json:"validityCountry"`
	Nationality      string `json:"nationality"`
	Holder           bool   `json:"holder"`
}

type Remarks struct {
	General []GeneralRemark `json:"general,omitempty"`
}

type GeneralRemark struct {
	SubType string `json:"subType"`
	Text    string `json:"text"`
}

type TicketingAgreement struct {
	Option string `json:"option"`
	Delay  string `json:"delay"`
}

type FlightOrderResponse struct {
	Data struct {
		Type              string             `json:"type"`
		ID                string             `json:"id"`
		QueuingOfficeId   string             `json:"queuingOfficeId"`
		AssociatedRecords []AssociatedRecord `json:"associatedRecords"`
		FlightOffers      []FlightOffer      `json:"flightOffers"`
		Travelers         []TravelerInfo     `json:"travelers"`
	} `json:"data"`
}

type AssociatedRecord struct {
	Reference        string `json:"reference"`
	CreationDate     string `json:"creationDate"`
	OriginSystemCode string `json:"originSystemCode"`
	FlightOfferId    string `json:"flightOfferId"`
}

// --- Methods ---

// SearchFlights searches for flight offers
func (c *Client) SearchFlights(ctx context.Context, origin, destination, departureDate, returnDate, arrivalBy string, adults int) (*FlightSearchResponse, error) {
	// Construct query parameters
	endpoint := fmt.Sprintf("/v2/shopping/flight-offers?originLocationCode=%s&destinationLocationCode=%s&adults=%d",
		origin, destination, adults)

	// Logic for Arrival Time Search:
	// If arrivalBy is provided, we might need to adjust logic.
	// However, Amadeus Flight Offers Search *requires* a departureDate.
	// If the user provides arrivalBy, we might infer a departure window?
	// For this MVP, we will assume the caller handles the date inference or we just pass whatever date is given.
	// BUT, if the user strictly wants "Arrival Time" search, they might not know the departure date.
	// We will rely on the caller to provide a departureDate.

	if departureDate != "" {
		endpoint += fmt.Sprintf("&departureDate=%s", departureDate)
	}

	if returnDate != "" {
		endpoint += fmt.Sprintf("&returnDate=%s", returnDate)
	}

	// Optimization: If arrivalBy is set, maybe we can pass it as a filter?
	// API doesn't seem to support arrivalBy filter directly in V2 GET.
	// We will handle filtering in the upper layer or just ignore for now in the raw plugin call.

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: %s", resp.Status)
	}

	var searchResp FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return &searchResp, nil
}

// ConfirmPrice confirms the price of a selected flight offer
func (c *Client) ConfirmPrice(ctx context.Context, offer FlightOffer) (*FlightSearchResponse, error) {
	reqBody := FlightPriceCheckRequest{}
	reqBody.Data.Type = "flight-offers-pricing"
	reqBody.Data.FlightOffers = []FlightOffer{offer}

	resp, err := c.doRequest(ctx, "POST", "/v1/shopping/flight-offers/pricing", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("price confirmation failed: %s", resp.Status)
	}

	var priceResp FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, err
	}

	return &priceResp, nil
}

// BookFlight creates a flight order
func (c *Client) BookFlight(ctx context.Context, offer FlightOffer, travelers []TravelerInfo) (*FlightOrderResponse, error) {
	reqBody := FlightOrderRequest{}
	reqBody.Data.Type = "flight-order"
	reqBody.Data.FlightOffers = []FlightOffer{offer}
	reqBody.Data.Travelers = travelers

	resp, err := c.doRequest(ctx, "POST", "/v1/booking/flight-orders", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("booking failed: %s", resp.Status)
	}

	var orderResp FlightOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, err
	}

	return &orderResp, nil
}
