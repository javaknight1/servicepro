// Package errors defines typed domain errors: failures the client caused or
// can act on (not found, conflicting state, validation, permission), each
// carrying an HTTP status, a stable machine code and a message that is safe
// to show to a user (T223).
//
// Services return (or wrap with %w) these errors; the API layer
// (internal/api/apierror) turns them into the standard error envelope
// documented in docs/api/errors.md. Any error that is NOT one of these is
// treated as an infrastructure failure and becomes a generic 500, so a
// database or driver message never reaches a client (T116).
//
// Import it as apperrors to avoid shadowing the standard library:
//
//	import apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
package errors

import (
	"errors"
	"net/http"
)

// Standard codes for the generic cases. Domain errors should prefer a more
// specific snake_case code (e.g. "cannot_accept") when a client may branch on it.
const (
	CodeBadRequest       = "bad_request"
	CodeValidation       = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeConflict         = "conflict"
	CodePayloadTooLarge  = "payload_too_large"
	CodeUnprocessable    = "unprocessable"
	CodeRateLimited      = "rate_limited"
	CodeInternal         = "internal_error"
	CodeUnavailable      = "service_unavailable"
)

// Error is a domain error that is safe to return to API clients.
type Error struct {
	// Status is the HTTP status the API answers with.
	Status int
	// Code is a stable snake_case identifier clients may branch on.
	Code string
	// Message is human-readable and safe to display.
	Message string
	// Fields maps request field names (JSON names) to a problem description,
	// for validation errors.
	Fields map[string]string

	cause  error
	origin *Error // the sentinel a WithCause/WithMessage copy was made from
}

// Error returns the safe message, so logging a domain error or comparing its
// text behaves like the plain sentinel errors it replaces.
func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

// Unwrap exposes the cause (for logging and errors.Is on the cause).
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether e was copied (WithCause/WithMessage) from target, so
// errors.Is(err, ErrSentinel) keeps working on decorated copies.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.origin != nil && e.origin == t
}

func (e *Error) clone() *Error {
	cp := *e
	if cp.origin == nil {
		cp.origin = e
	}
	return &cp
}

// WithCause returns a copy that records the underlying error (for logs only;
// the cause is never sent to clients).
func (e *Error) WithCause(cause error) *Error {
	cp := e.clone()
	cp.cause = cause
	return cp
}

// WithMessage returns a copy with a more specific safe message.
func (e *Error) WithMessage(message string) *Error {
	cp := e.clone()
	cp.Message = message
	return cp
}

// New creates a domain error with an explicit status and code.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// BadRequest is a 400 for a malformed or unusable request.
func BadRequest(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

// Validation is a 400 with per-field problems (JSON field name -> message).
func Validation(message string, fields map[string]string) *Error {
	e := New(http.StatusBadRequest, CodeValidation, message)
	e.Fields = fields
	return e
}

// NotFound is a 404. Cross-tenant reads use it too, so another tenant's
// rows are indistinguishable from rows that don't exist.
func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

// Forbidden is a 403: the caller may not do this (permissions, role).
// Not for state conflicts — use Conflict.
func Forbidden(code, message string) *Error {
	return New(http.StatusForbidden, code, message)
}

// Conflict is a 409: the resource's current state doesn't allow the change
// (invalid status transition, duplicate, concurrent update).
func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

// Unprocessable is a 422: the request is well-formed but breaks a business rule.
func Unprocessable(code, message string) *Error {
	return New(http.StatusUnprocessableEntity, code, message)
}

// As returns the domain error in err's chain, if any.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// CodeForStatus is the default code for an HTTP status, used when a response
// has no more specific code.
func CodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusConflict:
		return CodeConflict
	case http.StatusRequestEntityTooLarge:
		return CodePayloadTooLarge
	case http.StatusUnprocessableEntity:
		return CodeUnprocessable
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	}
	if status >= 500 {
		return CodeInternal
	}
	return CodeBadRequest
}

// MessageForStatus is the default user-facing message for an HTTP status.
func MessageForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "The request is invalid"
	case http.StatusUnauthorized:
		return "Authentication required"
	case http.StatusForbidden:
		return "You don't have permission to do this"
	case http.StatusNotFound:
		return "Not found"
	case http.StatusMethodNotAllowed:
		return "Method not allowed"
	case http.StatusConflict:
		return "The request conflicts with the current state of the resource"
	case http.StatusRequestEntityTooLarge:
		return "The request is too large"
	case http.StatusUnprocessableEntity:
		return "The request could not be processed"
	case http.StatusTooManyRequests:
		return "Too many requests, try again shortly"
	case http.StatusServiceUnavailable:
		return "The service is temporarily unavailable"
	}
	if status >= 500 {
		return "Something went wrong on our side. Please try again."
	}
	return "The request could not be processed"
}
