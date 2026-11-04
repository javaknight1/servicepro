package quickbooks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
	"github.com/javaknight1/servicepro/backend/pkg/database"
	"github.com/javaknight1/servicepro/backend/pkg/secretbox"
)

// Postgres-backed: needs SERVICEPRO_TEST_PG_URL (a server URL whose role can
// CREATE DATABASE); skipped otherwise.
func setupPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	base := os.Getenv("SERVICEPRO_TEST_PG_URL")
	if base == "" {
		t.Skip("SERVICEPRO_TEST_PG_URL not set; skipping Postgres-backed test")
	}
	admin, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	name := "sp_test_" + hex.EncodeToString(buf)
	require.NoError(t, admin.Exec("CREATE DATABASE "+name).Error)
	u, err := url.Parse(base)
	require.NoError(t, err)
	u.Path = "/" + name
	require.NoError(t, database.RunMigrations(u.String()))
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, tenancy.Register(db))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

type allowAll struct{ deny map[uuid.UUID]bool }

func (a allowAll) CheckPermission(_ context.Context, userID, _ uuid.UUID, _ string) (bool, error) {
	return !a.deny[userID], nil
}

type qbFixture struct {
	db    *gorm.DB
	fake  *qbo.Fake
	svc   *Service
	clock time.Time
	perms allowAll
}

type qbTenant struct {
	id, user, customer uuid.UUID
}

func newQBFixture(t *testing.T) *qbFixture {
	t.Helper()
	db := setupPostgres(t)
	box, err := secretbox.Random()
	require.NoError(t, err)
	f := &qbFixture{db: db, fake: qbo.NewFake("http://localhost:5173/api/v1/integrations/quickbooks/callback"),
		clock: time.Now(), perms: allowAll{deny: map[uuid.UUID]bool{}}}
	f.fake.Now = func() time.Time { return f.clock }
	f.svc = NewService(Config{DB: db, Client: f.fake, Box: box, Perms: f.perms, Now: func() time.Time { return f.clock }})
	return f
}

func (f *qbFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(context.Background())).Exec(sql, args...).Error)
}

func (f *qbFixture) tenant(t *testing.T, company string) *qbTenant {
	t.Helper()
	tn := &qbTenant{id: uuid.New(), user: uuid.New(), customer: uuid.New()}
	short := tn.id.String()[:8]
	f.exec(t, `INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`, tn.user, "owner-"+short+"@example.com")
	f.exec(t, `INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`, tn.id, "T "+short, "t-"+short, tn.user)
	f.exec(t, `INSERT INTO customers (id, tenant_id, first_name, last_name, company_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
		VALUES (?, ?, 'Dana', 'Reed', ?, ?, '555-555-5555', '1 Main St', 'Columbus', 'OH', '43215')`,
		tn.customer, tn.id, company, "dana-"+short+"@example.com")
	return tn
}

// invoice inserts a sent invoice with one taxable line (2 x 50.00, 8% tax).
func (f *qbFixture) invoice(t *testing.T, tn *qbTenant, number string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	today := f.clock.UTC().Format("2006-01-02")
	f.exec(t, `INSERT INTO invoices (id, tenant_id, customer_id, invoice_number, issue_date, due_date, status,
			subtotal, tax_amount, total_amount, amount_paid, created_by)
		VALUES (?, ?, ?, ?, ?, ?, 'sent', 100, 8, 108, 0, ?)`, id, tn.id, tn.customer, number, today, today, tn.user)
	f.exec(t, `INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, taxable, tax_rate, tax_amount)
		VALUES (?, 'Office cleaning', 2, 50, TRUE, 0.08, 8)`, id)
	return id
}

func (f *qbFixture) payment(t *testing.T, invoiceID uuid.UUID, amount string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	f.exec(t, `INSERT INTO invoice_payments (id, invoice_id, amount, payment_date, payment_method, reference_number)
		VALUES (?, ?, ?, CURRENT_DATE, 'check', '1042')`, id, invoiceID, amount)
	return id
}

