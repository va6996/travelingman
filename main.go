package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/va6996/travelingman/bootstrap"
	"github.com/va6996/travelingman/config"
	_ "github.com/va6996/travelingman/docs"
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
	// 0. Init Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1. Init DB
	db, err := InitDB("./travelingman.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := RunMigrations(db); err != nil {
		log.Fatal(err)
	}

	// 2. Init App Components (AI, Genkit, Tools, Agent)
	_, err = bootstrap.Setup(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

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
