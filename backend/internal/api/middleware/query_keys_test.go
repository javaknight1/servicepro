package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQueryKeys(t *testing.T) {
	cases := map[string]string{
		"":                               "",
		"token=secret123":                "token",
		"b=2&a=1&b=3":                    "a,b",
		"email=a%40b.test;code=9&flag":   "code,email,flag",
		"a%5Bx%5D=1&%zz=2&=nokey":        "%zz,a[x]",
		"x=1&\n%0Aevil=2":                "evil,x",
		"page=1&page_size=50&sort=-date": "page,page_size,sort",
	}
	for raw, want := range cases {
		if got := QueryKeys(raw); got != want {
			t.Errorf("QueryKeys(%q) = %q, want %q", raw, got, want)
		}
	}
}

// T239: the access log records the path and parameter names only — a token
// in the query string never reaches a log line.
func TestAccessLog_NeverLogsQueryValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(RequestID())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{Formatter: AccessLogFormatter, Output: &buf}))
	r.GET("/api/v1/auth/verify", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, target := range []string{
		"/api/v1/auth/verify?token=secret123",
		"/api/v1/auth/verify?email=victim%40example.com&token=secret123",
		"/api/v1/nope?token=secret123", // 404s are logged too
	} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	}

	out := buf.String()
	if strings.Count(out, "\n") != 3 {
		t.Fatalf("expected 3 access lines, got %q", out)
	}
	for _, secret := range []string{"secret123", "victim", "example.com", "?"} {
		if strings.Contains(out, secret) {
			t.Fatalf("access log leaks %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, `"/api/v1/auth/verify" | request_id=`) || !strings.Contains(out, `query_keys="email,token"`) {
		t.Fatalf("access log should keep the path and parameter names: %s", out)
	}
}