// connect runs the whole OAuth flow through the fake consent page.
func (f *qbFixture) connect(t *testing.T, tn *qbTenant) {
	t.Helper()
	authURL, err := f.svc.StartConnect(context.Background(), tn.id, tn.user)
	require.NoError(t, err)
	u, err := url.Parse(authURL)
	require.NoError(t, err)
	q := u.Query()
	got, err := f.svc.CompleteConnect(context.Background(), q.Get("code"), q.Get("state"), q.Get("realmId"))
	require.NoError(t, err)
	require.Equal(t, tn.id, got)
}

func (f *qbFixture) conn(t *testing.T, tn *qbTenant) models.QuickBooksConnection {
	t.Helper()
	var c models.QuickBooksConnection
	require.NoError(t, f.db.WithContext(tenancy.WithTenant(context.Background(), tn.id)).Where("tenant_id = ?", tn.id).First(&c).Error)
	return c
}

func (f *qbFixture) sync(t *testing.T, tn *qbTenant) RunResult {
	t.Helper()
	r, err := f.svc.SyncNow(context.Background(), tn.id, time.Minute)
	require.NoError(t, err)
	return r
}

func TestPostgresQuickBooksOAuth(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Harborview Property Management")
	ctx := context.Background()

	authURL, err := f.svc.StartConnect(ctx, a.id, a.user)
	require.NoError(t, err)
	q := mustQuery(t, authURL)
	state, code, realm := q.Get("state"), q.Get("code"), q.Get("realmId")

	t.Run("tampered state is refused", func(t *testing.T) {
		_, err := f.svc.CompleteConnect(ctx, code, state[:len(state)-2]+"xx", realm)
		assert.ErrorIs(t, err, ErrInvalidState)
		payload, mac, _ := strings.Cut(state, ".")
		_, err = f.svc.CompleteConnect(ctx, code, payload+"A."+mac, realm)
		assert.ErrorIs(t, err, ErrInvalidState)
	})

	t.Run("callback stores sealed tokens, state is single use", func(t *testing.T) {
		got, err := f.svc.CompleteConnect(ctx, code, state, realm)
		require.NoError(t, err)
		assert.Equal(t, a.id, got)
		c := f.conn(t, a)
		assert.Equal(t, models.QuickBooksConnected, c.Status)
		assert.Equal(t, realm, deref(c.RealmID))
		assert.Equal(t, a.user, *c.ConnectedBy)
		assert.True(t, strings.HasPrefix(deref(c.AccessTokenEnc), "v1:"))
		assert.NotContains(t, deref(c.AccessTokenEnc), "access-", "tokens are never stored in plaintext")
		assert.NotContains(t, deref(c.RefreshTokenEnc), "refresh-")
		assert.Nil(t, c.OAuthStateHash)

		_, err = f.svc.CompleteConnect(ctx, f.fake.IssueCode(), state, realm)
		assert.ErrorIs(t, err, ErrInvalidState, "replayed state")
	})

	t.Run("expired state is refused", func(t *testing.T) {
		authURL, err := f.svc.StartConnect(ctx, a.id, a.user)
		require.NoError(t, err)
		q := mustQuery(t, authURL)
		f.clock = f.clock.Add(StateTTL + time.Second)
		defer func() { f.clock = f.clock.Add(-StateTTL - time.Second) }()
		_, err = f.svc.CompleteConnect(ctx, q.Get("code"), q.Get("state"), q.Get("realmId"))
		assert.ErrorIs(t, err, ErrInvalidState)
	})

	t.Run("a user who lost the permission can't finish", func(t *testing.T) {
		authURL, err := f.svc.StartConnect(ctx, a.id, a.user)
		require.NoError(t, err)
		q := mustQuery(t, authURL)
		f.perms.deny[a.user] = true
		defer delete(f.perms.deny, a.user)
		_, err = f.svc.CompleteConnect(ctx, q.Get("code"), q.Get("state"), q.Get("realmId"))
		assert.ErrorIs(t, err, ErrInvalidState)
	})

	t.Run("disconnect revokes and forgets the tokens", func(t *testing.T) {
		require.NoError(t, f.svc.Disconnect(ctx, a.id))
		c := f.conn(t, a)
		assert.Equal(t, models.QuickBooksDisconnected, c.Status)
		assert.Nil(t, c.AccessTokenEnc)
		assert.Nil(t, c.RefreshTokenEnc)
		assert.Len(t, f.fake.Revoked(), 1)
		assert.ErrorIs(t, f.svc.Disconnect(ctx, a.id), ErrNotConnected)
	})
}

func mustQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u.Query()
}

func TestPostgresQuickBooksTokenRefresh(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Acme")
	f.connect(t, a)
	ctx := context.Background()

	t.Run("a valid token is reused", func(t *testing.T) {
		_, err := f.svc.session(ctx, a.id, false, "")
		require.NoError(t, err)
		assert.Zero(t, f.fake.CallCount("RefreshTokens"))
	})

	t.Run("concurrent refreshes refresh once", func(t *testing.T) {
		f.clock = f.clock.Add(2 * time.Hour) // access token expired
		var wg sync.WaitGroup
		errs := make([]error, 5)
		tokens := make([]string, 5)
		for i := range errs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				s, err := f.svc.session(ctx, a.id, false, "")
				errs[i], tokens[i] = err, s.AccessToken
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			require.NoError(t, err)
			assert.Equal(t, tokens[0], tokens[i])
		}
		assert.Equal(t, 1, f.fake.CallCount("RefreshTokens"), "rotated refresh token used once")
		assert.Equal(t, models.QuickBooksConnected, f.conn(t, a).Status)
	})

	t.Run("a 401 forces one refresh and the sync carries on", func(t *testing.T) {
		f.fake.ExpireAccessTokens()
		f.invoice(t, a, "INV-401")
		r := f.sync(t, a)
		assert.Equal(t, 2, r.Synced, "customer + invoice")
		assert.Equal(t, 2, f.fake.CallCount("RefreshTokens"))
	})

	t.Run("a rejected refresh token asks the tenant to reconnect", func(t *testing.T) {
		f.clock = f.clock.Add(2 * time.Hour)
		f.fake.FailNext("RefreshTokens", qbo.ErrInvalidGrant)
		_, err := f.svc.session(ctx, a.id, false, "")
		assert.ErrorIs(t, err, ErrReauthRequired)
		c := f.conn(t, a)
		assert.Equal(t, models.QuickBooksReauthRequired, c.Status)
		assert.Contains(t, deref(c.LastSyncError), "Reconnect")
		_, err = f.svc.SyncNow(ctx, a.id, time.Minute)
		assert.ErrorIs(t, err, ErrReauthRequired)
	})
}

