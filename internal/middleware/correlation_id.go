package middleware

import (
	"context"
	"net/http"

	"license-management-api/internal/logger"
	"github.com/google/uuid"
)

// CorrelationIDHeader is the HTTP header used to propagate the correlation ID.
const CorrelationIDHeader = "X-Request-ID"

// CorrelationIDMiddleware reads X-Request-ID from incoming requests (or generates
// a UUID v4 if absent), stores it in the request context, and echoes it back on
// every response via X-Request-ID.
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(CorrelationIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), logger.CorrelationIDContextKey, id)
		w.Header().Set(CorrelationIDHeader, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCorrelationID retrieves the correlation ID from the context.
// Returns an empty string if not present.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(logger.CorrelationIDContextKey).(string); ok {
		return id
	}
	return ""
}
