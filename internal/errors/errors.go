package errors

import "fmt"

type ErrorType string

const (
	ValidationError  ErrorType = "VALIDATION_ERROR"
	NotFoundError    ErrorType = "NOT_FOUND_ERROR"
	UnauthorizedError ErrorType = "UNAUTHORIZED_ERROR"
	ForbiddenError   ErrorType = "FORBIDDEN_ERROR"
	ConflictError    ErrorType = "CONFLICT_ERROR"
	InternalError    ErrorType = "INTERNAL_ERROR"
	BadRequestError  ErrorType = "BAD_REQUEST_ERROR"
	RateLimitError   ErrorType = "RATE_LIMIT_ERROR"
)

type ApiError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Status  int       `json:"status"`
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func NewApiError(errorType ErrorType, message string, status int) *ApiError {
	return &ApiError{
		Type:    errorType,
		Message: message,
		Status:  status,
	}
}

func NewValidationError(message string) *ApiError {
	return NewApiError(ValidationError, message, 400)
}

func NewNotFoundError(message string) *ApiError {
	return NewApiError(NotFoundError, message, 404)
}

func NewUnauthorizedError(message string) *ApiError {
	return NewApiError(UnauthorizedError, message, 401)
}

func NewForbiddenError(message string) *ApiError {
	return NewApiError(ForbiddenError, message, 403)
}

func NewConflictError(message string) *ApiError {
	return NewApiError(ConflictError, message, 409)
}

func NewInternalError(message string) *ApiError {
	return NewApiError(InternalError, message, 500)
}

func NewBadRequestError(message string) *ApiError {
	return NewApiError(BadRequestError, message, 400)
}
