package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Purchase orders with an authorised amount (T187). An invoice is matched to
// a PO by (customer, po_number). Billed = the customer's issued invoices
// (not draft, not cancelled) with that number; remaining = amount - billed.
var (
	// ErrPurchaseOrderNotFound: no such PO under that customer in the
	// caller's tenant.
	ErrPurchaseOrderNotFound = apperrors.NotFound("purchase_order_not_found", "Purchase order not found")
	// ErrPurchaseOrderExists: the customer already has a PO with that number.
	ErrPurchaseOrderExists = apperrors.Conflict("purchase_order_exists", "This customer already has a purchase order with that number")
	// ErrPOExceeded: sending the invoice would bill more than the PO's
	// remaining amount. Sending again with confirm_po_override goes ahead.
	ErrPOExceeded = apperrors.Conflict("po_exceeded", "This invoice is more than the purchase order's remaining balance")
	// ErrPOExpired: the invoice's PO has expired. Sending again with
	// confirm_po_override goes ahead.
	ErrPOExpired = apperrors.Conflict("po_expired", "The purchase order on this invoice has expired")
)

// maxPOAmount bounds a PO's amount (numeric(12,2)).
var maxPOAmount = decimal.New(1, 10)

// PurchaseOrderServiceInterface is the customer-PO use case the handler
// calls. The tenant always comes from ctx.
type PurchaseOrderServiceInterface interface {
	List(ctx context.Context, customerID uuid.UUID) ([]models.PurchaseOrderResponse, error)
	Create(ctx context.Context, customerID uuid.UUID, req *models.CreatePurchaseOrderRequest, userID uuid.UUID) (*models.PurchaseOrderResponse, error)
	Update(ctx context.Context, customerID, poID uuid.UUID, req *models.UpdatePurchaseOrderRequest) (*models.PurchaseOrderResponse, error)
	Delete(ctx context.Context, customerID, poID uuid.UUID) error
}

// PurchaseOrderService manages customer purchase orders.
type PurchaseOrderService struct {
	db  *gorm.DB
	pos repository.PurchaseOrderRepositoryInterface
}

// NewPurchaseOrderService creates the service.
func NewPurchaseOrderService(db *gorm.DB, pos repository.PurchaseOrderRepositoryInterface) *PurchaseOrderService {
	return &PurchaseOrderService{db: db, pos: pos}
}

func purchaseOrderError(err error) error {
	switch {
	case errors.Is(err, repository.ErrPurchaseOrderNotFound):
		return ErrPurchaseOrderNotFound
	case uniqueViolation(err):
		return ErrPurchaseOrderExists
	}
	return err
}

// tenantToday is the tenant-local calendar date (midnight UTC).
func tenantToday(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (time.Time, error) {
	loc, _, err := tenantLocation(tenancy.WithSystem(ctx), db, tenantID)
	if err != nil {
		return time.Time{}, err
	}
	return models.CivilDate(time.Now(), loc), nil
}

// List returns the customer's POs with their balances.
func (s *PurchaseOrderService) List(ctx context.Context, customerID uuid.UUID) ([]models.PurchaseOrderResponse, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	pos, err := s.pos.ListByCustomer(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	numbers := make([]string, len(pos))
	for i := range pos {
		numbers[i] = pos[i].PONumber
	}
	balances, err := s.pos.Balances(ctx, tenantID, customerID, numbers)
	if err != nil {
		return nil, err
	}
	today, err := tenantToday(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]models.PurchaseOrderResponse, len(pos))
	for i := range pos {
		out[i] = pos[i].ToResponse(balances[pos[i].PONumber], today)
	}
	return out, nil
}

// get returns one PO with its balance.
func (s *PurchaseOrderService) get(ctx context.Context, tenantID, customerID, poID uuid.UUID) (*models.PurchaseOrderResponse, error) {
	po, err := s.pos.GetForCustomer(ctx, tenantID, customerID, poID)
	if err != nil {
		return nil, purchaseOrderError(err)
	}
	balances, err := s.pos.Balances(ctx, tenantID, customerID, []string{po.PONumber})
	if err != nil {
		return nil, err
	}
	today, err := tenantToday(ctx, s.db, tenantID)
	if err != nil {
		return nil, err
	}
	resp := po.ToResponse(balances[po.PONumber], today)
	return &resp, nil
}

// validatePOAmount checks a PO amount and rounds it to cents.
func validatePOAmount(amount decimal.Decimal, fields map[string]string) decimal.Decimal {
	switch {
	case !amount.IsPositive():
		fields["amount"] = "must be more than zero"
	case amount.GreaterThanOrEqual(maxPOAmount):
		fields["amount"] = "is too large"
	}
	return amount.Round(2)
}

// parsePOExpiry parses an optional YYYY-MM-DD expiry ("" means none).
func parsePOExpiry(v string, fields map[string]string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	d, err := time.Parse(models.DateLayout, v)
	if err != nil {
		fields["expires_on"] = "must be a date (YYYY-MM-DD)"
		return nil
	}
	return &d
}

// Create records a PO for the customer.
func (s *PurchaseOrderService) Create(ctx context.Context, customerID uuid.UUID, req *models.CreatePurchaseOrderRequest, userID uuid.UUID) (*models.PurchaseOrderResponse, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	po := &models.CustomerPurchaseOrder{
		CustomerID: customerID,
		PONumber:   strings.TrimSpace(req.PONumber),
		Amount:     validatePOAmount(req.Amount, fields),
		Notes:      models.CleanPONumber(req.Notes),
		CreatedBy:  &userID,
	}
	if po.PONumber == "" {
		fields["po_number"] = "is required"
	}
	if req.ExpiresOn != nil {
		po.ExpiresOn = parsePOExpiry(*req.ExpiresOn, fields)
	}
	if len(fields) > 0 {
		return nil, apperrors.Validation("Invalid purchase order", fields)
	}
	if err := s.pos.Create(ctx, tenantID, po); err != nil {
		return nil, purchaseOrderError(err)
	}
	return s.get(ctx, tenantID, customerID, po.ID)
}

// Update changes the fields present in req.
func (s *PurchaseOrderService) Update(ctx context.Context, customerID, poID uuid.UUID, req *models.UpdatePurchaseOrderRequest) (*models.PurchaseOrderResponse, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	po, err := s.pos.GetForCustomer(ctx, tenantID, customerID, poID)
	if err != nil {
		return nil, purchaseOrderError(err)
	}
	fields := map[string]string{}
	if req.PONumber != nil {
		po.PONumber = strings.TrimSpace(*req.PONumber)
		if po.PONumber == "" {
			fields["po_number"] = "is required"
		}
	}
	if req.Amount != nil {
		po.Amount = validatePOAmount(*req.Amount, fields)
	}
	switch {
	case req.ClearExpiresOn:
		po.ExpiresOn = nil
	case req.ExpiresOn != nil:
		po.ExpiresOn = parsePOExpiry(*req.ExpiresOn, fields)
	}
	if req.Notes != nil {
		po.Notes = models.CleanPONumber(req.Notes)
	}
	if len(fields) > 0 {
		return nil, apperrors.Validation("Invalid purchase order", fields)
	}
	if err := s.pos.Update(ctx, tenantID, po); err != nil {
		return nil, purchaseOrderError(err)
	}
	return s.get(ctx, tenantID, customerID, po.ID)
}

// Delete removes one of the customer's POs.
func (s *PurchaseOrderService) Delete(ctx context.Context, customerID, poID uuid.UUID) error {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return err
	}
	return purchaseOrderError(s.pos.Delete(ctx, tenantID, customerID, poID))
}

