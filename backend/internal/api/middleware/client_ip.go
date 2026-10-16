package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
)

// Headers set by the Cloudflare Worker that proxies servicepro.sharkfins.xyz
// /api/* to Cloud Run. The Worker's fetch reaches Google from a Cloudflare
// egress address, so without these every visitor would share one IP (and one
// rate-limit bucket). The Worker forwards the visitor address in
// EdgeClientIPHeader together with the shared secret in EdgeSecretHeader.
const (
	EdgeSecretHeader   = "X-SP-Edge-Secret"
	EdgeClientIPHeader = "X-SP-Client-IP"
)

// clientIPContextKey stores the edge-verified visitor IP on the gin context.
const clientIPContextKey = "edge_client_ip"

// edgeVerifiedContextKey is set when the request carried the edge secret.
const edgeVerifiedContextKey = "edge_verified"

// EdgeClientIP must be the first middleware on the router. When the request
// carries EdgeSecretHeader matching secret (constant-time compare) and a
// valid EdgeClientIPHeader, it makes that address the request's client IP:
// r.RemoteAddr is rewritten and X-Forwarded-For / X-Real-IP are dropped so
// c.ClientIP() returns it regardless of Gin's trusted-proxy settings.
//
// The run.app URL is public, so the headers are honoured ONLY with the
// secret; otherwise they are ignored. Both headers are always stripped so the
// secret never reaches handlers, logs or error reports. An empty secret
// disables the feature (local development).
func EdgeClientIP(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if edgeSecretValid(c, secret) {
			c.Set(edgeVerifiedContextKey, true)
		}
		if ip := edgeClientIP(c, secret); ip != "" {
			c.Request.RemoteAddr = net.JoinHostPort(ip, "0")
			c.Request.Header.Del("X-Forwarded-For")
			c.Request.Header.Del("X-Real-IP")
			c.Set(clientIPContextKey, ip)
		}
		c.Request.Header.Del(EdgeSecretHeader)
		c.Request.Header.Del(EdgeClientIPHeader)
		c.Next()
	}
}

// edgeSecretValid reports whether the request carries the edge secret.
func edgeSecretValid(c *gin.Context, secret string) bool {
	if secret == "" {
		return false
	}
	got := c.GetHeader(EdgeSecretHeader)
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}

// RequireEdgeProxy rejects /api/* requests that did not come through the
// Cloudflare Worker (no valid edge secret), so the public *.run.app URL can't
// be used to bypass Cloudflare (WAF, rate limits, Access). It must run after
// EdgeClientIP. Paths outside /api/ — /health/*, /version, /internal/jobs/*
// (own token), /metrics — are unaffected. The Stripe webhook
// (/api/v1/stripe/webhooks) is deliberately kept behind the edge: Stripe
// calls the public domain, which the Worker proxies.
func RequireEdgeProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") && !c.GetBool(edgeVerifiedContextKey) {
			apierror.Write(c, http.StatusForbidden, "forbidden", "Direct access is not allowed")
			return
		}
		c.Next()
	}
}

// edgeClientIP returns the visitor address forwarded by the edge proxy, or ""
// when the secret is unconfigured, absent, wrong, or the address is invalid.
func edgeClientIP(c *gin.Context, secret string) string {
	if !edgeSecretValid(c, secret) {
		return ""
	}
	ip := strings.TrimSpace(c.GetHeader(EdgeClientIPHeader))
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

// ClientIP is the single helper rate limiters (and anything keying on the
// caller's address) should use: the edge-verified visitor IP when present,
// otherwise Gin's c.ClientIP().
func ClientIP(c *gin.Context) string {
	if ip := c.GetString(clientIPContextKey); ip != "" {
		return ip
	}
	return c.ClientIP()
}
