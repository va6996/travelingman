.PHONY: all proto build run server dev dev-frontend dev-backend clean setup-dev help

# Variables
PROTO_DIR = protos
PB_DIR = pb
BINARY_NAME = server

# Default target - production build
all: build

# Help target
help:
	@echo "Available targets:"
	@echo "  all               - Build production binary (default)"
	@echo "  proto             - Generate Go and TypeScript protobufs"
	@echo "  proto-go          - Generate Go protobufs only"
	@echo "  proto-web         - Generate TypeScript protobufs only"
	@echo "  build             - Build production binary with embedded UI"
	@echo "  run               - Build and run production server"
	@echo "  server            - Build and start production server"
	@echo "  dev               - Run both frontend and backend in dev mode"
	@echo "  dev-frontend      - Run frontend dev server only (Vite)"
	@echo "  dev-backend       - Run backend dev server only (Air hot reload)"
	@echo "  setup-dev         - Install development dependencies"
	@echo "  test              - Run all tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  clean             - Clean up generated files and binaries"
	@echo "  help              - Show this help message"
	@echo ""
	@echo "Development workflow:"
	@echo "  make dev          - Start both frontend (:5173) and backend (:8000)"
	@echo "  npm run dev       - Start frontend dev server only"
	@echo "  make dev-backend  - Start backend dev server only"
	@echo ""
	@echo "Production workflow:"
	@echo "  make build        - Build production binary with embedded UI"
	@echo "  ./server          - Run production server"

# Generate Go code from Protobuf files
proto-go:
	@echo "Generating Go protobufs..."
	@mkdir -p $(PB_DIR)
	@rm -f $(PB_DIR)/*.pb.go $(PB_DIR)/*.connect.go
	@rm -rf $(PB_DIR)/pbconnect
	export PATH="$(PATH):$$(go env GOPATH)/bin" && \
	protoc -I. --go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--connect-go_out=$(PB_DIR) --connect-go_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto && \
	mv $(PB_DIR)/protos/*.pb.go $(PB_DIR)/ && \
	(mv $(PB_DIR)/protos/pbconnect $(PB_DIR)/ || true) && \
	rmdir $(PB_DIR)/protos

PB_WEB_DIR = ui/src/gen

# Generate Web protobufs (Connect-ES)
proto-web:
	@echo "Generating Web protobufs..."
	@mkdir -p $(PB_WEB_DIR)
	@rm -rf $(PB_WEB_DIR)/*
	export PATH="$(PATH):$(PWD)/ui/node_modules/.bin" && \
	protoc -I. --es_out=$(PB_WEB_DIR) --es_opt=target=ts \
		--connect-es_out=$(PB_WEB_DIR) --connect-es_opt=target=ts \
		$(PROTO_DIR)/*.proto

# Generate all protos
proto: proto-go proto-web

# Build the application
build: proto
	@echo "Building application..."
	@echo "Building UI..."
	cd ui && npm run build
	@echo "Building Go binary with embedded UI..."
	go mod tidy
	go build -o $(BINARY_NAME) .

# Run the application
run: build
	@echo "Running application..."
	./$(BINARY_NAME)

# Run server (dedicated server command)
server: build
	@echo "Starting server..."
	./$(BINARY_NAME)

# Setup development dependencies
setup-dev:
	@echo "Installing development dependencies..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin"; \
	if ! command -v air >/dev/null 2>&1; then \
		echo "Installing air for Go hot reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "Installing UI dependencies..."
	@cd ui && npm install
	@echo "Configuring Genkit..."
	@if command -v genkit >/dev/null 2>&1; then \
		genkit config set updateNotificationsOptOut true || true; \
	else \
		echo "Genkit CLI not found - skipping notification config"; \
	fi

# Frontend development only (runs Vite dev server on :5173)
dev-frontend:
	@echo "Starting frontend development server..."
	@cd ui && npm run dev

# Backend development only (runs Air hot reload on :8000)
dev-backend: setup-dev
	@echo "Starting backend development server with hot reload..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && air

# Full development mode - runs both frontend and backend
dev: setup-dev
	@echo "Starting full development environment..."
	@echo "  Frontend: http://localhost:5173"
	@echo "  Backend:  http://localhost:8000"
	@echo "  Press Ctrl+C to stop both servers"
	@echo ""
	@# Use a subshell to handle both processes together
	@(export PATH="$$PATH:$$(go env GOPATH)/bin"; \
	air & \
	AIR_PID=$$!; \
	cd ui && npm run dev & \
	FRONTEND_PID=$$!; \
	trap "kill $$AIR_PID $$FRONTEND_PID 2>/dev/null; exit 0" INT TERM; \
	wait)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./plugins/integration/...

# Clean up generated files and binary
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(PB_DIR)/*.pb.go

