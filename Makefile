.PHONY: all proto build run clean

# Variables
PROTO_DIR = models
PB_DIR = pb
PB_TS_DIR = pb_ts
BINARY_NAME = server

# Default target
all: build

# Generate Go code from Protobuf files
proto-go:
	@echo "Generating Go protobufs..."
	@mkdir -p $(PB_DIR)
	@rm -f $(PB_DIR)/*.pb.go
	export PATH="$(PATH):$$(go env GOPATH)/bin" && \
	protoc -I$(PROTO_DIR) --go_out=$(PB_DIR) --go_opt=paths=source_relative $(PROTO_DIR)/*.proto

# Generate TS code from Protobuf files
proto-ts:
	@echo "Generating TS protobufs..."
	@mkdir -p $(PB_TS_DIR)
	@rm -f $(PB_TS_DIR)/*.ts
	protoc -I$(PROTO_DIR) --plugin=./node_modules/.bin/protoc-gen-ts_proto --ts_proto_out=$(PB_TS_DIR) --ts_proto_opt=esModuleInterop=true --ts_proto_opt=outputServices=grpc-js $(PROTO_DIR)/*.proto

# Generate all protos
proto: proto-go proto-ts

# Build the application
build: proto
	@echo "Building application..."
	go mod tidy
	go build -o $(BINARY_NAME) .

# Run the application
run: build
	@echo "Running application..."
	./$(BINARY_NAME)

# Run with hot reload
dev:
	@echo "Running with hot reload..."
	air

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
	rm -rf $(PB_TS_DIR)
