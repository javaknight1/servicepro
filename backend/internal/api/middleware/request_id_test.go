package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// requestIDRouter echoes what handlers see: the gin-context ID and the ID on
// the request context (what logging.* calls pick up).
func requestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"gin": RequestIDFrom(c),
			"ctx": logging.ExtractRequestID(c.Request.Context()),
		})
	})
	return r
}

func doRequestID(t *testing.T, inbound string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if inbound != "" {
		req.Header.Set(RequestIDHeader, inbound)
	}
	w := httptest.NewRecorder()
	requestIDRouter().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	id := w.Header().Get(RequestIDHeader)
	body := w.Body.String()
	if !strings.Contains(body, `"gin":"`+id+`"`) || !strings.Contains(body, `"ctx":"`+id+`"`) {
		t.Fatalf("handler did not see request ID %q: %s", id, body)
	}
	return w, id
}

func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	_, id := doRequestID(t, "")
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("generated ID %q is not a UUID", id)
	}
}

func TestRequestID_KeepsSaneInboundID(t *testing.T) {
	_, id := doRequestID(t, "edge-1234.abcd:5")
	if id != "edge-1234.abcd:5" {
		t.Fatalf("inbound ID not kept: %q", id)
	}
}

func TestRequestID_ReplacesUnsafeInboundID(t *testing.T) {
	for _, bad := range []string{
		"short",                      // too short
		"has space in it",            // spaces
		"line\r\nINJECTED log line",  // control characters
		strings.Repeat("a", 129),     // too long
		`quote"breaks-json-log-line`, // quote
	} {
		_, id := doRequestID(t, bad)
		if id == bad {
			t.Fatalf("unsafe inbound ID %q was kept", bad)
		}
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("replacement for %q is not a UUID: %q", bad, id)
		}
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	_, a := doRequestID(t, "")
	_, b := doRequestID(t, "")
	if a == b {
		t.Fatalf("two requests share ID %q", a)
	}
}

func TestAccessLogFormatter_IncludesRequestID(t *testing.T) {
	line := AccessLogFormatter(gin.LogFormatterParams{
		TimeStamp:  time.Now(),
		StatusCode: 500,
		Method:     http.MethodGet,
		Path:       "/api/v1/quotes",
		Keys:       map[any]any{requestIDKey: "req-abc-123"},
	})
	if !strings.Contains(line, "request_id=req-abc-123") || !strings.Contains(line, "500") {
		t.Fatalf("access log line missing request ID or status: %q", line)
	}
}
