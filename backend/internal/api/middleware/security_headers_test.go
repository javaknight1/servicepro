package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		config         *SecurityHeadersConfig
		expectedHeader string
		expectedValue  string
		checkContains  bool // if true, check Contains instead of exact match
	}{
		{
			name:           "X-Frame-Options is DENY by default",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "X-Frame-Options",
			expectedValue:  "DENY",
		},
		{
			name:           "X-Content-Type-Options is nosniff",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "X-Content-Type-Options",
			expectedValue:  "nosniff",
		},
		{
			name:           "X-XSS-Protection is enabled",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "X-XSS-Protection",
			expectedValue:  "1; mode=block",
		},
		{
			name:           "Referrer-Policy is strict-origin-when-cross-origin",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Referrer-Policy",
			expectedValue:  "strict-origin-when-cross-origin",
		},
		{
			name:           "Content-Security-Policy is set",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "default-src 'self'",
			checkContains:  true,
		},
		{
			name:           "CSP includes Stripe",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "https://js.stripe.com",
			checkContains:  true,
		},
		{
			name:           "Permissions-Policy is set",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Permissions-Policy",
			expectedValue:  "camera=()",
			checkContains:  true,
		},
		{
			name:           "Cross-Origin-Opener-Policy is same-origin",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Cross-Origin-Opener-Policy",
			expectedValue:  "same-origin",
		},
		{
			name:           "Cross-Origin-Resource-Policy is same-origin",
			config:         DefaultSecurityHeadersConfig("development"),
			expectedHeader: "Cross-Origin-Resource-Policy",
			expectedValue:  "same-origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeaders(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			got := w.Header().Get(tt.expectedHeader)
			if tt.checkContains {
				if !strings.Contains(got, tt.expectedValue) {
					t.Errorf("%s: expected header to contain %q, got %q", tt.expectedHeader, tt.expectedValue, got)
				}
			} else {
				if got != tt.expectedValue {
					t.Errorf("%s: expected %q, got %q", tt.expectedHeader, tt.expectedValue, got)
				}
			}
		})
	}
}

func TestSecurityHeadersMiddleware_HSTSOnlyInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		environment string
		expectHSTS  bool
	}{
		{
			name:        "HSTS not set in development",
			environment: "development",
			expectHSTS:  false,
		},
		{
			name:        "HSTS not set in staging",
			environment: "staging",
			expectHSTS:  false,
		},
		{
			name:        "HSTS set in production",
			environment: "production",
			expectHSTS:  true,
		},
		{
			name:        "HSTS set in prod (short form)",
			environment: "prod",
			expectHSTS:  true,
		},
		{
			name:        "HSTS set in PRODUCTION (uppercase)",
			environment: "PRODUCTION",
			expectHSTS:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSecurityHeadersConfig(tt.environment)
			router := gin.New()
			router.Use(SecurityHeaders(config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			hsts := w.Header().Get("Strict-Transport-Security")
			hasHSTS := hsts != ""

			if hasHSTS != tt.expectHSTS {
				if tt.expectHSTS {
					t.Errorf("expected HSTS header in %s environment, but it was not set", tt.environment)
				} else {
					t.Errorf("expected no HSTS header in %s environment, but got: %s", tt.environment, hsts)
				}
			}

			if tt.expectHSTS && !strings.Contains(hsts, "max-age=") {
				t.Errorf("HSTS header missing max-age: %s", hsts)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_CustomCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customCSP := "default-src 'none'; script-src 'self'"
	config := &SecurityHeadersConfig{
		Environment:           "development",
		ContentSecurityPolicy: customCSP,
	}

	router := gin.New()
	router.Use(SecurityHeaders(config))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	got := w.Header().Get("Content-Security-Policy")
	if got != customCSP {
		t.Errorf("expected custom CSP %q, got %q", customCSP, got)
	}
}

func TestSecurityHeadersMiddleware_FrameAncestors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &SecurityHeadersConfig{
		Environment:           "development",
		AllowedFrameAncestors: []string{"'self'", "https://trusted.example.com"},
	}

	router := gin.New()
	router.Use(SecurityHeaders(config))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// X-Frame-Options should NOT be set when frame-ancestors is used
	xfo := w.Header().Get("X-Frame-Options")
	if xfo != "" {
		t.Errorf("expected X-Frame-Options to be empty when AllowedFrameAncestors is set, got %q", xfo)
	}

	// CSP should include frame-ancestors
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors 'self' https://trusted.example.com") {
		t.Errorf("expected CSP to contain frame-ancestors directive, got %q", csp)
	}
}

func TestSecurityHeadersMiddleware_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(nil)) // Should not panic
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still set headers with defaults
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options to be set with nil config")
	}
}

func TestBuildHSTS(t *testing.T) {
	tests := []struct {
		name     string
		config   *SecurityHeadersConfig
		expected string
	}{
		{
			name: "default config",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           false,
			},
			expected: "max-age=31536000; includeSubDomains",
		},
		{
			name: "with preload",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name: "without includeSubDomains",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            86400,
				HSTSIncludeSubDomains: false,
				HSTSPreload:           false,
			},
			expected: "max-age=86400",
		},
		{
			name: "zero max-age uses default",
			config: &SecurityHeadersConfig{
				HSTSMaxAge: 0,
			},
			expected: "max-age=31536000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildHSTS(tt.config)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"production", true},
		{"prod", true},
		{"PRODUCTION", true},
		{"PROD", true},
		{"Production", true},
		{"development", false},
		{"dev", false},
		{"staging", false},
		{"test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			got := isProduction(tt.env)
			if got != tt.expected {
				t.Errorf("isProduction(%q) = %v, want %v", tt.env, got, tt.expected)
			}
		})
	}
}
