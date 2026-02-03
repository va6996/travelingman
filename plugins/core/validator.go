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

		// Validate nodes have required fields (INVARIANT 3)
		for i, node := range itinerary.Graph.Nodes {
			if node.Location == nil {
				errors = append(errors, fmt.Sprintf("Node %d (%s): Location is nil (INVARIANT 3 violation)", i, node.Id))
			}
			if node.ToTimestamp != nil && node.FromTimestamp != nil {
				// Validate temporal consistency within node
				fromTime := node.FromTimestamp.AsTime()
				toTime := node.ToTimestamp.AsTime()
				if !toTime.After(fromTime) {
					errors = append(errors, fmt.Sprintf("Node %d (%s): ToTimestamp must be after FromTimestamp", i, node.Id))
				}
			}

			// Validate accommodation if present (INVARIANT 6)
			if node.Stay != nil {
				if node.Stay.Location == nil {
					errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation.Location is nil (INVARIANT 3 violation)", i, node.Id))
				}
				if node.Stay.CheckIn == nil {
					errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation.CheckIn is nil (INVARIANT 6 violation)", i, node.Id))
				}
				if node.Stay.CheckOut == nil {
					errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation.CheckOut is nil (INVARIANT 6 violation)", i, node.Id))
				} else if node.Stay.CheckIn != nil {
					// Validate check-out is after check-in
					checkIn := node.Stay.CheckIn.AsTime()
					checkOut := node.Stay.CheckOut.AsTime()
					if !checkOut.After(checkIn) {
						errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation check-out must be after check-in", i, node.Id))
					}
				}
				if node.Stay.TravelerCount <= 0 {
					errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation.TravelerCount must be positive (INVARIANT 7 violation)", i, node.Id))
				}
				if node.Stay.Cost != nil && node.Stay.Cost.Currency == "" {
					errors = append(errors, fmt.Sprintf("Node %d (%s): Accommodation.Cost.Currency is empty (INVARIANT 8 violation)", i, node.Id))
				}
			}
		}

		// Validate edges have required fields (INVARIANT 2)
		for i, edge := range itinerary.Graph.Edges {
			if edge.Transport == nil {
				errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Transport is nil", i, edge.FromId, edge.ToId))
				continue
			}

			// INVARIANT 2: Transport locations must be non-nil
			if edge.Transport.OriginLocation == nil {
				errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Transport.OriginLocation is nil (INVARIANT 2 violation)", i, edge.FromId, edge.ToId))
			}
			if edge.Transport.DestinationLocation == nil {
				errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Transport.DestinationLocation is nil (INVARIANT 2 violation)", i, edge.FromId, edge.ToId))
			}

			// INVARIANT 7: Traveler count must be positive
			if edge.Transport.TravelerCount <= 0 {
				errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Transport.TravelerCount must be positive (INVARIANT 7 violation)", i, edge.FromId, edge.ToId))
			}

			// INVARIANT 8: Currency must be set
			if edge.Transport.Cost != nil && edge.Transport.Cost.Currency == "" {
				errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Transport.Cost.Currency is empty (INVARIANT 8 violation)", i, edge.FromId, edge.ToId))
			}

			// Validate flight details if transport type is flight (INVARIANT 5)
			if edge.Transport.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
				flight := edge.Transport.GetFlight()
				if flight == nil {
					errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Flight details missing for FLIGHT transport type", i, edge.FromId, edge.ToId))
				} else {
					if flight.DepartureTime == nil {
						errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Flight.DepartureTime is nil (INVARIANT 5 violation)", i, edge.FromId, edge.ToId))
					}
					if flight.ArrivalTime != nil && flight.DepartureTime != nil {
						// Validate arrival is after departure
						depTime := flight.DepartureTime.AsTime()
						arrTime := flight.ArrivalTime.AsTime()
						if !arrTime.After(depTime) {
							errors = append(errors, fmt.Sprintf("Edge %d (%s -> %s): Flight arrival must be after departure", i, edge.FromId, edge.ToId))
						}
					}
				}
			}
		}
	} else {
		errors = append(errors, "Graph is missing")
	}

	// 4. Check Journey Type
	switch itinerary.JourneyType {
	case pb.JourneyType_JOURNEY_TYPE_UNSPECIFIED:
		errors = append(errors, "Journey type is unspecified")
	case pb.JourneyType_JOURNEY_TYPE_RETURN:
		// A return trip must have a cycle in its graph
		if itinerary.Graph != nil && !tmcore.HasCycle(itinerary.Graph) {
			errors = append(errors, "Return trip itinerary graph must contain a cycle")
		}
	case pb.JourneyType_JOURNEY_TYPE_ONE_WAY:
		// No specific check for one way yet
		if itinerary.Graph != nil && tmcore.HasCycle(itinerary.Graph) {
			errors = append(errors, "One way trip itinerary graph must not contain a cycle")
		}
	}

	if len(errors) > 0 {
		errMsg := fmt.Sprintf("Validation Failed with %d errors:\n- %s", len(errors), strings.Join(errors, "\n- "))
		log.Errorf(ctx, "ValidateItinerary: %s", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	log.Debugf(ctx, "ValidateItinerary: Validation passed.")
	return nil
}
