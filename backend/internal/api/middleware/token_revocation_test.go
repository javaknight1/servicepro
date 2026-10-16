package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth_TokenTypeAndVersion(t *testing.T) {
	_, _, mr, jwtManager, pm := setupTestEnvironment(t)
	defer mr.Close()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	current := 2
	pm.SetTokenVersionLookup(func(id uuid.UUID) (int, error) {
		if id != userID {
			return 0, errors.New("no such user")
		}
		return current, nil
	})

	r := gin.New()
	r.GET("/me", pm.RequireAuth(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	call := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	access, refresh, err := jwtManager.GenerateTokenPairForVersion(userID.String(), "u@example.com", 2)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, call(access))
	assert.Equal(t, http.StatusUnauthorized, call(refresh), "refresh token is not an access token")

	current = 3 // logout / password reset
	assert.Equal(t, http.StatusUnauthorized, call(access), "revoked by token version bump")

	other, _, err := jwtManager.GenerateTokenPairForVersion(uuid.NewString(), "x@example.com", 0)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, call(other), "unknown user")
}
