package core

import (
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

// GetCurrencyForCountry returns the currency code for a given country code (ISO 3166-1 alpha-2).
// Defaults to "USD" if the country is not found or empty.
func GetCurrencyForCountry(countryCode string) string {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	if code == "" {
		return "USD"
	}

	// Parse the region (country code)
	region, err := language.ParseRegion(code)
	if err != nil {
		// Try mapping common non-standard codes if needed, or just return USD
		return "USD"
	}

	// Get currency for the region
	cur, ok := currency.FromRegion(region)
	if !ok {
		return "USD"
	}

	return cur.String()
}
