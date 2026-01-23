package amadeus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/tools"
)

const (
	BaseURLTest       = "https://test.api.amadeus.com"
	BaseURLProduction = "https://api.amadeus.com"
)

// Client is the main Amadeus API client
type Client struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTPClient   *http.Client
	Token        *AuthToken
	Cache        *SimpleCache
	Limits       struct {
		Flight int
		Hotel  int
	}
	FlightTool      *FlightTool
	HotelListTool   *HotelListTool
	HotelOffersTool *HotelOffersTool
	LocationTool    *LocationTool
}

// LocationSearchResponse wraps the API response for locations
type LocationSearchResponse struct {
	Data []LocationData `json:"data"`
}

// LocationData represents a single location result from Amadeus
type LocationData struct {
	SubType string  `json:"subType"`
	Name    string  `json:"name"`
	JobCode string  `json:"iataCode"`
	Address Address `json:"address"`
	GeoCode GeoCode `json:"geoCode"`
}

// Address contains location details
type Address struct {
	CityName    string `json:"cityName"`
	CityCode    string `json:"cityCode"`
	CountryName string `json:"countryName"`
	CountryCode string `json:"countryCode"`
}

// AuthToken represents the OAuth2 token response
type AuthToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Expiry      time.Time
}

// NewClient creates a new Amadeus client
// Returns an error if the client cannot be initialized
func NewClient(clientID, clientSecret string, isProduction bool, gk *genkit.Genkit, registry *tools.Registry, flightLimit, hotelLimit, timeout int) (*Client, error) {

	baseURL := BaseURLTest
	if isProduction {
		baseURL = BaseURLProduction
	}

	c := &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: time.Duration(timeout) * time.Second},
		Cache:        NewSimpleCache(),
	}
	c.Limits.Flight = flightLimit
	c.Limits.Hotel = hotelLimit

	// Initialize tools
	c.initTools(gk, registry)

	return c, nil
}

// initTools registers all Amadeus tools
func (c *Client) initTools(gk *genkit.Genkit, registry *tools.Registry) {
	if gk == nil || registry == nil {
		return
	}

	// Register Amadeus tools
	c.LocationTool = NewLocationTool(c, gk, registry)
	c.FlightTool = NewFlightTool(c, gk, registry)
	c.HotelListTool = NewHotelListTool(c, gk, registry)
	c.HotelOffersTool = NewHotelOffersTool(c, gk, registry)
}
func (c *Client) Authenticate() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.ClientID)
	data.Set("client_secret", c.ClientSecret)

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/security/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s", resp.Status)
	}

	var token AuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}

	// Set expiry time (subtract 10 seconds for buffer)
	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - 10*time.Second)
	c.Token = &token

	return nil
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	if c.Token == nil || time.Now().After(c.Token.Expiry) {
		if err := c.Authenticate(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	url := c.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Errorf(ctx, "Amadeus API request failed: %v", err)
		return nil, err
	}

	return resp, nil
}

// SearchLocations searches for airports and cities by keyword and returns protobuf Location objects
func (c *Client) SearchLocations(ctx context.Context, keyword string) ([]*pb.Location, error) {
	data := url.Values{}
	data.Set("keyword", keyword)
	data.Set("subType", "CITY,AIRPORT")
	data.Set("page[limit]", "5")

	endpoint := fmt.Sprintf("/v1/reference-data/locations?%s", data.Encode())
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Errorf(ctx, "SearchLocations: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(ctx, "SearchLocations: API returned status %s", resp.Status)
		return nil, fmt.Errorf("location search failed: %s", resp.Status)
	}

	var result LocationSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Errorf(ctx, "SearchLocations: failed to decode response: %v", err)
		return nil, err
	}

	var locations []*pb.Location
	var lat, lng float64
	foundCoordinates := false

	for _, l := range result.Data {
		loc := &pb.Location{
			Name:      l.Name,
			City:      l.Address.CityName,
			Country:   l.Address.CountryName,
			IataCodes: []string{l.JobCode},
			CityCode:  l.Address.CityCode,
			Geocode:   fmt.Sprintf("%f,%f", l.GeoCode.Latitude, l.GeoCode.Longitude),
		}
		locations = append(locations, loc)

		// Capture coordinates from the first result that has them
		if !foundCoordinates && (l.GeoCode.Latitude != 0 || l.GeoCode.Longitude != 0) {
			lat = l.GeoCode.Latitude
			lng = l.GeoCode.Longitude
			foundCoordinates = true
		}
	}

	// If we have coordinates, search for nearby airports
	if foundCoordinates {
		nearbyAirports, err := c.SearchNearbyAirports(ctx, lat, lng)
		if err == nil {
			// Add unique airports
			existingCodes := make(map[string]bool)
			for _, l := range locations {
				if len(l.IataCodes) > 0 {
					existingCodes[l.IataCodes[0]] = true
				}
			}

			for _, airport := range nearbyAirports {
				if len(airport.IataCodes) > 0 && !existingCodes[airport.IataCodes[0]] {
					locations = append(locations, airport)
					existingCodes[airport.IataCodes[0]] = true
				}
			}
		} else {
			log.Errorf(ctx, "SearchLocations: failed to search nearby airports: %v", err)
		}
	}

	return locations, nil
}

