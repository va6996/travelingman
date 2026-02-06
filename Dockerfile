# Multi-stage Dockerfile for Travelingman
# Build for specific platform:
#   AMD64: docker buildx build --platform linux/amd64 -t travelingman:latest .
#   ARM64: docker buildx build --platform linux/arm64 -t travelingman:latest .
#   Auto:  docker buildx build --platform linux/amd64,linux/arm64 -t travelingman:latest .

# Stage 1: Build UI (React + TypeScript)
FROM node:18-alpine AS ui-builder

WORKDIR /ui

# Copy package files
COPY ui/package*.json ./
RUN npm ci

# Copy UI source and build
COPY ui/ ./
RUN npm run build

# Stage 2: Build Go binary with embedded UI
FROM golang:alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make protobuf protobuf-dev nodejs npm build-base

# Install Go protobuf plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy UI package files and install dependencies
COPY ui/package*.json ./ui/
RUN cd ui && npm ci

# Copy source code (excluding local binaries via .dockerignore)
COPY . .

# Copy built UI from previous stage (overrides any local ui/dist)
COPY --from=ui-builder /ui/dist ./ui/dist

# Remove any local binaries to ensure we rebuild for Linux
RUN rm -f server travelingman *.db *.db-shm *.db-wal

# Set build environment for correct target architecture
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Cache busting - change this date to force rebuild
ARG CACHE_BUST=2025-02-06

# Build the binary with embedded UI for the target architecture
# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux make build

# Verify the binary was created
RUN ls -lh /app/server

# Stage 3: Minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder (cache busting ensures fresh copy)
ARG CACHE_BUST=2025-02-06
COPY --from=go-builder /app/server /app/server

# Copy optional config file if it exists
COPY config.yaml .

# Create directory for SQLite database
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the default port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/ || exit 1

# Set default environment variables
ENV PORT=8000
ENV LOG_LEVEL=info

# Run the application
CMD ["./server"]
