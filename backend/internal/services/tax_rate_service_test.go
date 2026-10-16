package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

func decPtr(s string) *decimal.Decimal { d := dec(s); return &d }

// seedGlobalRate inserts a global default (tenant_id NULL) tax rate.
func seedGlobalRate(t *testing.T, f *isolationFixture) *models.TaxRate {
	t.Helper()
	tr := &models.TaxRate{ID: uuid.New(), Name: "Global VAT", Rate: dec("0.2"), TaxType: models.TaxTypeVAT, IsActive: true}
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(context.Background())).Create(tr).Error)
	return tr
}

func tenantOf(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	tid, ok := tenancy.TenantID(ctx)
	require.True(t, ok)
	return tid
}

func rateIDs(rates []models.TaxRate) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rates))
	for _, r := range rates {
		ids = append(ids, r.ID)
	}
	return ids
}

// T237: a tenant lists its own rates plus the global defaults, never another
// tenant's; active_only hides inactive ones.
func TestTaxRates_ListIsTenantScoped(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA, tenantB := tenantOf(t, f.ctxA), tenantOf(t, f.ctxB)

	own := seedRate(t, f, f.ctxA, "0.0825", true)
	inactive := seedRate(t, f, f.ctxA, "0.05", false)
	foreign := seedRate(t, f, f.ctxB, "0.07", true)
	global := seedGlobalRate(t, f)

	rates, err := svc.List(f.ctxA, tenantA, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{own.ID, inactive.ID, global.ID}, rateIDs(rates))

	rates, err = svc.List(f.ctxA, tenantA, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{own.ID, global.ID}, rateIDs(rates))

	rates, err = svc.List(f.ctxB, tenantB, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{foreign.ID, global.ID}, rateIDs(rates))

	// The explicit tenant filter holds even if the context were for another tenant.
	rates, err = svc.List(tenancy.WithSystem(context.Background()), tenantA, false)
	require.NoError(t, err)
	assert.NotContains(t, rateIDs(rates), foreign.ID)
}

// T237: another tenant's rate is a 404 for get, update and delete, and is
// left unchanged.
func TestTaxRates_CrossTenantIsNotFound(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA := tenantOf(t, f.ctxA)
	foreign := seedRate(t, f, f.ctxB, "0.07", true)

	_, err := svc.Get(f.ctxA, tenantA, foreign.ID)
	assert.ErrorIs(t, err, ErrTaxRateMissing)

	_, err = svc.Update(f.ctxA, tenantA, foreign.ID, TaxRateInput{Rate: decPtr("0.5")})
	assert.ErrorIs(t, err, ErrTaxRateMissing)

	assert.ErrorIs(t, svc.Delete(f.ctxA, tenantA, foreign.ID), ErrTaxRateMissing)

	got, err := svc.Get(f.ctxB, tenantOf(t, f.ctxB), foreign.ID)
	require.NoError(t, err)
	assertDec(t, "0.07", got.Rate, "tenant B's rate is unchanged")
}

// T237: global default rates are readable but not changeable by tenants.
func TestTaxRates_GlobalRatesAreReadOnly(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA := tenantOf(t, f.ctxA)
	global := seedGlobalRate(t, f)

	got, err := svc.Get(f.ctxA, tenantA, global.ID)
	require.NoError(t, err)
	assert.Nil(t, got.TenantID)

	_, err = svc.Update(f.ctxA, tenantA, global.ID, TaxRateInput{Name: strPtr("Mine now")})
	assert.ErrorIs(t, err, ErrTaxRateReadOnly)
	assert.ErrorIs(t, svc.Delete(f.ctxA, tenantA, global.ID), ErrTaxRateReadOnly)

	got, err = svc.Get(f.ctxA, tenantA, global.ID)
	require.NoError(t, err)
	assert.Equal(t, "Global VAT", got.Name)
}

func TestTaxRates_CreateUpdateDelete(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA := tenantOf(t, f.ctxA)

	created, err := svc.Create(f.ctxA, tenantA, TaxRateInput{Name: strPtr("  City tax "), Rate: decPtr("0.0825")})
	require.NoError(t, err)
	require.NotNil(t, created.TenantID)
	assert.Equal(t, tenantA, *created.TenantID)
	assert.Equal(t, "City tax", created.Name)
	assert.Equal(t, models.TaxTypeSalesTax, created.TaxType)
	assert.True(t, created.IsActive)

	inactive := false
	updated, err := svc.Update(f.ctxA, tenantA, created.ID, TaxRateInput{Rate: decPtr("0.09"), IsActive: &inactive})
	require.NoError(t, err)
	assertDec(t, "0.09", updated.Rate, "rate updated")
	assert.False(t, updated.IsActive)
	assert.Equal(t, "City tax", updated.Name, "omitted fields unchanged")

	require.NoError(t, svc.Delete(f.ctxA, tenantA, created.ID))
	_, err = svc.Get(f.ctxA, tenantA, created.ID)
	assert.ErrorIs(t, err, ErrTaxRateMissing)
}

