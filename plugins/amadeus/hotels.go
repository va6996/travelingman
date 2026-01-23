package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/va6996/travelingman/log"
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

// HotelData represents basic hotel info in list response
type HotelData struct {
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
}

// HotelListResponse is the response from /v1/reference-data/locations/hotels/by-city
type HotelListResponse struct {
	Data []HotelData `json:"data"`
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
		log.Errorf(ctx, "SearchHotelsByCity: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(ctx, "SearchHotelsByCity: API returned status %s", resp.Status)
		// Log body for debugging
		return nil, fmt.Errorf("hotel list search failed: %s", resp.Status)
	}

	var listResp HotelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		log.Errorf(ctx, "SearchHotelsByCity: failed to decode response: %v", err)
		return nil, err
	}

	return &listResp, nil
}

// SearchHotelOffers searches for hotel offers and returns them as pb.Accommodation structs
func (c *Client) SearchHotelOffers(ctx context.Context, hotelIds []string, adults int, checkIn, checkOut string) ([]*pb.Accommodation, error) {
	// Amadeus API often has limits on the number of IDs (e.g. 50-100).
	// We chunk them to be safe (e.g., 20).
	const chunkSize = 20
	var accommodations []*pb.Accommodation

	// Chunk the hotel IDs
	for i := 0; i < len(hotelIds); i += chunkSize {
		end := i + chunkSize
		if end > len(hotelIds) {
			end = len(hotelIds)
		}

		batchIds := hotelIds[i:end]

		// construct hotelIds string for this batch
		ids := ""
		for j, id := range batchIds {
			if j > 0 {
				ids += ","
			}
			ids += id
		}

		endpoint := fmt.Sprintf("/v3/shopping/hotel-offers?hotelIds=%s&adults=%d&checkInDate=%s&checkOutDate=%s",
			ids, adults, checkIn, checkOut)

		// Check cache
		cacheKey := GenerateCacheKey("hotel_offers", endpoint)
		if val, ok := c.Cache.Get(cacheKey); ok {
			log.Debugf(ctx, "SearchHotelOffers: Cache hit for batch %d", (i/chunkSize)+1)
			accommodations = append(accommodations, val.([]*pb.Accommodation)...)
			continue
		}

		log.Debugf(ctx, "SearchHotelOffers: Requesting batch %d/%d: %s", (i/chunkSize)+1, (len(hotelIds)+chunkSize-1)/chunkSize, endpoint)

		resp, err := c.doRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			log.Errorf(ctx, "SearchHotelOffers: batch request failed: %v", err)
			continue // Try next batch
		}

		// 400 likely due to invalid parameters in this batch, or dates.
		// If dates are invalid, all batches will fail. If IDs are invalid, maybe just this batch.
		if resp.StatusCode != http.StatusOK {
			// Log detailed response if available for debugging
			var errBody map[string]interface{}

			if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
				if b, err := json.Marshal(errBody); err == nil {
					log.Errorf(ctx, "SearchHotelOffers: API error details: %s", string(b))
				} else {
					log.Errorf(ctx, "SearchHotelOffers: API error details: %v", errBody)
				}
			} else {
				log.Errorf(ctx, "SearchHotelOffers: API returned status %s (failed to parse error body)", resp.Status)
			}

			// We continue here because other batches might succeed
			resp.Body.Close()
			continue
		}

		var searchResp HotelSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
			log.Errorf(ctx, "SearchHotelOffers: failed to decode response: %v", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		var batchAccommodations []*pb.Accommodation
		for _, data := range searchResp.Data {
			batchAccommodations = append(batchAccommodations, data.ToAccommodations()...)
		}

		// Set cache for this batch (30 minutes TTL)
		c.Cache.Set(cacheKey, batchAccommodations, 30*time.Minute)
		accommodations = append(accommodations, batchAccommodations...)
	}

	if len(accommodations) == 0 && len(hotelIds) > 0 {
		return nil, fmt.Errorf("hotel offers search failed for all %d hotels (likely 400 Bad Request or no availability)", len(hotelIds))
	}

	// Apply limit
	limit := c.Limits.Hotel
	if limit > 0 && len(accommodations) > limit {
		accommodations = accommodations[:limit]
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
		log.Errorf(ctx, "BookHotel: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Errorf(ctx, "BookHotel: API returned status %s", resp.Status)
		return nil, fmt.Errorf("hotel booking failed: %s", resp.Status)
	}

	var orderResp HotelOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		log.Errorf(ctx, "BookHotel: failed to decode response: %v", err)
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
