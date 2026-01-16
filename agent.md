# Travelingman Agent Guide

## Overview
This document contains instructions for developing, building, testing, and running the Travelingman Go backend.

## Prerequisites
- Go 1.22+
- Make
- SQLite3
- `protoc` (Protocol Buffers compiler) with:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc` (if RPCs used later)
- `swag` (for API docs): `go install github.com/swaggo/swag/cmd/swag@latest`

## Project Structure
- `apis/v1`: API Handlers grouped by feature (auth, groups, search, bookings, itinerary).
- `models`: Protobuf definitions.
- `pb`: Generated Go code from Protobufs.
- `providers`: External service integrations (e.g., Amadeus).
- `migrations.go`: Database schema definitions.

## Key Commands

### Build
```bash
make build
# or
go build -o server .
```

### Run
```bash
./server
```
**Environment Variables Required for External APIs:**
- `AMADEUS_CLIENT_ID`
- `AMADEUS_CLIENT_SECRET`
- `GOOGLE_MAPS_API_KEY` (Backend - restrict by IP)

**Frontend Environment Variables:**
- `VITE_GOOGLE_MAPS_API_KEY` (Frontend - restrict by HTTP referrer)

### Hot Reload (Development)
```bash
make test
# or
go test ./...
```

### Generate Protobufs
```bash
make proto
```

### Generate API Documentation (Swagger)
1. Install Swag:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```
2. Generate Docs:
   ```bash
   swag init
   ```
3. Access Docs:
   Open `http://localhost:8081/swagger/index.html` after running the server.

## Database
The project uses SQLite (`travelingman.db`). The database is automatically initialized and migrated on server start if it doesn't exist.

## Adding New APIs
1. Define Request/Response in `models/service/`.
2. Run `make proto`.
3. Create Handler in `apis/v1/<feature>/`.
4. Register Handler in `main.go`.
5. Add Swagger annotations.
6. Run `swag init`.
7. Add Tests.
