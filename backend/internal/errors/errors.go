package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/getsentry/sentry-go"
)

// ErrorCode represents a unique error code
type ErrorCode string

// Error codes
const (
	CodeUnknown         ErrorCode = "UNKNOWN"
	CodeValidation      ErrorCode = "VALIDATION_ERROR"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	CodeForbidden       ErrorCode = "FORBIDDEN"
	CodeConflict        ErrorCode = "CONFLICT"
	CodeRateLimit       ErrorCode = "RATE_LIMIT"
	CodeInternal        ErrorCode = "INTERNAL_ERROR"
	CodeBadRequest      ErrorCode = "BAD_REQUEST"
	CodeTimeout         ErrorCode = "TIMEOUT"
	CodeServiceUnavail  ErrorCode = "SERVICE_UNAVAILABLE"
	CodeDatabaseError   ErrorCode = "DATABASE_ERROR"
	CodeExternalService ErrorCode = "EXTERNAL_SERVICE_ERROR"
	CodePaymentError    ErrorCode = "PAYMENT_ERROR"
	CodeEmailError      ErrorCode = "EMAIL_ERROR"
	CodeFileUploadError ErrorCode = "FILE_UPLOAD_ERROR"
)

// AppError represents an application-specific error
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Cause      error                  `json:"-"`
	Stack      string                 `json:"-"`
	Context    map[string]interface{} `json:"-"`
	Retryable  bool                   `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsRetryable returns whether the error is retryable
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// ToSentryLevel converts the error to a Sentry level
func (e *AppError) ToSentryLevel() sentry.Level {
	switch {
	case e.HTTPStatus >= 500:
		return sentry.LevelError
	case e.HTTPStatus >= 400:
		return sentry.LevelWarning
	default:
		return sentry.LevelInfo
	}
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
		Stack:      captureStack(),
	}
}

// Wrap wraps an existing error with an AppError
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
		Cause:      err,
		Stack:      captureStack(),
	}
}

// Error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, details ...string) *AppError {
	err := New(CodeValidation, message)
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message ...string) *AppError {
	msg := "Authentication required"
	if len(message) > 0 {
		msg = message[0]
	}
	return New(CodeUnauthorized, msg)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message ...string) *AppError {
	msg := "Access denied"
	if len(message) > 0 {
		msg = message[0]
	}
	return New(CodeForbidden, msg)
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *AppError {
	return New(CodeConflict, message)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError() *AppError {
	err := New(CodeRateLimit, "Rate limit exceeded")
	err.Retryable = true
	return err
}

// NewInternalError creates an internal error
func NewInternalError(message string) *AppError {
	return New(CodeInternal, message)
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(message string) *AppError {
	return New(CodeBadRequest, message)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(message string) *AppError {
	err := New(CodeTimeout, message)
	err.Retryable = true
	return err
}

// NewDatabaseError creates a database error
func NewDatabaseError(cause error) *AppError {
	return Wrap(cause, CodeDatabaseError, "Database operation failed")
}

// NewExternalServiceError creates an external service error
func NewExternalServiceError(service string, cause error) *AppError {
	err := Wrap(cause, CodeExternalService, fmt.Sprintf("External service error: %s", service))
	err.Retryable = true
	return err
}

// Error checking functions

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts an AppError from an error
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == CodeNotFound
}

// IsValidation checks if an error is a validation error
func IsValidation(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == CodeValidation
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == CodeUnauthorized
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Code == CodeForbidden
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Retryable
}

// Helper functions

func codeToHTTPStatus(code ErrorCode) int {
	switch code {
	case CodeValidation, CodeBadRequest:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimit:
		return http.StatusTooManyRequests
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func captureStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// ErrorResponse is the JSON response format for errors
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ToResponse converts an AppError to an ErrorResponse
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error:   e.Code.String(),
		Code:    string(e.Code),
		Message: e.Message,
		Details: e.Details,
	}
}

// String returns the string representation of an ErrorCode
func (c ErrorCode) String() string {
	return string(c)
}
