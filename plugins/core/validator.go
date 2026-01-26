package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	tmcore "github.com/va6996/travelingman/core"
	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
)

// ValidateItinerary checks itinerary logic for consistency
func ValidateItinerary(ctx context.Context, itinerary *pb.Itinerary) error {
	log.Debugf(ctx, "Validating itinerary: %s", itinerary.Title)

	// Perform Checks
	var errors []string

	// 1. Check basic fields
	if itinerary.Title == "" {
		errors = append(errors, "Title is missing")
	}

	// 2. Check Dates
	start := itinerary.StartTime.AsTime()
	end := itinerary.EndTime.AsTime()
	// Use yesterday as buffer to account for timezones
	yesterday := time.Now().AddDate(0, 0, -1)

	if !start.IsZero() {
		if start.Before(yesterday) {
			errors = append(errors, fmt.Sprintf("Start time (%s) is in the past", start))
		}
	} else {
		errors = append(errors, "Start time missing")
	}

	if !end.IsZero() {
		if !start.IsZero() && end.Before(start) {
			errors = append(errors, fmt.Sprintf("End time (%s) is before start time (%s)", end, start))
		}
	} else {
		errors = append(errors, "End time missing")
	}

	if itinerary.Travelers <= 0 {
		errors = append(errors, fmt.Sprintf("Invalid traveler count: %d", itinerary.Travelers))
	}

	// 3. Graph Logic
	if itinerary.Graph != nil {
		if err := tmcore.ValidateGraph(itinerary.Graph); err != nil {
			errors = append(errors, fmt.Sprintf("Graph validation failed: %v", err))
		}
	} else {
		errors = append(errors, "Graph is missing")
	}

	// 4. Check Journey Type
	if itinerary.JourneyType == pb.JourneyType_JOURNEY_TYPE_UNSPECIFIED {
		errors = append(errors, "Journey type is unspecified")
	} else if itinerary.JourneyType == pb.JourneyType_JOURNEY_TYPE_RETURN {
		// A return trip must have a cycle in its graph
		if itinerary.Graph != nil && !tmcore.HasCycle(itinerary.Graph) {
			errors = append(errors, "Return trip itinerary graph must contain a cycle")
		}
	} else if itinerary.JourneyType == pb.JourneyType_JOURNEY_TYPE_ONE_WAY {
		// No specific check for one way yet
	}

	if len(errors) > 0 {
		errMsg := fmt.Sprintf("Validation Failed with %d errors:\n- %s", len(errors), strings.Join(errors, "\n- "))
		log.Errorf(ctx, "ValidateItinerary: %s", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	log.Debugf(ctx, "ValidateItinerary: Validation passed.")
	return nil
}
