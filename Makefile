.PHONY: all proto build run server dev clean setup-dev help

# Variables
PROTO_DIR = protos
PB_DIR = pb
BINARY_NAME = server

# Default target
all: build

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build the application (default)"
	@echo "  proto        - Generate Go and TypeScript protobufs"
	@echo "  proto-go     - Generate Go protobufs only"
	@echo "  build        - Build the application binary"
	@echo "  run          - Build and run the server"
	@echo "  server       - Build and start the server (dedicated server command)"
	@echo "  dev          - Setup dev dependencies and run with hot reload"
	@echo "  setup-dev    - Install development dependencies (air, configure Genkit)"
	@echo "  test         - Run all tests"
	@echo "  test-integration - Run integration tests"
	@echo "  clean        - Clean up generated files and binaries"
	@echo "  help         - Show this help message"

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
		echo "Installing air for hot reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "Configuring Genkit..."
	@if command -v genkit >/dev/null 2>&1; then \
		genkit config set updateNotificationsOptOut true || true; \
	else \
		echo "Genkit CLI not found - skipping notification config"; \
	fi

# Run with hot reload
dev: setup-dev
	@echo "Running with hot reload..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && air

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

