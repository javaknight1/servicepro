package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

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
						if requestID := RequestIDFrom(c); requestID != "" {
							scope.SetTag("request_id", requestID)
						}
						scope.SetExtra("stack_trace", stack)
						scope.SetExtra("request_method", c.Request.Method)
						scope.SetExtra("request_path", c.Request.URL.Path)
						scope.SetExtra("request_query_keys", QueryKeys(c.Request.URL.RawQuery)) // names only (T239)
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

				apierror.Write(c, http.StatusInternalServerError, apperrors.CodeInternal, "An unexpected error occurred")
			}
		}()

		c.Next()
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		apierror.Write(c, http.StatusNotFound, apperrors.CodeNotFound, "The requested resource was not found")
	}
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		apierror.Write(c, http.StatusMethodNotAllowed, apperrors.CodeMethodNotAllowed, "The request method is not supported for this endpoint")
	}
}
