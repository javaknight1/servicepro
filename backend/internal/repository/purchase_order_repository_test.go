package repository

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T187: every purchase-order query is tenant-scoped; another tenant's PO is
// "not found" and its invoices never count towards a balance.
func TestPurchaseOrderRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "customer_purchase_orders")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA, custB := createTestCustomer(t, db, a), createTestCustomer(t, db, b)
	repo := NewPurchaseOrderRepository(db)

	po := &models.CustomerPurchaseOrder{CustomerID: custA, PONumber: "PO-7731", Amount: decimal.NewFromInt(1000)}
	// The tenant always comes from the caller: a TenantID on the value is overwritten.
	po.TenantID = b.ID
	require.NoError(t, repo.Create(a.Ctx, a.ID, po))
	assert.Equal(t, a.ID, po.TenantID)

	got, err := repo.GetForCustomer(a.Ctx, a.ID, custA, po.ID)
	require.NoError(t, err)
	assert.Equal(t, "PO-7731", got.PONumber)

	// Tenant B sees nothing and can't change or delete it.
	_, err = repo.GetForCustomer(b.Ctx, b.ID, custA, po.ID)
	require.ErrorIs(t, err, ErrPurchaseOrderNotFound)
	_, err = repo.GetForCustomer(b.Ctx, b.ID, custB, po.ID)
	require.ErrorIs(t, err, ErrPurchaseOrderNotFound)
	list, err := repo.ListByCustomer(b.Ctx, b.ID, custA)
	require.NoError(t, err)
	assert.Empty(t, list)
	require.ErrorIs(t, repo.Update(b.Ctx, b.ID, &models.CustomerPurchaseOrder{ID: po.ID, CustomerID: custA, PONumber: "pwned", Amount: decimal.NewFromInt(1)}), ErrPurchaseOrderNotFound)
	require.ErrorIs(t, repo.Delete(b.Ctx, b.ID, custA, po.ID), ErrPurchaseOrderNotFound)
	bal, err := repo.Balances(b.Ctx, b.ID, custA, []string{"PO-7731"})
	require.NoError(t, err)
	assert.Empty(t, bal)
	locked, err := LockPurchaseOrder(db.WithContext(b.Ctx), b.ID, custA, "PO-7731")
	require.NoError(t, err)
	assert.Nil(t, locked)

	// Under the wrong customer of the right tenant it is not found either.
	_, err = repo.GetForCustomer(a.Ctx, a.ID, createTestCustomer(t, db, a), po.ID)
	require.ErrorIs(t, err, ErrPurchaseOrderNotFound)

	got.Amount = decimal.NewFromInt(1500)
	require.NoError(t, repo.Update(a.Ctx, a.ID, got))
	list, err = repo.ListByCustomer(a.Ctx, a.ID, custA)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "1500.00", list[0].Amount.StringFixed(2))
	require.NoError(t, repo.Delete(a.Ctx, a.ID, custA, po.ID))
	_, err = repo.GetForCustomer(a.Ctx, a.ID, custA, po.ID)
	require.ErrorIs(t, err, ErrPurchaseOrderNotFound)
}
