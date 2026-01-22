package plugins

import (
	"context"

	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/plugins/googlemaps"
	"googlemaps.github.io/maps"
)

// FlightClient defines the interface for flight interaction
type FlightClient interface {
	SearchFlights(ctx context.Context, origin, destination, departureDate, returnDate, arrivalBy string, adults int) (*amadeus.FlightSearchResponse, error)
	ConfirmPrice(ctx context.Context, offer amadeus.FlightOffer) (*amadeus.FlightSearchResponse, error)
	BookFlight(ctx context.Context, offer amadeus.FlightOffer, travelers []*pb.User) (*amadeus.FlightOrderResponse, error)
}

// HotelClient defines the interface for hotel interaction
type HotelClient interface {
	SearchHotelsByCity(ctx context.Context, cityCode string) (*amadeus.HotelListResponse, error)
	SearchHotelOffers(ctx context.Context, hotelIds []string, adults int, checkIn, checkOut string) (*amadeus.HotelSearchResponse, error)
	BookHotel(ctx context.Context, offerId string, guests []amadeus.HotelGuest, payment amadeus.HotelPayment) (*amadeus.HotelOrderResponse, error)
}

// LLMClient defines the interface for LLM interaction
type LLMClient interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
}

// MapsClient defines the interface for maps interaction
type MapsClient interface {
	AutocompleteSearch(input string, location *googlemaps.Location, radius int) (*googlemaps.PlaceAutocompleteResponse, error)
	GetPlaceDetails(placeID string) (*googlemaps.PlaceDetails, error)
	GetCoordinates(address string) ([]maps.GeocodingResult, error)
}
