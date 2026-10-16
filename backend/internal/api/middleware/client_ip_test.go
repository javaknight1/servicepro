package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// runEdge sends one request through EdgeClientIP and reports what the handler
// saw: ClientIP(c), c.ClientIP(), and whether the edge headers leaked through.
func runEdge(t *testing.T, secret string, headers map[string]string) (helperIP, ginIP string, leaked bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(EdgeClientIP(secret))
	r.GET("/", func(c *gin.Context) {
		helperIP = ClientIP(c)
		ginIP = c.ClientIP()
		leaked = c.GetHeader(EdgeSecretHeader) != "" || c.GetHeader(EdgeClientIPHeader) != ""
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "104.16.0.1:443" // Cloudflare egress
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
	return helperIP, ginIP, leaked
}

func TestEdgeClientIP_ValidSecret(t *testing.T) {
	helperIP, ginIP, leaked := runEdge(t, "s3cret", map[string]string{
		EdgeSecretHeader:   "s3cret",
		EdgeClientIPHeader: "203.0.113.7",
		"X-Forwarded-For":  "198.51.100.9, 104.16.0.1",
	})
	if helperIP != "203.0.113.7" || ginIP != "203.0.113.7" {
		t.Fatalf("ClientIP = %q, c.ClientIP() = %q, want 203.0.113.7", helperIP, ginIP)
	}
	if leaked {
		t.Fatal("edge headers must be stripped before handlers")
	}
}

func TestEdgeClientIP_IPv6(t *testing.T) {
	helperIP, ginIP, _ := runEdge(t, "s3cret", map[string]string{
		EdgeSecretHeader:   "s3cret",
		EdgeClientIPHeader: "2001:db8::1",
	})
	if helperIP != "2001:db8::1" || ginIP != "2001:db8::1" {
		t.Fatalf("ClientIP = %q, c.ClientIP() = %q, want 2001:db8::1", helperIP, ginIP)
	}
}

func TestEdgeClientIP_Ignored(t *testing.T) {
	cases := map[string]struct {
		secret  string
		headers map[string]string
	}{
		"wrong secret":       {"s3cret", map[string]string{EdgeSecretHeader: "nope", EdgeClientIPHeader: "203.0.113.7"}},
		"missing secret":     {"s3cret", map[string]string{EdgeClientIPHeader: "203.0.113.7"}},
		"secret unset":       {"", map[string]string{EdgeSecretHeader: "", EdgeClientIPHeader: "203.0.113.7"}},
		"invalid ip":         {"s3cret", map[string]string{EdgeSecretHeader: "s3cret", EdgeClientIPHeader: "not-an-ip"}},
		"unset secret match": {"", map[string]string{EdgeSecretHeader: "anything", EdgeClientIPHeader: "203.0.113.7"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			helperIP, _, leaked := runEdge(t, tc.secret, tc.headers)
			if helperIP == "203.0.113.7" {
				t.Fatal("forged X-SP-Client-IP must not be honoured")
			}
			if helperIP != "104.16.0.1" {
				t.Fatalf("ClientIP = %q, want peer address 104.16.0.1", helperIP)
			}
			if leaked {
				t.Fatal("edge headers must be stripped even when not honoured")
			}
		})
	}
}
