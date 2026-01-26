package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"github.com/va6996/travelingman/bootstrap"
	"github.com/va6996/travelingman/config"
	logcontext "github.com/va6996/travelingman/context"
	"github.com/va6996/travelingman/log"
	pb "github.com/va6996/travelingman/pb"
	"github.com/va6996/travelingman/pb/pbconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type TravelServer struct {
	app *bootstrap.App
}

func (s *TravelServer) PlanTrip(ctx context.Context, req *connect.Request[pb.PlanTripRequest]) (*connect.Response[pb.PlanTripResponse], error) {
	query := req.Msg.Query
	if query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}

	// Generate request ID for tracking
	// Connect might already have one, but let's keep our context logic
	requestID := logcontext.NewRequestID()
	ctx = logcontext.WithRequestID(ctx, requestID)

	log.Infof(ctx, "Received planning request: %s", query)

	res, itineraries, err := s.app.TravelAgent.OrchestrateRequest(ctx, query, "")
	if err != nil {
		log.Errorf(ctx, "Error processing request: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &pb.PlanTripResponse{}

	if len(itineraries) > 0 {
		response.Itineraries = itineraries
	} else if res != "" {
		// Wrap text result (likely error or explanation) in an Itinerary with Error
		response.Itineraries = []*pb.Itinerary{
			{
				Error: &pb.Error{
					Message:  res,
					Severity: pb.ErrorSeverity_ERROR_SEVERITY_ERROR,
				},
			},
		}
	}

	return connect.NewResponse(response), nil
}

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

	// Create Connect handler
	traveler := &TravelServer{app: app}
	path, handler := pbconnect.NewTravelServiceHandler(traveler)
	mux.Handle(path, handler)

	// Simple CORS middleware
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow all origins for now (dev mode)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	// Use h2c for HTTP/2 without TLS (common for dev and internal services)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: h2c.NewHandler(corsHandler(mux), &http2.Server{}),
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
