package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
)

// TravelAgent is the main orchestrator
type TravelAgent struct {
	planner Planner
	desk    Assistant
}

// NewTravelAgent creates a new TravelAgent
func NewTravelAgent(p Planner, d Assistant) *TravelAgent {
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
		log.Debugf(ctx, "Orchestration iteration %d", i+1)

		// 1. Ask Planner for a plan
		log.Infof(ctx, "STEP 1: Requesting trip plan from TripPlanner...")
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
			log.Infof(ctx, "TripPlanner requests clarification: %q", planRes.Question)
			return planRes.Question, nil
		}

		if planRes.Itinerary == nil {
			log.Errorf(ctx, "ERROR: TripPlanner returned nil itinerary.")
			return "", fmt.Errorf("planner returned no itinerary and no question")
		}
		log.Debugf(ctx, "TripPlanner proposed itinerary: %q. Proceeding to verification.", planRes.Itinerary.Title)

		var successfulItineraries []*pb.Itinerary
		var errors []string

		// 2. Parallel Verification for each proposed itinerary
		log.Infof(ctx, "STEP 2: Verifying itineraries with TravelDesk...")

		itinerariesToCheck := []*pb.Itinerary{}
		if planRes.Itinerary != nil {
			itinerariesToCheck = append(itinerariesToCheck, planRes.Itinerary)
		}
		itinerariesToCheck = append(itinerariesToCheck, planRes.PossibleItineraries...)

		type deskResult struct {
			itinerary *pb.Itinerary
			err       error
		}

		resChan := make(chan deskResult, len(itinerariesToCheck))

		for _, it := range itinerariesToCheck {
			go func(it *pb.Itinerary) {
				itinerary, err := ta.desk.CheckAvailability(ctx, it)
				if err != nil {
					resChan <- deskResult{err: err}
					return
				}
				resChan <- deskResult{itinerary: itinerary}
			}(it)
		}

		for i := 0; i < len(itinerariesToCheck); i++ {
			res := <-resChan
			if res.err != nil {
				log.Errorf(ctx, "TravelDesk verification error: %v", res.err)
				continue
			}

			// Check for errors in the itinerary
			var itineraryIssues []string
			if res.itinerary.Graph != nil {
				// Check Flights
				for _, edge := range res.itinerary.Graph.Edges {
					if edge.Transport != nil && edge.Transport.Error != nil && edge.Transport.Error.Severity == pb.ErrorSeverity_ERROR_SEVERITY_ERROR {
						itineraryIssues = append(itineraryIssues, fmt.Sprintf("Transport error: %s", edge.Transport.Error.Message))
					}
				}
				// Check Accommodation
				for _, node := range res.itinerary.Graph.Nodes {
					if node.Stay != nil && node.Stay.Error != nil && node.Stay.Error.Severity == pb.ErrorSeverity_ERROR_SEVERITY_ERROR {
						itineraryIssues = append(itineraryIssues, fmt.Sprintf("Stay error: %s", node.Stay.Error.Message))
					}
				}
			}

			// Log itinerary as JSON
			if b, err := json.MarshalIndent(res.itinerary, "", "  "); err == nil {
				log.Debugf(ctx, "TravelDesk itinerary: %s", string(b))
			} else {
				log.Debugf(ctx, "TravelDesk itinerary: %v", res.itinerary)
			}

			if len(itineraryIssues) > 0 {
				log.Warnf(ctx, "TravelDesk issues for %s: %v", res.itinerary.Title, itineraryIssues)
				errors = append(errors, fmt.Sprintf("Plan '%s': %s", res.itinerary.Title, strings.Join(itineraryIssues, "; ")))
			} else {
				successfulItineraries = append(successfulItineraries, res.itinerary)
			}
		}
		close(resChan)

		// 3. check results
		if len(successfulItineraries) == 0 {
			log.Warnf(ctx, "STEP 3: All plans had issues. Initiating re-planning...")
			// Feed issues back to Planner
			issueStr := strings.Join(errors, "\n")
			currentHistory += fmt.Sprintf("\nSystem: The proposed plans had issues:\n%s\nPlease revise.", issueStr)
			continue // Loop back to planner
		}

		// Score, Tag and Sort Itineraries and Options
		ta.scoreAndTag(successfulItineraries)

		// 4. Success! Formulate final response
		finalResponse := fmt.Sprintf("Here are the valid trip options based on your request:\n\n%s\n\n", planRes.Reasoning)

		for i, itin := range successfulItineraries {
			finalResponse += fmt.Sprintf("### Option %d: %s %s\n", i+1, itin.Title, formatTags(itin.Tags))
			finalResponse += ta.formatItinerary(itin, 0)
			finalResponse += "\n"

			// Pretty print the itinerary JSON
			b, err := json.MarshalIndent(itin, "", "  ")
			if err == nil {
				log.Debugf(ctx, "Final Itinerary JSON (Option %d):\n%s", i+1, string(b))
			}
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
				Details: fmt.Sprintf("Stay at %s (%s). Ref: %s. Price: %s %s", acc.Name, acc.Address, acc.BookingReference, acc.PriceTotal, formatTags(acc.Tags)),
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

// scoreAndTag scores, tags, and selects the best options in the itineraries
func (ta *TravelAgent) scoreAndTag(itineraries []*pb.Itinerary) {
	for _, it := range itineraries {
		if it.Graph == nil {
			continue
		}

		var totalScore float64

		// 1. Edges (Transport)
		for _, edge := range it.Graph.Edges {
			if len(edge.TransportOptions) == 0 && edge.Transport != nil {
				// If no options but transport is set, consider it the only option
				edge.TransportOptions = []*pb.Transport{edge.Transport}
			}

			if len(edge.TransportOptions) > 0 {
				// Calculate scores and find min/max for tagging
				var minPrice float64 = math.MaxFloat64
				var minDuration int64 = math.MaxInt64

				for _, t := range edge.TransportOptions {
					if float64(t.PriceTotal) < minPrice {
						minPrice = float64(t.PriceTotal)
					}

					// Calculate duration
					var duration int64
					if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT && t.GetFlight() != nil {
						f := t.GetFlight()
						if f.ArrivalTime != nil && f.DepartureTime != nil {
							duration = f.ArrivalTime.Seconds - f.DepartureTime.Seconds
						}
					}

					if duration > 0 && duration < minDuration {
						minDuration = duration
					}
				}

				// Assign Tags and Score
				type scoredTransport struct {
					t     *pb.Transport
					score float64
				}
				var scored []*scoredTransport

				for _, t := range edge.TransportOptions {
					t.Tags = []string{} // Reset tags

					// Tagging
					if float64(t.PriceTotal) == minPrice {
						t.Tags = append(t.Tags, "Cheapest")
					}

					var duration int64
					if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT && t.GetFlight() != nil {
						f := t.GetFlight()
						if f.ArrivalTime != nil && f.DepartureTime != nil {
							duration = f.ArrivalTime.Seconds - f.DepartureTime.Seconds
						}
					}

					if duration > 0 && duration == minDuration {
						t.Tags = append(t.Tags, "Fastest")
					}

					// Scoring (Lower is better)
					// Base score = Price
					score := float64(t.PriceTotal)

					// Adjust for duration (value of time?)
					// Let's say we value 1 hour at $20
					if duration > 0 {
						hours := float64(duration) / 3600.0
						score += hours * 20.0
					}

					scored = append(scored, &scoredTransport{t: t, score: score})
				}

				// Identify Best Value (Lowest Score)
				sort.Slice(scored, func(i, j int) bool {
					return scored[i].score < scored[j].score
				})

				// Tag the best score as "Best Value"
				hasBestValue := false
				for _, tag := range scored[0].t.Tags {
					if tag == "Best Value" {
						hasBestValue = true
						break
					}
				}
				if !hasBestValue {
					scored[0].t.Tags = append(scored[0].t.Tags, "Best Value")
				}

				// Reorder options
				newOptions := make([]*pb.Transport, len(scored))
				for i, s := range scored {
					newOptions[i] = s.t
				}
				edge.TransportOptions = newOptions

				// Set best option
				edge.Transport = edge.TransportOptions[0]

				// Add to itinerary total score
				totalScore += float64(edge.Transport.PriceTotal)
			}
		}

		// 2. Nodes (Accommodation)
		for _, node := range it.Graph.Nodes {
			if len(node.StayOptions) == 0 && node.Stay != nil {
				node.StayOptions = []*pb.Accommodation{node.Stay}
			}

			if len(node.StayOptions) > 0 {
				var minPrice float64 = math.MaxFloat64

				for _, s := range node.StayOptions {
					p := parsePrice(s.PriceTotal)
					if p < minPrice {
						minPrice = p
					}
				}

				type scoredStay struct {
					s     *pb.Accommodation
					score float64
					price float64
				}
				var scored []*scoredStay

				for _, s := range node.StayOptions {
					s.Tags = []string{}
					p := parsePrice(s.PriceTotal)

					if p == minPrice {
						s.Tags = append(s.Tags, "Cheapest")
					}

					// Score = Price
					score := p

					scored = append(scored, &scoredStay{s: s, score: score, price: p})
				}

				// Sort
				sort.Slice(scored, func(i, j int) bool {
					return scored[i].score < scored[j].score
				})

				// Best Value tag for top 1
				scored[0].s.Tags = append(scored[0].s.Tags, "Best Value")

				newOptions := make([]*pb.Accommodation, len(scored))
				for i, s := range scored {
					newOptions[i] = s.s
				}
				node.StayOptions = newOptions
				node.Stay = node.StayOptions[0]

				totalScore += scored[0].price
			}
		}
	}

	// Second pass: Tag Itineraries
	if len(itineraries) > 0 {
		var minTotalScore float64 = math.MaxFloat64
		type scoredItin struct {
			it    *pb.Itinerary
			score float64
		}
		var scored []*scoredItin

		for _, it := range itineraries {
			score := calculateItineraryScore(it)
			if score < minTotalScore {
				minTotalScore = score
			}
			scored = append(scored, &scoredItin{it: it, score: score})
		}

		for _, s := range scored {
			s.it.Tags = []string{}
			if s.score == minTotalScore {
				s.it.Tags = append(s.it.Tags, "Lowest Overall Cost")
			}
		}

		// Sort itineraries by score
		sort.Slice(itineraries, func(i, j int) bool {
			si := calculateItineraryScore(itineraries[i])
			sj := calculateItineraryScore(itineraries[j])
			return si < sj
		})
	}
}

func calculateItineraryScore(it *pb.Itinerary) float64 {
	var total float64
	if it.Graph == nil {
		return 0
	}
	for _, e := range it.Graph.Edges {
		if e.Transport != nil {
			total += float64(e.Transport.PriceTotal)
		}
	}
	for _, n := range it.Graph.Nodes {
		if n.Stay != nil {
			total += parsePrice(n.Stay.PriceTotal)
		}
	}
	return total
}

func parsePrice(s string) float64 {
	re := regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)
	match := re.FindString(s)
	if match == "" {
		return 0
	}
	val, _ := strconv.ParseFloat(match, 64)
	return val
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return fmt.Sprintf("[%s]", strings.Join(tags, ", "))
}
