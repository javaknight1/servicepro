package services

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T187 on real Postgres (migration 000019): customer purchase orders with an
// amount, balances across issued invoices, and the send-time guard.

func newPOService(db *gorm.DB) *PurchaseOrderService {
	return NewPurchaseOrderService(db, repository.NewPurchaseOrderRepository(db))
}

// poDraft creates an untaxed draft for the tenant's customer under po.
func poDraft(t *testing.T, svc *InvoiceService, p *pgTenant, po, amount string) *models.Invoice {
	t.Helper()
	inv, err := svc.CreateInvoice(p.ctx, &models.Invoice{
		CustomerID: p.customerID, PONumber: po,
		Lines: []models.InvoiceLine{{Description: "Cleaning", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString(amount)}},
	}, p.userID)
	require.NoError(t, err)
	return inv
}

// poSent creates a draft under po and sends it (no PO override).
func poSent(t *testing.T, svc *InvoiceService, p *pgTenant, po, amount string) *models.Invoice {
	t.Helper()
	inv := poDraft(t, svc, p, po, amount)
	sent, err := svc.SendInvoice(p.ctx, inv.ID)
	require.NoError(t, err)
	return sent
}

func TestPostgres_PurchaseOrders_CRUDAndBalance(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	pos := newPOService(db)
	invoices := NewInvoiceService(db)

	expires := "2099-06-30"
	created, err := pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{
		PONumber: " PO-7731 ", Amount: decimal.RequireFromString("1000.004"), ExpiresOn: &expires,
	}, p.userID)
	require.NoError(t, err)
	assert.Equal(t, "PO-7731", created.PONumber)
	assert.Equal(t, "1000.00", created.Amount.StringFixed(2))
	assert.Equal(t, "1000.00", created.Remaining.StringFixed(2))
	require.NotNil(t, created.ExpiresOn)
	assert.Equal(t, "2099-06-30", *created.ExpiresOn)
	assert.False(t, created.Expired)

	// Same number again: 409.
	_, err = pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-7731", Amount: decimal.NewFromInt(5)}, p.userID)
	require.ErrorIs(t, err, ErrPurchaseOrderExists)

	// Validation.
	_, err = pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: " ", Amount: decimal.Zero}, p.userID)
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, 400, de.Status)
	assert.Contains(t, de.Fields, "po_number")
	assert.Contains(t, de.Fields, "amount")

	// Issued invoices with the PO count; drafts, cancelled invoices and
	// other POs don't.
	first := poSent(t, invoices, p, "PO-7731", "400.00")
	poSent(t, invoices, p, "PO-7731", "250.00")
	cancelled := poSent(t, invoices, p, "PO-7731", "100.00")
	_, err = invoices.CancelInvoice(p.ctx, cancelled.ID, "wrong amount")
	require.NoError(t, err)
	draft := poDraft(t, invoices, p, "PO-7731", "75.00")
	poSent(t, invoices, p, "PO-OTHER", "999.00")

	list, err := pos.List(p.ctx, p.customerID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "650.00", list[0].Billed.StringFixed(2))
	assert.Equal(t, "350.00", list[0].Remaining.StringFixed(2))
	assert.Equal(t, 2, list[0].InvoiceCount)

	// GET /invoices/{id} shows the PO's balance (drafts too); an
	// invoice without a tracked PO shows none.
	got, err := invoices.GetInvoice(p.ctx, first.ID)
	require.NoError(t, err)
	require.NotNil(t, got.PurchaseOrder)
	assert.Equal(t, "350.00", got.PurchaseOrder.Remaining.StringFixed(2))
	got, err = invoices.GetInvoice(p.ctx, draft.ID)
	require.NoError(t, err)
	require.NotNil(t, got.PurchaseOrder)
	assert.Equal(t, "PO-7731", got.PurchaseOrder.PONumber)

	// Update: amount down, expiry in the past, then cleared.
	past := "2020-01-31"
	amount := decimal.NewFromInt(600)
	updated, err := pos.Update(p.ctx, p.customerID, created.ID, &models.UpdatePurchaseOrderRequest{Amount: &amount, ExpiresOn: &past})
	require.NoError(t, err)
	assert.Equal(t, "-50.00", updated.Remaining.StringFixed(2))
	assert.True(t, updated.Expired)
	updated, err = pos.Update(p.ctx, p.customerID, created.ID, &models.UpdatePurchaseOrderRequest{ClearExpiresOn: true})
	require.NoError(t, err)
	assert.Nil(t, updated.ExpiresOn)
	assert.False(t, updated.Expired)

	require.NoError(t, pos.Delete(p.ctx, p.customerID, created.ID))
	list, err = pos.List(p.ctx, p.customerID)
	require.NoError(t, err)
	assert.Empty(t, list)
	got, err = invoices.GetInvoice(p.ctx, first.ID)
	require.NoError(t, err)
	assert.Nil(t, got.PurchaseOrder)
	assert.Equal(t, "PO-7731", got.PONumber, "invoices keep their PO number")
}

