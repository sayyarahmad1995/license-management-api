package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"license-management-api/internal/config"
)

// Context key constants. These are also defined in middleware.
const (
	CorrelationIDContextKey = "correlation_id"
	TraceIDContextKey       = "trace_id"
)

// Logger wraps slog with additional functionality
type Logger struct {
	*slog.Logger
	handler slog.Handler
}

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// LogConfig holds logger configuration
type LogConfig struct {
	Level        LogLevel
	IsProduction bool
	OutputFile   string // Optional file output
	MaxAge       int    // Log rotation max age in days
}

// New creates a new structured logger
func New(config LogConfig) *Logger {
	level := slog.LevelDebug
	switch config.Level {
	case DEBUG:
		level = slog.LevelDebug
	case INFO:
		level = slog.LevelInfo
	case WARN:
		level = slog.LevelWarn
	case ERROR:
		level = slog.LevelError
	}

	var handler slog.Handler

	if config.IsProduction {
		// JSON format for production
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		// Pretty text format for development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	// Add file output if configured
	if config.OutputFile != "" {
		file, err := openLogFile(config.OutputFile)
		if err != nil {
			fmt.Printf("Warning: Could not open log file %s: %v\n", config.OutputFile, err)
		} else {
			// Chain file handler with console handler
			multiWriter := io.MultiWriter(os.Stdout, file)
			if config.IsProduction {
				handler = slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
					Level: level,
				})
			} else {
				handler = slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
					Level: level,
				})
			}
		}
	}

	return &Logger{
		Logger:  slog.New(handler),
		handler: handler,
	}
}

// FromConfig creates a logger from AppConfig
func FromConfig(cfg *config.AppConfig) *Logger {
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	logLevel := INFO
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logLevel = DEBUG
	}

	return New(LogConfig{
		Level:        logLevel,
		IsProduction: isProduction,
		OutputFile:   "logs/app.log",
		MaxAge:       30,
	})
}

// WithContext returns a logger with context values attached
func (l *Logger) WithContext(ctx context.Context) *Logger {
	var args []interface{}

	// Extract correlation ID (from the new CorrelationIDMiddleware)
	if correlationID, ok := ctx.Value(CorrelationIDContextKey).(string); ok && correlationID != "" {
		args = append(args, "correlation_id", correlationID)
	}

	// Extract legacy trace ID for backward compatibility
	if traceID, ok := ctx.Value(TraceIDContextKey).(string); ok && traceID != "" {
		args = append(args, "trace_id", traceID)
	}

	if len(args) == 0 {
		return l
	}

	return &Logger{
		Logger:  l.Logger.With(args...),
		handler: l.handler,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, parseArgs(args)...)
}

// DebugCtx logs a debug message with context
func (l *Logger) DebugCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Logger.Debug(msg, parseArgs(args)...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, parseArgs(args)...)
}

// InfoCtx logs an info message with context
func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Logger.Info(msg, parseArgs(args)...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, parseArgs(args)...)
}

// WarnCtx logs a warning message with context
func (l *Logger) WarnCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Logger.Warn(msg, parseArgs(args)...)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, args ...interface{}) {
	allArgs := append([]interface{}{"error", err}, args...)
	l.Logger.Error(msg, parseArgs(allArgs)...)
}

// ErrorCtx logs an error message with context
func (l *Logger) ErrorCtx(ctx context.Context, msg string, err error, args ...interface{}) {
	l.WithContext(ctx).Error(msg, err, args...)
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0)
	for k, v := range fields {
		args = append(args, k, v)
	}
	newLogger := l.Logger.With(args...)
	return &Logger{
		Logger:  newLogger,
		handler: l.handler,
	}
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(method, path string, statusCode int, duration time.Duration, userID *int64, fields map[string]interface{}) {
	args := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status":      statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	if userID != nil {
		args["user_id"] = *userID
	}

	for k, v := range fields {
		args[k] = v
	}

	l.Logger.Info("HTTP Request", "method", method, "path", path, "status", statusCode, "duration_ms", duration.Milliseconds())
}

// LogError logs an error with full context
func (l *Logger) LogError(msg string, err error, fields map[string]interface{}) {
	args := []interface{}{"error", err}
	for k, v := range fields {
		args = append(args, k, v)
	}
	l.Logger.Error(msg, args...)
}

// LogDatabase logs database operations
func (l *Logger) LogDatabase(operation string, table string, duration time.Duration, rowsAffected int64, err error) {
	if err != nil {
		l.Logger.Error("Database Operation Failed",
			"operation", operation,
			"table", table,
			"duration_ms", duration.Milliseconds(),
			"error", err)
	} else {
		l.Logger.Info("Database Operation",
			"operation", operation,
			"table", table,
			"duration_ms", duration.Milliseconds(),
			"rows_affected", rowsAffected)
	}
}

// LogCacheOperation logs cache operations
func (l *Logger) LogCacheOperation(operation string, key string, hit bool, duration time.Duration, err error) {
	status := "MISS"
	if hit {
		status = "HIT"
	}

	if err != nil {
		l.Logger.Error("Cache Operation Failed",
			"operation", operation,
			"key", key,
			"cache_status", status,
			"duration_ms", duration.Milliseconds(),
			"error", err)
	} else {
		l.Logger.Info("Cache Operation",
			"operation", operation,
			"key", key,
			"cache_status", status,
			"duration_ms", duration.Milliseconds())
	}
}

// LogAuditEvent logs security and audit events
func (l *Logger) LogAuditEvent(eventType string, userID int64, action string, details map[string]interface{}) {
	args := []interface{}{
		"event_type", eventType,
		"user_id", userID,
		"action", action,
		"timestamp", time.Now().UTC(),
	}

	for k, v := range details {
		args = append(args, k, v)
	}

	l.Logger.Info("Audit Event", args...)
}

// LogPerformanceMetric logs performance metrics
func (l *Logger) LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	args := []interface{}{
		"metric", metric,
		"value", value,
		"unit", unit,
	}

	for k, v := range tags {
		args = append(args, k, v)
	}

	l.Logger.Info("Performance Metric", args...)
}

// parseArgs converts []interface{} to []any for slog
func parseArgs(args []interface{}) []any {
	result := make([]any, len(args))
	copy(result, args)
	return result
}

// openLogFile opens or creates the log file
func openLogFile(filename string) (*os.File, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// Global logger instance
var globalLogger *Logger

// Initialize sets up the global logger
func Initialize(cfg *config.AppConfig) {
	globalLogger = FromConfig(cfg)
}

// Get returns the global logger
func Get() *Logger {
	if globalLogger == nil {
		globalLogger = New(LogConfig{
			Level:        INFO,
			IsProduction: false,
			OutputFile:   "",
		})
	}
	return globalLogger
}

// Close gracefully closes the logger
func (l *Logger) Close() error {
	// slog doesn't require explicit close, but reserved for future use
	return nil
}