func TestPostgresQuickBooksSyncIsIdempotent(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Harborview Property Management")
	f.connect(t, a)
	inv := f.invoice(t, a, "INV-1042")
	pay := f.payment(t, inv, "50.00")
	ctx := context.Background()

	r := f.sync(t, a)
	assert.Equal(t, 3, r.Synced, "customer, invoice, payment")
	assert.Zero(t, r.Failed)

	customers, invoices, payments := f.fake.Snapshot(qbo.EntityKindCustomer), f.fake.Snapshot(qbo.EntityKindInvoice), f.fake.Snapshot(qbo.EntityKindPayment)
	require.Len(t, customers, 1)
	require.Len(t, invoices, 1)
	require.Len(t, payments, 1)
	assert.Equal(t, "Harborview Property Management", customers[0].Name)

	qInv := invoices[0].Payload.(qbo.Invoice)
	assert.Equal(t, "INV-1042", qInv.DocNumber)
	assert.Equal(t, customers[0].ID, qInv.CustomerRef.Value)
	require.Len(t, qInv.Line, 1)
	assert.Equal(t, "100.00", qInv.Line[0].Amount.String())
	assert.Equal(t, "1", qInv.Line[0].SalesItemLineDetail.ItemRef.Value, "the company's default Services item")
	assert.Equal(t, "TAX", qInv.Line[0].SalesItemLineDetail.TaxCodeRef.Value)
	assert.Equal(t, "8.00", qInv.TxnTaxDetail.TotalTax.String())

	qPay := payments[0].Payload.(qbo.Payment)
	assert.Equal(t, "50.00", qPay.TotalAmt.String())
	require.Len(t, qPay.Line, 1)
	assert.Equal(t, []qbo.LinkedTxn{{TxnID: invoices[0].ID, TxnType: "Invoice"}}, qPay.Line[0].LinkedTxn, "applied to the invoice")

	t.Run("re-running pushes nothing", func(t *testing.T) {
		before := f.fake.CallCount("CreateInvoice") + f.fake.CallCount("CreatePayment") + f.fake.CallCount("CreateCustomer")
		f.svc.EnqueueAfterCommit(ctx, a.id, models.QuickBooksEntityPayment, pay)
		f.svc.EnqueueAfterCommit(ctx, a.id, models.QuickBooksEntityInvoice, inv)
		r := f.sync(t, a)
		assert.Zero(t, r.Synced)
		after := f.fake.CallCount("CreateInvoice") + f.fake.CallCount("CreatePayment") + f.fake.CallCount("CreateCustomer")
		assert.Equal(t, before, after)
		assert.Len(t, f.fake.Snapshot(qbo.EntityKindPayment), 1, "payment applied once")
	})

	t.Run("an edit with the same content is skipped, a real change updates", func(t *testing.T) {
		f.exec(t, `UPDATE invoices SET notes = 'internal note' WHERE id = ?`, inv)
		r := f.sync(t, a)
		assert.Equal(t, 1, r.Skipped)
		assert.Zero(t, f.fake.CallCount("UpdateInvoice"))

		f.exec(t, `UPDATE invoices SET due_date = due_date + 30 WHERE id = ?`, inv)
		r = f.sync(t, a)
		assert.Equal(t, 1, r.Synced)
		assert.Equal(t, 1, f.fake.CallCount("UpdateInvoice"))
		assert.Len(t, f.fake.Snapshot(qbo.EntityKindInvoice), 1)
	})

	t.Run("an invoice edited in QuickBooks (stale SyncToken) is re-read and updated", func(t *testing.T) {
		f.fake.Bump(qbo.EntityKindInvoice, invoices[0].ID)
		f.exec(t, `UPDATE invoices SET due_date = due_date + 1 WHERE id = ?`, inv)
		r := f.sync(t, a)
		assert.Equal(t, 1, r.Synced)
		assert.Zero(t, r.Failed)
		assert.Equal(t, 1, f.fake.CallCount("Get"))
	})
}

