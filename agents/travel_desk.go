package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/va6996/travelingman/log"
	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
	"github.com/va6996/travelingman/plugins/core"
)

// TravelDesk is responsible for checking availability and booking
type TravelDesk struct {
	amadeus *amadeus.Client
}

// NewTravelDesk creates a new TravelDesk
func NewTravelDesk(client *amadeus.Client) *TravelDesk {
	return &TravelDesk{
		amadeus: client,
	}
}

// CheckAvailability validates the itinerary against real availability
func (td *TravelDesk) CheckAvailability(ctx context.Context, itinerary *pb.Itinerary) (*pb.Itinerary, error) {
	log.Infof(ctx, "TravelDesk: Starting availability check for: %s", itinerary.Title)

	// Enrich graph first (resolve codes, set currencies)
	td.EnrichGraph(ctx, itinerary)

	// Validate Itinerary first
	if err := core.ValidateItinerary(ctx, itinerary); err != nil {
		log.Errorf(ctx, "TravelDesk: Initial validation failed: %v", err)
		return nil, err
	}

	td.checkRecursive(ctx, itinerary)
	log.Infof(ctx, "TravelDesk: Finished check.")

	return itinerary, nil
}

// EnrichGraph resolves missing city codes, names and ensures global currency
func (td *TravelDesk) EnrichGraph(ctx context.Context, itinerary *pb.Itinerary) {
	if itinerary.Graph == nil {
		return
	}

	globalCurrency := "USD"

	// Apply global currency to all nodes and edges where missing
	for _, edge := range itinerary.Graph.Edges {
		if edge.Transport != nil {
			if edge.Transport.Cost == nil {
				edge.Transport.Cost = &pb.Cost{}
			}
			if edge.Transport.Cost.Currency == "" {
				edge.Transport.Cost.Currency = globalCurrency
			}
		}
	}
	for _, node := range itinerary.Graph.Nodes {
		if node.Stay != nil {
			if node.Stay.Cost == nil {
				node.Stay.Cost = &pb.Cost{}
			}
			if node.Stay.Cost.Currency == "" {
				node.Stay.Cost.Currency = globalCurrency
			}
		}
	}

	// Enrich location information
	for _, node := range itinerary.Graph.Nodes {
		if node.Stay == nil || node.Stay.Location == nil {
			continue
		}

		if err := td.enrichLocation(ctx, node.Stay.Location); err != nil {
			log.Errorf(ctx, "TravelDesk: Location enrichment failed for %s: %v", node.Stay.Location, err)
		}
	}

	// Enrich transport information
	for _, edge := range itinerary.Graph.Edges {
		if edge.Transport == nil || edge.Transport.OriginLocation == nil {
			continue
		}

		if err := td.enrichLocation(ctx, edge.Transport.OriginLocation); err != nil {
			log.Errorf(ctx, "TravelDesk: Location enrichment failed for %s: %v", edge.Transport.OriginLocation, err)
		}

		if edge.Transport.DestinationLocation != nil {
			if err := td.enrichLocation(ctx, edge.Transport.DestinationLocation); err != nil {
				log.Errorf(ctx, "TravelDesk: Location enrichment failed for %s: %v", edge.Transport.DestinationLocation, err)
			}
		}
	}
}

