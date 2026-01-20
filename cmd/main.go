package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	// 4. Run a query that will trigger multiple tool calls
	query := "find me hotels in paris for next weekend for 2 adults"
	log.Printf("Running TravelAgent with query: %q", query)

	result, err := app.TravelAgent.OrchestrateRequest(ctx, query, "")
	if err != nil {
		log.Fatalf("OrchestrateRequest failed: %v", err)
	}

	log.Println("\n---------------------------------------------------")
	log.Println("Agent Response:")
	log.Println(result)
	log.Println("---------------------------------------------------")
}
