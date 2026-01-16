package main

import (
	"log"
	"net/http"
	"os"

	_ "example.com/travelingman/docs"
	"example.com/travelingman/providers/amadeus"
	"example.com/travelingman/providers/gemini"
	"example.com/travelingman/tools/toolcalling"
	_ "github.com/mattn/go-sqlite3"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Travelingman API
// @version         1.0
// @description     This is the API server for the Travelingman group travel application.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8081
// @BasePath  /api/v1

func main() {
	// 1. Init DB
	db, err := InitDB("./travelingman.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := RunMigrations(db); err != nil {
		log.Fatal(err)
	}

	// 2. Init Amadeus
	clientID := os.Getenv("AMADEUS_CLIENT_ID")
	clientSecret := os.Getenv("AMADEUS_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("Error: AMADEUS_CLIENT_ID and AMADEUS_CLIENT_SECRET must be set")
	}
	amadeusClient, err := amadeus.NewClient(clientID, clientSecret, false) // false = test env
	if err != nil {
		log.Fatalf("Error: Failed to initialize Amadeus client: %v", err)
	}

	// 3. Init Gemini AI
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("Error: GEMINI_API_KEY must be set")
	}
	geminiClient, err := gemini.NewClient(geminiAPIKey)
	if err != nil {
		log.Fatalf("Error: Failed to initialize Gemini client: %v", err)
	}

	// 4. Init Genkit
	travelAI, err := toolcalling.InitAgent(amadeusClient, geminiClient)
	if err != nil {
		log.Fatalf("Error: Failed to initialize Genkit: %v", err)
	}
	_ = travelAI // Suppress unused var (until used in handler)

	// 4. Routes
	// Swagger
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// CORS Middleware
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	log.Println("Server starting on :8081...")
	if err := http.ListenAndServe(":8081", corsHandler(http.DefaultServeMux)); err != nil {
		log.Fatal(err)
	}
}
