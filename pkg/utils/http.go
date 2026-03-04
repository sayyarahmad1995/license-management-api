package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"license-management-api/internal/errors"
	"github.com/go-chi/chi/v5"
)

// GetIDFromURL extracts an ID parameter from the URL
func GetIDFromURL(r *http.Request, paramName string) (int, *errors.ApiError) {
	idStr := chi.URLParam(r, paramName)
	if idStr == "" {
		return 0, errors.NewBadRequestError(fmt.Sprintf("Missing %s parameter", paramName))
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errors.NewBadRequestError(fmt.Sprintf("Invalid %s parameter", paramName))
	}

	return id, nil
}

// GetPaginationParams extracts pagination parameters from query string
func GetPaginationParams(r *http.Request) (int, int, *errors.ApiError) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page := 1
	pageSize := 10

	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return 0, 0, errors.NewBadRequestError("Invalid page parameter")
		}
		page = p
	}

	if pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 || ps > 100 {
			return 0, 0, errors.NewBadRequestError("Invalid pageSize parameter (1-100)")
		}
		pageSize = ps
	}

	return page, pageSize, nil
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (nginx, reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// WriteErrorResponse writes an error response with JSON encoding
func WriteErrorResponse(w http.ResponseWriter, status int, errType errors.ErrorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errResp := map[string]interface{}{
		"error": map[string]string{
			"type":    string(errType),
			"message": message,
		},
	}
	json.NewEncoder(w).Encode(errResp)
}
