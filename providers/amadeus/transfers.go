package amadeus

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Structs for Transfer Search ---

type TransferSearchResponse struct {
	Data []TransferOffer `json:"data"`
}

type TransferOffer struct {
	Type                     string            `json:"type"`
	ID                       string            `json:"id"`
	TransferType             string            `json:"transferType"` // PRIVATE, SHARED, TXI
	Start                    TransferPoint     `json:"start"`
	End                      TransferPoint     `json:"end"`
	Vehicle                  TransferVehicle   `json:"vehicle"`
	ServiceProvider          ServiceProvider   `json:"serviceProvider"`
	Quotation                TransferQuotation `json:"quotation"`
	MethodsOfPaymentAccepted []string          `json:"methodsOfPaymentAccepted"`
}

type TransferPoint struct {
	DateTime     string         `json:"dateTime"`
	LocationCode string         `json:"locationCode,omitempty"` // IATA code
	Address      *PostalAddress `json:"address,omitempty"`
	GeoCode      *GeoCode       `json:"geoCode,omitempty"`
}

type GeoCode struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type TransferVehicle struct {
	Code        string `json:"code"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Seats       []Seat `json:"seats"`
	Baggage     []Bag  `json:"baggage"`
}

type Seat struct {
	Count int    `json:"count"`
	Type  string `json:"type"` // ADULT, CHILD
}

type Bag struct {
	Count int    `json:"count"`
	Type  string `json:"type"` // HOLD, HAND
}

type ServiceProvider struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type TransferQuotation struct {
	MonetaryAmount string `json:"monetaryAmount"`
	CurrencyCode   string `json:"currencyCode"`
	IsEstimated    bool   `json:"isEstimated"`
}

// --- Structs for Transfer Booking ---

type TransferOrderRequest struct {
	Data struct {
		Type          string             `json:"type"` // "transfer-order"
		OfferID       string             `json:"offerId"`
		Start         *TransferPoint     `json:"start,omitempty"` // Optional confirmation
		End           *TransferPoint     `json:"end,omitempty"`   // Optional confirmation
		Travelers     []TransferTraveler `json:"travelers"`
		Payment       *TransferPayment   `json:"payment,omitempty"`
		ExtraServices []ExtraService     `json:"extraServices,omitempty"`
	} `json:"data"`
}

type TransferTraveler struct {
	ID        string   `json:"id"` // Unique ref
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Title     string   `json:"title,omitempty"`
	Contacts  *Contact `json:"contacts,omitempty"` // Reusing Contact from flights.go (ensure accessible or redefine)
}

type TransferPayment struct {
	CreditCard *PaymentCard `json:"creditCard,omitempty"` // Reusing PaymentCard from hotels.go
}

type ExtraService struct {
	Code   string `json:"code"`
	ItemID string `json:"itemId"`
	Count  int    `json:"count"`
}

type TransferOrderResponse struct {
	Data struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		// ...
	} `json:"data"`
}

// --- Methods ---

// SearchTransfers searches for transfers
func (c *Client) SearchTransfers(startLocationCode, endLocationCode, startDateTime string, passengers int) (*TransferSearchResponse, error) {
	endpoint := fmt.Sprintf("/v1/shopping/transfer-offers?startLocationCode=%s&endLocationCode=%s&startDateTime=%s&passengers=%d",
		startLocationCode, endLocationCode, startDateTime, passengers)

	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transfer search failed: %s", resp.Status)
	}

	var searchResp TransferSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return &searchResp, nil
}

// BookTransfer creates a transfer order
func (c *Client) BookTransfer(offerId string, travelers []TransferTraveler, payment *TransferPayment) (*TransferOrderResponse, error) {
	reqBody := TransferOrderRequest{}
	reqBody.Data.Type = "transfer-booking" // Check API spec, usually "transfer-order" or similar, stick to standard guess if undefined
	// Correction: API spec says "transfer-order" for structure but endpoint is /ordering/transfer-orders
	reqBody.Data.Type = "transfer-order"
	reqBody.Data.OfferID = offerId
	reqBody.Data.Travelers = travelers
	reqBody.Data.Payment = payment

	resp, err := c.doRequest("POST", "/v1/ordering/transfer-orders", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("transfer booking failed: %s", resp.Status)
	}

	var orderResp TransferOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, err
	}

	return &orderResp, nil
}
