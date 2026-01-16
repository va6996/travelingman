package testutils

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	queries := []string{
		`CREATE TABLE users (
			"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			"email" TEXT UNIQUE,
			"password_hash" TEXT,
			"full_name" TEXT,
			"created_at" DATETIME
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
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// CreateTestUser helps creating a user for tests
func CreateTestUser(db *sql.DB, email, name string) (int64, error) {
	res, err := db.Exec("INSERT INTO users (email, password_hash, full_name, created_at) VALUES (?, ?, ?, ?)",
		email, "hash", name, time.Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
