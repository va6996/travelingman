package core

import (
	"context"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/va6996/travelingman/tools"
)

// CurrencyTool wraps GetCurrencyForCountry
type CurrencyTool struct{}

type CurrencyInput struct {
	CountryCode string `json:"country_code" description:"ISO 3166-1 alpha-2 country code"`
}

func NewCurrencyTool(gk *genkit.Genkit, registry *tools.Registry) *CurrencyTool {
	t := &CurrencyTool{}
	if gk == nil || registry == nil {
		return t
	}

	registry.Register(genkit.DefineTool[*CurrencyInput, string](
		gk,
		"core_get_currency",
		"Returns the currency code for a given country code (ISO 3166-1 alpha-2).",
		func(ctx *ai.ToolContext, input *CurrencyInput) (string, error) {
			return GetCurrencyForCountry(input.CountryCode), nil
		},
	), func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		countryCode, _ := args["country_code"].(string)
		return GetCurrencyForCountry(countryCode), nil
	})

	return t
}
