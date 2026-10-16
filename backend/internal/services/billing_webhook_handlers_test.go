package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

type fakeBillingStore struct {
	updated, deleted []string
	recorded         []*models.BillingEvent
	tenants          map[string]uuid.UUID
	lookupErr        error
	sawSystemCtx     bool
}

func (f *fakeBillingStore) HandleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error {
	f.sawSystemCtx = tenancy.IsSystem(ctx)
	f.updated = append(f.updated, sub.ID)
	return nil
}

func (f *fakeBillingStore) HandleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error {
	f.sawSystemCtx = tenancy.IsSystem(ctx)
	f.deleted = append(f.deleted, sub.ID)
	return nil
}

func (f *fakeBillingStore) TenantIDForStripeCustomer(_ context.Context, id string) (uuid.UUID, error) {
	if f.lookupErr != nil {
		return uuid.Nil, f.lookupErr
	}
	t, ok := f.tenants[id]
	if !ok {
		return uuid.Nil, repository.ErrTenantNotFound
	}
	return t, nil
}

func (f *fakeBillingStore) RecordBillingEvent(_ context.Context, tenantID uuid.UUID, e *models.BillingEvent) error {
	e.TenantID = tenantID
	f.recorded = append(f.recorded, e)
	return nil
}

func billingEvent(id, typ, raw string) *stripe.Event {
	return &stripe.Event{ID: id, Type: stripe.EventType(typ), Data: &stripe.EventData{Raw: []byte(raw)}}
}

// Regression for T122: subscription and invoice events used to go to the
// legacy /webhooks/stripe handler, which panicked on a context type assertion.
// They are now handled by the processor behind /stripe/webhooks.
func TestBillingWebhookHandlers_SubscriptionEvents(t *testing.T) {
	store := &fakeBillingStore{}
	p := stripeClient.NewEventProcessor()
	RegisterBillingWebhookHandlers(p, store)

	ctx := context.Background()
	for _, typ := range []string{stripeClient.EventSubscriptionCreated, stripeClient.EventSubscriptionUpdated} {
		if err := p.ProcessEvent(ctx, billingEvent("evt_"+typ, typ, `{"id":"sub_1","status":"active"}`)); err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
	}
	if err := p.ProcessEvent(ctx, billingEvent("evt_del", stripeClient.EventSubscriptionDeleted, `{"id":"sub_1","status":"canceled"}`)); err != nil {
		t.Fatalf("deleted: %v", err)
	}
	if len(store.updated) != 2 || len(store.deleted) != 1 || store.deleted[0] != "sub_1" {
		t.Fatalf("updated=%v deleted=%v", store.updated, store.deleted)
	}
	if !store.sawSystemCtx {
		t.Fatal("handlers must run with a system tenancy context")
	}
}

func TestBillingWebhookHandlers_InvoiceEvents(t *testing.T) {
	tenantID := uuid.New()
	store := &fakeBillingStore{tenants: map[string]uuid.UUID{"cus_1": tenantID}}
	p := stripeClient.NewEventProcessor()
	RegisterBillingWebhookHandlers(p, store)
	ctx := context.Background()

	if err := p.ProcessEvent(ctx, billingEvent("evt_paid", stripeClient.EventInvoicePaid,
		`{"id":"in_1","customer":"cus_1","amount_paid":4900,"amount_due":4900,"currency":"usd","hosted_invoice_url":"https://x"}`)); err != nil {
		t.Fatal(err)
	}
	if err := p.ProcessEvent(ctx, billingEvent("evt_failed", stripeClient.EventInvoicePaymentFailed,
		`{"id":"in_2","customer":"cus_1","amount_paid":0,"amount_due":9900,"currency":"usd"}`)); err != nil {
		t.Fatal(err)
	}
	// Unknown customer is acknowledged, not retried.
	if err := p.ProcessEvent(ctx, billingEvent("evt_other", stripeClient.EventInvoicePaid, `{"id":"in_3","customer":"cus_unknown"}`)); err != nil {
		t.Fatalf("unknown customer should be acknowledged: %v", err)
	}

	if len(store.recorded) != 2 {
		t.Fatalf("recorded %d events, want 2", len(store.recorded))
	}
	paid, failed := store.recorded[0], store.recorded[1]
	if paid.TenantID != tenantID || paid.Status != models.BillingEventStatusSucceeded || *paid.AmountCents != 4900 || paid.StripeEventID != "evt_paid" {
		t.Fatalf("paid event = %+v", paid)
	}
	if failed.Status != models.BillingEventStatusFailed || *failed.AmountCents != 9900 || failed.StripeInvoiceURL != nil {
		t.Fatalf("failed event = %+v", failed)
	}
}

func TestBillingWebhookHandlers_TransientLookupErrorIsRetried(t *testing.T) {
	store := &fakeBillingStore{lookupErr: errors.New("db down")}
	p := stripeClient.NewEventProcessor()
	RegisterBillingWebhookHandlers(p, store)
	if err := p.ProcessEvent(context.Background(), billingEvent("evt", stripeClient.EventInvoicePaid, `{"id":"in_1","customer":"cus_1"}`)); err == nil {
		t.Fatal("transient errors must surface so Stripe retries")
	}
}
