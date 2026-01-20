package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/va6996/travelingman/pb"
)

// TravelAgent is the main orchestrator
type TravelAgent struct {
	planner Planner
	desk    *TravelDesk
}

// NewTravelAgent creates a new TravelAgent
func NewTravelAgent(p Planner, d *TravelDesk) *TravelAgent {
	return &TravelAgent{
		planner: p,
		desk:    d,
	}
}

// OrchestrateRequest handles the end-to-end planning process
func (ta *TravelAgent) OrchestrateRequest(ctx context.Context, userQuery string, history string) (string, error) {
	currentHistory := history
	maxIterations := 5

	for i := 0; i < maxIterations; i++ {
		log.Printf("Orchestration iteration %d", i+1)

		// 1. Ask Planner for a plan
		log.Println("STEP 1: Requesting trip plan from TripPlanner...")
		planReq := PlanRequest{
			UserQuery: userQuery,
			History:   currentHistory,
		}

		planRes, err := ta.planner.Plan(ctx, planReq)
		if err != nil {
			return "", fmt.Errorf("planner error: %w", err)
		}

		// If Planner needs user clarification, return immediately
		if planRes.NeedsClarification {
			log.Printf("TripPlanner requests clarification: %q", planRes.Question)
			return planRes.Question, nil
		}

		if planRes.Itinerary == nil {
			log.Println("ERROR: TripPlanner returned nil itinerary.")
			return "", fmt.Errorf("planner returned no itinerary and no question")
		}
		log.Printf("TripPlanner proposed itinerary: %q. Proceeding to verification.", planRes.Itinerary.Title)

		var successfulItineraries []*pb.Itinerary
		var errors []string

		// 2. Parallel Verification for each proposed itinerary
		log.Println("STEP 2: Verifying itineraries with TravelDesk...")

		itinerariesToCheck := []*pb.Itinerary{}
		if planRes.Itinerary != nil {
			itinerariesToCheck = append(itinerariesToCheck, planRes.Itinerary)
		}
		itinerariesToCheck = append(itinerariesToCheck, planRes.PossibleItineraries...)

		type deskResult struct {
			itinerary *pb.Itinerary
			issues    []string
			err       error
		}

		resChan := make(chan deskResult, len(itinerariesToCheck))

		for _, it := range itinerariesToCheck {
			go func(it *pb.Itinerary) {
				deskReq := BookingRequest{
					Itinerary: it,
				}
				deskRes, err := ta.desk.CheckAvailabilityAndBook(ctx, deskReq)
				if err != nil {
					resChan <- deskResult{err: err}
					return
				}
				resChan <- deskResult{itinerary: deskRes.Itinerary, issues: deskRes.Issues}
			}(it)
		}

		for i := 0; i < len(itinerariesToCheck); i++ {
			res := <-resChan
			if res.err != nil {
				log.Printf("TravelDesk verification error: %v", res.err)
				continue
			}
			if len(res.issues) > 0 {
				log.Printf("TravelDesk issues for %s: %v", res.itinerary.Title, res.issues)
				errors = append(errors, fmt.Sprintf("Plan '%s': %s", res.itinerary.Title, strings.Join(res.issues, "; ")))
			} else {
				successfulItineraries = append(successfulItineraries, res.itinerary)
			}
		}
		close(resChan)

		// 3. check results
		if len(successfulItineraries) == 0 {
			log.Printf("STEP 3: All plans had issues. Initiating re-planning...")
			// Feed issues back to Planner
			issueStr := strings.Join(errors, "\n")
			currentHistory += fmt.Sprintf("\nSystem: The proposed plans had issues:\n%s\nPlease revise.", issueStr)
			continue // Loop back to planner
		}
		// 4. Success! Formulate final response
		finalResponse := fmt.Sprintf("Here are the valid trip options based on your request:\n\n%s\n\n", planRes.Reasoning)

		for i, itin := range successfulItineraries {
			finalResponse += fmt.Sprintf("### Option %d: %s\n", i+1, itin.Title)
			finalResponse += ta.formatItinerary(itin, 0)
			finalResponse += "\n"
		}

		return finalResponse, nil
	}

	return "I'm having trouble finding a plan that works with current availability. Can we try adjusting your criteria?", nil
}

type itineraryItem struct {
	Time    string
	EndTime string
	Details string
	SortKey string
}

func (ta *TravelAgent) formatItinerary(it *pb.Itinerary, indentLevel int) string {
	var items []itineraryItem
	indent := strings.Repeat("  ", indentLevel)

	if it.Graph == nil {
		return ""
	}

	// Collect Accommodation (Nodes)
	for _, node := range it.Graph.Nodes {
		if acc := node.Stay; acc != nil {
			start := acc.CheckIn.AsTime()
			end := acc.CheckOut.AsTime()
			items = append(items, itineraryItem{
				Time:    start.Format("Jan 02 15:04"),
				EndTime: end.Format("Jan 02 15:04"),
				Details: fmt.Sprintf("Stay at %s (%s). Ref: %s. Price: %s", acc.Name, acc.Address, acc.BookingReference, acc.PriceTotal),
				SortKey: start.Format(time.RFC3339),
			})
		}
	}

	// Collect Transport (Edges)
	for _, edge := range it.Graph.Edges {
		if t := edge.Transport; t != nil {
			// Try to find a time for sorting
			var sortTime string
			var description string

			if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
				if f := t.GetFlight(); f != nil {
					dep := f.DepartureTime.AsTime()
					sortTime = dep.Format(time.RFC3339)

					origin := "Unknown"
					if t.OriginLocation != nil {
						if len(t.OriginLocation.IataCodes) > 0 {
							origin = t.OriginLocation.IataCodes[0]
						} else {
							origin = t.OriginLocation.CityCode
						}
					}

					dest := "Unknown"
					if t.DestinationLocation != nil {
						if len(t.DestinationLocation.IataCodes) > 0 {
							dest = t.DestinationLocation.IataCodes[0]
						} else {
							dest = t.DestinationLocation.CityCode
						}
					}

					description = fmt.Sprintf("Flight %s %s from %s to %s. Departs: %s.",
						f.CarrierCode, f.FlightNumber, origin, dest, dep.Format("Jan 02 15:04"))
				}
			} else {
				// fallback
				sortTime = "9999" // end of list if unknown
				description = fmt.Sprintf("Transport: %s", t.Type)
			}

			items = append(items, itineraryItem{
				Time:    "", // Already in description if relevant
				Details: fmt.Sprintf("%s Ref: %s", description, t.ReferenceNumber),
				SortKey: sortTime,
			})
		}
	}

	// Collect Sub-Graph
	if it.Graph.SubGraph != nil {
		subDetails := ta.formatItinerary(&pb.Itinerary{Graph: it.Graph.SubGraph}, indentLevel+1)
		items = append(items, itineraryItem{
			Time:    "",
			Details: fmt.Sprintf("Sub-Trip Details:\n%s", subDetails),
			SortKey: "9999",
		})
	}

	// Sort items
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].SortKey > items[j].SortKey {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Build string
	var sb strings.Builder
	for _, item := range items {
		if item.Time != "" {
			sb.WriteString(fmt.Sprintf("%s- [%s] %s\n", indent, item.Time, item.Details))
		} else {
			sb.WriteString(fmt.Sprintf("%s- %s\n", indent, item.Details))
		}
	}
	return sb.String()
}
