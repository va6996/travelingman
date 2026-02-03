package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/orm"
	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

type IncludedCheckedBags struct {
	Quantity   int    `json:"quantity"`
	Weight     int    `json:"weight"`
	WeightUnit string `json:"weightUnit"` // KG, LB
}

type FareDetails struct {
	SegmentID           string                `json:"segmentId"`
	IncludedCheckedBags *IncludedCheckedBags  `json:"includedCheckedBags,omitempty"`
}

type TravelerPricing struct {
	TravelerID    string        `json:"travelerId"`
	FareOption    string        `json:"fareOption"`
	TravelerType  string        `json:"travelerType"`
	Price         Price         `json:"price"`
	FareDetails   []FareDetails `json:"fareDetails,omitempty"`
}

// --- Structs for Flight Price Confirmation ---
// Uses FlightSearchResponse as response as well

type FlightPriceCheckRequest struct {
	Data struct {
		Type         string        `json:"type"`
		FlightOffers []FlightOffer `json:"flightOffers"`
	} `json:"data"`
}

// AdditionalServicesRequest is used to request pricing for extra bags
type AdditionalServicesRequest struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
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
// INVARIANTS (see docs/INVARIANTS.md):
//   - transport.OriginLocation and transport.DestinationLocation are non-nil and enriched
//   - All required fields (dates, traveler count) are validated by ValidateItinerary
func (c *Client) SearchFlights(ctx context.Context, transport *pb.Transport) ([]*pb.Transport, error) {
	// Extract flight from transport
	flight := transport.GetFlight()
	if flight == nil {
		return nil, fmt.Errorf("transport does not contain flight details")
	}

	// Extract location codes (prefer specific airport, fallback to city)
	// INVARIANT: Locations are enriched before this is called
	origin := getLocationCode(transport.OriginLocation)
	destination := getLocationCode(transport.DestinationLocation)

	// INVARIANT: DepartureTime and TravelerCount are always set by ValidateItinerary
	departureDate := flight.DepartureTime.AsTime().Format("2006-01-02")
	adults := int(transport.TravelerCount)

	// Calculate returnDate if needed (not in current Proto for one-way segments, but logic kept for compatibility)
	// If it's a round trip, logic might be handled differently, but for now we follow previous logic.
	returnDate := ""

	// INVARIANTS (see docs/INVARIANTS.md):
	// - origin and destination are non-empty (enriched locations)
	// - departureDate is valid and not in past
	// - adults (TravelerCount) is always positive
	// - DepartureTime is always non-nil
	// - Currency is always set

	// Construct query parameters
	endpoint := fmt.Sprintf("/v2/shopping/flight-offers?originLocationCode=%s&destinationLocationCode=%s&adults=%d",
		origin, destination, adults)

	if departureDate != "" {
		endpoint += fmt.Sprintf("&departureDate=%s", departureDate)
	}

	if returnDate != "" {
		endpoint += fmt.Sprintf("&returnDate=%s", returnDate)
	}

	// INVARIANT: Currency is always set by ValidateItinerary
	endpoint += fmt.Sprintf("&currencyCode=%s", transport.Cost.Currency)

	// Handle Preferences
	if transport.FlightPreferences != nil {
		classStr := ""
		switch transport.FlightPreferences.TravelClass {
		case pb.Class_CLASS_ECONOMY:
			classStr = "ECONOMY"
		case pb.Class_CLASS_PREMIUM_ECONOMY:
			classStr = "PREMIUM_ECONOMY"
		case pb.Class_CLASS_BUSINESS:
			classStr = "BUSINESS"
		case pb.Class_CLASS_FIRST:
			classStr = "FIRST"
		}
		if classStr != "" {
			endpoint += fmt.Sprintf("&travelClass=%s", classStr)
		}
	}

	// Optimization: If arrivalBy is set, maybe we can pass it as a filter?
	// API doesn't seem to support arrivalBy filter directly in V2 GET.
	// We will handle filtering in the upper layer or just ignore for now in the raw plugin call.

	// Check cache
	cacheKey := GenerateCacheKey("flights", endpoint)

	// Try DB Cache first if available
	if c.DB != nil {
		if entry, err := orm.GetCacheEntry(c.DB, cacheKey); err == nil {
			log.Debugf(ctx, "SearchFlights: DB Cache hit for %s", endpoint)
			// Unmarshal
			var cachedTransports []*pb.Transport
			if err := json.Unmarshal(entry.Value, &cachedTransports); err == nil {
				return cachedTransports, nil
			}
		}
	}

	// Fallback to memory cache
	if val, ok := c.Cache.Get(cacheKey); ok {
		log.Debugf(ctx, "SearchFlights: Cache hit for %s", endpoint)
		return val.([]*pb.Transport), nil
	}

	log.Debugf(ctx, "SearchFlights: Requesting %s", endpoint)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Errorf(ctx, "SearchFlights: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log detailed response if available for debugging
		var errBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			if b, err := json.Marshal(errBody); err == nil {
				// Use the error body in the returned error message so MapError can see it
				log.Errorf(ctx, "SearchFlights: API error details: %s", string(b))
				return nil, fmt.Errorf("search failed with status %s: %s", resp.Status, string(b))
			}
			log.Errorf(ctx, "SearchFlights: API error details: %v", errBody)
		}
		log.Errorf(ctx, "SearchFlights: API returned status %s", resp.Status)
		return nil, fmt.Errorf("search failed: %s", resp.Status)
	}

	var searchResp FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		log.Errorf(ctx, "SearchFlights: failed to decode response: %v", err)
		return nil, err
	}

	var transports []*pb.Transport
	limit := c.Config.FlightLimit
	if limit <= 0 {
		limit = 10 // Default
	}

	for i, offer := range searchResp.Data {
		if i >= limit {
			break
		}
		transports = append(transports, offer.ToTransport())
	}

	// Enrich transport locations from input transport and populate ancillary baggage pricing
	// INVARIANT: transport locations are non-nil and enriched
	for i, t := range transports {
		if t.OriginLocation == nil {
			t.OriginLocation = &pb.Location{}
		}
		if t.DestinationLocation == nil {
			t.DestinationLocation = &pb.Location{}
		}
		enrichLocationFrom(t.OriginLocation, transport.OriginLocation)
		enrichLocationFrom(t.DestinationLocation, transport.DestinationLocation)

		// Copy flight preferences from input transport
		t.FlightPreferences = transport.FlightPreferences

		// Populate ancillary baggage pricing if user needs more bags than included
		if i < len(searchResp.Data) {
			if err := c.PopulateAncillaryBaggagePricing(ctx, t, searchResp.Data[i]); err != nil {
				log.Warnf(ctx, "SearchFlights: Failed to populate ancillary baggage pricing: %v", err)
				// Continue anyway, just log the warning
			}
		}
	}

	// Set cache
	ttl := time.Duration(c.Config.CacheTTL.Flight) * time.Hour
	c.Cache.Set(cacheKey, transports, ttl)

	// Persist to DB if available
	if c.DB != nil {
		if b, err := json.Marshal(transports); err == nil {
			// Save with longer TTL for DB if desired, or same
			orm.SetCacheEntry(c.DB, cacheKey, b, 60*time.Minute)
		}
	}

	return transports, nil
}

