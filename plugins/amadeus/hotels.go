package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Structs for Hotel Search ---

type HotelSearchResponse struct {
	Data []HotelOfferData `json:"data"`
}

type HotelOfferData struct {
	Type      string       `json:"type"`
	Hotel     HotelInfo    `json:"hotel"`
	Available bool         `json:"available"`
	Offers    []HotelOffer `json:"offers"`
	Self      string       `json:"self"`
}

type HotelInfo struct {
	Type      string  `json:"type"`
	HotelId   string  `json:"hotelId"`
	ChainCode string  `json:"chainCode"`
	DupeId    string  `json:"dupeId"`
	Name      string  `json:"name"`
	CityCode  string  `json:"cityCode"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type HotelOffer struct {
	ID                  string `json:"id"`
	CheckInDate         string `json:"checkInDate"`
	CheckOutDate        string `json:"checkOutDate"`
	RateCode            string `json:"rateCode"`
	RateFamilyEstimated struct {
		Code string `json:"code"`
		Type string `json:"type"`
	} `json:"rateFamilyEstimated"`
	Room     HotelRoom     `json:"room"`
	Guests   HotelGuests   `json:"guests"`
	Price    HotelPrice    `json:"price"`
	Policies HotelPolicies `json:"policies"`
	Self     string        `json:"self"`
}

type HotelRoom struct {
	Type          string `json:"type"`
	TypeEstimated struct {
		Category string `json:"category"`
		Beds     int    `json:"beds"`
		BedType  string `json:"bedType"`
	} `json:"typeEstimated"`
	Description struct {
		Text string `json:"text"`
		Lang string `json:"lang"`
	} `json:"description"`
}

type HotelGuests struct {
	Adults int `json:"adults"`
}

type HotelPrice struct {
	Currency   string `json:"currency"`
	Base       string `json:"base"`
	Total      string `json:"total"`
	Variations struct {
		Average struct {
			Base string `json:"base"`
		} `json:"average"`
	} `json:"variations"`
}

type HotelPolicies struct {
	BoookingHoldPolicy struct {
		Deadline string `json:"deadline"`
	} `json:"bookingHoldPolicy"`
	Guarantee struct {
		AcceptedPayments struct {
			CreditCards []string `json:"creditCards"`
			Methods     []string `json:"methods"`
		} `json:"acceptedPayments"`
	} `json:"guarantee"`
	PaymentType  string `json:"paymentType"`
	Cancellation struct {
		Deadline string `json:"deadline"`
	} `json:"cancellation"`
}

// --- Structs for Hotel Booking ---

type HotelOrderRequest struct {
	Data struct {
		Type             string            `json:"type"` // "hotel-order"
		RoomAssociations []RoomAssociation `json:"roomAssociations"`
		Guests           []HotelGuest      `json:"guests"`
		Payments         []HotelPayment    `json:"payments"`
		TravelAgent      *TravelAgent      `json:"travelAgent,omitempty"`
	} `json:"data"`
}

type RoomAssociation struct {
	GuestReferences []GuestReference `json:"guestReferences"`
	HotelOfferId    string           `json:"hotelOfferId"`
}

type GuestReference struct {
	GuestReferenceId string `json:"guestReferenceId"`
}

type HotelGuest struct {
	Tid       int    `json:"tid"` // Traveler ID
	Title     string `json:"title,omitempty"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone,omitempty"`
	Email     string `json:"email,omitempty"`
}

type HotelPayment struct {
	Method string       `json:"method"` // CREDIT_CARD
	Card   *PaymentCard `json:"card,omitempty"`
}

type PaymentCard struct {
	VendorCode string `json:"vendorCode"` // VI, CA, AX
	CardNumber string `json:"cardNumber"`
	ExpiryDate string `json:"expiryDate"` // YYYY-MM
}

type TravelAgent struct {
	Contact struct {
		Email string `json:"email"`
	} `json:"contact"`
}

type HotelOrderResponse struct {
	Data []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		// Other fields omitted for brevity
	} `json:"data"`
}

// --- Methods ---

