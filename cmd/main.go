package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/travelingman/providers/amadeus"
	"example.com/travelingman/providers/gemini"
	"example.com/travelingman/tools/toolcalling"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present
	_ = godotenv.Load()

	log.Println("Testing Agent Data Storage...")

	// 1. Setup Amadeus Client
	amadeusClientID := os.Getenv("AMADEUS_CLIENT_ID")
	amadeusClientSecret := os.Getenv("AMADEUS_CLIENT_SECRET")
	if amadeusClientID == "" || amadeusClientSecret == "" {
		log.Fatal("Error: AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}

	amadeusClient, err := amadeus.NewClient(amadeusClientID, amadeusClientSecret, false)
	if err != nil {
		log.Fatalf("Failed to create Amadeus client: %v", err)
	}

	// 2. Setup Gemini Client
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("Error: GEMINI_API_KEY must be set")
	}

	geminiClient, err := gemini.NewClient(geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer geminiClient.Close()

	// 3. Init Agent
	log.Println("Initializing Agent...")
	agent, err := toolcalling.InitAgent(amadeusClient, geminiClient)
	if err != nil {
		log.Fatalf("Failed to initialize Agent: %v", err)
	}

	// 4. Run a query that will trigger multiple tool calls
	query := "find me hotels in paris for next weekend for 2 adults"
	log.Printf("Running PlanTrip with query: %q", query)

	ctx := context.Background()
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
