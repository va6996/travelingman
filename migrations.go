package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"email" TEXT UNIQUE,
			"password_hash" TEXT,
			"full_name" TEXT,
			"created_at" DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS passports (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"user_id" INTEGER,
			"number" TEXT,
			"issuing_country" TEXT,
			"expiry_date" DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS drivers_licenses (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"user_id" INTEGER,
			"number" TEXT,
			"issuing_country" TEXT,
			"expiry_date" DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS travel_groups (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"name" TEXT,
			"organizer_id" INTEGER,
			"destination" TEXT,
			"travel_date" DATETIME,
			FOREIGN KEY(organizer_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS group_members (
			"group_id" INTEGER,
			"user_id" INTEGER,
			"role" TEXT,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY(group_id) REFERENCES travel_groups(id),
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS flight_offers (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"group_id" INTEGER,
			"amadeus_offer_id" TEXT,
			"carrier_code" TEXT,
			"flight_number" TEXT,
			"departure_airport" TEXT,
			"arrival_airport" TEXT,
			"departure_time" DATETIME,
			"arrival_time" DATETIME,
			"price_total" TEXT,
			"currency" TEXT,
			"raw_data" TEXT,
			FOREIGN KEY(group_id) REFERENCES travel_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS bookings (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"user_id" INTEGER,
			"type" TEXT,
			"provider" TEXT,
			"status" TEXT,
			"external_booking_reference" TEXT,
			"created_at" DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS payments (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"booking_id" INTEGER,
			"user_id" INTEGER,
			"amount" TEXT,
			"currency" TEXT,
			"status" TEXT,
			"transaction_id" TEXT,
			"created_at" DATETIME,
			FOREIGN KEY(booking_id) REFERENCES bookings(id),
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS itinerary_items (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"group_id" INTEGER,
			"type" TEXT,
			"title" TEXT,
			"start_time" DATETIME,
			"end_time" DATETIME,
			"details_json" BLOB,
			FOREIGN KEY(group_id) REFERENCES travel_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS accommodations (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"group_id" INTEGER,
			"name" TEXT,
			"address" TEXT,
			"check_in" DATETIME,
			"check_out" DATETIME,
			"price_total" TEXT,
			"booking_reference" TEXT,
			"status" TEXT,
			FOREIGN KEY(group_id) REFERENCES travel_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS hotel_offers (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"group_id" INTEGER,
			"hotel_name" TEXT,
			"check_in" DATETIME,
			"check_out" DATETIME,
			"price_total" TEXT,
			"currency" TEXT,
			"offer_id" TEXT,
			FOREIGN KEY(group_id) REFERENCES travel_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS transports (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"booking_id" INTEGER,
			"type" TEXT,
			"provider" TEXT,
			"departure_location" TEXT,
			"arrival_location" TEXT,
			"departure_time" DATETIME,
			"arrival_time" DATETIME,
			"reference_number" TEXT,
			FOREIGN KEY(booking_id) REFERENCES bookings(id)
		);`,
		`CREATE TABLE IF NOT EXISTS car_rentals (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"booking_id" INTEGER,
			"company" TEXT,
			"pickup_location" TEXT,
			"dropoff_location" TEXT,
			"pickup_time" DATETIME,
			"dropoff_time" DATETIME,
			"car_type" TEXT,
			"price_total" TEXT,
			FOREIGN KEY(booking_id) REFERENCES bookings(id)
		);`,
		`CREATE TABLE IF NOT EXISTS trips (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"group_id" INTEGER NOT NULL,
			"name" TEXT NOT NULL,
			"destination" TEXT,
			"start_date" DATETIME,
			"end_date" DATETIME,
			"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(group_id) REFERENCES travel_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS trip_days (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"trip_id" INTEGER NOT NULL,
			"day_number" INTEGER NOT NULL,
			"date" DATE NOT NULL,
			"location" TEXT,
			FOREIGN KEY(trip_id) REFERENCES trips(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS places (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"trip_day_id" INTEGER NOT NULL,
			"name" TEXT NOT NULL,
			"address" TEXT,
			"place_type" TEXT,
			"latitude" REAL,
			"longitude" REAL,
			"notes" TEXT,
			"visit_time" DATETIME,
			"order_index" INTEGER DEFAULT 0,
			FOREIGN KEY(trip_day_id) REFERENCES trip_days(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS trip_travelers (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"trip_id" INTEGER NOT NULL,
			"user_id" INTEGER NOT NULL,
			"color" TEXT NOT NULL,
			FOREIGN KEY(trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(trip_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS trip_transports (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"trip_id" INTEGER NOT NULL,
			"from_location" TEXT NOT NULL,
			"to_location" TEXT NOT NULL,
			"transport_mode" TEXT NOT NULL,
			"departure_time" DATETIME,
			"arrival_time" DATETIME,
			"details_json" TEXT,
			FOREIGN KEY(trip_id) REFERENCES trips(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS trip_transport_travelers (
			"trip_transport_id" INTEGER NOT NULL,
			"user_id" INTEGER NOT NULL,
			PRIMARY KEY (trip_transport_id, user_id),
			FOREIGN KEY(trip_transport_id) REFERENCES trip_transports(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	log.Println("Migrations completed.")
	return nil

}
