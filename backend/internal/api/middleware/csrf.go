package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// CSRFHeader / CSRFHeaderValue must accompany every state-changing request
// that carries an auth cookie. Browsers never let a cross-site form or a
// simple fetch set custom headers, and adding one forces a CORS preflight,
// so a forged request from another origin (including a same-site sibling
// such as another *.sharkfins.xyz app, which SameSite=Lax does not stop)
// fails here (F-16). The frontend axios client sends it on every request.
const (
	CSRFHeader      = "X-Requested-With"
	CSRFHeaderValue = "XMLHttpRequest"
)

// RequireCSRFHeader rejects cookie-authenticated POST/PUT/PATCH/DELETE
// requests without the CSRF header. Requests without auth cookies (bearer
// tokens, Stripe webhooks, public token routes) are unaffected.
func RequireCSRFHeader(cookieNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		hasAuthCookie := false
		for _, name := range cookieNames {
			if v, err := c.Cookie(name); err == nil && v != "" {
				hasAuthCookie = true
				break
			}
		}
		if hasAuthCookie && c.GetHeader(CSRFHeader) != CSRFHeaderValue {
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "csrf_check_failed",
				Message: "Missing " + CSRFHeader + " header",
			})
			return
		}
		c.Next()
	}
}
