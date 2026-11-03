package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// fakeConnect is an in-memory ConnectAPI.
type fakeConnect struct {
	creates   []stripeClient.CreateAccountInput
	links     []string // account IDs
	retrieves int
	status    map[string]*stripeClient.AccountStatus
	err       error
}

func (f *fakeConnect) CreateAccount(_ context.Context, in stripeClient.CreateAccountInput) (*stripeClient.AccountStatus, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.creates = append(f.creates, in)
	id := "acct_" + in.TenantID[:8]
	st := &stripeClient.AccountStatus{AccountID: id, CardPaymentsStatus: "pending", ACHPaymentsStatus: "pending", RequirementsStatus: "currently_due"}
	if f.status == nil {
		f.status = map[string]*stripeClient.AccountStatus{}
	}
	f.status[id] = st
	return st, nil
}

func (f *fakeConnect) CreateOnboardingLink(_ context.Context, accountID, returnURL, refreshURL, key string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.links = append(f.links, accountID)
	return "https://connect.stripe.test/setup/" + accountID + "?return=" + returnURL + "&refresh=" + refreshURL + "&key=" + key, nil
}

func (f *fakeConnect) RetrieveAccount(_ context.Context, accountID string) (*stripeClient.AccountStatus, error) {
	f.retrieves++
	if f.err != nil {
		return nil, f.err
	}
	st, ok := f.status[accountID]
	if !ok {
		return nil, errors.New("no such account")
	}
	return st, nil
}

type connectFixture struct {
	*isolationFixture
	api      *fakeConnect
	svc      *PaymentsConnectService
	tenantA  uuid.UUID
	tenantB  uuid.UUID
	accounts *repository.ConnectedAccountRepository
}

func setupConnect(t *testing.T, paymentsEnabled bool) *connectFixture {
	t.Helper()
	f := setupIsolation(t)
	createSQLiteTableForModel(t, f.db, &models.StripeConnectedAccount{}, "")
	require.NoError(t, f.db.Exec(`CREATE UNIQUE INDEX idx_sca_tenant ON stripe_connected_accounts (tenant_id)`).Error)
	a, _ := tenancy.TenantID(f.ctxA)
	b, _ := tenancy.TenantID(f.ctxB)
	for _, id := range []uuid.UUID{a, b} {
		require.NoError(t, f.db.Create(&models.Tenant{ID: id, Name: "Brightline " + id.String()[:4], Slug: "t-" + id.String()[:8],
			OwnerID: f.userA, Email: "billing@" + id.String()[:4] + ".test"}).Error)
	}
	api := &fakeConnect{}
	repo := repository.NewConnectedAccountRepository(f.db)
	svc := NewPaymentsConnectService(PaymentsConnectServiceConfig{
		DB: f.db, Repo: repo, API: api, FrontendURL: "https://app.test", PaymentsEnabled: paymentsEnabled,
	})
	return &connectFixture{isolationFixture: f, api: api, svc: svc, tenantA: a, tenantB: b, accounts: repo}
}

func (c *connectFixture) activate(id string, ach bool) {
	st := c.api.status[id]
	st.CardPaymentsStatus = models.CapabilityActive
	st.PayoutsStatus = models.CapabilityActive
	st.RequirementsStatus = ""
	if ach {
		st.ACHPaymentsStatus = models.CapabilityActive
	}
}

// T169: the first click creates the tenant's Stripe account (idempotency key
// per tenant); later clicks only make a fresh onboarding link.
func TestPaymentsConnect_StartOnboardingCreatesAccountOnce(t *testing.T) {
	c := setupConnect(t, true)

	url, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Contains(t, url, "return=https://app.test/settings/payments?onboarding=return")
	_, err = c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)

	require.Len(t, c.api.creates, 1, "one Stripe account per tenant")
	in := c.api.creates[0]
	assert.Equal(t, "connect-account:"+c.tenantA.String(), in.IdempotencyKey)
	assert.Equal(t, c.tenantA.String(), in.TenantID)
	assert.Contains(t, in.DisplayName, "Brightline")
	assert.Len(t, c.api.links, 2)

	acct, err := c.accounts.GetByTenant(c.ctxA, c.tenantA)
	require.NoError(t, err)
	require.NotNil(t, acct)
	assert.Equal(t, "acct_"+c.tenantA.String()[:8], acct.StripeAccountID)
}

