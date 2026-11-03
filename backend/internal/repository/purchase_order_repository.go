package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ErrPurchaseOrderNotFound: no such PO under that customer in the caller's
// tenant. Other tenants' POs look exactly like missing ones.
var ErrPurchaseOrderNotFound = errors.New("purchase order not found")

// PurchaseOrderRepositoryInterface is data access for customer purchase
// orders (T187). Every method takes ctx (which carries the tenant for the
// tenancy plugin and the RLS backstop) and the tenant ID, and filters
// tenant_id explicitly too.
type PurchaseOrderRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, po *models.CustomerPurchaseOrder) error
	GetForCustomer(ctx context.Context, tenantID, customerID, id uuid.UUID) (*models.CustomerPurchaseOrder, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]models.CustomerPurchaseOrder, error)
	// Update saves the editable columns of po (matched by tenant, customer
	// and ID).
	Update(ctx context.Context, tenantID uuid.UUID, po *models.CustomerPurchaseOrder) error
	Delete(ctx context.Context, tenantID, customerID, id uuid.UUID) error
	// Balances sums the customer's issued invoices (sent, not cancelled)
	// per PO number, for the given numbers.
	Balances(ctx context.Context, tenantID, customerID uuid.UUID, poNumbers []string) (map[string]models.POBalance, error)
}

// PurchaseOrderRepository implements PurchaseOrderRepositoryInterface with GORM.
type PurchaseOrderRepository struct {
	db *gorm.DB
}

// NewPurchaseOrderRepository creates a purchase-order repository.
func NewPurchaseOrderRepository(db *gorm.DB) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{db: db}
}

// Create inserts po into tenantID (the tenant always comes from the
// caller, never from the value).
func (r *PurchaseOrderRepository) Create(ctx context.Context, tenantID uuid.UUID, po *models.CustomerPurchaseOrder) error {
	po.TenantID = tenantID
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(po).Error
}

// GetForCustomer returns one of the customer's POs.
func (r *PurchaseOrderRepository) GetForCustomer(ctx context.Context, tenantID, customerID, id uuid.UUID) (*models.CustomerPurchaseOrder, error) {
	var po models.CustomerPurchaseOrder
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id).
		First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPurchaseOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &po, nil
}

// ListByCustomer returns the customer's POs, newest first.
func (r *PurchaseOrderRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]models.CustomerPurchaseOrder, error) {
	pos := []models.CustomerPurchaseOrder{}
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("created_at DESC, id").Find(&pos).Error; err != nil {
		return nil, err
	}
	return pos, nil
}

// purchaseOrderColumns are the columns Update writes (Select writes nil too).
var purchaseOrderColumns = []string{"po_number", "amount", "expires_on", "notes", "updated_at"}

// Update saves the PO's editable columns.
func (r *PurchaseOrderRepository) Update(ctx context.Context, tenantID uuid.UUID, po *models.CustomerPurchaseOrder) error {
	cp := *po
	cp.TenantID = tenantID
	res := r.db.WithContext(ctx).Model(&models.CustomerPurchaseOrder{}).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, po.CustomerID, po.ID).
		Select(purchaseOrderColumns).Updates(&cp)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPurchaseOrderNotFound
	}
	return nil
}

// Delete removes one of the customer's POs. Invoices keep their PO number;
// they are just no longer tracked against an amount.
func (r *PurchaseOrderRepository) Delete(ctx context.Context, tenantID, customerID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id).
		Delete(&models.CustomerPurchaseOrder{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPurchaseOrderNotFound
	}
	return nil
}

// Balances sums the customer's issued invoices per PO number.
func (r *PurchaseOrderRepository) Balances(ctx context.Context, tenantID, customerID uuid.UUID, poNumbers []string) (map[string]models.POBalance, error) {
	return POBalances(r.db.WithContext(ctx), tenantID, customerID, poNumbers, nil)
}

// POBalances sums the customer's issued invoices (neither draft nor
// cancelled, not deleted) per PO number, leaving out excludeInvoice when
// given. Drafts don't count until they are sent, so a stale draft never
// uses up a PO. db carries the tenant
// context (a request db or a transaction).
func POBalances(db *gorm.DB, tenantID, customerID uuid.UUID, poNumbers []string, excludeInvoice *uuid.UUID) (map[string]models.POBalance, error) {
	out := make(map[string]models.POBalance, len(poNumbers))
	if len(poNumbers) == 0 {
		return out, nil
	}
	q := db.Model(&models.Invoice{}).
		Select("po_number, COALESCE(SUM(total_amount), 0) AS billed, COUNT(*) AS invoice_count").
		Where("tenant_id = ? AND customer_id = ? AND po_number IN ? AND status NOT IN ?",
			tenantID, customerID, poNumbers, []models.InvoiceStatus{models.InvoiceStatusDraft, models.InvoiceStatusCancelled})
	if excludeInvoice != nil {
		q = q.Where("id <> ?", *excludeInvoice)
	}
	var rows []struct {
		PONumber     string
		Billed       decimal.Decimal
		InvoiceCount int
	}
	if err := q.Group("po_number").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PONumber] = models.POBalance{Billed: row.Billed, InvoiceCount: row.InvoiceCount}
	}
	return out, nil
}

// LockPurchaseOrder returns the customer's PO with poNumber, locked FOR
// UPDATE inside tx (so concurrent sends against one PO serialise), or nil
// when the customer has no such PO.
func LockPurchaseOrder(tx *gorm.DB, tenantID, customerID uuid.UUID, poNumber string) (*models.CustomerPurchaseOrder, error) {
	var pos []models.CustomerPurchaseOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND customer_id = ? AND po_number = ?", tenantID, customerID, poNumber).
		Limit(1).Find(&pos).Error; err != nil {
		return nil, err
	}
	if len(pos) == 0 {
		return nil, nil
	}
	return &pos[0], nil
}