func TestPostgres_PurchaseOrders_SendGuard(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	pos := newPOService(db)
	invoices := NewInvoiceService(db)
	_, err := pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-1", Amount: decimal.NewFromInt(1000)}, p.userID)
	require.NoError(t, err)

	// 600 fits; the next 500 would make 1100 > 1000.
	poSent(t, invoices, p, "PO-1", "600.00")
	b := poDraft(t, invoices, p, "PO-1", "500.00")
	_, err = invoices.SendInvoice(p.ctx, b.ID)
	require.ErrorIs(t, err, ErrPOExceeded)
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, de.Status)
	assert.Equal(t, "po_exceeded", de.Code)
	assert.Contains(t, de.Message, "400.00 remaining")
	got, err := invoices.GetInvoice(p.ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, got.Status, "nothing sent")

	// The office confirms: it goes out.
	sent, err := invoices.SendInvoiceWithOptions(p.ctx, b.ID, SendInvoiceOptions{ConfirmPOOverride: true})
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, sent.Status)

	// Exactly the remaining amount is fine.
	_, err = pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-2", Amount: decimal.NewFromInt(300)}, p.userID)
	require.NoError(t, err)
	poSent(t, invoices, p, "PO-2", "300.00")

	// An untracked PO number is never blocked.
	poSent(t, invoices, p, "PO-UNTRACKED", "5000.00")

	// An expired PO warns too, and can be overridden.
	past := "2020-01-31"
	_, err = pos.Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-OLD", Amount: decimal.NewFromInt(9000), ExpiresOn: &past}, p.userID)
	require.NoError(t, err)
	old := poDraft(t, invoices, p, "PO-OLD", "10.00")
	_, err = invoices.SendInvoice(p.ctx, old.ID)
	require.ErrorIs(t, err, ErrPOExpired)
	_, err = invoices.SendInvoiceWithOptions(p.ctx, old.ID, SendInvoiceOptions{ConfirmPOOverride: true})
	require.NoError(t, err)
}

// Two drafts that each fit but not together, sent at once: the PO row lock
// serialises the sends, so exactly one goes out.
func TestPostgres_PurchaseOrders_ConcurrentSends(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	invoices := NewInvoiceService(db)
	_, err := newPOService(db).Create(p.ctx, p.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-RACE", Amount: decimal.NewFromInt(1000)}, p.userID)
	require.NoError(t, err)
	x := poDraft(t, invoices, p, "PO-RACE", "600.00")
	y := poDraft(t, invoices, p, "PO-RACE", "600.00")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []uuid.UUID{x.ID, y.ID} {
		wg.Add(1)
		go func(i int, id uuid.UUID) {
			defer wg.Done()
			_, errs[i] = invoices.SendInvoice(p.ctx, id)
		}(i, id)
	}
	wg.Wait()
	ok, exceeded := 0, 0
	for _, e := range errs {
		switch {
		case e == nil:
			ok++
		case assert.ErrorIs(t, e, ErrPOExceeded):
			exceeded++
		}
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, 1, exceeded)
}

func TestPostgres_PurchaseOrders_CrossTenant(t *testing.T) {
	db := setupPostgres(t)
	a, b := seedPGTenant(t, db), seedPGTenant(t, db)
	pos := newPOService(db)
	invoices := NewInvoiceService(db)
	created, err := pos.Create(a.ctx, a.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-A", Amount: decimal.NewFromInt(100)}, a.userID)
	require.NoError(t, err)

	// Tenant B can't reach A's customer or PO.
	_, err = pos.List(b.ctx, a.customerID)
	require.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = pos.Create(b.ctx, a.customerID, &models.CreatePurchaseOrderRequest{PONumber: "PO-B", Amount: decimal.NewFromInt(1)}, b.userID)
	require.ErrorIs(t, err, ErrCustomerNotFound)
	amount := decimal.NewFromInt(1)
	_, err = pos.Update(b.ctx, b.customerID, created.ID, &models.UpdatePurchaseOrderRequest{Amount: &amount})
	require.ErrorIs(t, err, ErrPurchaseOrderNotFound)
	require.ErrorIs(t, pos.Delete(b.ctx, b.customerID, created.ID), ErrPurchaseOrderNotFound)

	// B's invoice with the same PO number neither shows A's PO nor is
	// guarded by it, and never counts towards A's balance.
	invB := poSent(t, invoices, b, "PO-A", "5000.00")
	got, err := invoices.GetInvoice(b.ctx, invB.ID)
	require.NoError(t, err)
	assert.Nil(t, got.PurchaseOrder)

	list, err := pos.List(a.ctx, a.customerID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "0.00", list[0].Billed.StringFixed(2))
}