func (td *TravelDesk) enrichLocation(ctx context.Context, loc *pb.Location) error {
	keywords := []string{}

	// Prioritize IATA code, then City Code, then City Name
	if len(loc.IataCodes) > 0 {
		keywords = append(keywords, loc.IataCodes[0])
	}
	if loc.CityCode != "" {
		keywords = append(keywords, loc.CityCode)
	}
	if loc.City != "" {
		keywords = append(keywords, loc.City)
	}

	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}

		location, err := td.amadeus.SearchLocations(ctx, keyword)
		if err != nil {
			log.Warnf(ctx, "TravelDesk: Location search failed for '%s': %v. Trying next fallback.", keyword, err)
			continue
		}

		if len(location) > 0 {
			// Found a match, populate and return
			bestMatch := location[0]
			maxScore := -1

			for _, l := range location {
				score := 0

				// Check IATA codes
				for _, code := range loc.IataCodes {
					for _, candidateCode := range l.IataCodes {
						if code == candidateCode {
							score += 5
							break
						}
					}
				}

				// Check City Code
				if loc.CityCode != "" && l.CityCode == loc.CityCode {
					score += 4
				}

				// Check City Name
				if loc.City != "" {
					if strings.EqualFold(l.City, loc.City) {
						score += 3
					} else if strings.Contains(strings.ToLower(l.City), strings.ToLower(loc.City)) {
						score += 1
					}
				}

				// Check Country
				if loc.Country != "" && strings.EqualFold(l.Country, loc.Country) {
					score += 2
				}

				// Prefer results with City populated
				if l.City != "" {
					score += 1
				}

				if score > maxScore {
					maxScore = score
					bestMatch = l
				}
			}

			loc.City = bestMatch.City
			loc.Country = bestMatch.Country
			loc.CityCode = bestMatch.CityCode
			loc.IataCodes = bestMatch.IataCodes
			return nil
		}
	}

	log.Warnf(ctx, "TravelDesk: Could not enrich location for %v", loc)
	return nil // Not strictly an error, just failed to enrich
}

