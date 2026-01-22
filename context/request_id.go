// Package context provides context utilities for request tracking
package context

import (
	stdctx "context"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey int

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey contextKey = iota
)

// NewRequestID generates a new unique request ID
func NewRequestID() string {
	return uuid.New().String()
}

// WithRequestID adds a request ID to the context
func WithRequestID(parent stdctx.Context, requestID string) stdctx.Context {
	return stdctx.WithValue(parent, RequestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context
func RequestIDFromContext(ctx stdctx.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
