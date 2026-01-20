package agents

import (
	"context"
	"fmt"
	"log"

	"github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/plugins/amadeus"
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

// BookingRequest contains the itinerary to book/check
type BookingRequest struct {
	Itinerary *pb.Itinerary
}

// BookingResult contains the updated itinerary with booking details
type BookingResult struct {
	Itinerary *pb.Itinerary
	Issues    []string
}

// CheckAvailabilityAndBook validates the itinerary against real availability
func (td *TravelDesk) CheckAvailabilityAndBook(ctx context.Context, req BookingRequest) (*BookingResult, error) {
	log.Printf("TravelDesk: Starting availability check for: %s", req.Itinerary.Title)
	issues := td.checkRecursive(ctx, req.Itinerary)
	log.Printf("TravelDesk: Finished check. Total issues found: %d", len(issues))

	return &BookingResult{
		Itinerary: req.Itinerary,
		Issues:    issues,
	}, nil
}

func (td *TravelDesk) checkRecursive(ctx context.Context, itinerary *pb.Itinerary) []string {
	var issues []string
	if itinerary.Graph == nil {
		return issues
	}

	// 1. Check Flights (Edges)
	for _, edge := range itinerary.Graph.Edges {
		if t := edge.Transport; t != nil {
			if t.Type == pb.TransportType_TRANSPORT_TYPE_FLIGHT {
				if flight := t.GetFlight(); flight != nil {
					log.Printf("TravelDesk: Checking flights on %s", flight.DepartureTime.AsTime().Format("2006-01-02"))

					// SearchFlights handles location extraction internally
					resp, err := td.amadeus.SearchFlights(ctx, t)

					if err != nil {
						issues = append(issues, fmt.Sprintf("Flight search failed: %v", err))
					} else if len(resp.Data) > 0 {
						// Collect ALL flight options
						var options []*pb.Transport
						for _, offer := range resp.Data {
							options = append(options, offer.ToTransport())
						}
						edge.TransportOptions = options

						// Set first option as selected (backward compatibility)
						offer := resp.Data[0]
						t.Status = "AVAILABLE"
						t.ReferenceNumber = offer.ID

						// Update flight details
						if len(offer.Itineraries) > 0 && len(offer.Itineraries[0].Segments) > 0 {
							seg := offer.Itineraries[0].Segments[0]
							t.Details.(*pb.Transport_Flight).Flight.FlightNumber = seg.Number
							t.Details.(*pb.Transport_Flight).Flight.CarrierCode = seg.CarrierCode
						}
						log.Printf("TravelDesk: Found %d flight options, selected: %s price: %s", len(options), offer.ID, offer.Price.Total)
					}
				}
			}
		}
	}

	// 2. Check Hotels (Nodes)
	for _, node := range itinerary.Graph.Nodes {
		if acc := node.Stay; acc != nil {
			log.Printf("TravelDesk: Checking hotels in city %s", acc.Address)

			// Direct API Flow:
			// A. Search hotels by city to            // Use preferences
			listResp, err := td.amadeus.SearchHotelsByCity(ctx, acc) // acc.Address holds CityCode
			if err != nil {
				issues = append(issues, fmt.Sprintf("Hotel city search failed for %s: %v", acc.Address, err))
				continue
			}

			if len(listResp.Data) == 0 {
				issues = append(issues, fmt.Sprintf("No hotels found in city %s", acc.Address))
				continue
			}

			// B. Pick top hotels to check for offers
			var hotelIds []string
			limit := 5
			if len(listResp.Data) < limit {
				limit = len(listResp.Data)
			}
			for i := 0; i < limit; i++ {
				hotelIds = append(hotelIds, listResp.Data[i].HotelId)
			}

			// C. Search offers for these hotels
			checkIn := acc.CheckIn.AsTime().Format("2006-01-02")
			checkOut := acc.CheckOut.AsTime().Format("2006-01-02")

			// Use traveler count from accommodation
			adults := int(acc.TravelerCount)
			if adults <= 0 {
				adults = 1
			}

			log.Printf("TravelDesk: Checking offers for %d hotels for %d adults...", len(hotelIds), adults)
			accommodations, err := td.amadeus.SearchHotelOffersAsAccommodations(ctx, hotelIds, adults, checkIn, checkOut)
			if err != nil {
				// SearchHotelOffersAsAccommodations might error if none available or API error
				issues = append(issues, fmt.Sprintf("Hotel offers search failed: %v", err))
			} else if len(accommodations) > 0 {
				node.StayOptions = accommodations

				// Set first option as selected (backward compatibility)
				firstOffer := accommodations[0]
				acc.Status = "AVAILABLE"
				acc.BookingReference = firstOffer.BookingReference
				acc.PriceTotal = firstOffer.PriceTotal
				acc.Name = firstOffer.Name

				log.Printf("TravelDesk: Found %d hotel options, selected: %s price: %s", len(accommodations), acc.Name, acc.PriceTotal)
			} else {
				// No data returned
				acc.Status = "NO_OFFERS"
				issues = append(issues, fmt.Sprintf("No hotel offers found in %s", acc.Address))
			}
		}
	}

	// 3. Recurse for sub-graph if needed
	if itinerary.Graph.SubGraph != nil {
		subItin := &pb.Itinerary{Graph: itinerary.Graph.SubGraph}
		subIssues := td.checkRecursive(ctx, subItin)
		issues = append(issues, subIssues...)
	}

	return issues
}
