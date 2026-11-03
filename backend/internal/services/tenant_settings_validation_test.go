package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// settingsFields saves settings for the fixture's tenant A through
// TenantService.UpdateTenant and returns the field errors (nil on success).
func settingsFields(t *testing.T, f *isolationFixture, svc *TenantService, settings models.TenantSettings) map[string]string {
	t.Helper()
	tenantID, _ := tenancy.TenantID(f.ctxA)
	_, err := svc.UpdateTenant(f.ctxA, tenantID, &models.UpdateTenantRequest{Settings: &settings})
	if err == nil {
		return nil
	}
	require.True(t, errors.Is(err, ErrInvalidTenantSettings), "unexpected error: %v", err)
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, 400, de.Status)
	assert.Equal(t, apperrors.CodeValidation, de.Code)
	return de.Fields
}

func newSettingsService(t *testing.T) (*isolationFixture, *TenantService) {
	t.Helper()
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{})
	return f, NewTenantService(repository.NewTenantRepository(f.db), nil)
}

// T235: valid settings are stored, normalised.
func TestUpdateTenantSettings_ValidIsStoredNormalised(t *testing.T) {
	f, svc := newSettingsService(t)
	term := seedTerm(t, f, f.ctxA, 30)
	rate := seedRate(t, f, f.ctxA, "0.08", true)

	fields := settingsFields(t, f, svc, models.TenantSettings{
		Currency:             " cad ",
		InvoicePrefix:        " BF-INV- ",
		QuotePrefix:          "Q-",
		JobPrefix:            "WO-",
		Timezone:             "America/New_York",
		DefaultPaymentTermID: &term.ID,
		DefaultTaxRateID:     &rate.ID,
		Trade:                models.TenantTradeCommercialCleaning,
		TechnicianCount:      7,
		OnboardingStep:       2,
		BusinessCity:         "Columbus",
		BusinessState:        "OH",
	})
	require.Nil(t, fields)

	tenantID, _ := tenancy.TenantID(f.ctxA)
	stored := models.TenantSettingsInTx(f.db, tenantID)
	assert.Equal(t, "CAD", stored.Currency)
	assert.Equal(t, "BF-INV-", stored.InvoicePrefix)
	assert.Equal(t, term.ID, *stored.DefaultPaymentTermID)
	assert.Equal(t, rate.ID, *stored.DefaultTaxRateID)
	assert.Equal(t, 7, stored.TechnicianCount)
}

// T235: each bad value is a 400 naming the setting.
func TestUpdateTenantSettings_RejectsBadValues(t *testing.T) {
	f, svc := newSettingsService(t)
	fields := settingsFields(t, f, svc, models.TenantSettings{
		Currency:        "XYZ",
		InvoicePrefix:   strings.Repeat("A", 21),
		QuotePrefix:     "Q\n-",
		JobPrefix:       "J\x00",
		Timezone:        "Mars/Olympus_Mons",
		Trade:           "plumbing",
		TechnicianCount: -1,
		OnboardingStep:  9,
		BusinessCity:    strings.Repeat("c", 101),
	})
	for _, key := range []string{"currency", "invoicePrefix", "quotePrefix", "jobPrefix", "timezone", "trade",
		"technicianCount", "onboardingStep", "businessCity"} {
		assert.Contains(t, fields, key)
	}

	// "Local" is the server's zone, not an IANA name a tenant can pick.
	assert.Contains(t, settingsFields(t, f, svc, models.TenantSettings{Timezone: "Local"}), "timezone")
	// A 20-character prefix is fine; empty values mean "use the default".
	assert.Nil(t, settingsFields(t, f, svc, models.TenantSettings{InvoicePrefix: strings.Repeat("A", 20)}))
	assert.Nil(t, settingsFields(t, f, svc, models.TenantSettings{}))
}

