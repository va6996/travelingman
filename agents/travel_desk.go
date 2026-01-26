package agents

import (
	"context"
	"fmt"

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

	// Validate Itinerary first
	if err := core.ValidateItinerary(ctx, itinerary); err != nil {
		log.Errorf(ctx, "TravelDesk: Initial validation failed: %v", err)
		return nil, err
	}

	td.checkRecursive(ctx, itinerary)
	log.Infof(ctx, "TravelDesk: Finished check.")

	return itinerary, nil
}

func (td *TravelDesk) checkRecursive(ctx context.Context, itinerary *pb.Itinerary) {
	if itinerary.Graph == nil {
		return
	}

	// 0. Determine Global Currency
	// Try to find a currency from the first pricing element or default to something sensible
	globalCurrency := "USD"
	foundCurrency := false

	// Check edges for existing currency
	for _, edge := range itinerary.Graph.Edges {
		if edge.Transport != nil && edge.Transport.Cost != nil && edge.Transport.Cost.Currency != "" {
			globalCurrency = edge.Transport.Cost.Currency
			foundCurrency = true
			break
		}
	}
	// If not found, check nodes
	if !foundCurrency {
		for _, node := range itinerary.Graph.Nodes {
			if node.Stay != nil && node.Stay.Cost != nil && node.Stay.Cost.Currency != "" {
				globalCurrency = node.Stay.Cost.Currency
				foundCurrency = true
				break
			}
		}
	}
	// If still not found, infer from Start Location of first transport
	if !foundCurrency && len(itinerary.Graph.Edges) > 0 {
		edge := itinerary.Graph.Edges[0]
		if edge.Transport != nil && edge.Transport.OriginLocation != nil {
			globalCurrency = core.GetCurrencyForCountry(edge.Transport.OriginLocation.Country)
			if globalCurrency == "" {
				globalCurrency = "USD" // Fallback
			}
		}
	}

	log.Infof(ctx, "TravelDesk: Enforcing global currency: %s", globalCurrency)

	// 1. Check Flights (Edges)
	for _, edge := range itinerary.Graph.Edges {
		if t := edge.Transport; t != nil {
			if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
				if flight := t.GetFlight(); flight != nil {
					log.Debugf(ctx, "TravelDesk: Checking flights on %s", flight.DepartureTime.AsTime().Format("2006-01-02"))

					// Enforce global currency
					if t.Cost == nil {
						t.Cost = &pb.Cost{}
					}
					t.Cost.Currency = globalCurrency

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
			limit := td.amadeus.Limits.Hotel

			count := 0
			for _, hotel := range listResp.Data {
				if count >= limit {
					break
				}
				hotelIds = append(hotelIds, hotel.HotelId)
				count++
			}

			// C. Search offers for these hotels
			checkIn := acc.CheckIn.AsTime().Format("2006-01-02")
			checkOut := acc.CheckOut.AsTime().Format("2006-01-02")

			// Use traveler count from accommodation
			adults := int(acc.TravelerCount)
			if adults <= 0 {
				adults = 1
			}

			// Enforce global currency
			if acc.Cost == nil {
				acc.Cost = &pb.Cost{}
			}
			acc.Cost.Currency = globalCurrency

			log.Debugf(ctx, "TravelDesk: Checking offers for %d hotels for %d adults...", len(hotelIds), adults)
			accommodations, err := td.amadeus.SearchHotelOffers(ctx, hotelIds, adults, checkIn, checkOut, globalCurrency)
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
