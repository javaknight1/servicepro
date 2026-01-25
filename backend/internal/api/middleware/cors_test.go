package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"https://app.servicepro.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name           string
		origin         string
		expectAllowed  bool
		expectedOrigin string
	}{
		{
			name:           "allowed origin - production",
			origin:         "https://app.servicepro.com",
			expectAllowed:  true,
			expectedOrigin: "https://app.servicepro.com",
		},
		{
			name:           "allowed origin - localhost",
			origin:         "http://localhost:3000",
			expectAllowed:  true,
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:          "disallowed origin",
			origin:        "https://evil.com",
			expectAllowed: false,
		},
		{
			name:          "no origin header",
			origin:        "",
			expectAllowed: true, // Same-origin requests have no Origin header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")

			if tt.expectAllowed && tt.origin != "" {
				if allowOrigin != tt.expectedOrigin {
					t.Errorf("expected Access-Control-Allow-Origin %q, got %q", tt.expectedOrigin, allowOrigin)
				}
			} else if !tt.expectAllowed && tt.origin != "" {
				if allowOrigin != "" {
					t.Errorf("expected no Access-Control-Allow-Origin header for disallowed origin, got %q", allowOrigin)
				}
			}
		})
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"https://app.servicepro.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Custom-Header"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.POST("/api/data", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "https://app.servicepro.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check status code (should be 204 No Content)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d for preflight, got %d", http.StatusNoContent, w.Code)
	}

	// Check Access-Control-Allow-Origin
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.servicepro.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://app.servicepro.com', got %q", got)
	}

	// Check Access-Control-Allow-Methods
	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}

	// Check Access-Control-Allow-Headers
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}

	// Check Access-Control-Max-Age (preflight caching)
	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("expected Access-Control-Max-Age '3600', got %q", maxAge)
	}

	// Check Access-Control-Allow-Credentials
	allowCreds := w.Header().Get("Access-Control-Allow-Credentials")
	if allowCreds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got %q", allowCreds)
	}
}

func TestCORSMiddleware_PreflightRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.servicepro.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.POST("/api/data", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Preflight from disallowed origin
	req := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d for rejected preflight, got %d", http.StatusForbidden, w.Code)
	}

	// Should NOT have CORS headers
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for rejected origin, got %q", got)
	}
}

func TestCORSMiddleware_Credentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		allowCredentials bool
		expectHeader     bool
	}{
		{
			name:             "credentials enabled",
			allowCredentials: true,
			expectHeader:     true,
		},
		{
			name:             "credentials disabled",
			allowCredentials: false,
			expectHeader:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"https://app.servicepro.com"},
				AllowCredentials: tt.allowCredentials,
			}

			router := gin.New()
			router.Use(CORSMiddleware(cfg, "development"))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://app.servicepro.com")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			credHeader := w.Header().Get("Access-Control-Allow-Credentials")
			hasHeader := credHeader != ""

			if hasHeader != tt.expectHeader {
				if tt.expectHeader {
					t.Error("expected Access-Control-Allow-Credentials header")
				} else {
					t.Errorf("expected no Access-Control-Allow-Credentials header, got %q", credHeader)
				}
			}
		})
	}
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.servicepro.com"},
		ExposedHeaders: []string{"X-Request-ID", "X-RateLimit-Remaining"},
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.servicepro.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	exposed := w.Header().Get("Access-Control-Expose-Headers")
	if exposed != "X-Request-ID, X-RateLimit-Remaining" {
		t.Errorf("expected exposed headers 'X-Request-ID, X-RateLimit-Remaining', got %q", exposed)
	}
}

func TestCORSMiddleware_VaryHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.servicepro.com"},
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.servicepro.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Vary header is important for caching
	vary := w.Header().Get("Vary")
	if vary != "Origin" {
		t.Errorf("expected Vary header 'Origin', got %q", vary)
	}
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:        false,
		AllowedOrigins: []string{"https://app.servicepro.com"},
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.servicepro.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// When disabled, no CORS headers should be set
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS headers when disabled, got Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORSMiddleware_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORSMiddleware(nil, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.servicepro.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not panic and should work like disabled
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false, // Wildcard with credentials is invalid
	}

	router := gin.New()
	router.Use(CORSMiddleware(cfg, "development"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// With wildcard and no credentials, should return "*"
	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
	}
}

func TestNewCORSConfig_Development(t *testing.T) {
	cfg := NewCORSConfig("http://localhost:8080", "development")

	// Should include common dev URLs
	expectedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
		"http://localhost:8080", // Custom frontend URL
	}

	for _, expected := range expectedOrigins {
		found := false
		for _, origin := range cfg.AllowedOrigins {
			if origin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected origin %q in development config", expected)
		}
	}

	// Should have credentials enabled
	if !cfg.AllowCredentials {
		t.Error("expected AllowCredentials to be true in development")
	}
}

func TestNewCORSConfig_Production(t *testing.T) {
	cfg := NewCORSConfig("https://app.servicepro.com", "production")

	// Should ONLY include the frontend URL
	if len(cfg.AllowedOrigins) != 1 {
		t.Errorf("expected exactly 1 allowed origin in production, got %d", len(cfg.AllowedOrigins))
	}

	if cfg.AllowedOrigins[0] != "https://app.servicepro.com" {
		t.Errorf("expected origin 'https://app.servicepro.com', got %q", cfg.AllowedOrigins[0])
	}

	// Should NOT include localhost
	for _, origin := range cfg.AllowedOrigins {
		if origin == "http://localhost:3000" || origin == "http://localhost:5173" {
			t.Errorf("unexpected localhost origin in production: %q", origin)
		}
	}
}
