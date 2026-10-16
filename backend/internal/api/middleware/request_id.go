package middleware

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	// RequestIDHeader carries the request's correlation ID in both directions.
	RequestIDHeader = "X-Request-ID"
	// requestIDKey is the gin context key holding the request ID.
	requestIDKey = "request_id"
)

// validRequestID bounds what an inbound X-Request-ID may look like: short,
// printable, no spaces or control characters, so a caller cannot inject log
// lines or oversized values. Anything else is replaced by a fresh ID.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// RequestID gives every request a correlation ID (T223). A sane inbound
// X-Request-ID (e.g. from the edge proxy or a client retry) is kept;
// otherwise a UUID is generated. The ID is returned in the X-Request-ID
// response header, stored on the gin context (RequestIDFrom), and put on the
// request context so every logging.* call made with c.Request.Context()
// carries it. Mount it first so even early rejections are tagged.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}

		c.Set(requestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), id))

		c.Next()
	}
}

// RequestIDFrom returns the request's correlation ID ("" when the RequestID
// middleware is not mounted, e.g. in narrow handler tests).
func RequestIDFrom(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

// AccessLogFormatter is gin's default access-log line with the request ID
// appended, so an access line can be matched to the app's log lines and to
// the "Reference" a user reports from the error UI. gin hands it the path
// with the raw query attached; only the path and the sorted query parameter
// names are logged, never their values (T239).
func AccessLogFormatter(param gin.LogFormatterParams) string {
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	requestID, _ := param.Keys[requestIDKey].(string)
	path, rawQuery := splitQuery(param.Path)
	queryKeys := ""
	if keys := QueryKeys(rawQuery); keys != "" {
		queryKeys = fmt.Sprintf(" | query_keys=%q", keys)
	}
	return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v | request_id=%s%s\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		path,
		requestID,
		queryKeys,
		param.ErrorMessage,
	)
}