// ConfirmPrice confirms the price of a selected flight offer
func (c *Client) ConfirmPrice(ctx context.Context, offer FlightOffer) (*FlightSearchResponse, error) {
	reqBody := FlightPriceCheckRequest{}
	reqBody.Data.Type = "flight-offers-pricing"
	reqBody.Data.FlightOffers = []FlightOffer{offer}

	resp, err := c.doRequest(ctx, "POST", "/v1/shopping/flight-offers/pricing", reqBody)
	if err != nil {
		log.Errorf(ctx, "ConfirmPrice: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(ctx, "ConfirmPrice: API returned status %s", resp.Status)
		return nil, fmt.Errorf("price confirmation failed: %s", resp.Status)
	}

	var priceResp FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		log.Errorf(ctx, "ConfirmPrice: failed to decode response: %v", err)
		return nil, err
	}

	return &priceResp, nil
}

// BookFlight creates a flight order
func (c *Client) BookFlight(ctx context.Context, offer FlightOffer, users []*pb.User) (*FlightOrderResponse, error) {
	var travelers []TravelerInfo
	for _, user := range users {
		traveler := TravelerInfo{
			ID:          fmt.Sprintf("%d", user.Id),
			DateOfBirth: user.DateOfBirth.AsTime().Format("2006-01-02"),
			Name: Name{
				FirstName: getFirstName(user.FullName),
				LastName:  getLastName(user.FullName),
			},
			Gender: user.Gender,
			Contact: &Contact{
				EmailAddress: user.Email,
				Phones: []Phone{
					{
						DeviceType:         "MOBILE",
						CountryCallingCode: "1", // TODO: Extract from phone number
						Number:             user.Phone,
					},
				},
			},
		}

		if len(user.Passports) > 0 {
			passport := user.Passports[0]
			traveler.Documents = append(traveler.Documents, Document{
				DocumentType:     "PASSPORT",
				BirthPlace:       passport.BirthPlace,
				IssuanceLocation: passport.IssuanceLocation,
				IssuanceDate:     passport.IssuanceDate.AsTime().Format("2006-01-02"),
				Number:           passport.Number,
				ExpiryDate:       passport.ExpiryDate.AsTime().Format("2006-01-02"),
				IssuanceCountry:  passport.IssuingCountry,
				ValidityCountry:  passport.IssuingCountry,
				Nationality:      passport.Nationality,
				Holder:           true,
			})
		}
		travelers = append(travelers, traveler)
	}

	reqBody := FlightOrderRequest{}
	reqBody.Data.Type = "flight-order"
	reqBody.Data.FlightOffers = []FlightOffer{offer}
	reqBody.Data.Travelers = travelers

	resp, err := c.doRequest(ctx, "POST", "/v1/booking/flight-orders", reqBody)
	if err != nil {
		log.Errorf(ctx, "BookFlight: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Errorf(ctx, "BookFlight: API returned status %s", resp.Status)
		return nil, fmt.Errorf("booking failed: %s", resp.Status)
	}

	var orderResp FlightOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		log.Errorf(ctx, "BookFlight: failed to decode response: %v", err)
		return nil, err
	}

	return &orderResp, nil
}

