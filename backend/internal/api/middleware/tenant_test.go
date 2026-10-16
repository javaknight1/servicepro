package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// fakeTenantRepo is an in-memory TenantRepository: memberships[tenant][user] = role.
type fakeTenantRepo struct {
	tenants     map[uuid.UUID]*models.Tenant
	memberships map[uuid.UUID]map[uuid.UUID]*models.Role
}

func (f *fakeTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Tenant, error) {
	if t, ok := f.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeTenantRepo) GetBySlug(context.Context, string) (*models.Tenant, error) {
	return nil, errors.New("not found")
}
func (f *fakeTenantRepo) UserBelongsToTenant(_ context.Context, userID, tenantID uuid.UUID) (bool, error) {
	_, ok := f.memberships[tenantID][userID]
	return ok, nil
}
func (f *fakeTenantRepo) GetUserTenants(_ context.Context, userID uuid.UUID) ([]models.Tenant, error) {
	var out []models.Tenant
	for tid, members := range f.memberships {
		if _, ok := members[userID]; ok {
			out = append(out, *f.tenants[tid])
		}
	}
	return out, nil
}
func (f *fakeTenantRepo) GetUserTenantMembership(_ context.Context, userID, tenantID uuid.UUID) (*models.TenantUser, error) {
	role, ok := f.memberships[tenantID][userID]
	if !ok {
		return nil, errors.New("user not in tenant")
	}
	return &models.TenantUser{TenantID: tenantID, UserID: userID, RoleID: role.ID, Role: role, IsActive: true}, nil
}

func TestRequireTenantFromPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := uuid.New()
	mine, other := uuid.New(), uuid.New()
	admin := &models.Role{ID: uuid.New(), Name: "admin", HierarchyLevel: 80}
	repo := &fakeTenantRepo{
		tenants: map[uuid.UUID]*models.Tenant{
			mine:  {ID: mine, IsActive: true},
			other: {ID: other, IsActive: true},
		},
		memberships: map[uuid.UUID]map[uuid.UUID]*models.Role{
			mine: {user: admin},
		},
	}
	tm := NewTenantMiddleware(repo)

	newRouter := func() *gin.Engine {
		r := gin.New()
		r.GET("/tenants/:id/members",
			func(c *gin.Context) { c.Set(userIDKey, user) },
			tm.RequireTenantFromPath("id"),
			func(c *gin.Context) {
				ctxTenant, _ := tenancy.TenantID(c.Request.Context())
				ginTenant, _ := GetTenantID(c)
				if ctxTenant != ginTenant {
					c.Status(http.StatusTeapot)
					return
				}
				c.JSON(http.StatusOK, gin.H{"tenant": ginTenant, "level": GetTenantRoleLevel(c)})
			})
		return r
	}

	t.Run("member of path tenant", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tenants/"+mine.String()+"/members", nil)
		// A header naming another tenant must not override the path.
		req.Header.Set(TenantHeader, other.String())
		newRouter().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), mine.String())
		assert.Contains(t, w.Body.String(), `"level":80`)
	})

	t.Run("not a member of path tenant", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tenants/"+other.String()+"/members", nil)
		// Even with a header naming a tenant the user does belong to.
		req.Header.Set(TenantHeader, mine.String())
		newRouter().ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid tenant id", func(t *testing.T) {
		w := httptest.NewRecorder()
		newRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tenants/nope/members", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRequireTenant_HeaderMustBeMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := uuid.New()
	mine, other := uuid.New(), uuid.New()
	role := &models.Role{ID: uuid.New(), Name: "user", HierarchyLevel: 40}
	repo := &fakeTenantRepo{
		tenants: map[uuid.UUID]*models.Tenant{
			mine:  {ID: mine, IsActive: true},
			other: {ID: other, IsActive: true},
		},
		memberships: map[uuid.UUID]map[uuid.UUID]*models.Role{mine: {user: role}},
	}
	tm := NewTenantMiddleware(repo)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { c.Set(userIDKey, user) }, tm.RequireTenant(), func(c *gin.Context) {
		id, _ := tenancy.TenantID(c.Request.Context())
		c.String(http.StatusOK, id.String())
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(TenantHeader, other.String())
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mine.String(), w.Body.String())
}
