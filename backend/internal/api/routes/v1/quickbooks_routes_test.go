package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/internal/services/quickbooks"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
	"github.com/javaknight1/servicepro/backend/pkg/secretbox"
)

// T192 end to end through the real middleware: integrations.manage gates
// every QuickBooks route (owners/admins only), per tenant; the OAuth
// callback needs no session and always redirects to Settings.
func TestPostgresQuickBooksRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := techTestDB(t)
	sys := tenancy.WithSystem(context.Background())
	exec := func(sql string, args ...any) { require.NoError(t, db.WithContext(sys).Exec(sql, args...).Error) }

	tenantA, tenantB := uuid.New(), uuid.New()
	adminA, managerA, adminB := uuid.New(), uuid.New(), uuid.New()
	for _, u := range []uuid.UUID{adminA, managerA, adminB} {
		exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`, u, "u-"+u.String()[:8]+"@example.com")
	}
	exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, 'A', ?, ?)`, tenantA, "qa-"+tenantA.String()[:8], adminA)
	exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, 'B', ?, ?)`, tenantB, "qb-"+tenantB.String()[:8], adminB)
	member := func(tenant, user uuid.UUID, role string) {
		exec(`INSERT INTO tenant_users (tenant_id, user_id, role_id, is_active, accepted_at) VALUES (?, ?, ?, TRUE, NOW())`,
			tenant, user, roleID(t, db, role))
	}
	member(tenantA, adminA, "admin")
	member(tenantA, managerA, "manager")
	member(tenantB, adminB, "admin")

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwt := auth.NewJWTManager(&config.JWTConfig{Secret: "qb-test-secret", AccessExpiration: time.Hour, RefreshExpiration: time.Hour})
	checker := permissionsSvc.NewPermissionChecker(repository.NewPermissionRepository(db), redisClient)
	permMW := middleware.NewPermissionMiddleware(checker, jwt, nil, "access_token")
	tenantMW := middleware.NewTenantMiddleware(repository.NewTenantRepository(db))

	fake := qbo.NewFake("http://app.test/api/v1/integrations/quickbooks/callback")
	box, err := secretbox.Random()
	require.NoError(t, err)
	svc := quickbooks.NewService(quickbooks.Config{DB: db, Client: fake, Box: box, Perms: checker})
	cfg := &routeconfigs.RouteConfig{
		Middleware: &routeconfigs.Middleware{PermMiddleware: permMW, TenantMW: tenantMW, PermChecker: checker},
		Handlers:   &routeconfigs.Handlers{QuickBooks: handlers.NewQuickBooksHandler(svc, "http://app.test")},
	}
	engine := gin.New()
	SetupQuickBooksRoutes(engine.Group("/api/v1"), cfg)

	do := func(user, tenant uuid.UUID, method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1"+path, nil)
		token, err := jwt.GenerateAccessToken(user.String(), "u@example.com")
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", tenant.String())
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}
	pathA := "/tenants/" + tenantA.String() + "/integrations/quickbooks"

	t.Run("managers can't manage integrations", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, do(managerA, tenantA, http.MethodGet, pathA).Code)
		assert.Equal(t, http.StatusForbidden, do(managerA, tenantA, http.MethodPost, pathA+"/connect").Code)
	})

	t.Run("another tenant's admin can't reach tenant A", func(t *testing.T) {
		w := do(adminB, tenantB, http.MethodGet, pathA)
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, w.Code, w.Body.String())
		w = do(adminB, tenantA, http.MethodPost, pathA+"/connect")
		assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, w.Code, w.Body.String())
	})

	t.Run("admin connects through the callback and sees the status", func(t *testing.T) {
		w := do(adminA, tenantA, http.MethodGet, pathA)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var st models.QuickBooksStatusResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &st))
		assert.True(t, st.Configured)
		assert.Equal(t, models.QuickBooksDisconnected, st.Status)

		w = do(adminA, tenantA, http.MethodPost, pathA+"/connect")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var cr models.QuickBooksConnectResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
		u, err := url.Parse(cr.URL)
		require.NoError(t, err)

		// Intuit's redirect: no Authorization header, no tenant header.
		cb := httptest.NewRecorder()
		engine.ServeHTTP(cb, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/quickbooks/callback?"+u.RawQuery, nil))
		require.Equal(t, http.StatusFound, cb.Code)
		assert.Equal(t, "http://app.test/settings/organization?quickbooks=connected&tab=integrations", cb.Header().Get("Location"))

		// Replay: refused, still a redirect (never an error page with the code).
		cb = httptest.NewRecorder()
		engine.ServeHTTP(cb, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/quickbooks/callback?"+u.RawQuery, nil))
		require.Equal(t, http.StatusFound, cb.Code)
		assert.True(t, strings.HasSuffix(cb.Header().Get("Location"), "quickbooks=error&reason=expired&tab=integrations"), cb.Header().Get("Location"))

		w = do(adminA, tenantA, http.MethodGet, pathA)
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &st))
		assert.Equal(t, models.QuickBooksConnected, st.Status)
		assert.Equal(t, "Sandbox Company_US_1", st.CompanyName)

		w = do(adminA, tenantA, http.MethodPost, pathA+"/sync")
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		w = do(adminA, tenantA, http.MethodPost, pathA+"/records/"+uuid.NewString()+"/retry")
		assert.Equal(t, http.StatusNotFound, w.Code)
		w = do(adminA, tenantA, http.MethodPost, pathA+"/records/not-a-uuid/retry")
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Tenant B is untouched.
		w = do(adminB, tenantB, http.MethodGet, "/tenants/"+tenantB.String()+"/integrations/quickbooks")
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &st))
		assert.Equal(t, models.QuickBooksDisconnected, st.Status)

		w = do(adminA, tenantA, http.MethodPost, pathA+"/disconnect")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &st))
		assert.Equal(t, models.QuickBooksDisconnected, st.Status)
	})

	t.Run("user cancelled at Intuit", func(t *testing.T) {
		cb := httptest.NewRecorder()
		engine.ServeHTTP(cb, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/quickbooks/callback?error=access_denied&state=x", nil))
		require.Equal(t, http.StatusFound, cb.Code)
		assert.True(t, strings.HasSuffix(cb.Header().Get("Location"), "reason=cancelled&tab=integrations"))
	})
}

// Without Intuit keys the status says "not configured" and connect is 503.
func TestQuickBooksHandlerNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handlers.NewQuickBooksHandler(quickbooks.NewService(quickbooks.Config{}), "http://app.test")
	r := gin.New()
	tenantID, userID := uuid.New(), uuid.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
	})
	r.GET("/status", h.Status)
	r.POST("/connect", h.Connect)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"configured":false`)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/connect", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "quickbooks_not_configured")
}