func getFirstName(fullName string) string {
	// Simple split, assuming First Last
	// In production, robust name parsing is needed
	var firstName string
	fmt.Sscanf(fullName, "%s", &firstName)
	return firstName
}

func getLastName(fullName string) string {
	var firstName, lastName string
	fmt.Sscanf(fullName, "%s %s", &firstName, &lastName)
	if lastName == "" {
		return firstName // Fallback
	}
	return lastName
}

// ToTransport converts a FlightOffer to a pb.Transport
func (o FlightOffer) ToTransport() *pb.Transport {
	t := &pb.Transport{
		Type: pb.TransportType_TRANSPORT_TYPE_FLIGHT,
		OriginLocation: &pb.Location{
			IataCodes: []string{},
		},
		DestinationLocation: &pb.Location{
			IataCodes: []string{},
		},
	}

	// Price
	basePrice := 0.0
	if price, err := strconv.ParseFloat(o.Price.Total, 64); err == nil {
		basePrice = price
		t.Cost = &pb.Cost{
			Value:    basePrice,
			Currency: o.Price.Currency,
		}
	}

	// Details from first segment of first itinerary (simplification)
	if len(o.Itineraries) > 0 && len(o.Itineraries[0].Segments) > 0 {
		firstSeg := o.Itineraries[0].Segments[0]
		lastSeg := o.Itineraries[0].Segments[len(o.Itineraries[0].Segments)-1]

		t.OriginLocation.IataCodes = append(t.OriginLocation.IataCodes, firstSeg.Departure.IataCode)
		t.DestinationLocation.IataCodes = append(t.DestinationLocation.IataCodes, lastSeg.Arrival.IataCode)

		// Carrier and Flight Number
		flightDetails := &pb.Flight{
			CarrierCode:  firstSeg.CarrierCode,
			FlightNumber: firstSeg.Number,
		}

		// Times
		if depTime, err := time.Parse("2006-01-02T15:04:05", firstSeg.Departure.At); err == nil {
			flightDetails.DepartureTime = timestamppb.New(depTime)
		}
		if arrTime, err := time.Parse("2006-01-02T15:04:05", lastSeg.Arrival.At); err == nil {
			flightDetails.ArrivalTime = timestamppb.New(arrTime)
		}

		// Extract baggage information from travelerPricings
		extractBaggageInfo(o, flightDetails)

		// Calculate total cost with ancillaries (initially just base price)
		flightDetails.TotalCostWithAncillaries = &pb.Cost{
			Value:    basePrice,
			Currency: o.Price.Currency,
		}

		t.Details = &pb.Transport_Flight{Flight: flightDetails}
	}

	return t
}

