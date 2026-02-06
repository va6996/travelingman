package main

import (
	"embed"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	pathpkg "path"
	"strings"
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

//go:embed ui/dist
var uiFS embed.FS

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

	// Create a sub-filesystem for ui/dist
	uiSubFS, err := fs.Sub(uiFS, "ui/dist")
	if err != nil {
		log.Fatalf(context.Background(), "Failed to create UI sub-filesystem: %v", err)
	}

	// Create file server for embedded UI
	uiFileServer := http.FileServer(http.FS(uiSubFS))

	// SPA fallback handler - serves index.html for non-API routes
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set proper MIME types
		ext := strings.ToLower(pathpkg.Ext(r.URL.Path))
		switch ext {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".html":
			w.Header().Set("Content-Type", "text/html")
		case ".woff":
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		case ".otf":
			w.Header().Set("Content-Type", "font/otf")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg":
		case ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		}

		// Try to serve the file from the embedded filesystem
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if cleanPath == "" {
			cleanPath = "."
		}

		_, err := uiSubFS.Open(cleanPath)
		if err == nil {
			// File exists, serve it
			uiFileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist, fallback to index.html for SPA routing
		indexFile, err := uiSubFS.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()

		// Get file info for Content-Type header
		stat, _ := indexFile.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(interface {
			io.ReadSeeker
		}))
	})

	// Register UI handler for all non-API routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes go to Connect handler
		if strings.HasPrefix(r.URL.Path, "/TravelService") {
			handler.ServeHTTP(w, r)
			return
		}
		// All other routes go to SPA handler
		spaHandler.ServeHTTP(w, r)
	})

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
