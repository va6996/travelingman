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

						log.Infof(ctx, "TravelDesk: Found %d flight options", len(transports))
					} else {
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
			for _, hotel := range listResp.Data {
				hotelIds = append(hotelIds, hotel.HotelId)
			}

			// C. Search offers for these hotels
			checkIn := acc.CheckIn.AsTime().Format("2006-01-02")
			checkOut := acc.CheckOut.AsTime().Format("2006-01-02")

			// Use traveler count from accommodation
			adults := int(acc.TravelerCount)
			if adults <= 0 {
				adults = 1
			}

			log.Debugf(ctx, "TravelDesk: Checking offers for %d hotels for %d adults...", len(hotelIds), adults)
			accommodations, err := td.amadeus.SearchHotelOffers(ctx, hotelIds, adults, checkIn, checkOut)
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