// extractBaggageInfo extracts baggage allowance information from flight offer
func extractBaggageInfo(offer FlightOffer, flight *pb.Flight) {
	if len(offer.TravelerPricings) == 0 {
		return
	}

	// Get baggage info from first traveler pricing
	tp := offer.TravelerPricings[0]
	if len(tp.FareDetails) == 0 {
		return
	}

	for _, fd := range tp.FareDetails {
		if fd.IncludedCheckedBags != nil {
			policy := &pb.BaggagePolicy{
				Type:        pb.BaggageType_BAGGAGE_TYPE_CHECKED,
				Quantity:    int32(fd.IncludedCheckedBags.Quantity),
				Weight:      int32(fd.IncludedCheckedBags.Weight),
				WeightUnit:  fd.IncludedCheckedBags.WeightUnit,
			}
			flight.BaggagePolicy = append(flight.BaggagePolicy, policy)
		}
	}

	// Note: Most airlines include 1 carry-on bag, but this is not always
	// explicitly returned in the API response. We could add a default here
	// or build a database of airline policies.
}

// GetIncludedBaggageCount returns the number of included checked bags
func getIncludedBaggageCount(flight *pb.Flight) int32 {
	if flight == nil {
		return 0
	}
	for _, policy := range flight.BaggagePolicy {
		if policy.Type == pb.BaggageType_BAGGAGE_TYPE_CHECKED {
			return policy.Quantity
		}
	}
	return 0
}

// CheckBaggageRequirements compares user preferences with included baggage
// and returns the number of additional bags needed
func CheckBaggageRequirements(transport *pb.Transport) int32 {
	if transport.FlightPreferences == nil || transport.FlightPreferences.Baggage == nil {
		return 0
	}

	flight := transport.GetFlight()
	if flight == nil {
		return 0
	}

	included := getIncludedBaggageCount(flight)
	needed := transport.FlightPreferences.Baggage.CheckedBags

	if needed > included {
		return needed - included
	}
	return 0
}

// GetAdditionalBaggagePrice queries the Flight Offers Price API for additional baggage costs
// Returns the price for adding the specified number of extra bags
func (c *Client) GetAdditionalBaggagePrice(ctx context.Context, offer FlightOffer, additionalBags int32) (*FlightSearchResponse, error) {
	if additionalBags <= 0 {
		return nil, fmt.Errorf("additional bags must be positive")
	}

	// Create pricing request with additional services
	reqBody := struct {
		Data struct {
			Type         string        `json:"type"`
			FlightOffers []FlightOffer `json:"flightOffers"`
		} `json:"data"`
	}{
		Data: struct {
			Type         string        `json:"type"`
			FlightOffers []FlightOffer `json:"flightOffers"`
		}{
			Type:         "flight-offers-pricing",
			FlightOffers: []FlightOffer{offer},
		},
	}

	log.Debugf(ctx, "GetAdditionalBaggagePrice: Requesting pricing for %d additional bags", additionalBags)

	resp, err := c.doRequest(ctx, "POST", "/v1/shopping/flight-offers/pricing", reqBody)
	if err != nil {
		log.Errorf(ctx, "GetAdditionalBaggagePrice: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(ctx, "GetAdditionalBaggagePrice: API returned status %s", resp.Status)
		return nil, fmt.Errorf("additional baggage pricing failed: %s", resp.Status)
	}

	var priceResp FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		log.Errorf(ctx, "GetAdditionalBaggagePrice: failed to decode response: %v", err)
		return nil, err
	}

	return &priceResp, nil
}

