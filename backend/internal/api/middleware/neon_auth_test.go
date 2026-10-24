package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/pkg/auth/neonauth"
)

// stubNeon accepts exactly one token and returns fixed claims for it.
type stubNeon struct {
	token  string
	claims *neonauth.Claims
}

func (s stubNeon) Verify(_ context.Context, token string) (*neonauth.Claims, error) {
	if token != s.token {
		return nil, errors.New("bad token")
	}
	return s.claims, nil
}

// neonShapedToken has an EdDSA header, so RequireAuth routes it to Neon Auth.
func neonShapedToken(suffix string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","kid":"k"}`)) + ".payload." + suffix
}

func TestRequireAuth_NeonAuth(t *testing.T) {
	_, _, mr, jwtManager, pm := setupTestEnvironment(t)
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	linkedUser := uuid.New()
	good := neonShapedToken("good")
	pm.SetNeonAuth(
		stubNeon{token: good, claims: &neonauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "neon-linked"},
			Email:            "linked@example.com",
		}},
		func(authUserID string) (uuid.UUID, error) {
			if authUserID == "neon-linked" {
				return linkedUser, nil
			}
			return uuid.Nil, errors.New("not linked")
		},
	)

	router := gin.New()
	router.GET("/test", pm.RequireAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, c.MustGet(userIDKey).(uuid.UUID).String())
	})
	call := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("valid Neon token for a linked user", func(t *testing.T) {
		w := call(good)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, linkedUser.String(), w.Body.String())
	})

	t.Run("Neon token that fails verification", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, call(neonShapedToken("forged")).Code)
	})

	t.Run("valid Neon token, no linked user", func(t *testing.T) {
		unlinked := neonShapedToken("unlinked")
		pm.SetNeonAuth(stubNeon{token: unlinked, claims: &neonauth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "neon-stranger"},
		}}, pm.authUsers)
		w := call(unlinked)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "No ServicePro account is linked")
	})

	t.Run("the API's own JWTs still work", func(t *testing.T) {
		token, err := jwtManager.GenerateAccessToken(uuid.NewString(), "own@example.com")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, call(token).Code)
	})
}

func TestRequireAuth_NeonTokenRejectedWhenNeonAuthOff(t *testing.T) {
	_, _, mr, _, pm := setupTestEnvironment(t)
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", pm.RequireAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+neonShapedToken("x"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