func (td *TravelDesk) checkRecursive(ctx context.Context, itinerary *pb.Itinerary) {
	if itinerary.Graph == nil {
		return
	}

	// 1. Check Flights (Edges)
	for _, edge := range itinerary.Graph.Edges {
		if t := edge.Transport; t != nil {
			if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
				if flight := t.GetFlight(); flight != nil {
					log.Debugf(ctx, "TravelDesk: Checking flights on %s", flight.DepartureTime.AsTime().Format("2006-01-02"))

					// SearchFlights handles location extraction internally
					transports, err := td.amadeus.SearchFlights(ctx, t)

					if err != nil {
						errMsg := fmt.Sprintf("Flight search failed: %s", err)
						log.Errorf(ctx, "TravelDesk: ISSUE: %s", errMsg)
						t.Error = &pb.Error{
							Message:  errMsg,
							Code:     td.amadeus.MapError(err),
							Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
						}
					} else if len(transports) > 0 {
						// Collect ALL flight options
						edge.TransportOptions = transports

						// ENRICH LOCATIONS
						// Amadeus SearchFlights returns IATA codes but no city names. We need to fetch them.
						// We'll gather all unique codes from the results and do a lookup.
						codesToEnrich := make(map[string]bool)
						for _, tr := range transports {
							if tr.OriginLocation != nil && len(tr.OriginLocation.IataCodes) > 0 {
								codesToEnrich[tr.OriginLocation.IataCodes[0]] = true
							}
							if tr.DestinationLocation != nil && len(tr.DestinationLocation.IataCodes) > 0 {
								codesToEnrich[tr.DestinationLocation.IataCodes[0]] = true
							}
						}

						// Loop and enrich (Naive approach: 1 lookup per code. In prod, use cache/batch)
						// We can use td.amadeus.SearchLocations or a local cache
						// For now, let's just do it
						locationDetails := make(map[string]string) // Code -> CityName

						for code := range codesToEnrich {
							// Use existing LocationTool logic or direct Client call
							// Client.SearchLocations returns list.
							// We can try to use the cache in Client if we implemented it, or just call it.
							// Given this is a demo/prototype, calling it is fine, but maybe limit concurrency.
							locs, err := td.amadeus.SearchLocations(ctx, code)
							if err == nil && len(locs) > 0 {
								// find the one matching the code
								for _, l := range locs {
									// SearchLocations returns matches. Need to filter.
									// Actually SearchLocations("SFO") returns SFO airport details which includes City Name
									// We'll just take the city name from the first result if it matches or seems relevant
									if l.City != "" {
										locationDetails[code] = l.City
										break
									}
								}
							}
						}

						// Apply enrichment
						for _, tr := range transports {
							// Origin
							if tr.OriginLocation != nil && len(tr.OriginLocation.IataCodes) > 0 {
								code := tr.OriginLocation.IataCodes[0]
								if name, ok := locationDetails[code]; ok {
									tr.OriginLocation.City = name
								}
							}
							// Dest
							if tr.DestinationLocation != nil && len(tr.DestinationLocation.IataCodes) > 0 {
								code := tr.DestinationLocation.IataCodes[0]
								if name, ok := locationDetails[code]; ok {
									tr.DestinationLocation.City = name
								}
							}
						}

						log.Infof(ctx, "TravelDesk: Found %d flight options", len(transports))
					} else {
						// ... existing error handling ...
						errMsg := fmt.Sprintf("No flights found for %s on %s", t.OriginLocation.IataCodes, flight.DepartureTime.AsTime().Format("2006-01-02"))
						log.Errorf(ctx, "TravelDesk: ISSUE: %s", errMsg)
						t.Error = &pb.Error{
							Message:  errMsg,
							Code:     pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND,
							Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
						}
					}
				}
			}
		}
	}

	// 2. Check Hotels (Nodes)
	for _, node := range itinerary.Graph.Nodes {
		if acc := node.Stay; acc != nil {
			log.Debugf(ctx, "TravelDesk: Checking hotels in city %s", acc.Address)

			// Direct API Flow:
			// A. Search hotels by city to            // Use preferences
			listResp, err := td.amadeus.SearchHotelsByCity(ctx, acc) // acc.Address holds CityCode
			if err != nil {
				errMsg := fmt.Sprintf("Hotel city search failed for %s: %s", acc.Address, err)
				log.Errorf(ctx, "TravelDesk: ISSUE: %s", errMsg)
				acc.Error = &pb.Error{
					Message:  errMsg,
					Code:     td.amadeus.MapError(err),
					Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
				}
				continue
			}

			if len(listResp.Data) == 0 {
				errMsg := fmt.Sprintf("No hotels found in city %s", acc.Address)
				log.Errorf(ctx, "TravelDesk: ISSUE: %s", errMsg)
				acc.Error = &pb.Error{
					Message:  errMsg,
					Code:     pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND,
					Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
				}
				continue
			}

			// B. Pick top hotels to check for offers
			var hotelIds []string
			limit := td.amadeus.Config.HotelLimit

			count := 0
			for _, hotel := range listResp.Data {
				if count >= limit {
					break
				}
				hotelIds = append(hotelIds, hotel.HotelId)
				count++
			}

			// C. Search offers for these hotels

			// Use traveler count from accommodation
			adults := int(acc.TravelerCount)
			if adults <= 0 {
				adults = 1
			}

			// Enforce global currency
			if acc.Cost == nil {
				acc.Cost = &pb.Cost{}
			}

			log.Debugf(ctx, "TravelDesk: Checking offers for %d hotels for %d adults...", len(hotelIds), adults)
			accommodations, err := td.amadeus.SearchHotelOffers(ctx, hotelIds, acc)
			if err != nil {
				// SearchHotelOffers might error if none available or API error
				errMsg := fmt.Sprintf("Hotel offers search failed: %s", err)
				log.Infof(ctx, "TravelDesk: %s", errMsg)
				acc.Error = &pb.Error{
					Message:  errMsg,
					Code:     td.amadeus.MapError(err),
					Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
				}
				// Do not add to issues, just log and continue
				continue
			} else if len(accommodations) > 0 {
				node.StayOptions = accommodations

				log.Infof(ctx, "TravelDesk: Found %d hotel options", len(accommodations))
			} else {
				// No data returned
				acc.Status = "NO_OFFERS"
				errMsg := fmt.Sprintf("No hotel offers found in %s", acc.Address)
				acc.Error = &pb.Error{
					Message:  errMsg,
					Code:     pb.ErrorCode_ERROR_CODE_DATA_NOT_FOUND,
					Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
				}
				log.Infof(ctx, "TravelDesk: %s", errMsg)
			}
		}
	}

	// 3. Recurse for sub-graph if needed
	if itinerary.Graph.SubGraph != nil {
		subItin := &pb.Itinerary{Graph: itinerary.Graph.SubGraph}
		td.checkRecursive(ctx, subItin)
	}
}
