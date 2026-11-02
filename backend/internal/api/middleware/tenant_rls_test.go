package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/internal/tenancy/rlstest"
)

// TestRequireTenant_SetsRLSTenant drives a request through the real tenant
// middleware against Postgres with RLS enforced (T154): the handler's
// queries carry the caller's tenant to the database, so even an unfiltered
// query only sees that tenant's rows.
func TestRequireTenant_SetsRLSTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := rlstest.Setup(t)

	type seeded struct{ user, tenant, customer uuid.UUID }
	adminRole := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	seed := func() seeded {
		s := seeded{user: uuid.New(), tenant: uuid.New(), customer: uuid.New()}
		short := s.tenant.String()[:8]
		require.NoError(t, env.Owner.Exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`,
			s.user, "u-"+short+"@example.com").Error)
		require.NoError(t, env.Owner.Exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`,
			s.tenant, "Tenant "+short, "t-"+short, s.user).Error)
		require.NoError(t, env.Owner.Exec(`INSERT INTO tenant_users (tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, true)`,
			s.tenant, s.user, adminRole).Error)
		require.NoError(t, env.Owner.Exec(`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
				billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
			 VALUES (?, ?, 'Pat', 'Customer', ?, '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701')`,
			s.customer, s.tenant, "pat-"+short+"@example.com").Error)
		return s
	}
	a, b := seed(), seed()

	db := env.OpenApp(t, true)
	tm := NewTenantMiddleware(repository.NewTenantRepository(db))
	customers := repository.NewCustomerRepository(db)

	r := gin.New()
	r.GET("/probe/:customer",
		func(c *gin.Context) { c.Set(userIDKey, a.user) }, // what RequireAuth sets
		tm.RequireTenant(),
		func(c *gin.Context) {
			ctx := c.Request.Context()
			var setting string
			var visible []uuid.UUID
			if err := db.WithContext(ctx).Raw(`SELECT current_setting(?, true)`, tenancy.SettingTenantID).Scan(&setting).Error; err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			// Deliberately unfiltered: only RLS scopes it.
			if err := db.WithContext(ctx).Raw(`SELECT id FROM customers`).Scan(&visible).Error; err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			id, _ := uuid.Parse(c.Param("customer"))
			_, err := customers.WithContext(ctx).GetByID(id)
			c.JSON(http.StatusOK, gin.H{"setting": setting, "visible": visible, "found": err == nil})
		})

	probe := func(customer uuid.UUID) (setting string, visible []uuid.UUID, found bool) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/probe/"+customer.String(), nil)
		req.Header.Set(TenantHeader, a.tenant.String())
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body struct {
			Setting string      `json:"setting"`
			Visible []uuid.UUID `json:"visible"`
			Found   bool        `json:"found"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Setting, body.Visible, body.Found
	}

	setting, visible, found := probe(a.customer)
	assert.Equal(t, a.tenant.String(), setting, "the request's queries carry the tenant")
	assert.Equal(t, []uuid.UUID{a.customer}, visible)
	assert.True(t, found)

	_, _, found = probe(b.customer)
	assert.False(t, found, "another tenant's customer is a not-found")

	// A member of A naming tenant B in the header is refused before any query.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe/"+b.customer.String(), nil)
	req.Header.Set(TenantHeader, b.tenant.String())
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

}