// HotelListResponse is the response from /v1/reference-data/locations/hotels/by-city
type HotelListResponse struct {
	Data []struct {
		ChainCode string `json:"chainCode"`
		IataCode  string `json:"iataCode"`
		DupeId    int    `json:"dupeId"`
		Name      string `json:"name"`
		HotelId   string `json:"hotelId"`
		GeoCode   struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"geoCode"`
		Address struct {
			CountryCode string `json:"countryCode"`
		} `json:"address"`
	} `json:"data"`
}

// SearchHotelsByCity searches for hotels in a specific city
func (c *Client) SearchHotelsByCity(ctx context.Context, acc *pb.Accommodation) (*HotelListResponse, error) {
	cityCode := ""
	if acc.Location != nil {
		if len(acc.Location.IataCodes) > 0 {
			cityCode = acc.Location.IataCodes[0]
		} else {
			cityCode = acc.Location.CityCode
		}
	}
	if cityCode == "" {
		cityCode = acc.Address // Fallback
	}

	// Step 1: Get list of hotels in city
	endpoint := fmt.Sprintf("/v1/reference-data/locations/hotels/by-city?cityCode=%s", cityCode)

	if acc.Preferences != nil {
		if acc.Preferences.Rating > 0 {
			endpoint += fmt.Sprintf("&ratings=%d", acc.Preferences.Rating)
		}

		// Amenities is comma separated list
		if len(acc.Preferences.Amenities) > 0 {
			// simple join
			joined := ""
			for i, a := range acc.Preferences.Amenities {
				if i > 0 {
					joined += ","
				}
				joined += a
			}
			endpoint += fmt.Sprintf("&amenities=%s", joined)
		}
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Printf("SearchHotelsByCity: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("SearchHotelsByCity: API returned status %s", resp.Status)
		// Log body for debugging
		return nil, fmt.Errorf("hotel list search failed: %s", resp.Status)
	}

	var listResp HotelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		log.Printf("SearchHotelsByCity: failed to decode response: %v", err)
		return nil, err
	}

	return &listResp, nil
}

// SearchHotelOffers searches for offers for a specific hotel
func (c *Client) SearchHotelOffers(ctx context.Context, hotelIds []string, adults int, checkIn, checkOut string) (*HotelSearchResponse, error) {
	// construct hotelIds string
	ids := ""
	for i, id := range hotelIds {
		if i > 0 {
			ids += ","
		}
		ids += id
	}

	endpoint := fmt.Sprintf("/v3/shopping/hotel-offers?hotelIds=%s&adults=%d&checkInDate=%s&checkOutDate=%s",
		ids, adults, checkIn, checkOut)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Printf("SearchHotelOffers: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("SearchHotelOffers: API returned status %s", resp.Status)
		return nil, fmt.Errorf("hotel offers search failed: %s", resp.Status)
	}

	var searchResp HotelSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		log.Printf("SearchHotelOffers: failed to decode response: %v", err)
		return nil, err
	}

	return &searchResp, nil
}

// SearchHotelOffersAsAccommodations searches for hotel offers and returns them as pb.Accommodation structs
func (c *Client) SearchHotelOffersAsAccommodations(ctx context.Context, hotelIds []string, adults int, checkIn, checkOut string) ([]*pb.Accommodation, error) {
	resp, err := c.SearchHotelOffers(ctx, hotelIds, adults, checkIn, checkOut)
	if err != nil {
		return nil, err
	}

	var accommodations []*pb.Accommodation
	for _, data := range resp.Data {
		accommodations = append(accommodations, data.ToAccommodations()...)
	}

	return accommodations, nil
}

// BookHotel creates a hotel booking
func (c *Client) BookHotel(ctx context.Context, offerId string, guests []HotelGuest, payment HotelPayment) (*HotelOrderResponse, error) {
	reqBody := HotelOrderRequest{}
	reqBody.Data.Type = "hotel-order"

	// Map guests to room
	guestRefs := make([]GuestReference, len(guests))
	for i := range guests {
		guestRefs[i] = GuestReference{GuestReferenceId: fmt.Sprintf("%d", guests[i].Tid)}
	}

	reqBody.Data.RoomAssociations = []RoomAssociation{
		{
			GuestReferences: guestRefs,
			HotelOfferId:    offerId,
		},
	}
	reqBody.Data.Guests = guests
	reqBody.Data.Payments = []HotelPayment{payment}

	resp, err := c.doRequest(ctx, "POST", "/v2/booking/hotel-orders", reqBody)
	if err != nil {
		log.Printf("BookHotel: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("BookHotel: API returned status %s", resp.Status)
		return nil, fmt.Errorf("hotel booking failed: %s", resp.Status)
	}

	var orderResp HotelOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		log.Printf("BookHotel: failed to decode response: %v", err)
		return nil, err
	}

	return &orderResp, nil
}

// ToAccommodations converts HotelOfferData to a list of pb.Accommodation
func (d HotelOfferData) ToAccommodations() []*pb.Accommodation {
	var accs []*pb.Accommodation
	for _, offer := range d.Offers {
		acc := &pb.Accommodation{
			Name:    d.Hotel.Name,
			Address: d.Hotel.ChainCode, // Using ChainCode as address placeholder or similar
			Location: &pb.Location{
				CityCode: d.Hotel.CityCode,
				Name:     d.Hotel.Name,
				Geocode:  fmt.Sprintf("%f,%f", d.Hotel.Latitude, d.Hotel.Longitude),
			},
			Preferences: &pb.AccommodationPreferences{
				RoomType: offer.Room.TypeEstimated.Category,
				Amenities: []string{
					offer.Room.Description.Text,
				},
				// Rating not directly in offer, maybe in HotelInfo but struct definition doesn't show it (it was in request params)
			},
			PriceTotal: offer.Price.Total,
			Status:     "AVAILABLE",
		}

		if t, err := time.Parse("2006-01-02", offer.CheckInDate); err == nil {
			acc.CheckIn = timestamppb.New(t)
		}
		if t, err := time.Parse("2006-01-02", offer.CheckOutDate); err == nil {
			acc.CheckOut = timestamppb.New(t)
		}

		// If guests info is available
		if offer.Guests.Adults > 0 {
			acc.TravelerCount = int32(offer.Guests.Adults)
		}

		accs = append(accs, acc)
	}
	return accs
}