// T235 cross-tenant: tenant B's payment term / tax rate can't become tenant
// A's default; unknown and inactive rows are rejected the same way; global
// rows (tenant_id IS NULL) are allowed.
func TestUpdateTenantSettings_DefaultsMustBeTenantsOwn(t *testing.T) {
	f, svc := newSettingsService(t)
	termB := seedTerm(t, f, f.ctxB, 30)
	rateB := seedRate(t, f, f.ctxB, "0.05", true)
	fields := settingsFields(t, f, svc, models.TenantSettings{DefaultPaymentTermID: &termB.ID, DefaultTaxRateID: &rateB.ID})
	assert.Equal(t, "was not found", fields["defaultPaymentTermId"])
	assert.Equal(t, "was not found", fields["defaultTaxRateId"])

	unknown := uuid.New()
	inactive := seedRate(t, f, f.ctxA, "0.07", false)
	fields = settingsFields(t, f, svc, models.TenantSettings{DefaultPaymentTermID: &unknown, DefaultTaxRateID: &inactive.ID})
	assert.Contains(t, fields, "defaultPaymentTermId")
	assert.Contains(t, fields, "defaultTaxRateId")

	global := &models.PaymentTerm{ID: uuid.New(), Name: "Net 30", TermType: models.PaymentTermType("net_30"), DaysUntilDue: 30, IsActive: true}
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(f.ctxA)).Create(global).Error)
	assert.Nil(t, settingsFields(t, f, svc, models.TenantSettings{DefaultPaymentTermID: &global.ID}))
}

// T235: a value stored before validation existed doesn't block saving the
// rest of the settings unchanged.
func TestUpdateTenantSettings_UnchangedLegacyValuesPass(t *testing.T) {
	f, svc := newSettingsService(t)
	gone := uuid.New()
	legacy := models.TenantSettings{Currency: "usd$", InvoicePrefix: strings.Repeat("X", 30), DefaultTaxRateID: &gone}
	f.setTenantSettings(t, f.ctxA, legacy)

	next := legacy
	next.BusinessCity = "Columbus"
	assert.Nil(t, settingsFields(t, f, svc, next))

	next.Currency = "EURO"
	assert.Contains(t, settingsFields(t, f, svc, next), "currency")
}

// T235: settings on create are validated; default IDs can't be set before the
// tenant has any rows.
func TestValidateTenantSettings_OnCreate(t *testing.T) {
	id := uuid.New()
	err := validateTenantSettings(t.Context(), nil, uuid.Nil, &models.TenantSettings{},
		&models.TenantSettings{Currency: "usd", DefaultTaxRateID: &id})
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Contains(t, de.Fields, "defaultTaxRateId")
	assert.NotContains(t, de.Fields, "currency")

	assert.NoError(t, validateTenantSettings(t.Context(), nil, uuid.Nil, &models.TenantSettings{},
		&models.TenantSettings{Currency: "usd", OnboardingStep: 1}))
}

// T030: the hourly labor rate is a non-negative amount with at most 2
// decimals; it round-trips through the settings JSON as a decimal.
func TestUpdateTenantSettings_LaborRate(t *testing.T) {
	f, svc := newSettingsService(t)
	rate := func(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }

	for _, bad := range []string{"-1", "10000.01", "85.125"} {
		assert.Contains(t, settingsFields(t, f, svc, models.TenantSettings{LaborRate: rate(bad)}), "laborRate", bad)
	}
	assert.Nil(t, settingsFields(t, f, svc, models.TenantSettings{LaborRate: rate("85.50")}))
	tenantID, _ := tenancy.TenantID(f.ctxA)
	stored := models.TenantSettingsInTx(f.db, tenantID)
	require.NotNil(t, stored.LaborRate)
	assert.Equal(t, "85.50", stored.LaborRate.StringFixed(2))

	// Unset is fine (no labor line on job invoices).
	assert.Nil(t, settingsFields(t, f, svc, models.TenantSettings{}))
}

func TestIsISO4217Currency(t *testing.T) {
	for _, c := range []string{"USD", "usd", " EUR ", "JPY", "CAD"} {
		assert.True(t, models.IsISO4217Currency(c), c)
	}
	for _, c := range []string{"", "US", "XYZ", "XAU", "USDT"} {
		assert.False(t, models.IsISO4217Currency(c), c)
	}
}