func TestPostgresQuickBooksLostResponseDoesNotDuplicate(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Acme")
	f.connect(t, a)
	inv := f.invoice(t, a, "INV-7")
	f.payment(t, inv, "108.00")

	f.fake.LoseNextResponse("CreatePayment")
	f.fake.LoseNextResponse("CreateInvoice")
	r := f.sync(t, a)
	assert.Equal(t, 1, r.Synced, "the customer")
	assert.Equal(t, 2, r.Failed, "invoice (answer lost) and its payment")

	st, err := f.svc.Status(context.Background(), a.id)
	require.NoError(t, err)
	assert.Equal(t, int64(2), st.Counts.Pending, "outages stay pending, not errors")
	assert.Zero(t, st.Counts.Errors)

	f.clock = f.clock.Add(time.Hour) // past the backoff
	r = f.sync(t, a)
	assert.Equal(t, 1, r.Synced, "the invoice, found by DocNumber")
	assert.Equal(t, 1, r.Failed, "the payment: created, answer lost")
	f.clock = f.clock.Add(time.Hour)
	r = f.sync(t, a)
	assert.Equal(t, 1, r.Synced, "the payment, replayed by requestid")
	f.clock = f.clock.Add(time.Hour)
	r = f.sync(t, a)
	assert.Zero(t, r.Synced+r.Failed)
	assert.Equal(t, 2, f.fake.CallCount("CreatePayment"))

	assert.Len(t, f.fake.Snapshot(qbo.EntityKindInvoice), 1, "same requestid / DocNumber lookup: no duplicate invoice")
	assert.Len(t, f.fake.Snapshot(qbo.EntityKindPayment), 1, "same requestid: payment applied once")
}

func TestPostgresQuickBooksErrorsAndSettings(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Acme")
	f.connect(t, a)
	ctx := context.Background()
	// A company without the default "Services" item.
	f.fake.Items = map[string]*qbo.FakeRecord{}
	inv := f.invoice(t, a, "INV-9")

	r := f.sync(t, a)
	assert.Equal(t, 1, r.Synced, "customer")
	assert.Equal(t, 1, r.Failed)
	st, err := f.svc.Status(ctx, a.id)
	require.NoError(t, err)
	require.Len(t, st.Errors, 1)
	assert.Equal(t, inv, st.Errors[0].LocalID)
	assert.Contains(t, st.Errors[0].Label, "Invoice INV-9")
	assert.Contains(t, st.Errors[0].Label, "$108.00")
	assert.Contains(t, st.Errors[0].LastError, "income account")

	opts, err := f.svc.Options(ctx, a.id)
	require.NoError(t, err)
	require.NotEmpty(t, opts.IncomeAccounts)

	_, err = f.svc.UpdateSettings(ctx, a.id, models.QuickBooksSettingsRequest{IncomeAccountID: "nope"})
	assert.ErrorIs(t, err, ErrInvalidSetting)

	st, err = f.svc.UpdateSettings(ctx, a.id, models.QuickBooksSettingsRequest{IncomeAccountID: "79", TaxCodeID: "3", DepositAccountID: "35"})
	require.NoError(t, err)
	assert.Equal(t, "Cleaning Services", st.IncomeAccount.Name)
	assert.Equal(t, "ServicePro - Cleaning Services", st.Item.Name)
	assert.Equal(t, "OH 8.0%", st.TaxCode.Name)
	assert.Empty(t, st.Errors, "failed records are retried after a settings change")
	assert.Equal(t, int64(1), st.Counts.Pending)

	r = f.sync(t, a)
	assert.Equal(t, 1, r.Synced)
	invs := f.fake.Snapshot(qbo.EntityKindInvoice)
	require.Len(t, invs, 1)
	q := invs[0].Payload.(qbo.Invoice)
	assert.Equal(t, st.Item.ID, q.Line[0].SalesItemLineDetail.ItemRef.Value)
	assert.Equal(t, "3", q.TxnTaxDetail.TxnTaxCodeRef.Value)

	// Saving again reuses the item.
	_, err = f.svc.UpdateSettings(ctx, a.id, models.QuickBooksSettingsRequest{IncomeAccountID: "79"})
	require.NoError(t, err)
	assert.Equal(t, 1, f.fake.CallCount("CreateItem"))
}

