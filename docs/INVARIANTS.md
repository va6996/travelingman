# System Invariants

This document describes the key invariants maintained throughout the TravelingMan system to ensure data consistency and simplify code.

## Core Data Invariants

### INVARIANT 1: Location Enrichment Happens Before API Calls

**Established**: `agents/travel_desk.go` in `CheckAvailability()` → `EnrichGraph()` → `enrichLocation()`

**Guarantee**: All `Location` objects in the itinerary graph are enriched with complete information before any external API calls are made.

**Details**:
- `Location.IataCodes` populated when available (e.g., ["JFK"])
- `Location.City` populated (e.g., "New York")
- `Location.Country` populated (e.g., "United States")
- `Location.CityCode` populated for city-level searches (e.g., "NYC")

**Relied Upon By**:
- `plugins/amadeus/flights.go:SearchFlights()` - assumes locations are enriched
- `plugins/amadeus/hotels.go:SearchHotelsByCity()` - assumes city codes are available
- All Amadeus API methods

---

### INVARIANT 2: Transport Always Has Non-Nil Locations

**Established**: Graph construction in `agents/trip_planner.go`

**Guarantee**: Every `Transport` object has non-nil `OriginLocation` and `DestinationLocation`.

**Details**:
```go
transport.OriginLocation != nil
transport.DestinationLocation != nil
// At least one of these is populated after enrichment:
len(transport.OriginLocation.IataCodes) > 0 || transport.OriginLocation.CityCode != ""
len(transport.DestinationLocation.IataCodes) > 0 || transport.DestinationLocation.CityCode != ""
```

**Relied Upon By**:
- All transport processing code can safely dereference locations
- `SearchFlights()` uses `getLocationCode()` without nil checks

---

### INVARIANT 3: Accommodation Always Has Non-Nil Location

**Established**: Graph construction and ORM layer

**Guarantee**: Every `Accommodation` has a non-nil `Location` object with city/country information.

**Details**:
```go
accommodation.Location != nil
accommodation.Location.City != "" || accommodation.Location.CityCode != ""
```

**Relied Upon By**:
- Hotel search APIs
- UI rendering of accommodation locations
- `SearchHotelsByCity()` uses `getLocationCode()` without nil checks

---

## Validation Invariants

### INVARIANT 4: All Required Fields Validated Before API Calls

**Established**: `ValidateItinerary()` called in `travel_desk.go:CheckAvailability()`

**Guarantee**: All fields required for API calls are validated before any external API calls are made.

**Details**:
- Departure dates are present and valid (not in past, not year 1)
- Check-in/check-out dates are present and valid
- Traveler counts are positive (>= 1)
- Flight details exist when transport type is flight
- All required fields are non-empty

**Relied Upon By**:
- `SearchFlights()` - skips date/traveler validation
- `SearchHotels()` - skips required field checks
- All Amadeus API methods

---

### INVARIANT 5: Flight Departure Time Always Set

**Established**: `ValidateItinerary()` validates flight details

**Guarantee**: Flight departure time is always non-nil and valid.

**Details**:
```go
flight.DepartureTime != nil
// Departure time is not in the past
// Departure time is a valid date
```

**Relied Upon By**:
- `SearchFlights()` directly formats departure time without nil check

---

### INVARIANT 6: Accommodation Check-In/Out Always Set

**Established**: `ValidateItinerary()` validates accommodation dates

**Guarantee**: Accommodation check-in and check-out times are always non-nil and valid.

**Details**:
```go
accommodation.CheckIn != nil
accommodation.CheckOut != nil
// check-out is after check-in
```

**Relied Upon By**:
- Hotel search APIs can directly use check-in/out without nil checks

---

### INVARIANT 7: Traveler Count Always Positive

**Established**: `ValidateItinerary()` validates traveler counts

**Guarantee**: Traveler count is always >= 1 for both transport and accommodation.

**Details**:
```go
transport.TravelerCount >= 1
accommodation.TravelerCount >= 1
```

**Relied Upon By**:
- API methods don't need to default traveler count to 1
- No need for `if adults <= 0 { adults = 1 }` checks

---

## System Invariants

### INVARIANT 8: Currency Always Set

**Established**: `agents/travel_desk.go:EnrichGraph()` sets global currency

**Guarantee**: All cost objects have currency populated.

**Details**:
```go
cost.Currency != ""
// Global currency is set from itinerary or defaults to USD
```

**Relied Upon By**:
- API methods use currency without nil/empty checks
- `SearchFlights()` directly uses `transport.Cost.Currency`

---

### INVARIANT 9: Database Always Initialized

**Established**: Application initialization

**Guarantee**: The database connection is always initialized.

**Details**:
```go
client.DB != nil
// DB is initialized at startup
```

**Relied Upon By**:
- No need to check `if c.DB != nil` before DB operations
- Cache persistence operations

---

### INVARIANT 10: API Call Order

**Order**: Enrichment → Validation → API Calls

**Ensures**: All data is complete and valid before making expensive external API calls.

**Flow**:
1. `CheckAvailability()` receives itinerary
2. `EnrichGraph()` enriches all locations and sets currency
3. `ValidateItinerary()` validates all required fields
4. `checkRecursive()` makes API calls to Amadeus

---

## Code Simplification Benefits

These invariants allow us to:

1. **Remove defensive nil checks** - Locations, dates, and other fields are guaranteed to exist
2. **Simplify extraction logic** - Direct access to location codes via `getLocationCode()`
3. **Fail fast** - Invariant violations indicate bugs, not expected cases
4. **Improve performance** - No redundant validation in API methods
5. **Clearer code** - Intent is documented through invariants

## Maintaining Invariants

**When adding new code**:
- Ensure location enrichment happens in `EnrichGraph()` before API calls
- Don't create Transport/Accommodation without Location objects
- Add validation checks to `ValidateItinerary()` for new required fields
- Document any new invariants in this file

**When debugging**:
- If location data is missing, check `enrichLocation()` was called
- Verify the order: enrichment → validation → API calls
- Check that graph construction populates all required fields
- Look for invariant violation error messages

## Verification Checklist

To verify invariants are maintained:

- [ ] All locations enriched before `ValidateItinerary()` call
- [ ] `ValidateItinerary()` called before any API calls
- [ ] No nil checks for locations in API methods
- [ ] No date validation in API methods (handled by `ValidateItinerary()`)
- [ ] No traveler count defaults in API methods
- [ ] No currency nil checks in API methods
- [ ] No DB nil checks before cache operations

---

### INVARIANT 11: Service Configuration Always Pre-Initialized

**Established**: `bootstrap/setup.go` in `Setup()`

**Guarantee**: All service and plugin configuration (API keys, Base URLs, Models) is fully resolved and populated in the struct before `Init()` is called.

**Details**:
```go
plugin := &Plugin{
    APIKey:  cfg.Key, // Loaded from config/env by setup logic
    BaseURL: cfg.URL,
}
// Plugin.Init() is called on fully configured struct
```

**Relied Upon By**:
- `bootstrap/zai/zai.go:Init()` - assumes APIKey and BaseURL are set
- Reduces need for `os.Getenv` calls inside plugin logic
- Centralizes configuration management in `bootstrap/setup.go`
