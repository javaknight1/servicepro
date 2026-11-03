package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

type arAgingFixture struct {
	db     *gorm.DB
	ctx    context.Context
	userID uuid.UUID
	tag    string
}

func newARAgingFixture(t *testing.T) *arAgingFixture {
	t.Helper()
	db := openProvisioningTestDB(t)
	f := &arAgingFixture{db: db, ctx: tenancy.WithSystem(context.Background()), tag: uuid.NewString()[:8]}
	require.NoError(t, db.WithContext(f.ctx).Raw(
		`INSERT INTO users (email, password_hash) VALUES (?, 'x') RETURNING id`, "aging-"+f.tag+"@example.com",
	).Row().Scan(&f.userID))
	t.Cleanup(func() {
		ctx := tenancy.WithSystem(context.Background())
		like := "aging-" + f.tag + "%"
		for _, q := range []string{
			`DELETE FROM invoice_payments WHERE invoice_id IN (SELECT id FROM invoices WHERE invoice_number LIKE ?)`,
			`DELETE FROM invoices WHERE invoice_number LIKE ?`,
			`DELETE FROM customers WHERE email LIKE ?`,
			`DELETE FROM tenants WHERE slug LIKE ?`,
			`DELETE FROM users WHERE email LIKE ?`,
		} {
			_ = db.WithContext(ctx).Exec(q, like).Error
		}
	})
	return f
}

func (f *arAgingFixture) tenant(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.db.WithContext(f.ctx).Raw(
		`INSERT INTO tenants (name, slug, owner_id) VALUES (?, ?, ?) RETURNING id`, name, "aging-"+f.tag+"-"+name, f.userID,
	).Row().Scan(&id))
	return id
}

func (f *arAgingFixture) customer(t *testing.T, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.db.WithContext(f.ctx).Raw(
		`INSERT INTO customers (tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
		VALUES (?, ?, 'Test', ?, '+16145550100', '1 Main St', 'Columbus', 'OH', '43215') RETURNING id`,
		tenantID, name, "aging-"+f.tag+"-"+name+"@example.com",
	).Row().Scan(&id))
	return id
}

// invoice is sent, unpaid, total `amount`, due `daysPastDue` days before asOf.
func (f *arAgingFixture) invoice(t *testing.T, tenantID, customerID uuid.UUID, n int, amount string, due time.Time) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.db.WithContext(f.ctx).Raw(
		`INSERT INTO invoices (tenant_id, customer_id, invoice_number, issue_date, due_date, status,
			subtotal, total_amount, amount_paid, created_by)
		VALUES (?, ?, ?, ?, ?, 'sent', ?, ?, 0, ?) RETURNING id`,
		tenantID, customerID, "aging-"+f.tag+"-"+string(rune('A'+n)), due.AddDate(0, 0, -30), due, amount, amount, f.userID,
	).Row().Scan(&id))
	return id
}

// The per-customer A/R aging buckets were always zero (GORM mapped
// Days1To30 to "days1_to30", not the query's "days_1_to_30") and customers
// with no payments got last_payment 0001-01-01.
func TestARAgingCustomerSummary_BucketsAndLastPayment(t *testing.T) {
	f := newARAgingFixture(t)
	tenantA := f.tenant(t, "a")
	tenantB := f.tenant(t, "b")
	asOf := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

	paying := f.customer(t, tenantA, "paying")
	f.invoice(t, tenantA, paying, 0, "100.00", asOf.AddDate(0, 0, 5))   // current
	f.invoice(t, tenantA, paying, 1, "200.00", asOf.AddDate(0, 0, -10)) // 1–30
	f.invoice(t, tenantA, paying, 2, "300.00", asOf.AddDate(0, 0, -45)) // 31–60
	f.invoice(t, tenantA, paying, 3, "400.00", asOf.AddDate(0, 0, -75)) // 61–90
	partly := f.invoice(t, tenantA, paying, 4, "500.00", asOf.AddDate(0, 0, -120))
	require.NoError(t, f.db.WithContext(f.ctx).Exec(
		`INSERT INTO invoice_payments (invoice_id, amount, payment_date) VALUES (?, 50, '2026-10-20')`, partly).Error)
	require.NoError(t, f.db.WithContext(f.ctx).Exec(`UPDATE invoices SET amount_paid = 50 WHERE id = ?`, partly).Error)

	silent := f.customer(t, tenantA, "silent")
	f.invoice(t, tenantA, silent, 5, "75.00", asOf.AddDate(0, 0, -3))

	other := f.customer(t, tenantB, "other") // another tenant's debt stays out
	f.invoice(t, tenantB, other, 6, "999.00", asOf.AddDate(0, 0, -10))

	svc := NewARAgingService(f.db)
	got, err := svc.GetCustomerSummary(tenancy.WithTenant(context.Background(), tenantA), tenantA, &models.ARAgingQuery{AsOfDate: &asOf})
	require.NoError(t, err)
	require.Len(t, got, 2)

	byID := map[uuid.UUID]models.ARAgingCustomerSummary{}
	for _, s := range got {
		byID[s.CustomerID] = s
	}
	dec := decimal.RequireFromString

	p := byID[paying]
	assert.True(t, p.Current.Equal(dec("100")), "current = %s", p.Current)
	assert.True(t, p.Days1To30.Equal(dec("200")), "1–30 = %s", p.Days1To30)
	assert.True(t, p.Days31To60.Equal(dec("300")), "31–60 = %s", p.Days31To60)
	assert.True(t, p.Days61To90.Equal(dec("400")), "61–90 = %s", p.Days61To90)
	assert.True(t, p.Days90Plus.Equal(dec("450")), "90+ = %s", p.Days90Plus)
	assert.True(t, p.TotalOwed.Equal(dec("1450")), "total = %s", p.TotalOwed)
	require.NotNil(t, p.LastPayment)
	assert.Equal(t, "2026-10-20", p.LastPayment.Format("2006-01-02"))

	s := byID[silent]
	assert.True(t, s.Days1To30.Equal(dec("75")), "1–30 = %s", s.Days1To30)
	assert.Nil(t, s.LastPayment, "no payments → no last_payment")
}
