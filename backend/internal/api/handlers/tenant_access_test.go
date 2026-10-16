package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// accessTenantRepo is an in-memory middleware.TenantRepository:
// memberships[tenant][user] = role.
type accessTenantRepo struct {
	tenants     map[uuid.UUID]*models.Tenant
	memberships map[uuid.UUID]map[uuid.UUID]*models.Role
}

func (f *accessTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Tenant, error) {
	if t, ok := f.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}
func (f *accessTenantRepo) GetBySlug(context.Context, string) (*models.Tenant, error) {
	return nil, errors.New("not found")
}
func (f *accessTenantRepo) UserBelongsToTenant(_ context.Context, userID, tenantID uuid.UUID) (bool, error) {
	_, ok := f.memberships[tenantID][userID]
	return ok, nil
}
func (f *accessTenantRepo) GetUserTenants(context.Context, uuid.UUID) ([]models.Tenant, error) {
	return nil, nil
}
func (f *accessTenantRepo) GetUserTenantMembership(_ context.Context, userID, tenantID uuid.UUID) (*models.TenantUser, error) {
	role, ok := f.memberships[tenantID][userID]
	if !ok {
		return nil, errors.New("user not in tenant")
	}
	return &models.TenantUser{TenantID: tenantID, UserID: userID, RoleID: role.ID, Role: role, IsActive: true}, nil
}

// fakeEffectivePermissions returns perms[tenant][user].
type fakeEffectivePermissions struct {
	perms map[uuid.UUID]map[uuid.UUID][]string
	err   error
}

func (f *fakeEffectivePermissions) EffectivePermissionNames(_ context.Context, userID, tenantID uuid.UUID) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.perms[tenantID][userID], nil
}

// T230: GET /tenants/:id/me returns the caller's role and effective
// permissions in that tenant only; non-members are refused.
func TestTenantHandler_GetMyAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := uuid.New()
	tenantA, tenantB, foreign := uuid.New(), uuid.New(), uuid.New()
	admin := &models.Role{ID: uuid.New(), Name: "admin", HierarchyLevel: 80}
	tech := &models.Role{ID: uuid.New(), Name: "technician", HierarchyLevel: 40}
	repo := &accessTenantRepo{
		tenants: map[uuid.UUID]*models.Tenant{
			tenantA: {ID: tenantA, IsActive: true},
			tenantB: {ID: tenantB, IsActive: true},
			foreign: {ID: foreign, IsActive: true},
		},
		memberships: map[uuid.UUID]map[uuid.UUID]*models.Role{
			tenantA: {user: admin},
			tenantB: {user: tech},
			foreign: {uuid.New(): admin},
		},
	}
	perms := &fakeEffectivePermissions{perms: map[uuid.UUID]map[uuid.UUID][]string{
		tenantA: {user: {"customers.read", "tenants.manage", "tenants.update"}},
		tenantB: {user: {"jobs.read"}},
		foreign: {user: {"tenants.delete"}}, // must never be reachable
	}}

	h := NewTenantHandler(nil)
	h.SetPermissions(perms)
	tm := middleware.NewTenantMiddleware(repo)

	get := func(tenant string) *httptest.ResponseRecorder {
		r := gin.New()
		r.GET("/tenants/:id/me",
			func(c *gin.Context) { c.Set("user_id", user) },
			tm.RequireTenantFromPath("id"),
			h.GetMyAccess)
		req := httptest.NewRequest(http.MethodGet, "/tenants/"+tenant+"/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("member gets role and permissions for that tenant", func(t *testing.T) {
		w := get(tenantA.String())
		require.Equal(t, http.StatusOK, w.Code)
		var resp models.TenantAccessResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, tenantA, resp.TenantID)
		assert.Equal(t, admin.ID, resp.RoleID)
		assert.Equal(t, "admin", resp.Role)
		assert.Equal(t, 80, resp.RoleLevel)
		assert.Equal(t, []string{"customers.read", "tenants.manage", "tenants.update"}, resp.Permissions)
	})

	t.Run("a role in tenant A grants nothing in tenant B", func(t *testing.T) {
		w := get(tenantB.String())
		require.Equal(t, http.StatusOK, w.Code)
		var resp models.TenantAccessResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "technician", resp.Role)
		assert.Equal(t, []string{"jobs.read"}, resp.Permissions)
	})

	t.Run("non-member is refused (cross-tenant)", func(t *testing.T) {
		w := get(foreign.String())
		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp models.ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "tenant_access_denied", resp.Error)
		assert.NotContains(t, w.Body.String(), "tenants.delete")
	})

	t.Run("bad tenant id", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, get("nope").Code)
	})

	t.Run("checker failure is a 500 without details", func(t *testing.T) {
		perms.err = errors.New("redis down: secret detail")
		defer func() { perms.err = nil }()
		w := get(tenantA.String())
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), "secret detail")
	})
}
