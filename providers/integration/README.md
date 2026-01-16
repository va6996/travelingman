# Integration Tests

This directory contains integration tests for all external provider integrations.

## Providers Tested

1. **Amadeus API** - Flight and hotel booking services
2. **Google Maps API** - Places, geocoding, and location services
3. **Gemini API** - AI content generation

## Prerequisites

Set the following environment variables:

```bash
# Amadeus API
export AMADEUS_CLIENT_ID="your_client_id"
export AMADEUS_CLIENT_SECRET="your_client_secret"
export AMADEUS_PRODUCTION="false"  # Set to "true" for production

# Google Maps API
export GOOGLE_MAPS_API_KEY="your_api_key"

# Gemini API
export GEMINI_API_KEY="your_api_key"
```

## Running Tests

### Run All Integration Tests

```bash
go test -v -tags=integration ./providers/integration/...
```

### Run Specific Provider Tests

```bash
# Amadeus only
go test -v -tags=integration -run TestAmadeusIntegration ./providers/integration/

# Google Maps only
go test -v -tags=integration -run TestGoogleMapsIntegration ./providers/integration/

# Gemini only
go test -v -tags=integration -run TestGeminiIntegration ./providers/integration/
```

### Using Make

Add to your Makefile:

```makefile
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./providers/integration/...
```

Then run:
```bash
make test-integration
```

## Test Behavior

- Tests make **real API calls** to external services
- Tests may **fail** if:
  - API credentials are invalid
  - API access is not enabled for your account
  - Rate limits are exceeded
  - Network issues occur

## Notes

- Integration tests require network access
- These tests may consume API quota/credits
- Some tests may require specific API permissions in your developer accounts
- Tests are designed to be non-destructive (read-only operations)

## CI/CD Integration

To run integration tests in CI/CD:

1. Set environment variables as secrets
2. Use the `integration` build tag to separate from unit tests
3. Consider rate limiting and API quota management

Example GitHub Actions:

```yaml
- name: Run Integration Tests
  env:
    AMADEUS_CLIENT_ID: ${{ secrets.AMADEUS_CLIENT_ID }}
    AMADEUS_CLIENT_SECRET: ${{ secrets.AMADEUS_CLIENT_SECRET }}
    GOOGLE_MAPS_API_KEY: ${{ secrets.GOOGLE_MAPS_API_KEY }}
    GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
  run: go test -v -tags=integration ./providers/integration/...
```

