package nager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/tools"
)

// Client handles Nager.Date API requests
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Nager.Date API client and initializes tools
func NewClient(gk *genkit.Genkit, registry *tools.Registry) *Client {
	c := &Client{
		BaseURL:    "https://date.nager.at/api/v3",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Initialize tools
	c.initTools(gk, registry)

	return c
}

// initTools registers all Nager tools
func (c *Client) initTools(gk *genkit.Genkit, registry *tools.Registry) {
	if gk == nil || registry == nil {
		return
	}

	// Register Nager tools
	NewAvailableCountriesTool(c, gk, registry)
	NewPublicHolidaysTool(c, gk, registry)
	NewLongWeekendsTool(c, gk, registry)
	NewIsTodayHolidayTool(c, gk, registry)
}

// Country represents a country from Nager.Date API
type Country struct {
	CountryCode string `json:"countryCode"`
	Name        string `json:"name"`
}

// Holiday represents a public holiday from Nager.Date API
type Holiday struct {
	Date        string   `json:"date"`
	LocalName   string   `json:"localName"`
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"`
	LaunchYear  *int     `json:"launchYear"`
	Types       []string `json:"types"`
}

// LongWeekend represents a long weekend from Nager.Date API
type LongWeekend struct {
	StartDate          string `json:"startDate"`
	EndDate            string `json:"endDate"`
	DayCount           int    `json:"dayCount"`
	NeedBridgeDay      bool   `json:"needBridgeDay"`
	BridgeDayType      string `json:"bridgeDayType"`
	WorkdayCount       int    `json:"workdayCount"`
	WorkdayCountBefore int    `json:"workdayCountBefore"`
	WorkdayCountAfter  int    `json:"workdayCountAfter"`
	UniqueHolidayCount int    `json:"uniqueHolidayCount"`
}

// GetAvailableCountries returns a list of available countries
func (c *Client) GetAvailableCountries(ctx context.Context) ([]Country, error) {
	url := fmt.Sprintf("%s/AvailableCountries", c.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get available countries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var countries []Country
	if err := json.NewDecoder(resp.Body).Decode(&countries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return countries, nil
}

// GetPublicHolidays returns public holidays for a specific country and year
func (c *Client) GetPublicHolidays(ctx context.Context, year int, countryCode string) ([]Holiday, error) {
	url := fmt.Sprintf("%s/PublicHolidays/%d/%s", c.BaseURL, year, countryCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get public holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var holidays []Holiday
	if err := json.NewDecoder(resp.Body).Decode(&holidays); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return holidays, nil
}

// GetLongWeekends returns long weekends for a specific country and year
func (c *Client) GetLongWeekends(ctx context.Context, year int, countryCode string) ([]LongWeekend, error) {
	url := fmt.Sprintf("%s/LongWeekend/%d/%s", c.BaseURL, year, countryCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get long weekends: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var weekends []LongWeekend
	if err := json.NewDecoder(resp.Body).Decode(&weekends); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return weekends, nil
}

// IsTodayPublicHoliday checks if today is a public holiday for a specific country
func (c *Client) IsTodayPublicHoliday(ctx context.Context, countryCode string) (bool, error) {
	today := time.Now()
	year := today.Year()

	holidays, err := c.GetPublicHolidays(ctx, year, countryCode)
	if err != nil {
		return false, err
	}

	todayStr := today.Format("2006-01-02")
	for _, holiday := range holidays {
		if holiday.Date == todayStr {
			return true, nil
		}
	}

	return false, nil
}

// MapError categorizes an error into a protobuf ErrorCode
func (c *Client) MapError(err error) pb.ErrorCode {
	if err == nil {
		return pb.ErrorCode_ERROR_CODE_UNSPECIFIED
	}

	errMsg := err.Error()

	if fmt.Sprintf("%v", http.StatusNotFound) == "404" && (bytes.Contains([]byte(errMsg), []byte("404")) || bytes.Contains([]byte(errMsg), []byte("Not Found"))) {
		return pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND
	}

	// Nager errors are often simple HTTP status errors based on our implementation
	// "API request failed with status 404"
	if bytes.Contains([]byte(errMsg), []byte("status 404")) {
		return pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND
	}
	if bytes.Contains([]byte(errMsg), []byte("status 429")) {
		return pb.ErrorCode_ERROR_CODE_API_LIMIT_REACHED
	}
	if bytes.Contains([]byte(errMsg), []byte("status 400")) {
		return pb.ErrorCode_ERROR_CODE_INVALID_INPUT
	}
	if bytes.Contains([]byte(errMsg), []byte("status 500")) {
		return pb.ErrorCode_ERROR_CODE_INTERNAL_SERVER_ERROR
	}

	return pb.ErrorCode_ERROR_CODE_SEARCH_FAILED
}
