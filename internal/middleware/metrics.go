package middleware

import (
	"net/http"
	"time"

	"license-management-api/internal/service"
)

// MetricsMiddleware creates middleware that records HTTP metrics
func MetricsMiddleware(metricsService *service.MetricsService) func(http.Handler) http.Handler {
	if metricsService == nil {
		// Return a no-op middleware if metrics service is not available
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code and response size
			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Increment active connections
			metricsService.IncrementActiveConnections()
			defer metricsService.DecrementActiveConnections()

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			path := r.URL.Path
			method := r.Method
			statusCode := wrapped.statusCode
			responseBytes := wrapped.written

			metricsService.RecordHTTPRequest(method, path, statusCode, duration, responseBytes)
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and response size
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

// WriteHeader captures the status code
func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written
func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Flush flushes the response writer if it implements http.Flusher
func (rw *metricsResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
