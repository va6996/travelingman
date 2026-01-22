// Package log provides a simple wrapper around logrus
// with a familiar API (Printf, Infof, Errorf, etc.)
package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance
var Logger = logrus.New()

// CustomFormatter implements logrus.Formatter for the desired output format
type CustomFormatter struct {
	TimestampFormat string
}

// Format formats a log entry as [<time>] [LEVEL] [file:line] <message>
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Timestamp
	timestamp := entry.Time.Format(f.TimestampFormat)
	fmt.Fprintf(b, "[%s] ", timestamp)

	// Level
	level := strings.ToUpper(entry.Level.String())
	fmt.Fprintf(b, "[%s] ", level)

	// File and line
	// We walk the stack to find the caller, skipping logrus internals and our log wrapper
	pcs := make([]uintptr, 32)
	// Skip runtime.Callers and Format
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	file := ""
	line := 0

	for {
		frame, more := frames.Next()

		// Skip logrus internals
		if strings.Contains(frame.File, "github.com/sirupsen/logrus") {
			if !more {
				break
			}
			continue
		}

		// Skip this log package
		if strings.HasSuffix(frame.File, "log/log.go") {
			if !more {
				break
			}
			continue
		}

		// Skip runtime functions
		if strings.Contains(frame.File, "runtime/") {
			if !more {
				break
			}
			continue
		}

		file = frame.File
		line = frame.Line
		break
	}

	if file != "" {
		// Extract just filename
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]
		fmt.Fprintf(b, "[%s:%d] ", filename, line)
	}

	// Message
	b.WriteString(entry.Message)

	// Add fields if any (handle request_id specially)
	if len(entry.Data) > 0 {
		// Check if request_id is in the data and not empty
		if requestID, ok := entry.Data["request_id"].(string); ok && requestID != "" {
			fmt.Fprintf(b, " [req:%s]", requestID)
		}

		// Add any other fields (excluding request_id which we already handled)
		for key, value := range entry.Data {
			if key != "request_id" {
				fmt.Fprintf(b, " %s=%v", key, value)
			}
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

// requestIDFromContext safely extracts request ID from context
func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// Try to get from our custom context package
	if requestID, ok := ctx.Value(contextKey(0)).(string); ok && requestID != "" {
		return requestID
	}
	return ""
}

// contextKey is a custom type for context keys
type contextKey int

const (
	// RequestIDKey matches the key used in context/request_id.go
	RequestIDKey contextKey = iota
)

// Helper to add request ID as a field to the log entry
func withRequestIDField(ctx context.Context) *logrus.Entry {
	requestID := requestIDFromContext(ctx)
	if requestID != "" {
		return Logger.WithField("request_id", requestID)
	}
	return Logger.WithField("request_id", "")
}

// Infof logs formatted message at info level
func Infof(ctx context.Context, format string, args ...interface{}) {
	withRequestIDField(ctx).Infof(format, args...)
}

// Info logs a message at info level
func Info(ctx context.Context, args ...interface{}) {
	withRequestIDField(ctx).Info(args...)
}

// Debugf logs formatted message at debug level
func Debugf(ctx context.Context, format string, args ...interface{}) {
	withRequestIDField(ctx).Debugf(format, args...)
}

// Debug logs a message at debug level
func Debug(ctx context.Context, args ...interface{}) {
	withRequestIDField(ctx).Debug(args...)
}

// Warnf logs formatted message at warning level
func Warnf(ctx context.Context, format string, args ...interface{}) {
	withRequestIDField(ctx).Warnf(format, args...)
}

// Warn logs a message at warning level
func Warn(ctx context.Context, args ...interface{}) {
	withRequestIDField(ctx).Warn(args...)
}

// Errorf logs formatted message at error level
func Errorf(ctx context.Context, format string, args ...interface{}) {
	withRequestIDField(ctx).Errorf(format, args...)
}

// Error logs a message at error level
func Error(ctx context.Context, args ...interface{}) {
	withRequestIDField(ctx).Error(args...)
}

// Fatalf logs formatted message at fatal level and exits
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	withRequestIDField(ctx).Fatalf(format, args...)
}

// Fatal logs a message at fatal level and exits
func Fatal(ctx context.Context, args ...interface{}) {
	withRequestIDField(ctx).Fatal(args...)
}

// SetLevel sets the global log level
func SetLevel(level logrus.Level) {
	Logger.SetLevel(level)
}

// SetFormatter sets the global log formatter
func SetFormatter(formatter logrus.Formatter) {
	Logger.SetFormatter(formatter)
}

// SetOutput sets the global log output
func SetOutput(out io.Writer) {
	Logger.SetOutput(out)
}

// Init initializes the logger with default settings
func Init() {
	Logger.SetFormatter(&CustomFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	// Caller reporting handled manually in Format
	Logger.SetLevel(logrus.InfoLevel)
}

// WithFields creates a logger with predefined fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Logger.WithFields(fields)
}

// WithField creates a logger with predefined field
func WithField(key string, value interface{}) *logrus.Entry {
	return Logger.WithField(key, value)
}

// WithRequestID creates a logger entry with a request ID field
func WithRequestID(requestID string) *logrus.Entry {
	return Logger.WithField("request_id", requestID)
}
