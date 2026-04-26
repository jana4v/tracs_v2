package logging

import (
	"log/slog"
	"os"
)

// NewLogger creates a structured JSON logger with the given service name.
// All services use slog for structured logging per SRS Section 15.
func NewLogger(serviceName string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler).With("service", serviceName)
}

// ParseLevel converts a string log level to slog.Level.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