// invoicePurchaseOrder is the tracked PO an invoice is billed under, with
// its balance (this invoice included once it is issued), or nil when
// the invoice has no PO number or the customer has no PO with that number.
func invoicePurchaseOrder(ctx context.Context, db *gorm.DB, invoice *models.Invoice) (*models.PurchaseOrderResponse, error) {
	number := strings.TrimSpace(invoice.PONumber)
	if number == "" {
		return nil, nil
	}
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var pos []models.CustomerPurchaseOrder
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND po_number = ?", tenantID, invoice.CustomerID, number).
		Limit(1).Find(&pos).Error; err != nil {
		return nil, fmt.Errorf("load purchase order: %w", err)
	}
	if len(pos) == 0 {
		return nil, nil
	}
	balances, err := repository.POBalances(db.WithContext(ctx), tenantID, invoice.CustomerID, []string{number}, nil)
	if err != nil {
		return nil, fmt.Errorf("load purchase order balance: %w", err)
	}
	today, err := tenantToday(ctx, db, tenantID)
	if err != nil {
		return nil, err
	}
	resp := pos[0].ToResponse(balances[number], today)
	return &resp, nil
}

// checkPOForSend is the PO guard on sending a draft (T187), inside the send
// transaction: when the invoice's PO number is a tracked PO of the
// customer, the PO row is locked (concurrent sends against one PO
// serialise) and sending fails with ErrPOExpired when the PO has expired,
// or ErrPOExceeded when the customer's other issued invoices on the
// PO plus this one come to more than its amount. Untracked PO numbers pass.
func checkPOForSend(ctx context.Context, tx *gorm.DB, invoice *models.Invoice) error {
	number := strings.TrimSpace(invoice.PONumber)
	if number == "" {
		return nil
	}
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return err
	}
	po, err := repository.LockPurchaseOrder(tx, tenantID, invoice.CustomerID, number)
	if err != nil {
		return fmt.Errorf("lock purchase order: %w", err)
	}
	if po == nil {
		return nil
	}
	today, err := tenantToday(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	if po.ExpiresOn != nil && po.ExpiresOn.UTC().Before(today) {
		return ErrPOExpired.WithMessage(fmt.Sprintf("Purchase order %s expired on %s", po.PONumber, po.ExpiresOn.UTC().Format(models.DateLayout)))
	}
	others, err := repository.POBalances(tx, tenantID, invoice.CustomerID, []string{number}, &invoice.ID)
	if err != nil {
		return fmt.Errorf("load purchase order balance: %w", err)
	}
	remaining := po.Amount.Sub(others[number].Billed)
	if invoice.TotalAmount.GreaterThan(remaining) {
		return ErrPOExceeded.WithMessage(fmt.Sprintf("This invoice (%s) is more than the %s remaining on purchase order %s",
			invoice.TotalAmount.StringFixed(2), remaining.StringFixed(2), po.PONumber))
	}
	return nil
}
