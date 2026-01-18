package googlemaps

import (
	"context"
	"fmt"

	"googlemaps.github.io/maps"
)

// Client handles Google Maps API requests
type Client struct {
	APIKey     string
	MapsClient *maps.Client
}

// NewClient creates a new Google Maps API client
// Returns an error if the client cannot be initialized
func NewClient(apiKey string) (*Client, error) {

	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create maps client: %w", err)
	}

	return &Client{
		APIKey:     apiKey,
		MapsClient: c,
	}, nil
}

// PlaceAutocompleteResponse represents the autocomplete API response
type PlaceAutocompleteResponse struct {
	Predictions []Prediction `json:"predictions"`
	Status      string       `json:"status"`
}

// Prediction represents a single autocomplete prediction
type Prediction struct {
	Description      string           `json:"description"`
	PlaceID          string           `json:"place_id"`
	StructuredFormat StructuredFormat `json:"structured_formatting"`
	Types            []string         `json:"types"`
}

// StructuredFormat provides structured place information
type StructuredFormat struct {
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

// PlaceDetailsResponse represents the place details API response
type PlaceDetailsResponse struct {
	Result PlaceDetails `json:"result"`
	Status string       `json:"status"`
}

// PlaceDetails contains detailed information about a place
type PlaceDetails struct {
	PlaceID          string   `json:"place_id"`
	Name             string   `json:"name"`
	FormattedAddress string   `json:"formatted_address"`
	Geometry         Geometry `json:"geometry"`
	Types            []string `json:"types"`
	Rating           float64  `json:"rating,omitempty"`
	PriceLevel       int      `json:"price_level,omitempty"`
	Photos           []Photo  `json:"photos,omitempty"`
}

// Geometry contains location coordinates
type Geometry struct {
	Location Location `json:"location"`
}

// Location represents latitude and longitude
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Photo represents a place photo reference
type Photo struct {
	PhotoReference string `json:"photo_reference"`
	Height         int    `json:"height"`
	Width          int    `json:"width"`
}

// AutocompleteSearch searches for places using autocomplete
func (c *Client) AutocompleteSearch(input string, location *Location, radius int) (*PlaceAutocompleteResponse, error) {
	if c.MapsClient == nil {
		return nil, fmt.Errorf("maps client not initialized")
	}

	req := &maps.PlaceAutocompleteRequest{
		Input: input,
	}

	if location != nil {
		req.Location = &maps.LatLng{
			Lat: location.Lat,
			Lng: location.Lng,
		}
		if radius > 0 {
			req.Radius = uint(radius)
		}
	}

	resp, err := c.MapsClient.PlaceAutocomplete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("autocomplete request failed: %w", err)
	}

	// Convert SDK response to our custom response type
	result := &PlaceAutocompleteResponse{
		Status:      "OK", // SDK doesn't return status, assume OK if no error
		Predictions: make([]Prediction, len(resp.Predictions)),
	}

	for i, pred := range resp.Predictions {
		result.Predictions[i] = Prediction{
			Description: pred.Description,
			PlaceID:     pred.PlaceID,
			Types:       pred.Types,
			StructuredFormat: StructuredFormat{
				MainText:      pred.StructuredFormatting.MainText,
				SecondaryText: pred.StructuredFormatting.SecondaryText,
			},
		}
	}

	// Handle zero results
	if len(result.Predictions) == 0 {
		result.Status = "ZERO_RESULTS"
	}

	return result, nil
}

// GetPlaceDetails retrieves detailed information about a place
func (c *Client) GetPlaceDetails(placeID string) (*PlaceDetails, error) {
	if c.MapsClient == nil {
		return nil, fmt.Errorf("maps client not initialized")
	}

	// Parse field masks for the fields we need
	fields := []maps.PlaceDetailsFieldMask{
		maps.PlaceDetailsFieldMaskPlaceID,
		maps.PlaceDetailsFieldMaskName,
		maps.PlaceDetailsFieldMaskFormattedAddress,
		maps.PlaceDetailsFieldMaskGeometry,
		maps.PlaceDetailsFieldMaskTypes,
		maps.PlaceDetailsFieldMaskRatings,
		maps.PlaceDetailsFieldMaskPriceLevel,
		maps.PlaceDetailsFieldMaskPhotos,
	}

	req := &maps.PlaceDetailsRequest{
		PlaceID: placeID,
		Fields:  fields,
	}

	result, err := c.MapsClient.PlaceDetails(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("place details request failed: %w", err)
	}

	// Convert SDK response to our custom response type
	placeDetails := &PlaceDetails{
		PlaceID:          result.PlaceID,
		Name:             result.Name,
		FormattedAddress: result.FormattedAddress,
		Types:            result.Types,
		Rating:           float64(result.Rating),
		PriceLevel:       result.PriceLevel,
		Geometry: Geometry{
			Location: Location{
				Lat: result.Geometry.Location.Lat,
				Lng: result.Geometry.Location.Lng,
			},
		},
		Photos: make([]Photo, len(result.Photos)),
	}

	for i, photo := range result.Photos {
		placeDetails.Photos[i] = Photo{
			PhotoReference: photo.PhotoReference,
			Height:         photo.Height,
			Width:          photo.Width,
		}
	}

	return placeDetails, nil
}

// GetCoordinates retrieves the latitude and longitude for a given address
func (c *Client) GetCoordinates(address string) ([]maps.GeocodingResult, error) {
	if c.MapsClient == nil {
		return nil, fmt.Errorf("maps client not initialized")
	}

	r := &maps.GeocodingRequest{Address: address}
	return c.MapsClient.Geocode(context.Background(), r)
}
