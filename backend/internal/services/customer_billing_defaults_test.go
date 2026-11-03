package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T235: new customers inherit the tenant's onboarding billing defaults for
// the rules their create request leaves out.

func createInheriting(t *testing.T, f *isolationFixture, ctx context.Context, req *models.BillingRulesRequest) *models.Customer {
	t.Helper()
	c := &models.Customer{ID: uuid.New(), FirstName: "New", LastName: "Customer", Email: uuid.NewString() + "@x.test",
		CustomerType: "commercial", Status: "active"}
	require.NoError(t, req.Apply(&c.BillingRules))
	c.InheritTenantBillingDefaults(req)
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(ctx).Create(c))
	var stored models.Customer
	require.NoError(t, f.db.WithContext(ctx).First(&stored, "id = ?", c.ID).Error)
	return &stored
}

func TestCustomerCreate_InheritsTenantBillingDefaults(t *testing.T) {
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultConsolidatedBilling: true, DefaultPORequired: true})

	c := createInheriting(t, f, f.ctxA, &models.BillingRulesRequest{})
	assert.True(t, c.ConsolidatedBilling, "omitted: tenant default")
	assert.True(t, c.PORequired, "omitted: tenant default")

	no := false
	c = createInheriting(t, f, f.ctxA, &models.BillingRulesRequest{ConsolidatedBilling: &no, PORequired: &no})
	assert.False(t, c.ConsolidatedBilling, "the request wins")
	assert.False(t, c.PORequired, "the request wins")

	// Tenant B has no defaults: tenant A's never leak into it.
	c = createInheriting(t, f, f.ctxB, &models.BillingRulesRequest{})
	assert.False(t, c.ConsolidatedBilling)
	assert.False(t, c.PORequired)
}

// Customers created without opting in (imports, fixtures) are unchanged.
func TestCustomerCreate_WithoutInheritKeepsRules(t *testing.T) {
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultConsolidatedBilling: true, DefaultPORequired: true})
	c := &models.Customer{ID: uuid.New(), FirstName: "Plain", LastName: "Customer", Email: "plain@x.test",
		CustomerType: "commercial", Status: "active"}
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(f.ctxA).Create(c))
	assert.False(t, c.ConsolidatedBilling)
	assert.False(t, c.PORequired)
}
