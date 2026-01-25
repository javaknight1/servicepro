package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig configures the security headers middleware
type SecurityHeadersConfig struct {
	// Environment determines if HSTS should be enabled (only in production)
	Environment string

	// ContentSecurityPolicy allows customizing the CSP directive
	// If empty, a sensible default is used
	ContentSecurityPolicy string

	// AllowedFrameAncestors overrides X-Frame-Options with CSP frame-ancestors
	// Use this if you need to allow specific domains to embed your app
	AllowedFrameAncestors []string

	// ReportURI for CSP violation reporting (optional)
	CSPReportURI string

	// HSTSMaxAge in seconds (default: 31536000 = 1 year)
	HSTSMaxAge int

	// HSTSIncludeSubDomains includes subdomains in HSTS
	HSTSIncludeSubDomains bool

	// HSTSPreload adds preload directive to HSTS
	HSTSPreload bool
}

// DefaultSecurityHeadersConfig returns a configuration suitable for ServicePro
func DefaultSecurityHeadersConfig(env string) *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Environment:           env,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubDomains: true,
		HSTSPreload:           false, // Don't preload by default - requires submission to preload list
	}
}

// SecurityHeaders returns a middleware that adds security-related HTTP headers
// to all responses. These headers help protect against common web vulnerabilities.
func SecurityHeaders(config *SecurityHeadersConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityHeadersConfig("development")
	}

	// Build CSP once at initialization
	csp := buildCSP(config)

	// Build HSTS header once
	hsts := buildHSTS(config)

	return func(c *gin.Context) {
		// X-Frame-Options: Prevents clickjacking attacks by controlling
		// whether the page can be embedded in frames
		// DENY = never allow framing (strictest)
		if len(config.AllowedFrameAncestors) == 0 {
			c.Header("X-Frame-Options", "DENY")
		}
		// If AllowedFrameAncestors is set, CSP frame-ancestors takes precedence

		// X-Content-Type-Options: Prevents MIME type sniffing attacks
		// Browsers may ignore declared Content-Type and guess based on content
		// "nosniff" forces browsers to respect the declared Content-Type
		c.Header("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection: Legacy XSS filter for older browsers
		// Modern browsers have deprecated this in favor of CSP, but it's
		// still useful for IE11 and older Safari versions
		// "1; mode=block" enables filter and blocks page instead of sanitizing
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: Controls how much URL info is sent in Referer header
		// "strict-origin-when-cross-origin":
		// - Same-origin: full URL
		// - Cross-origin HTTPS→HTTPS: origin only (no path)
		// - Cross-origin HTTPS→HTTP: nothing
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy: Whitelist for allowed content sources
		// This is the most powerful security header - it can prevent XSS
		// exploitation even if vulnerabilities exist
		if csp != "" {
			c.Header("Content-Security-Policy", csp)
		}

		// Strict-Transport-Security (HSTS): Forces HTTPS
		// Only enable in production to avoid issues during development
		// This header tells browsers to only use HTTPS for this domain
		if isProduction(config.Environment) && hsts != "" {
			c.Header("Strict-Transport-Security", hsts)
		}

		// Permissions-Policy: Controls browser features (formerly Feature-Policy)
		// Disables sensitive features that aren't needed
		c.Header("Permissions-Policy", buildPermissionsPolicy())

		// Cross-Origin-Opener-Policy: Isolates browsing context
		// "same-origin" prevents cross-origin documents from sharing a browsing context
		c.Header("Cross-Origin-Opener-Policy", "same-origin")

		// Cross-Origin-Resource-Policy: Controls cross-origin resource sharing
		// "same-origin" only allows same-origin requests to load resources
		// Use "cross-origin" if you serve public APIs or assets
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		c.Next()
	}
}

// buildCSP constructs the Content-Security-Policy header value
func buildCSP(config *SecurityHeadersConfig) string {
	if config.ContentSecurityPolicy != "" {
		return config.ContentSecurityPolicy
	}

	// Default CSP for ServicePro:
	// - Allow scripts from self and Stripe (for payment integration)
	// - Allow styles from self with unsafe-inline (many UI frameworks need this)
	// - Allow images from self, data URIs, and HTTPS sources
	// - Allow fonts from self and common CDNs
	// - Allow connections to self and Stripe API
	// - Allow frames from Stripe (for payment elements)
	directives := []string{
		// Default fallback for all resource types
		"default-src 'self'",

		// JavaScript sources
		// 'self' = same origin
		// js.stripe.com = Stripe.js for payment integration
		// 'unsafe-inline' is avoided for scripts - use nonces in production if needed
		"script-src 'self' https://js.stripe.com",

		// CSS sources
		// 'unsafe-inline' needed for many UI frameworks (Tailwind, styled-components, etc.)
		"style-src 'self' 'unsafe-inline'",

		// Image sources
		// data: = inline base64 images
		// https: = allow images from any HTTPS source (avatars, logos, etc.)
		// blob: = for dynamically generated images
		"img-src 'self' data: https: blob:",

		// Font sources
		"font-src 'self' data:",

		// AJAX/WebSocket/fetch connections
		// api.stripe.com = Stripe payment API
		"connect-src 'self' https://api.stripe.com",

		// Embedded frames (iframes)
		// js.stripe.com = Stripe payment element iframes
		"frame-src https://js.stripe.com",

		// Form submission targets
		"form-action 'self'",

		// Base URI for relative URLs
		"base-uri 'self'",

		// Object/embed/applet (Flash, Java) - block all
		"object-src 'none'",
	}

	// Add frame-ancestors if configured (overrides X-Frame-Options)
	if len(config.AllowedFrameAncestors) > 0 {
		directives = append(directives,
			fmt.Sprintf("frame-ancestors %s", strings.Join(config.AllowedFrameAncestors, " ")))
	} else {
		directives = append(directives, "frame-ancestors 'none'")
	}

	// Add report-uri if configured
	if config.CSPReportURI != "" {
		directives = append(directives,
			fmt.Sprintf("report-uri %s", config.CSPReportURI))
	}

	return strings.Join(directives, "; ")
}

// buildHSTS constructs the Strict-Transport-Security header value
func buildHSTS(config *SecurityHeadersConfig) string {
	if config.HSTSMaxAge <= 0 {
		config.HSTSMaxAge = 31536000 // 1 year default
	}

	hsts := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)

	if config.HSTSIncludeSubDomains {
		hsts += "; includeSubDomains"
	}

	if config.HSTSPreload {
		hsts += "; preload"
	}

	return hsts
}

// buildPermissionsPolicy constructs the Permissions-Policy header
// This disables browser features that ServicePro doesn't need
func buildPermissionsPolicy() string {
	// Disable features we don't use to reduce attack surface
	policies := []string{
		"accelerometer=()",          // Motion sensors
		"camera=()",                 // Camera access
		"geolocation=()",            // GPS/location (enable if needed for field service)
		"gyroscope=()",              // Orientation sensors
		"magnetometer=()",           // Compass
		"microphone=()",             // Microphone access
		"payment=(self)",            // Payment Request API - allow for Stripe
		"usb=()",                    // USB device access
		"interest-cohort=()",        // FLoC tracking (deprecated but still good to block)
		"browsing-topics=()",        // Topics API (successor to FLoC)
		"join-ad-interest-group=()", // FLEDGE/Protected Audience API
	}

	return strings.Join(policies, ", ")
}

// isProduction checks if the environment is production
func isProduction(env string) bool {
	env = strings.ToLower(env)
	return env == "production" || env == "prod"
}
