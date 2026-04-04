// Package logger provides a simple wrapper around slog for structured logging with
// support for context, log levels, output formats, and output destinations.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// defaultLogger is the global logger instance used by package-level logging functions.
var defaultLogger = slog.Default()

/*
Init configures the global logger with the specified output path, log level, and format.

Params:

- outputPath: "stdout", "stderr", or a file path to write logs to.
- logLevel: "debug", "info", "warn", or "error" (case-insensitive).
- logFormat: "json" for JSON output, anything else for plain text.

Returns:

- none. This function initializes the global logger and does not return an error. If the output path is invalid, it falls back to stdout and logs a warning to stderr.
*/
func Init(outputPath, logLevel, logFormat string) {
	defaultLogger = new(outputPath, logLevel, logFormat)
}

// Debug logs a message at debug level using the global logger.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// DebugContext logs a message at debug level with context using the global logger.
func DebugContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

// Info logs a message at info level using the global logger.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// InfoContext logs a message at info level with context using the global logger.
func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// Warn logs a message at warning level using the global logger.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// WarnContext logs a message at warning level with context using the global logger.
func WarnContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

// Error logs a message at error level using the global logger.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// ErrorContext logs a message at error level with context using the global logger.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

// logFileMode is the file mode used when creating log files.
const logFileMode = os.FileMode(0644)

// new is an internal helper to construct a slog.Logger with the given configuration.
func new(outputPath, logLevel, logFormat string) *slog.Logger {
	output := openLogOutput(outputPath)
	level := resolveLevel(logLevel)

	var handler slog.Handler
	switch strings.ToLower(logFormat) {
	case "json":
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})
	}

	logger := slog.New(handler)
	return logger
}

// openLogOutput opens the log output destination based on the given path.
func openLogOutput(logFilePath string) *os.File {
	switch logFilePath {
	case "", "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFileMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "log: failed to open %q: %v; falling back to stdout\n", logFilePath, err)
			return os.Stdout
		}
		return f
	}
}

// resolveLevel converts a string log level to slog.Level
func resolveLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
