package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/internal/services"
)

// T229: invitation handler errors carry a machine code in `error` and a human
// `message`, like the other handlers.
func TestInvitationHandler_ErrorsHaveCodeAndMessage(t *testing.T) {
	// An empty database: token lookups that reach it find nothing.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	h := NewInvitationHandler(services.NewInvitationService(db, nil, "http://app.test"))

	tenantID := uuid.New()
	withUser := func(c *gin.Context) { c.Set("user_id", uuid.New()) }
	noop := func(c *gin.Context) {}

	cases := []struct {
		name       string
		method     string
		route      string
		path       string
		body       string
		pre        gin.HandlerFunc
		handler    gin.HandlerFunc
		wantStatus int
		wantCode   string
	}{
		{"invite: bad tenant id", http.MethodPost, "/tenants/:id/invitations", "/tenants/nope/invitations",
			`{"email":"a@b.test","role_id":"` + uuid.NewString() + `"}`, withUser, h.InviteMember, http.StatusBadRequest, "invalid_tenant_id"},
		{"invite: invalid body", http.MethodPost, "/tenants/:id/invitations", "/tenants/" + tenantID.String() + "/invitations",
			`{"email":"not-an-email"}`, withUser, h.InviteMember, http.StatusBadRequest, "invalid_request"},
		{"invite: no user", http.MethodPost, "/tenants/:id/invitations", "/tenants/" + tenantID.String() + "/invitations",
			`{"email":"a@b.test","role_id":"` + uuid.NewString() + `"}`, noop, h.InviteMember, http.StatusUnauthorized, "unauthorized"},
		{"cancel: bad invitation id", http.MethodDelete, "/tenants/:id/invitations/:invitation_id", "/tenants/" + tenantID.String() + "/invitations/nope",
			"", noop, h.CancelInvitation, http.StatusBadRequest, "invalid_invitation_id"},
		{"resend: no tenant context", http.MethodPost, "/tenants/:id/invitations/:invitation_id/resend", "/tenants/" + tenantID.String() + "/invitations/" + uuid.NewString() + "/resend",
			"", noop, h.ResendInvitation, http.StatusBadRequest, "invalid_tenant_id"},
		{"get by token: unknown token", http.MethodGet, "/invitations/:token", "/invitations/tok",
			"", noop, h.GetInvitationByToken, http.StatusNotFound, "invitation_not_found"},
		{"accept: no user", http.MethodPost, "/invitations/:token/accept", "/invitations/tok/accept",
			"", noop, h.AcceptInvitation, http.StatusUnauthorized, "unauthorized"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupTestRouter()
			router.Handle(tc.method, tc.route, tc.pre, tc.handler)

			req, _ := http.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			var resp ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantCode, resp.Error)
			assert.NotEmpty(t, resp.Message, "a human message accompanies the code")
		})
	}
}
