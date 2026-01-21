package nager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	toolspkg "github.com/va6996/travelingman/tools"
)

// NagerInput defines input for Nager holiday tool
type NagerInput struct {
	Action      string `json:"action" description:"Action to perform: 'countries', 'holidays', 'long_weekends', 'is_today_holiday'"`
	CountryCode string `json:"country_code" description:"ISO country code (e.g., 'US', 'GB', 'DE')"`
	Year        *int   `json:"year" description:"Year for holiday queries (optional, defaults to current year)"`
}

// NagerToolOutput defines output from Nager holiday tool
type NagerToolOutput struct {
	Count     int                      `json:"count"`
	Countries []map[string]interface{} `json:"countries,omitempty"`
	Holidays  []map[string]interface{} `json:"holidays,omitempty"`
	Weekends  []map[string]interface{} `json:"weekends,omitempty"`
	IsHoliday *bool                    `json:"is_holiday,omitempty"`
	Date      string                   `json:"date,omitempty"`
}

// NagerTool provides access to Nager.Date API for holiday information
type NagerTool struct {
	client *Client
}

// NewNagerTool creates a new NagerTool and registers it
func NewNagerTool(client *Client, gk *genkit.Genkit, registry *toolspkg.Registry) *NagerTool {
	t := &NagerTool{
		client: client,
	}

	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*NagerInput, *NagerToolOutput](
		gk,
		"holiday_tool",
		t.Description(),
		func(ctx *ai.ToolContext, input *NagerInput) (*NagerToolOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Use JSON roundtrip for robust mapping
		b, _ := json.Marshal(args)
		var input NagerInput
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}

		result, err := t.Execute(ctx, &input)
		if err != nil {
			return nil, err
		}

		// Return struct value directly, not pointer
		return *result, nil
	})

	return t
}

func (t *NagerTool) Description() string {
	return "Queries public holidays and long weekends for countries worldwide. Provides information about available countries, public holidays by year/country, long weekends, and checks if today is a holiday."
}

func (t *NagerTool) Execute(ctx context.Context, input *NagerInput) (*NagerToolOutput, error) {
	if t.client == nil {
		return nil, fmt.Errorf("nager client not initialized")
	}

	inputJSON, _ := json.Marshal(input)
	fmt.Printf("[DEBUG] NagerTool Executing with input: %s\n", string(inputJSON))

	switch input.Action {
	case "countries":
		return t.getCountries(ctx)

	case "holidays":
		if input.CountryCode == "" {
			return nil, fmt.Errorf("country_code is required for holidays action")
		}
		year := input.Year
		if year == nil {
			currentYear := time.Now().Year()
			year = &currentYear
		}

		holidays, err := t.client.GetPublicHolidays(ctx, *year, input.CountryCode)
		if err != nil {
			return nil, fmt.Errorf("failed to get public holidays: %w", err)
		}

		// Format for better readability
		result := make([]map[string]interface{}, len(holidays))
		for i, holiday := range holidays {
			holidayInfo := map[string]interface{}{
				"date":       holiday.Date,
				"local_name": holiday.LocalName,
				"name":       holiday.Name,
				"country":    holiday.CountryCode,
				"fixed":      holiday.Fixed,
				"global":     holiday.Global,
				"types":      holiday.Types,
			}

			if holiday.LaunchYear != nil {
				holidayInfo["launch_year"] = *holiday.LaunchYear
			}

			if len(holiday.Counties) > 0 {
				holidayInfo["counties"] = holiday.Counties
			}

			result[i] = holidayInfo
		}

		return &NagerToolOutput{
			Count:    len(holidays),
			Holidays: result,
		}, nil

	case "long_weekends":
		if input.CountryCode == "" {
			return nil, fmt.Errorf("country_code is required for long_weekends action")
		}
		year := input.Year
		if year == nil {
			currentYear := time.Now().Year()
			year = &currentYear
		}

		weekends, err := t.client.GetLongWeekends(ctx, *year, input.CountryCode)
		if err != nil {
			return nil, fmt.Errorf("failed to get long weekends: %w", err)
		}

		// Format for better readability
		result := make([]map[string]interface{}, len(weekends))
		for i, weekend := range weekends {
			result[i] = map[string]interface{}{
				"start_date":           weekend.StartDate,
				"end_date":             weekend.EndDate,
				"day_count":            weekend.DayCount,
				"need_bridge_day":      weekend.NeedBridgeDay,
				"bridge_day_type":      weekend.BridgeDayType,
				"workday_count":        weekend.WorkdayCount,
				"workday_count_before": weekend.WorkdayCountBefore,
				"workday_count_after":  weekend.WorkdayCountAfter,
				"unique_holiday_count": weekend.UniqueHolidayCount,
			}
		}

		return &NagerToolOutput{
			Count:    len(weekends),
			Weekends: result,
		}, nil

	case "is_today_holiday":
		if input.CountryCode == "" {
			return nil, fmt.Errorf("country_code is required for is_today_holiday action")
		}

		isHoliday, err := t.client.IsTodayPublicHoliday(ctx, input.CountryCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check if today is a holiday: %w", err)
		}

		return &NagerToolOutput{
			IsHoliday: &isHoliday,
			Date:      time.Now().Format("2006-01-02"),
		}, nil

	default:
		return nil, fmt.Errorf("invalid action: %s. Must be one of: countries, holidays, long_weekends, is_today_holiday", input.Action)
	}
}

func (t *NagerTool) getCountries(ctx context.Context) (*NagerToolOutput, error) {
	countries, err := t.client.GetAvailableCountries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available countries: %w", err)
	}

	// Format for better readability
	result := make([]map[string]interface{}, len(countries))
	for i, country := range countries {
		result[i] = map[string]interface{}{
			"code": country.CountryCode,
			"name": country.Name,
		}
	}

	return &NagerToolOutput{
		Count:     len(countries),
		Countries: result,
	}, nil
}
