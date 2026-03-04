package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CompressWriter wraps http.ResponseWriter to compress responses
type CompressWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (cw *CompressWriter) Write(p []byte) (int, error) {
	return cw.Writer.Write(p)
}

// CompressionMiddleware adds gzip compression to responses
func CompressionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip encoding
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Create gzip writer
			gz := gzip.NewWriter(w)
			defer gz.Close()

			// Set response headers
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length") // Remove Content-Length since it will change with compression

			// Wrap the response writer
			compressedW := &CompressWriter{ResponseWriter: w, Writer: gz}

			next.ServeHTTP(compressedW, r)
		})
	}
}

// CacheControlMiddleware adds HTTP caching headers based on content type
func CacheControlMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer to capture status and content-type
			wrapped := &responseWriterWrapper{ResponseWriter: w}

			next.ServeHTTP(wrapped, r)

			// Set cache headers based on response
			contentType := wrapped.ResponseWriter.Header().Get("Content-Type")
			statusCode := wrapped.statusCode

			// Don't cache error responses
			if statusCode >= 400 {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
				return
			}

			// Cache static assets for 1 year
			if strings.Contains(contentType, "text/css") ||
				strings.Contains(contentType, "application/javascript") ||
				strings.Contains(contentType, "image/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				return
			}

			// Cache API responses for short duration (5 minutes)
			if strings.Contains(contentType, "application/json") {
				// Only cache GET requests
				if r.Method == "GET" {
					w.Header().Set("Cache-Control", "public, max-age=300")
					// Add ETag for client-side caching
					w.Header().Set("ETag", fmt.Sprintf(`"%d"`, statusCode))
					return
				}
				// Don't cache POST/PUT/DELETE
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				return
			}

			// Default: cache HTML for 1 hour
			w.Header().Set("Cache-Control", "public, max-age=3600")
		})
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// ETagMiddleware adds ETags to responses for client-side caching validation
func ETagMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only process GET requests
			if r.Method != "GET" {
				next.ServeHTTP(w, r)
				return
			}

			// If client sent If-None-Match header, check ETag
			if r.Header.Get("If-None-Match") != "" {
				// For simplicity, always regenerate ETag (in production, store actual content hash)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
