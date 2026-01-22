package nager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/log"
	toolspkg "github.com/va6996/travelingman/tools"
)

// --- Available Countries Tool ---

type AvailableCountriesInput struct{}

type AvailableCountriesOutput struct {
	Countries []Country `json:"countries"`
	Count     int       `json:"count"`
}

type AvailableCountriesTool struct {
	client *Client
}

func NewAvailableCountriesTool(client *Client, gk *genkit.Genkit, registry *toolspkg.Registry) *AvailableCountriesTool {
	t := &AvailableCountriesTool{client: client}
	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*AvailableCountriesInput, *AvailableCountriesOutput](
		gk,
		"nager_available_countries",
		"Returns a list of all available countries supported by the Nager.Date API.",
		func(ctx *ai.ToolContext, input *AvailableCountriesInput) (*AvailableCountriesOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return t.Execute(ctx, &AvailableCountriesInput{})
	})
	return t
}

func (t *AvailableCountriesTool) Execute(ctx context.Context, input *AvailableCountriesInput) (*AvailableCountriesOutput, error) {
	log.Debugf(ctx, "AvailableCountriesTool executing")

	if t.client == nil {
		return nil, fmt.Errorf("nager client not initialized")
	}

	countries, err := t.client.GetAvailableCountries(ctx)
	if err != nil {
		log.Errorf(ctx, "AvailableCountriesTool failed: %v", err)
		return nil, err
	}

	log.Debugf(ctx, "AvailableCountriesTool completed successfully. Found %d countries.", len(countries))
	return &AvailableCountriesOutput{
		Countries: countries,
		Count:     len(countries),
	}, nil
}

// --- Public Holidays Tool ---

type PublicHolidaysInput struct {
	CountryCode string `json:"country_code" description:"ISO country code (e.g., 'US', 'GB')"`
	Year        int    `json:"year" description:"Year (e.g., 2024)"`
}

type PublicHolidaysOutput struct {
	Holidays []Holiday `json:"holidays"`
	Count    int       `json:"count"`
}

type PublicHolidaysTool struct {
	client *Client
}

func NewPublicHolidaysTool(client *Client, gk *genkit.Genkit, registry *toolspkg.Registry) *PublicHolidaysTool {
	t := &PublicHolidaysTool{client: client}
	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*PublicHolidaysInput, *PublicHolidaysOutput](
		gk,
		"nager_public_holidays",
		"Returns public holidays for a specific country and year.",
		func(ctx *ai.ToolContext, input *PublicHolidaysInput) (*PublicHolidaysOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		b, _ := json.Marshal(args)
		var input PublicHolidaysInput
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		// Default to current year if 0
		if input.Year == 0 {
			input.Year = time.Now().Year()
		}
		return t.Execute(ctx, &input)
	})
	return t
}

func (t *PublicHolidaysTool) Execute(ctx context.Context, input *PublicHolidaysInput) (*PublicHolidaysOutput, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "PublicHolidaysTool executing with input: %s", string(inputJSON))

	if t.client == nil {
		return nil, fmt.Errorf("nager client not initialized")
	}
	if input.CountryCode == "" {
		return nil, fmt.Errorf("country_code is required")
	}
	if input.Year == 0 {
		input.Year = time.Now().Year()
	}

	holidays, err := t.client.GetPublicHolidays(ctx, input.Year, input.CountryCode)
	if err != nil {
		log.Errorf(ctx, "PublicHolidaysTool failed: %v", err)
		return nil, err
	}

	log.Debugf(ctx, "PublicHolidaysTool completed successfully. Found %d holidays.", len(holidays))
	return &PublicHolidaysOutput{
		Holidays: holidays,
		Count:    len(holidays),
	}, nil
}

// --- Long Weekends Tool ---

type LongWeekendsInput struct {
	CountryCode string `json:"country_code" description:"ISO country code"`
	Year        int    `json:"year" description:"Year"`
}

type LongWeekendsOutput struct {
	Weekends []LongWeekend `json:"long_weekends"`
	Count    int           `json:"count"`
}

type LongWeekendsTool struct {
	client *Client
}

func NewLongWeekendsTool(client *Client, gk *genkit.Genkit, registry *toolspkg.Registry) *LongWeekendsTool {
	t := &LongWeekendsTool{client: client}
	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*LongWeekendsInput, *LongWeekendsOutput](
		gk,
		"nager_long_weekends",
		"Returns long weekends for a specific country and year.",
		func(ctx *ai.ToolContext, input *LongWeekendsInput) (*LongWeekendsOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		b, _ := json.Marshal(args)
		var input LongWeekendsInput
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		if input.Year == 0 {
			input.Year = time.Now().Year()
		}
		return t.Execute(ctx, &input)
	})
	return t
}

func (t *LongWeekendsTool) Execute(ctx context.Context, input *LongWeekendsInput) (*LongWeekendsOutput, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "LongWeekendsTool executing with input: %s", string(inputJSON))

	if t.client == nil {
		return nil, fmt.Errorf("nager client not initialized")
	}
	if input.CountryCode == "" {
		return nil, fmt.Errorf("country_code is required")
	}
	if input.Year == 0 {
		input.Year = time.Now().Year()
	}

	weekends, err := t.client.GetLongWeekends(ctx, input.Year, input.CountryCode)
	if err != nil {
		log.Errorf(ctx, "LongWeekendsTool failed: %v", err)
		return nil, err
	}

	log.Debugf(ctx, "LongWeekendsTool completed successfully. Found %d weekends.", len(weekends))
	return &LongWeekendsOutput{
		Weekends: weekends,
		Count:    len(weekends),
	}, nil
}

// --- Is Today Holiday Tool ---

type IsTodayHolidayInput struct {
	CountryCode string `json:"country_code" description:"ISO country code"`
}

type IsTodayHolidayOutput struct {
	IsHoliday bool   `json:"is_holiday"`
	Date      string `json:"date"`
}

type IsTodayHolidayTool struct {
	client *Client
}

func NewIsTodayHolidayTool(client *Client, gk *genkit.Genkit, registry *toolspkg.Registry) *IsTodayHolidayTool {
	t := &IsTodayHolidayTool{client: client}
	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*IsTodayHolidayInput, *IsTodayHolidayOutput](
		gk,
		"nager_is_today_holiday",
		"Checks if today is a public holiday in the specified country.",
		func(ctx *ai.ToolContext, input *IsTodayHolidayInput) (*IsTodayHolidayOutput, error) {
			return t.Execute(ctx, input)
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		b, _ := json.Marshal(args)
		var input IsTodayHolidayInput
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
		return t.Execute(ctx, &input)
	})
	return t
}

func (t *IsTodayHolidayTool) Execute(ctx context.Context, input *IsTodayHolidayInput) (*IsTodayHolidayOutput, error) {
	inputJSON, _ := json.Marshal(input)
	log.Debugf(ctx, "IsTodayHolidayTool executing with input: %s", string(inputJSON))

	if t.client == nil {
		return nil, fmt.Errorf("nager client not initialized")
	}
	if input.CountryCode == "" {
		return nil, fmt.Errorf("country_code is required")
	}

	isHoliday, err := t.client.IsTodayPublicHoliday(ctx, input.CountryCode)
	if err != nil {
		log.Errorf(ctx, "IsTodayHolidayTool failed: %v", err)
		return nil, err
	}

	log.Debugf(ctx, "IsTodayHolidayTool completed successfully. IsHoliday: %v", isHoliday)
	return &IsTodayHolidayOutput{
		IsHoliday: isHoliday,
		Date:      time.Now().Format("2006-01-02"),
	}, nil
}
