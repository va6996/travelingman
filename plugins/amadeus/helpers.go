package amadeus

import (
	"github.com/va6996/travelingman/pb"
)

// enrichLocationFrom enriches the target location with data from the source location.
// It performs a union of IATA codes and fills in missing fields.
// INVARIANT: Both locations are non-nil (see docs/INVARIANTS.md)
func enrichLocationFrom(target, source *pb.Location) {
	if target == nil || source == nil {
		return
	}

	// Union IATA codes (merge unique codes)
	if len(source.IataCodes) > 0 {
		codeMap := make(map[string]bool)
		// Add existing target codes
		for _, code := range target.IataCodes {
			codeMap[code] = true
		}
		// Add source codes if not already present
		for _, code := range source.IataCodes {
			if !codeMap[code] {
				target.IataCodes = append(target.IataCodes, code)
				codeMap[code] = true
			}
		}
	}

	// Fill in missing fields
	if target.City == "" {
		target.City = source.City
	}
	if target.Country == "" {
		target.Country = source.Country
	}
	if target.CityCode == "" {
		target.CityCode = source.CityCode
	}
	if target.Name == "" {
		target.Name = source.Name
	}
}

// getLocationCode extracts the best available location code from a Location object.
// Prefers specific airport codes (IataCodes) over city codes.
// INVARIANT: Location is non-nil and enriched (see docs/INVARIANTS.md)
func getLocationCode(loc *pb.Location) string {
	if len(loc.IataCodes) > 0 {
		return loc.IataCodes[0]
	}
	return loc.CityCode
}