func TestPaymentsConnect_ConnectNotEnabledIsTyped(t *testing.T) {
	c := setupConnect(t, true)
	c.api.err = errors.Join(errors.New("create account"), stripeClient.ErrConnectNotEnabled)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	assert.ErrorIs(t, err, ErrConnectUnavailable)

	noStripe := NewPaymentsConnectService(PaymentsConnectServiceConfig{DB: c.db, Repo: c.accounts})
	_, err = noStripe.StartOnboarding(c.ctxA, c.tenantA)
	assert.ErrorIs(t, err, ErrPaymentsNotConfigured)
}

// Status re-reads Stripe while onboarding is unfinished, then serves the cache.
func TestPaymentsConnect_StatusRefreshesUntilActive(t *testing.T) {
	c := setupConnect(t, true)
	st, err := c.svc.Status(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Equal(t, models.PaymentsStateNotConnected, st.State)
	assert.Equal(t, "0.4", st.Fees.CardPercent)
	assert.Equal(t, "1", st.Fees.ACHPercent)
	assert.Equal(t, "10.00", st.Fees.ACHCap)

	_, err = c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctID := "acct_" + c.tenantA.String()[:8]

	st, err = c.svc.Status(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Equal(t, models.PaymentsStateOnboarding, st.State)
	assert.False(t, st.PayLinksActive)

	c.activate(acctID, true)
	st, err = c.svc.Status(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Equal(t, models.PaymentsStateActive, st.State)
	assert.True(t, st.PayLinksActive)
	assert.Equal(t, models.CapabilityActive, st.ACHPaymentsStatus)

	before := c.api.retrieves
	_, err = c.svc.Status(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Equal(t, before, c.api.retrieves, "an active account is served from the cache")
}

// T169 cross-tenant: webhooks update only the account's own tenant, and
// tenant B never sees tenant A's account.
func TestPaymentsConnect_SyncAccountIsTenantScoped(t *testing.T) {
	c := setupConnect(t, true)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctID := "acct_" + c.tenantA.String()[:8]
	c.activate(acctID, false)

	require.NoError(t, c.svc.SyncAccount(context.Background(), acctID))
	require.NoError(t, c.svc.SyncAccount(context.Background(), "acct_not_ours"), "unknown accounts are ignored")

	assert.True(t, c.svc.CanAcceptPayments(context.Background(), c.tenantA))
	assert.False(t, c.svc.CanAcceptPayments(context.Background(), c.tenantB))

	stB, err := c.svc.Status(c.ctxB, c.tenantB)
	require.NoError(t, err)
	assert.False(t, stB.Connected, "B has no account of its own")

	tenant, ok, err := c.svc.TenantForStripeAccount(context.Background(), acctID)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, c.tenantA, tenant)
}

// The master switch (T261) still wins over an active account.
func TestPaymentsConnect_MasterSwitchOff(t *testing.T) {
	c := setupConnect(t, false)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctID := "acct_" + c.tenantA.String()[:8]
	c.activate(acctID, true)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctID))

	assert.False(t, c.svc.CanAcceptPayments(context.Background(), c.tenantA))
	st, err := c.svc.Status(c.ctxA, c.tenantA)
	require.NoError(t, err)
	assert.Equal(t, models.PaymentsStateActive, st.State)
	assert.False(t, st.PayLinksActive)
}

// account.updated on the Connect endpoint re-syncs the event's account.
func TestConnectWebhook_AccountUpdatedSyncs(t *testing.T) {
	var synced []string
	p := stripeClient.NewEventProcessor()
	RegisterConnectWebhookHandlers(p, syncFunc(func(_ context.Context, id string) error {
		synced = append(synced, id)
		return nil
	}))
	require.NoError(t, p.ProcessEvent(context.Background(), &stripe.Event{
		ID: "evt_a", Type: "account.updated", Account: "acct_1", Data: &stripe.EventData{Raw: json.RawMessage(`{"id":"acct_1"}`)},
	}))
	assert.Equal(t, []string{"acct_1"}, synced)
}

type syncFunc func(context.Context, string) error

func (f syncFunc) SyncAccount(ctx context.Context, id string) error { return f(ctx, id) }

// --- direct-charge checkout (T169) ---

// public is the context the public pay route runs in (token-authenticated).
var public = tenancy.WithSystem(context.Background())

func newPaymentService(t *testing.T, c *connectFixture, enabled bool) (*InvoicePaymentService, *[]*stripe.CheckoutSessionParams) {
	t.Helper()
	var sessions []*stripe.CheckoutSessionParams
	svc := &InvoicePaymentService{
		invoiceService: NewInvoiceService(c.db),
		accounts:       c.svc,
		successURL:     "https://app.test/ok",
		cancelURL:      "https://app.test/cancel",
		enabled:        enabled,
		newSession: func(p *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
			sessions = append(sessions, p)
			return &stripe.CheckoutSession{ID: "cs_test_1", URL: "https://checkout.stripe.test/cs_test_1"}, nil
		},
	}
	return svc, &sessions
}

func payableInvoice(t *testing.T, c *connectFixture, total int64) (*models.Invoice, string) {
	t.Helper()
	inv := newSentInvoice(t, c.isolationFixture, total, models.InvoiceStatusSent)
	token := uuid.NewString()
	require.NoError(t, c.db.WithContext(c.ctxA).Model(&models.Invoice{}).Where("id = ?", inv.ID).
		Updates(map[string]any{"payment_token": token, "payment_token_expires_at": time.Now().Add(time.Hour)}).Error)
	return inv, token
}

func TestDirectCharge_NoActiveAccountNoSession(t *testing.T) {
	c := setupConnect(t, true)
	pay, sessions := newPaymentService(t, c, true)
	_, token := payableInvoice(t, c, 100)

	_, err := pay.CreateCheckoutSessionForInvoice(public, token, "")
	assert.ErrorIs(t, err, ErrInvoicePaymentDisabled, "no connected account")

	_, err = c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	_, err = pay.CreateCheckoutSessionForInvoice(public, token, "")
	assert.ErrorIs(t, err, ErrInvoicePaymentDisabled, "account still onboarding")
	assert.Empty(t, *sessions)
}

func TestDirectCharge_BankAndCardFees(t *testing.T) {
	c := setupConnect(t, true)
	pay, sessions := newPaymentService(t, c, true)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctID := "acct_" + c.tenantA.String()[:8]
	c.activate(acctID, false)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctID))
	inv, token := payableInvoice(t, c, 5000) // $5,000.00

	_, err = pay.CreateCheckoutSessionForInvoice(public, token, "bank")
	assert.ErrorIs(t, err, ErrPaymentMethodUnavailable, "ACH not active yet")
	_, err = pay.CreateCheckoutSessionForInvoice(public, token, "crypto")
	assert.ErrorIs(t, err, ErrInvalidPaymentMethod)

	res, err := pay.CreateCheckoutSessionForInvoice(public, token, "")
	require.NoError(t, err)
	assert.Equal(t, "card", res.Method, "card when ACH isn't active")
	assert.Equal(t, int64(2000), res.ApplicationFee, "0.4% of $5,000")

	c.activate(acctID, true)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctID))
	res, err = pay.CreateCheckoutSessionForInvoice(public, token, "")
	require.NoError(t, err)
	assert.Equal(t, "bank", res.Method, "bank is the default once ACH is active")
	assert.Equal(t, int64(1000), res.ApplicationFee, "$10 on a $5,000 bank payment")

	require.Len(t, *sessions, 2)
	bank := (*sessions)[1]
	require.NotNil(t, bank.StripeAccount)
	assert.Equal(t, acctID, *bank.StripeAccount, "direct charge on the tenant's own account")
	assert.Equal(t, []string{"us_bank_account"}, strVals(bank.AllowedPaymentMethodTypes))
	assert.Equal(t, int64(1000), *bank.PaymentIntentData.ApplicationFeeAmount)
	assert.Equal(t, inv.ID.String(), bank.Metadata["invoice_id"])
	assert.Equal(t, c.tenantA.String(), bank.Metadata["tenant_id"])
	assert.Contains(t, *bank.IdempotencyKey, acctID)

	card := (*sessions)[0]
	assert.Equal(t, []string{"us_bank_account"}, strVals(card.ExcludedPaymentMethodTypes))
	assert.NotEqual(t, *card.IdempotencyKey, *bank.IdempotencyKey)
}