func TestPostgresQuickBooksCrossTenant(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Tenant A Customer")
	b := f.tenant(t, "Tenant B Customer")
	f.connect(t, a)
	f.connect(t, b)
	ctx := context.Background()
	invA := f.invoice(t, a, "A-1")
	f.invoice(t, b, "B-1")

	// B enqueues A's invoice ID (a forged or buggy event): B's sync must
	// not find or push it.
	require.NoError(t, f.svc.Enqueue(ctx, b.id, models.QuickBooksEntityInvoice, invA))
	r := f.sync(t, b)
	assert.Equal(t, 2, r.Synced, "B's customer and invoice only")
	for _, inv := range f.fake.Snapshot(qbo.EntityKindInvoice) {
		assert.NotEqual(t, "A-1", inv.Name)
	}

	f.sync(t, a)
	stA, err := f.svc.Status(ctx, a.id)
	require.NoError(t, err)
	stB, err := f.svc.Status(ctx, b.id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stA.Counts.Invoices)
	assert.Equal(t, int64(1), stB.Counts.Invoices)
	for _, rec := range stB.Recent {
		assert.NotContains(t, rec.Label, "Tenant A")
	}

	// B can't retry A's records.
	var recA models.QuickBooksSyncRecord
	require.NoError(t, f.db.WithContext(tenancy.WithTenant(ctx, a.id)).Where("tenant_id = ?", a.id).First(&recA).Error)
	f.exec(t, `UPDATE quickbooks_sync_map SET status = 'error' WHERE id = ?`, recA.ID)
	assert.ErrorIs(t, f.svc.Retry(ctx, b.id, recA.ID), ErrRecordNotFound)
	require.NoError(t, f.svc.Retry(ctx, a.id, recA.ID))

	// A state minted for A can't be re-pointed at B (signed).
	authURL, err := f.svc.StartConnect(ctx, a.id, a.user)
	require.NoError(t, err)
	q := mustQuery(t, authURL)
	tenant, ok := f.svc.StateTenant(q.Get("state"))
	assert.True(t, ok)
	assert.Equal(t, a.id, tenant)
	payload, mac, _ := strings.Cut(q.Get("state"), ".")
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	require.NoError(t, err)
	forged := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(string(raw), a.id.String(), b.id.String(), 1))) + "." + mac
	_, err = f.svc.CompleteConnect(ctx, q.Get("code"), forged, q.Get("realmId"))
	assert.ErrorIs(t, err, ErrInvalidState)

	// The job syncs every connected tenant, each into its own company.
	f.invoice(t, a, "A-2")
	f.invoice(t, b, "B-2")
	res, err := f.svc.ProcessPending(ctx, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Tenants)
	assert.Equal(t, 2, res.Synced)
}

func TestPostgresQuickBooksEnqueueOnlyWhenConnected(t *testing.T) {
	f := newQBFixture(t)
	a := f.tenant(t, "Acme")
	ctx := context.Background()
	inv := f.invoice(t, a, "INV-1")
	require.NoError(t, f.svc.Enqueue(ctx, a.id, models.QuickBooksEntityInvoice, inv))
	var n int64
	require.NoError(t, f.db.WithContext(tenancy.WithTenant(ctx, a.id)).Model(&models.QuickBooksSyncRecord{}).Count(&n).Error)
	assert.Zero(t, n, "no connection, nothing queued")

	f.connect(t, a)
	require.NoError(t, f.svc.Enqueue(ctx, a.id, models.QuickBooksEntityInvoice, inv))
	require.NoError(t, f.db.WithContext(tenancy.WithTenant(ctx, a.id)).Model(&models.QuickBooksSyncRecord{}).Count(&n).Error)
	assert.Equal(t, int64(1), n)
}

func TestNotConfigured(t *testing.T) {
	svc := NewService(Config{})
	assert.False(t, svc.Configured())
	st, err := svc.Status(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, st.Configured)
	_, err = svc.StartConnect(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotConfigured)
	require.NoError(t, svc.Enqueue(context.Background(), uuid.New(), "invoice", uuid.New()))
	res, err := svc.ProcessPending(context.Background(), time.Second)
	require.NoError(t, err)
	assert.Zero(t, res.Tenants)
}
