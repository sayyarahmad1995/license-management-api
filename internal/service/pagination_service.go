package service

import (
	"math"
	"strings"

	"license-management-api/internal/dto"
)

// PaginationService provides pagination utilities
type PaginationService interface {
	CalculateOffset(page, pageSize int64) int64
	CalculateTotalPages(totalCount, pageSize int64) int64
	ApplyPagination(page, pageSize int64) (int64, int64)
	ValidatePaginationParams(page, pageSize int64) (int64, int64)
	BuildPaginatedResponse(data interface{}, page, pageSize, totalCount int64) *dto.PaginatedResponse
	ParseSortParam(sortBy string) (string, string)
}

type paginationService struct{}

// NewPaginationService creates a new pagination service
func NewPaginationService() PaginationService {
	return &paginationService{}
}

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// ValidatePaginationParams validates and normalizes pagination parameters
func (ps *paginationService) ValidatePaginationParams(page, pageSize int64) (int64, int64) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// CalculateOffset calculates the database offset from page and pageSize
func (ps *paginationService) CalculateOffset(page, pageSize int64) int64 {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

// CalculateTotalPages calculates the total number of pages
func (ps *paginationService) CalculateTotalPages(totalCount, pageSize int64) int64 {
	if pageSize <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(totalCount) / float64(pageSize)))
}

// ApplyPagination applies pagination and returns offset and limit
func (ps *paginationService) ApplyPagination(page, pageSize int64) (int64, int64) {
	page, pageSize = ps.ValidatePaginationParams(page, pageSize)
	offset := ps.CalculateOffset(page, pageSize)
	return offset, pageSize
}

// BuildPaginatedResponse builds a paginated response
func (ps *paginationService) BuildPaginatedResponse(data interface{}, page, pageSize, totalCount int64) *dto.PaginatedResponse {
	totalPages := ps.CalculateTotalPages(totalCount, pageSize)
	return &dto.PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}

// ParseSortParam parses sort parameter (e.g., "name:asc" or "id:desc")
func (ps *paginationService) ParseSortParam(sortBy string) (string, string) {
	defaultColumn := "id"
	defaultOrder := "asc"

	if sortBy == "" {
		return defaultColumn, defaultOrder
	}

	parts := strings.Split(sortBy, ":")
	if len(parts) == 0 {
		return defaultColumn, defaultOrder
	}

	column := strings.TrimSpace(parts[0])
	if column == "" {
		column = defaultColumn
	}

	order := defaultOrder
	if len(parts) > 1 {
		o := strings.ToLower(strings.TrimSpace(parts[1]))
		if o == "desc" || o == "descending" {
			order = "desc"
		}
	}

	// Validate column name to prevent SQL injection (basic validation)
	validColumns := map[string]bool{
		"id":           true,
		"name":         true,
		"email":        true,
		"created_at":   true,
		"updated_at":   true,
		"status":       true,
		"role":         true,
		"license_key":  true,
		"expires_at":   true,
		"activated_at": true,
		"last_seen_at": true,
	}

	if !validColumns[column] {
		column = defaultColumn
	}

	return column, order
}

// FilterBySearch applies a text search filter to a list
func (ps *paginationService) FilterBySearch(data interface{}, searchTerm string) interface{} {
	// This would be implemented with type switching based on the data type
	// For now, it's a placeholder for the pagination service interface
	return data
}
