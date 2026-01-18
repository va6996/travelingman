package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/va6996/travelingman/bootstrap"
	"github.com/va6996/travelingman/config"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C (SIGINT)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	go func() {
		<-sigChan
		log.Println("\nProgram terminated externally. Exiting...")
		cancel()
	}()

	log.Println("Testing Agent Data Storage...")

	// 0. Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1-3. Init App Components using Bootstrap
	app, err := bootstrap.Setup(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	agent := app.Agent

	// 4. Run a query that will trigger multiple tool calls
	query := "find me hotels in paris for next weekend for 2 adults"
	log.Printf("Running PlanTrip with query: %q", query)

	result, err := agent.PlanTrip(ctx, query)
	if err != nil {
		log.Fatalf("PlanTrip failed: %v", err)
	}

	log.Println("\n---------------------------------------------------")
	log.Println("Agent Response:")
	log.Println(result)
	log.Println("---------------------------------------------------")

	// 5. Access stored data
	log.Println("\n=== STORED TRIP DATA ===")

	tripData := agent.GetTripData()
	if tripData != nil {
		log.Printf("Query: %s", tripData.Query)
		log.Printf("Created At: %s", tripData.CreatedAt.Format(time.RFC3339))

		// Get flights
		flights := agent.GetFlights()
		log.Printf("Flights found: %d", len(flights))
		for i, flight := range flights {
			log.Printf("  Flight %d: %+v", i+1, flight)
		}

		// Get hotels
		hotels := agent.GetHotels()
		log.Printf("Hotels found: %d", len(hotels))
		for i, hotel := range hotels {
			log.Printf("  Hotel %d: %+v", i+1, hotel)
		}

		// Get raw results
		rawResults := agent.GetRawResults()
		log.Printf("Total tool calls: %d", len(rawResults))
		for i, result := range rawResults {
			log.Printf("  Call %d: %s -> %v", i+1, result.ToolName, result.Output)
		}

		// Export to JSON
		jsonData, err := agent.ToJSON()
		if err != nil {
			log.Printf("Failed to export to JSON: %v", err)
		} else {
			log.Println("\n=== JSON EXPORT ===")
			fmt.Println(string(jsonData))
		}
	}
}
