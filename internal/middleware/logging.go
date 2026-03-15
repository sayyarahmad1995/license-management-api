package middleware

import (
	"context"
	"net/http"
	"time"

	"license-management-api/internal/logger"
)

// LoggingMiddleware logs HTTP requests with structured logging
func LoggingMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Correlation ID is already in context from CorrelationIDMiddleware
			// Logger will extract it automatically in WithContext()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Record start time
			startTime := time.Now()

			// Call next handler with current context (which includes correlation ID)
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(startTime)

			// Extract user ID from context if available
			var userID *int64
			if id, ok := r.Context().Value("user_id").(*int64); ok {
				userID = id
			}

			// Skip logging health checks to avoid spam
			if r.URL.Path == "/health" && wrapped.statusCode == http.StatusOK {
				return
			}

			// Log request with structured fields
			fields := map[string]interface{}{
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.Header.Get("User-Agent"),
			}

			log.LogRequest(
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				userID,
				fields,
			)

			// Log slow requests (> 1 second)
			if duration > time.Second {
				log.WarnCtx(r.Context(), "Slow Request Detected",
					"duration_ms", duration.Milliseconds(),
					"method", r.Method,
					"path", r.URL.Path,
				)
			}

			// Log error responses
			if wrapped.statusCode >= 400 {
				log.WarnCtx(r.Context(), "Error Response",
					"status", wrapped.statusCode,
					"method", r.Method,
					"path", r.URL.Path,
				)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// generateTraceID generates a unique trace ID for request tracking
func generateTraceID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string for trace ID
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// ErrorLoggingMiddleware logs errors with context
func ErrorLoggingMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add logger to context
			ctx := context.WithValue(r.Context(), "logger", log)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLoggerFromContext retrieves the logger from the request context
func GetLoggerFromContext(r *http.Request) *logger.Logger {
	if log, ok := r.Context().Value("logger").(*logger.Logger); ok {
		return log
	}
	return logger.Get()
}

// GetTraceIDFromContext retrieves the trace ID from the request context
func GetTraceIDFromContext(r *http.Request) string {
	if traceID, ok := r.Context().Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}