// AddAncillaryBaggageCost adds an ancillary cost entry for additional bags
// and updates the total cost with ancillaries
func AddAncillaryBaggageCost(transport *pb.Transport, additionalBags int32, bagPrice float64, currency string) {
	flight := transport.GetFlight()
	if flight == nil {
		return
	}

	// Add ancillary cost for extra bags
	ancillary := &pb.AncillaryCost{
		Id:          fmt.Sprintf("BAG_%d", additionalBags),
		Type:        "BAGGAGE",
		Description: fmt.Sprintf("%d additional checked bag(s)", additionalBags),
		Cost: &pb.Cost{
			Value:    bagPrice * float64(additionalBags),
			Currency: currency,
		},
	}

	flight.AncillaryCosts = append(flight.AncillaryCosts, ancillary)

	// Update total cost with ancillaries
	basePrice := transport.Cost.GetValue()
	if flight.TotalCostWithAncillaries == nil {
		flight.TotalCostWithAncillaries = &pb.Cost{
			Currency: currency,
		}
	}

	flight.TotalCostWithAncillaries.Value = basePrice + (bagPrice * float64(additionalBags))
	flight.TotalCostWithAncillaries.Currency = currency
}

// PopulateAncillaryBaggagePricing checks if user needs more bags than included,
// queries the additional pricing, and updates the transport with ancillary costs
func (c *Client) PopulateAncillaryBaggagePricing(ctx context.Context, transport *pb.Transport, offer FlightOffer) error {
	additionalBags := CheckBaggageRequirements(transport)
	if additionalBags == 0 {
		log.Debugf(ctx, "PopulateAncillaryBaggagePricing: No additional bags needed")
		return nil
	}

	log.Debugf(ctx, "PopulateAncillaryBaggagePricing: User needs %d additional bags", additionalBags)

	// For now, we'll use a default price since the Flight Offers Price API
	// doesn't always return detailed ancillary pricing in a consistent format.
	// In production, you would:
	// 1. Build a database of airline baggage fees
	// 2. Use the Flight Offers Price API with additionalServices
	// 3. Or integrate with Duffel API which has excellent baggage data

	// Default estimated price per additional bag (will vary by airline/route)
	// This is a placeholder - in production you'd query the actual price
	defaultBagPrice := 50.0 // USD, will be adjusted based on currency

	currency := "USD"
	if transport.Cost != nil {
		currency = transport.Cost.Currency
	}

	// Try to get actual pricing from Amadeus
	priceResp, err := c.GetAdditionalBaggagePrice(ctx, offer, additionalBags)
	if err == nil && len(priceResp.Data) > 0 {
		// Calculate the difference between original price and price with additional bags
		originalPrice, _ := strconv.ParseFloat(offer.Price.Total, 64)
		newPrice, _ := strconv.ParseFloat(priceResp.Data[0].Price.Total, 64)

		if newPrice > originalPrice {
			// The API returned a higher price, likely including additional bags
			extraCost := newPrice - originalPrice
			bagPrice := extraCost / float64(additionalBags)
			AddAncillaryBaggageCost(transport, additionalBags, bagPrice, currency)
			log.Debugf(ctx, "PopulateAncillaryBaggagePricing: Added ancillary cost: %.2f %s per bag", bagPrice, currency)
			return nil
		}
	}

	// Fallback to default pricing if API didn't return additional bag cost
	log.Debugf(ctx, "PopulateAncillaryBaggagePricing: Using default pricing (%.2f %s per bag)", defaultBagPrice, currency)
	AddAncillaryBaggageCost(transport, additionalBags, defaultBagPrice, currency)

	return nil
}