// --- webhooks from connected accounts (T169) ---

func checkoutEvent(t *testing.T, id, typ, account string, inv *models.Invoice, status string) *stripe.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"id": "cs_" + inv.ID.String()[:8], "object": "checkout.session", "payment_status": status, "status": "complete",
		"amount_total": 10000, "currency": "usd", "payment_intent": "pi_" + inv.ID.String()[:8],
		"metadata": map[string]string{"invoice_id": inv.ID.String(), "payment_type": "invoice"},
	})
	require.NoError(t, err)
	return &stripe.Event{ID: id, Type: stripe.EventType(typ), Account: account, Data: &stripe.EventData{Raw: raw}}
}

func TestConnectedCheckoutWebhook_RecordsAgainstTheAccountsTenantOnce(t *testing.T) {
	c := setupConnect(t, true)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctA := "acct_" + c.tenantA.String()[:8]
	_, err = c.svc.StartOnboarding(c.ctxB, c.tenantB)
	require.NoError(t, err)
	acctB := "acct_" + c.tenantB.String()[:8]

	invSvc := NewInvoiceService(c.db)
	adapter := NewInvoicePaymentHandlerAdapter(invSvc)
	adapter.SetConnectedAccounts(c.svc)
	handle := adapter.CreateCheckoutCompletedHandler()
	inv := newSentInvoice(t, c.isolationFixture, 100, models.InvoiceStatusSent)

	// Tenant B's account claiming tenant A's invoice: ignored, nothing recorded.
	require.NoError(t, handle(context.Background(), checkoutEvent(t, "evt_b", "checkout.session.completed", acctB, inv, "paid")))
	// An account that isn't ours: ignored.
	require.NoError(t, handle(context.Background(), checkoutEvent(t, "evt_x", "checkout.session.completed", "acct_stranger", inv, "paid")))
	assert.True(t, reloadInvoice(t, c.isolationFixture, inv.ID).AmountPaid.IsZero())

	// ACH: completed while unpaid records nothing; async success records once,
	// and a replayed completion can't credit again.
	require.NoError(t, handle(context.Background(), checkoutEvent(t, "evt_1", "checkout.session.completed", acctA, inv, "unpaid")))
	assert.True(t, reloadInvoice(t, c.isolationFixture, inv.ID).AmountPaid.IsZero())
	require.NoError(t, handle(context.Background(), checkoutEvent(t, "evt_2", "checkout.session.async_payment_succeeded", acctA, inv, "paid")))
	require.NoError(t, handle(context.Background(), checkoutEvent(t, "evt_3", "checkout.session.completed", acctA, inv, "paid")))

	got := reloadInvoice(t, c.isolationFixture, inv.ID)
	assertDec(t, "100", got.AmountPaid, "credited exactly once")
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
}

func strVals(ps []*string) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, *p)
	}
	return out
}
