package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation error details
type ValidationErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

// ErrorHandler middleware handles errors and returns standardized responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Handle different error types
			switch e := err.Err.(type) {
			case validator.ValidationErrors:
				handleValidationError(c, e)
				return
			default:
				// Generic error handler
				handleGenericError(c, err.Err)
			}
		}
	}
}

// handleValidationError handles validation errors
func handleValidationError(c *gin.Context, errs validator.ValidationErrors) {
	fields := make(map[string]string)

	for _, err := range errs {
		field := err.Field()
		switch err.Tag() {
		case "required":
			fields[field] = fmt.Sprintf("%s is required", field)
		case "email":
			fields[field] = fmt.Sprintf("%s must be a valid email", field)
		case "min":
			fields[field] = fmt.Sprintf("%s must be at least %s", field, err.Param())
		case "max":
			fields[field] = fmt.Sprintf("%s must be at most %s", field, err.Param())
		case "gt":
			fields[field] = fmt.Sprintf("%s must be greater than %s", field, err.Param())
		case "gte":
			fields[field] = fmt.Sprintf("%s must be greater than or equal to %s", field, err.Param())
		case "lt":
			fields[field] = fmt.Sprintf("%s must be less than %s", field, err.Param())
		case "lte":
			fields[field] = fmt.Sprintf("%s must be less than or equal to %s", field, err.Param())
		case "uuid":
			fields[field] = fmt.Sprintf("%s must be a valid UUID", field)
		default:
			fields[field] = fmt.Sprintf("%s is invalid", field)
		}
	}

	c.JSON(http.StatusBadRequest, ValidationErrorResponse{
		Error:   "Validation failed",
		Message: "One or more fields are invalid",
		Fields:  fields,
	})
}

// handleGenericError handles generic errors
func handleGenericError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := "Internal server error"

	// You can add custom error type checks here
	if err.Error() == "record not found" {
		statusCode = http.StatusNotFound
		message = "Resource not found"
	}

	c.JSON(statusCode, ErrorResponse{
		Error:   message,
		Message: err.Error(),
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error.
// If an error tracking client is provided, panics are captured and sent to the tracking service.
func RecoveryMiddleware(errorTracker errortracking.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Convert panic value to error
				var err error
				switch v := rec.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("%s", v)
				default:
					err = fmt.Errorf("%v", v)
				}

				// Get stack trace
				stack := string(debug.Stack())

				// Send to error tracking if client is available
				if errorTracker != nil {
					// Configure scope with request context
					errorTracker.ConfigureScope(c.Request.Context(), func(scope errortracking.Scope) {
						scope.SetTag("panic", "true")
						scope.SetExtra("stack_trace", stack)
						scope.SetExtra("request_method", c.Request.Method)
						scope.SetExtra("request_path", c.Request.URL.Path)
						scope.SetExtra("request_query", c.Request.URL.RawQuery)
						scope.SetExtra("client_ip", c.ClientIP())

						// Add user context if available
						if userID, exists := c.Get("user_id"); exists {
							scope.SetUser(&errortracking.User{
								ID: fmt.Sprintf("%v", userID),
							})
						}
					})

					// Capture the exception
					errorTracker.CaptureException(c.Request.Context(), err)
				}

				// Log the panic using structured logging
				logging.Error(c.Request.Context(), "Panic recovered", map[string]any{
					"error":       err.Error(),
					"stack_trace": stack,
					"method":      c.Request.Method,
					"path":        c.Request.URL.Path,
					"client_ip":   c.ClientIP(),
				})

				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "Internal server error",
					Message: "An unexpected error occurred",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "The requested resource was not found",
		})
	}
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "Method not allowed",
			Message: "The request method is not supported for this endpoint",
		})
	}
}
