package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/va6996/travelingman/bootstrap"
	"github.com/va6996/travelingman/config"
	logcontext "github.com/va6996/travelingman/context"
	"github.com/va6996/travelingman/log"
)

func main() {
	// Initialize logging
	log.Init()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C (SIGINT)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	go func() {
		<-sigChan
		log.Info(context.Background(), "\nProgram terminated externally. Exiting...")
		cancel()
	}()

	// 0. Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf(context.Background(), "Failed to load config: %v", err)
	}

	// 1-3. Init App Components using Bootstrap
	app, err := bootstrap.Setup(context.Background(), cfg)
	if err != nil {
		log.Fatalf(context.Background(), "Setup failed: %v", err)
	}

	// 4. Start API Server
	port := envPort()
	if port == "" {
		port = "8000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/plan", func(w http.ResponseWriter, r *http.Request) {
		handlePlanTrip(w, r, app)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Info(context.Background(), "Shutting down server...")
		srv.Shutdown(context.Background())
	}()

	log.Infof(context.Background(), "Starting server on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf(context.Background(), "Server failed: %v", err)
	}
}

func envPort() string {
	return os.Getenv("PORT")
}

type PlanTripRequest struct {
	Query string `json:"query"`
}

type PlanTripResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func handlePlanTrip(w http.ResponseWriter, r *http.Request, app *bootstrap.App) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate request ID for tracking
	requestID := logcontext.NewRequestID()

	// Add request ID to context
	ctx := logcontext.WithRequestID(r.Context(), requestID)

	var req PlanTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	log.Infof(ctx, "Received planning request: %s", req.Query)

	res, err := app.TravelAgent.OrchestrateRequest(ctx, req.Query, "")

	resp := PlanTripResponse{}
	if err != nil {
		log.Errorf(ctx, "Error processing request: %v", err)
		resp.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		resp.Result = res
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