func TestTaxRates_Validation(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA := tenantOf(t, f.ctxA)
	bad := models.TaxType("luxury")

	for name, in := range map[string]TaxRateInput{
		"missing rate":    {Name: strPtr("X")},
		"blank name":      {Name: strPtr("  "), Rate: decPtr("0.1")},
		"rate above 1":    {Name: strPtr("X"), Rate: decPtr("8.25")},
		"negative rate":   {Name: strPtr("X"), Rate: decPtr("-0.01")},
		"too precise":     {Name: strPtr("X"), Rate: decPtr("0.08255")},
		"unknown type":    {Name: strPtr("X"), Rate: decPtr("0.1"), TaxType: &bad},
		"name too long":   {Name: strPtr(string(make([]byte, 101))), Rate: decPtr("0.1")},
		"region too long": {Name: strPtr("X"), Rate: decPtr("0.1"), Region: strPtr(string(make([]byte, 101)))},
	} {
		_, err := svc.Create(f.ctxA, tenantA, in)
		assert.ErrorIs(t, err, ErrInvalidTaxRateInput, name)
	}
}

// T237: a rate an invoice uses can't be deleted (409), only deactivated.
func TestTaxRates_DeleteInUseIsConflict(t *testing.T) {
	f := setupIsolation(t)
	svc := NewTaxRateService(f.db)
	tenantA := tenantOf(t, f.ctxA)
	rate := seedRate(t, f, f.ctxA, "0.1", true)

	inv := newDraftInvoice(f)
	inv.TaxRateID = &rate.ID
	_, err := NewInvoiceService(f.db).CreateInvoice(f.ctxA, inv, f.userA)
	require.NoError(t, err)

	assert.ErrorIs(t, svc.Delete(f.ctxA, tenantA, rate.ID), ErrTaxRateInUse)
	_, err = svc.Get(f.ctxA, tenantA, rate.ID)
	assert.NoError(t, err)
}

// T237: switching an invoice from a configured tax rate to a direct rate
// clears tax_rate_id; picking a configured rate sets it again.
func TestInvoiceUpdate_DirectRateClearsTaxRateID(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	rate := seedRate(t, f, f.ctxA, "0.1", true)

	inv := newDraftInvoice(f)
	inv.TaxRateID = &rate.ID
	inv, err := svc.CreateInvoice(f.ctxA, inv, f.userA)
	require.NoError(t, err)

	direct := dec("0.05")
	got, err := svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{TaxRate: &direct}, f.userA)
	require.NoError(t, err)
	assert.Nil(t, got.TaxRateID, "a direct rate replaces the configured one")
	assertDec(t, "5", got.TaxAmount, "tax at the direct rate")

	got, err = svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{TaxRateID: &rate.ID}, f.userA)
	require.NoError(t, err)
	require.NotNil(t, got.TaxRateID)
	assert.Equal(t, rate.ID, *got.TaxRateID)
	assertDec(t, "10", got.TaxAmount, "tax at the configured rate")

	got, err = svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{Notes: strPtr("x")}, f.userA)
	require.NoError(t, err)
	require.NotNil(t, got.TaxRateID, "other edits keep the configured rate")
}

// T237 (Postgres): migration 000014 grants tax_rates.manage to super_admin
// and admin only, tax_rates.read to every role; the service works against the
// real schema (gen_random_uuid ids, tax_type enum, rate CHECK).
func TestTaxRates_Postgres(t *testing.T) {
	db := setupPostgres(t)
	sys := tenancy.WithSystem(context.Background())

	type grant struct{ Role, Perm string }
	var grants []grant
	require.NoError(t, db.WithContext(sys).Raw(`SELECT r.name AS role, p.name AS perm FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id JOIN permissions p ON p.id = rp.permission_id
		WHERE p.name LIKE 'tax_rates.%' ORDER BY 1, 2`).Scan(&grants).Error)
	assert.ElementsMatch(t, []grant{
		{"admin", "tax_rates.manage"}, {"admin", "tax_rates.read"},
		{"guest", "tax_rates.read"}, {"manager", "tax_rates.read"},
		{"super_admin", "tax_rates.manage"}, {"super_admin", "tax_rates.read"},
		{"user", "tax_rates.read"},
	}, grants)

	a, b := seedPGTenant(t, db), seedPGTenant(t, db)
	svc := NewTaxRateService(db)
	vat := models.TaxTypeVAT
	created, err := svc.Create(a.ctx, a.tenantID, TaxRateInput{Name: strPtr("EU VAT"), Rate: decPtr("0.2"), TaxType: &vat})
	require.NoError(t, err)

	got, err := svc.Get(a.ctx, a.tenantID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaxTypeVAT, got.TaxType)
	assertDec(t, "0.2", got.Rate, "stored rate")

	_, err = svc.Get(b.ctx, b.tenantID, created.ID)
	assert.ErrorIs(t, err, ErrTaxRateMissing, "tenant B can't see A's rate")
	_, err = svc.Update(b.ctx, b.tenantID, created.ID, TaxRateInput{Rate: decPtr("0.5")})
	assert.ErrorIs(t, err, ErrTaxRateMissing, "tenant B can't change A's rate")
	assert.ErrorIs(t, svc.Delete(b.ctx, b.tenantID, created.ID), ErrTaxRateMissing, "tenant B can't delete A's rate")

	updated, err := svc.Update(a.ctx, a.tenantID, created.ID, TaxRateInput{Rate: decPtr("0.21")})
	require.NoError(t, err)
	assertDec(t, "0.21", updated.Rate, "updated rate")
	require.NoError(t, svc.Delete(a.ctx, a.tenantID, created.ID))
}