// SearchNearbyAirports searches for airports near a specific latitude and longitude
func (c *Client) SearchNearbyAirports(ctx context.Context, lat, lng float64) ([]*pb.Location, error) {
	data := url.Values{}
	data.Set("latitude", fmt.Sprintf("%f", lat))
	data.Set("longitude", fmt.Sprintf("%f", lng))
	data.Set("radius", "100") // 100km radius
	data.Set("page[limit]", "5")

	endpoint := fmt.Sprintf("/v1/reference-data/locations/airports?%s", data.Encode())
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Errorf(ctx, "SearchNearbyAirports: request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(ctx, "SearchNearbyAirports: API returned status %s", resp.Status)
		return nil, fmt.Errorf("nearby airport search failed: %s", resp.Status)
	}

	var result LocationSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Errorf(ctx, "SearchNearbyAirports: failed to decode response: %v", err)
		return nil, err
	}

	var locations []*pb.Location
	for _, l := range result.Data {
		loc := &pb.Location{
			Name:      l.Name,
			City:      l.Address.CityName,
			Country:   l.Address.CountryName,
			IataCodes: []string{l.JobCode},
			CityCode:  l.Address.CityCode,
			Geocode:   fmt.Sprintf("%f,%f", l.GeoCode.Latitude, l.GeoCode.Longitude),
		}
		locations = append(locations, loc)
	}

	return locations, nil
}

// MapError categorizes an error into a protobuf ErrorCode
func (c *Client) MapError(err error) pb.ErrorCode {
	if err == nil {
		return pb.ErrorCode_ERROR_CODE_UNSPECIFIED
	}

	// Check for Amadeus API errors (if we had a custom error struct, we'd check that)
	// For now, we'll parse the error string or check for common net/http errors
	errMsg := err.Error()

	if bytes.Contains([]byte(errMsg), []byte("404")) || bytes.Contains([]byte(errMsg), []byte("Not Found")) {
		return pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND
	}
	if bytes.Contains([]byte(errMsg), []byte("429")) || bytes.Contains([]byte(errMsg), []byte("Too Many Requests")) {
		return pb.ErrorCode_ERROR_CODE_API_LIMIT_REACHED
	}
	if bytes.Contains([]byte(errMsg), []byte("400")) || bytes.Contains([]byte(errMsg), []byte("Bad Request")) {
		return pb.ErrorCode_ERROR_CODE_INVALID_INPUT
	}
	if bytes.Contains([]byte(errMsg), []byte("401")) || bytes.Contains([]byte(errMsg), []byte("Unauthorized")) {
		return pb.ErrorCode_ERROR_CODE_AUTHENTICATION_FAILED
	}
	if bytes.Contains([]byte(errMsg), []byte("403")) || bytes.Contains([]byte(errMsg), []byte("Forbidden")) {
		return pb.ErrorCode_ERROR_CODE_AUTHENTICATION_FAILED
	}
	if bytes.Contains([]byte(errMsg), []byte("500")) || bytes.Contains([]byte(errMsg), []byte("Internal Server Error")) {
		return pb.ErrorCode_ERROR_CODE_INTERNAL_SERVER_ERROR
	}
	if bytes.Contains([]byte(errMsg), []byte("timeout")) || bytes.Contains([]byte(errMsg), []byte("connection refused")) {
		return pb.ErrorCode_ERROR_CODE_CONNECTION_FAILED
	}

	return pb.ErrorCode_ERROR_CODE_SEARCH_FAILED
}
