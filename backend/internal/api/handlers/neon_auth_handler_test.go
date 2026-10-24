package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth/neonauth"
)

type stubVerifier struct{ token string }

func (s stubVerifier) Verify(_ context.Context, token string) (*neonauth.Claims, error) {
	if token != s.token {
		return nil, errors.New("bad token")
	}
	return &neonauth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "neon-sub"},
		Email:            "a@example.com", EmailVerified: true,
	}, nil
}

type stubProvisioner struct {
	err      error
	gotID    services.NeonIdentity
	gotCode  *string
	gotToken *string
}

func (s *stubProvisioner) Provision(_ context.Context, id services.NeonIdentity, code, token *string) (*services.NeonProvisionResult, error) {
	s.gotID, s.gotCode, s.gotToken = id, code, token
	if s.err != nil {
		return nil, s.err
	}
	return &services.NeonProvisionResult{UserID: uuid.MustParse("00000000-0000-0000-0000-0000000000aa"), Created: true}, nil
}

func neonToken(suffix string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","kid":"k"}`)) + ".p." + suffix
}

func callProvision(h *NeonAuthHandler, token, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/p", h.Provision)
	req := httptest.NewRequest(http.MethodPost, "/p", strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestNeonProvision_Disabled(t *testing.T) {
	w := callProvision(NewNeonAuthHandler(nil, &stubProvisioner{}), neonToken("good"), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNeonProvision_RequiresValidNeonToken(t *testing.T) {
	h := NewNeonAuthHandler(stubVerifier{token: neonToken("good")}, &stubProvisioner{})
	assert.Equal(t, http.StatusUnauthorized, callProvision(h, "", "").Code)
	assert.Equal(t, http.StatusUnauthorized, callProvision(h, "not.an-eddsa.token", "").Code)
	assert.Equal(t, http.StatusUnauthorized, callProvision(h, neonToken("forged"), "").Code)
}

func TestNeonProvision_SuccessPassesIdentityAndInvite(t *testing.T) {
	p := &stubProvisioner{}
	h := NewNeonAuthHandler(stubVerifier{token: neonToken("good")}, p)

	w := callProvision(h, neonToken("good"), `{"invite_code":"ABC-123"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "00000000-0000-0000-0000-0000000000aa", body["user_id"])
	assert.Equal(t, true, body["created"])

	assert.Equal(t, services.NeonIdentity{AuthUserID: "neon-sub", Email: "a@example.com", EmailVerified: true}, p.gotID)
	require.NotNil(t, p.gotCode)
	assert.Equal(t, "ABC-123", *p.gotCode)
	assert.Nil(t, p.gotToken)

	// No body at all is fine (existing user being linked).
	assert.Equal(t, http.StatusOK, callProvision(h, neonToken("good"), "").Code)
}

func TestNeonProvision_ErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{services.ErrNeonEmailNotVerified, http.StatusForbidden, "email_not_verified"},
		{services.ErrRegistrationClosed, http.StatusForbidden, "registration_closed"},
		{services.ErrInvalidInviteCode, http.StatusForbidden, "invalid_invite_code"},
		{services.ErrInvalidInvitation, http.StatusForbidden, "invalid_invitation"},
		{services.ErrAuthIdentityConflict, http.StatusConflict, "identity_conflict"},
		{errors.New("boom"), http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		h := NewNeonAuthHandler(stubVerifier{token: neonToken("good")}, &stubProvisioner{err: tc.err})
		w := callProvision(h, neonToken("good"), "")
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
		if tc.code != "" {
			assert.Contains(t, w.Body.String(), `"`+tc.code+`"`)
		}
		assert.NotContains(t, w.Body.String(), "boom")
	}
}
