package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/internal/utils"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	// PaymentTokenExpiry is the duration a payment token is valid (30 days)
	PaymentTokenExpiry = 30 * 24 * time.Hour
	// PaymentTokenLength is the number of random bytes for the token
	PaymentTokenLength = 32
)

var (
	// ErrInvoiceNotFound is returned when an invoice is not found
	ErrInvoiceNotFound = errors.New("invoice not found")
	// ErrInvalidInvoiceData is returned when invoice data is invalid
	ErrInvalidInvoiceData = errors.New("invalid invoice data")
	// ErrInvoiceAlreadyPaid is returned when trying to modify a paid invoice
	ErrInvoiceAlreadyPaid = errors.New("cannot modify paid invoice")
	// ErrInvalidPaymentToken is returned when a payment token is invalid
	ErrInvalidPaymentToken = errors.New("invalid payment token")
	// ErrPaymentTokenExpired is returned when a payment token has expired
	ErrPaymentTokenExpired = errors.New("payment token has expired")
	// ErrPaymentTokenVoided is returned when a payment token has been voided
	ErrPaymentTokenVoided = errors.New("payment token has been voided")
	// ErrInvoiceNotPayable is returned when invoice is not in a payable status
	ErrInvoiceNotPayable = errors.New("invoice is not in a payable status")
	// ErrPaymentNotAllowed is returned when recording a payment on a draft,
	// cancelled or refunded invoice
	ErrPaymentNotAllowed = errors.New("payments can only be recorded on sent, viewed, overdue or partially paid invoices")
	// ErrOverpayment is returned when a payment exceeds the balance due
	ErrOverpayment = errors.New("payment amount exceeds outstanding balance")
	// ErrInvoiceHasPayments is returned when deleting or cancelling an invoice
	// that has payments recorded against it
	ErrInvoiceHasPayments = errors.New("invoice has recorded payments; refund them first")
	// ErrAlreadyInvoiced is returned when the job or quote already has an invoice
	ErrAlreadyInvoiced = errors.New("this job or quote has already been invoiced")
	// ErrInvoiceSourceNotFound is returned when the job or quote doesn't exist (in this tenant)
	ErrInvoiceSourceNotFound = errors.New("job or quote not found")
	// ErrInvoiceSourceCustomer is returned when the job or quote is for another customer
	ErrInvoiceSourceCustomer = errors.New("job or quote belongs to a different customer")
	// ErrInvoiceSourceStatus is returned when invoicing a cancelled job
	ErrInvoiceSourceStatus = errors.New("cannot invoice a cancelled job")
	// ErrQuoteNotAccepted is returned when invoicing a quote that wasn't accepted
	ErrQuoteNotAccepted = errors.New("only accepted quotes can be invoiced")
	// ErrInvoiceClosed is returned when editing a paid, cancelled or refunded invoice
	ErrInvoiceClosed = errors.New("paid, cancelled and refunded invoices can't be edited")
	// ErrInvalidInvoiceTransition is returned for a status change the invoice state machine forbids
	ErrInvalidInvoiceTransition = errors.New("this invoice status change is not allowed")
	// ErrTaxRateNotFound is returned for an unknown tax_rate_id
	ErrTaxRateNotFound = errors.New("tax rate not found")
)

// ErrReceiptNotAvailable is returned when asking for the receipt of an
// invoice that isn't paid (T223: a typed 409 instead of a generic 400).
var ErrReceiptNotAvailable = apperrors.Conflict("receipt_not_available", "A receipt is only available for paid invoices")

// InvoiceService handles invoice business logic
type InvoiceService struct {
	db          *gorm.DB
	emailClient email.Client
	pdfService  *DocumentPDFService
	frontendURL string
	backendURL  string
}

// InvoiceServiceConfig holds configuration for the invoice service
type InvoiceServiceConfig struct {
	EmailClient email.Client
	PDFService  *DocumentPDFService
	FrontendURL string
	BackendURL  string
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(db *gorm.DB) *InvoiceService {
	return &InvoiceService{db: db}
}

// NewInvoiceServiceWithConfig creates a new invoice service with configuration
func NewInvoiceServiceWithConfig(db *gorm.DB, cfg *InvoiceServiceConfig) *InvoiceService {
	svc := &InvoiceService{db: db}
	if cfg != nil {
		svc.emailClient = cfg.EmailClient
		svc.pdfService = cfg.PDFService
		svc.frontendURL = cfg.FrontendURL
		svc.backendURL = cfg.BackendURL
	}
	return svc
}

// CreateInvoice creates a new invoice with business logic
func (s *InvoiceService) CreateInvoice(ctx context.Context, invoice *models.Invoice, userID uuid.UUID) (*models.Invoice, error) {
	// Set metadata
	invoice.CreatedBy = userID
	invoice.Status = models.InvoiceStatusDraft

	// Set default dates if not provided
	if invoice.IssueDate.IsZero() {
		invoice.IssueDate = time.Now()
	}

	if err := s.applyTenantInvoiceDefaults(ctx, invoice); err != nil {
		return nil, err
	}

	// Calculate due date based on payment terms
	if invoice.DueDate.IsZero() {
		dueDate, err := s.calculateDueDate(ctx, invoice.IssueDate, invoice.PaymentTermID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate due date: %w", err)
		}
		invoice.DueDate = dueDate
	}

	// The invoice number is allocated from the tenant's counter inside the
	// create transaction (models.Invoice.BeforeCreate).

	// Validate invoice
	if err := s.validateInvoice(invoice); err != nil {
		return nil, err
	}

	// The customer must belong to the caller's tenant (scoped lookup).
	if err := s.requireCustomer(ctx, invoice.CustomerID); err != nil {
		return nil, err
	}

	// Create invoice in transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A job or quote is invoiced at most once (audit 7.M5). Locking the
		// source row serialises concurrent creates for the same job/quote.
		if err := guardInvoiceSource(tx, invoice); err != nil {
			return err
		}

		// Totals are always computed here, never taken from the client.
		rate, err := resolveTaxRate(tx, invoice.ApplyTaxRate, invoice.TaxRateID, nil)
		if err != nil {
			return err
		}
		totals, err := calculateInvoiceTotals(invoice.Lines, invoice.DiscountAmount, rate)
		if err != nil {
			return err
		}
		setInvoiceTotals(invoice, totals)
		for i := range invoice.Lines {
			if invoice.Lines[i].ID == uuid.Nil {
				invoice.Lines[i].ID = uuid.New()
			}
		}

		if err := tx.Create(invoice).Error; err != nil {
			return fmt.Errorf("failed to create invoice: %w", err)
		}
		if err := moveInvoiceJob(tx, invoice, models.JobStatusCompleted, models.JobStatusInvoiced,
			fmt.Sprintf("Invoice %s created", invoice.InvoiceNumber), userID); err != nil {
			return err
		}

		// Load relationships
		if err := tx.Preload("Lines").First(invoice, invoice.ID).Error; err != nil {
			return fmt.Errorf("failed to load invoice: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return invoice, nil
}

// GetInvoice retrieves an invoice by ID
func (s *InvoiceService) GetInvoice(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	var invoice models.Invoice

	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Lines", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("Payments", func(db *gorm.DB) *gorm.DB {
			return db.Order("payment_date DESC")
		}).
		Preload("Refunds", func(db *gorm.DB) *gorm.DB {
			return db.Order("refund_date DESC")
		}).
		Preload("PaymentTerm").
		Preload("TaxRate").
		First(&invoice, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	return &invoice, nil
}

// GetInvoiceByNumber retrieves an invoice by invoice number
func (s *InvoiceService) GetInvoiceByNumber(ctx context.Context, invoiceNumber string) (*models.Invoice, error) {
	var invoice models.Invoice

	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Lines", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("Payments").
		Preload("PaymentTerm").
		Preload("TaxRate").
		First(&invoice, "invoice_number = ?", invoiceNumber).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	return &invoice, nil
}

// InvoiceUpdate is a partial invoice update: nil fields are left unchanged.
// Status is deliberately absent — it changes only through send, payment,
// cancel and refund.
type InvoiceUpdate struct {
	DueDate            *time.Time
	PaymentTermID      *uuid.UUID
	TaxRateID          *uuid.UUID
	TaxRate            *decimal.Decimal // direct rate (0.0825 = 8.25%); overrides TaxRateID's rate
	PONumber           *string
	Notes              *string
	TermsAndConditions *string
	DiscountAmount     *decimal.Decimal
	Lines              []models.InvoiceLine // nil = keep the current lines
}

// UpdateInvoice applies a partial update and recomputes the totals inside
// the same transaction, whatever changed (T225 / audit 7.H3): a discount or
// tax-rate change alone updates total_amount, and sending lines without a
// tax rate keeps the invoice's current rate instead of zeroing the tax.
func (s *InvoiceService) UpdateInvoice(ctx context.Context, id uuid.UUID, upd *InvoiceUpdate, userID uuid.UUID) (*models.Invoice, error) {
	if upd == nil {
		upd = &InvoiceUpdate{}
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}
		if existing.Status == models.InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		if !existing.Status.IsEditable() {
			return ErrInvoiceClosed
		}

		fields := map[string]interface{}{"updated_by": userID}
		if upd.DueDate != nil {
			if upd.DueDate.Before(existing.IssueDate) {
				return invalidInvoice("due_date must be after issue_date")
			}
			fields["due_date"] = *upd.DueDate
		}
		if upd.PaymentTermID != nil {
			fields["payment_term_id"] = *upd.PaymentTermID
		}
		if upd.TaxRateID != nil {
			fields["tax_rate_id"] = *upd.TaxRateID
		}
		if upd.PONumber != nil {
			fields["po_number"] = *upd.PONumber
		}
		if upd.Notes != nil {
			fields["notes"] = *upd.Notes
		}
		if upd.TermsAndConditions != nil {
			fields["terms_and_conditions"] = *upd.TermsAndConditions
		}
		discount := existing.DiscountAmount
		if upd.DiscountAmount != nil {
			discount = *upd.DiscountAmount
		}

		var current []models.InvoiceLine
		if err := tx.Where("invoice_id = ?", id).Order("sort_order ASC").Find(&current).Error; err != nil {
			return err
		}

		// Rate: the request's direct rate or new tax_rate_id; otherwise keep
		// what the invoice already uses (its lines' rate, else its tax_rate_id).
		var rate decimal.Decimal
		switch {
		case upd.TaxRate != nil || upd.TaxRateID != nil:
			rate, err = resolveTaxRate(tx, upd.TaxRate, upd.TaxRateID, nil)
		case len(current) > 0:
			rate, err = resolveTaxRate(tx, nil, nil, current)
		default:
			rate, err = resolveTaxRate(tx, nil, existing.TaxRateID, nil)
		}
		if err != nil {
			return err
		}

		lines := current
		replaceLines := upd.Lines != nil
		if replaceLines {
			lines = upd.Lines
		}
		totals, err := calculateInvoiceTotals(lines, discount, rate)
		if err != nil {
			return err
		}
		if totals.Total.LessThan(existing.AmountPaid) {
			return invalidInvoice("invoice total cannot be less than the amount already paid")
		}
		fields["subtotal"] = totals.Subtotal
		fields["discount_amount"] = totals.Discount
		fields["tax_amount"] = totals.Tax
		fields["total_amount"] = totals.Total

		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(fields).Error; err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		if !replaceLines {
			return saveLineTaxes(tx, lines)
		}
		if err := tx.Where("invoice_id = ?", id).Delete(&models.InvoiceLine{}).Error; err != nil {
			return fmt.Errorf("failed to delete old line items: %w", err)
		}
		for i := range lines {
			lines[i].ID = uuid.New()
			lines[i].InvoiceID = id
			if err := tx.Create(&lines[i]).Error; err != nil {
				return fmt.Errorf("failed to create line item: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetInvoice(ctx, id)
}

// saveLineTaxes writes back the per-line fields calculateInvoiceTotals fills.
func saveLineTaxes(tx *gorm.DB, lines []models.InvoiceLine) error {
	for _, l := range lines {
		if err := tx.Model(&models.InvoiceLine{}).Where("id = ? AND invoice_id = ?", l.ID, l.InvoiceID).Updates(map[string]interface{}{
			"discount_amount": l.DiscountAmount,
			"tax_rate":        l.TaxRate,
			"tax_amount":      l.TaxAmount,
		}).Error; err != nil {
			return fmt.Errorf("failed to update line item: %w", err)
		}
	}
	return nil
}

// setInvoiceTotals copies calculated totals onto an invoice.
func setInvoiceTotals(invoice *models.Invoice, t invoiceTotals) {
	invoice.Subtotal = t.Subtotal
	invoice.DiscountAmount = t.Discount
	invoice.TaxAmount = t.Tax
	invoice.TotalAmount = t.Total
}

// resolveTaxRate picks the rate for an invoice: a direct rate, else the
// configured tax rate (global or the caller's tenant — the tenancy plugin
// hides other tenants' rates), else the rate already on the invoice's lines,
// else zero.
func resolveTaxRate(tx *gorm.DB, direct *decimal.Decimal, taxRateID *uuid.UUID, currentLines []models.InvoiceLine) (decimal.Decimal, error) {
	if direct != nil {
		return *direct, nil
	}
	if taxRateID != nil {
		var tr models.TaxRate
		if err := tx.First(&tr, "id = ?", *taxRateID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return decimal.Zero, ErrTaxRateNotFound
			}
			return decimal.Zero, err
		}
		return tr.Rate, nil
	}
	for _, l := range currentLines {
		if l.TaxRate.IsPositive() {
			return l.TaxRate, nil
		}
	}
	return decimal.Zero, nil
}

// applyTenantInvoiceDefaults fills a new invoice's payment term and tax rate
// from the tenant's settings when the request gave neither (T127). The
// defaults are looked up within the caller's tenant (models.DefaultPaymentTerm
// / DefaultTaxRate), so a stale or foreign ID in the settings is ignored.
func (s *InvoiceService) applyTenantInvoiceDefaults(ctx context.Context, invoice *models.Invoice) error {
	needTerm := invoice.PaymentTermID == nil
	needTax := invoice.TaxRateID == nil && invoice.ApplyTaxRate == nil
	if !needTerm && !needTax {
		return nil
	}
	tenantID, ok := tenancy.TenantID(ctx)
	if !ok {
		return nil
	}
	db := s.db.WithContext(ctx)
	settings := models.TenantSettingsInTx(db, tenantID)
	if needTerm {
		term, err := models.DefaultPaymentTerm(db, tenantID, settings)
		if err != nil {
			return fmt.Errorf("load default payment term: %w", err)
		}
		if term != nil {
			invoice.PaymentTermID = &term.ID
		}
	}
	if needTax {
		rate, err := models.DefaultTaxRate(db, tenantID, settings)
		if err != nil {
			return fmt.Errorf("load default tax rate: %w", err)
		}
		if rate != nil {
			invoice.TaxRateID = &rate.ID
		}
	}
	return nil
}

// guardInvoiceSource locks the job and/or quote an invoice is created from
// and refuses a second invoice for it (audit 7.M5): a quote must be accepted
// and for the same customer; a job must belong to the customer, not be
// cancelled and not already be invoiced. Cancelled invoices don't count.
func guardInvoiceSource(tx *gorm.DB, invoice *models.Invoice) error {
	if invoice.JobID != nil {
		var job models.Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", *invoice.JobID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceSourceNotFound
			}
			return err
		}
		if job.CustomerID != invoice.CustomerID {
			return ErrInvoiceSourceCustomer
		}
		switch job.Status {
		case models.JobStatusInvoiced, models.JobStatusPaid:
			return ErrAlreadyInvoiced
		case models.JobStatusCancelled:
			return ErrInvoiceSourceStatus
		}
		if err := requireNoActiveInvoice(tx, "job_id", *invoice.JobID); err != nil {
			return err
		}
	}
	if invoice.QuoteID != nil {
		var quote models.Quote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&quote, "id = ?", *invoice.QuoteID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceSourceNotFound
			}
			return err
		}
		if quote.CustomerID != invoice.CustomerID {
			return ErrInvoiceSourceCustomer
		}
		if quote.Status != models.QuoteStatusAccepted {
			return ErrQuoteNotAccepted
		}
		if err := requireNoActiveInvoice(tx, "quote_id", *invoice.QuoteID); err != nil {
			return err
		}
		// An invoice for a quote bills in the quote's currency (T127).
		if quote.Currency != "" {
			invoice.Currency = quote.Currency
		}
	}
	return nil
}

// requireNoActiveInvoice fails when a non-cancelled invoice already exists
// for the given job_id / quote_id column.
func requireNoActiveInvoice(tx *gorm.DB, column string, id uuid.UUID) error {
	var n int64
	if err := tx.Model(&models.Invoice{}).
		Where(column+" = ? AND status <> ?", id, models.InvoiceStatusCancelled).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrAlreadyInvoiced
	}
	return nil
}

// DeleteInvoice soft deletes an invoice
func (s *InvoiceService) DeleteInvoice(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock so a payment can't land between the check and the delete.
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}
		if invoice.Status == models.InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		paid, err := hasPayments(tx, invoice)
		if err != nil {
			return err
		}
		if paid {
			return ErrInvoiceHasPayments
		}

		result := tx.Delete(&models.Invoice{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete invoice: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrInvoiceNotFound
		}
		return nil
	})
}

// ListInvoices retrieves invoices with filtering and pagination
func (s *InvoiceService) ListInvoices(ctx context.Context, filter *models.InvoiceFilter) (*models.InvoiceListResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.Invoice{})

	// Apply filters
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.FromDate != nil {
		query = query.Where("issue_date >= ?", *filter.FromDate)
	}

	if filter.ToDate != nil {
		query = query.Where("issue_date <= ?", *filter.ToDate)
	}

	if filter.MinAmount != nil {
		query = query.Where("total_amount >= ?", *filter.MinAmount)
	}

	if filter.MaxAmount != nil {
		query = query.Where("total_amount <= ?", *filter.MaxAmount)
	}

	if filter.IsOverdue != nil && *filter.IsOverdue {
		query = query.Where("due_date < ? AND status IN (?)", time.Now(), []models.InvoiceStatus{
			models.InvoiceStatusSent,
			models.InvoiceStatusViewed,
			models.InvoiceStatusPartiallyPaid,
			models.InvoiceStatusOverdue,
		})
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("invoice_number ILIKE ? OR notes ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count invoices: %w", err)
	}

	// Apply sorting (with SQL injection protection)
	orderClause := utils.SafeOrderClause(
		filter.SortBy,
		filter.SortOrder,
		utils.InvoiceAllowedSortColumns,
		"created_at",
	)
	query = query.Order(orderClause)

	// Apply pagination
	page := 1
	if filter.Page > 0 {
		page = filter.Page
	}
	pageSize := 20
	if filter.PageSize > 0 {
		pageSize = filter.PageSize
	}
	offset := (page - 1) * pageSize

	query = query.Offset(offset).Limit(pageSize)

	// Preload relationships
	query = query.
		Preload("Customer").
		Preload("PaymentTerm").
		Preload("TaxRate")

	// Execute query
	var invoices []models.Invoice
	if err := query.Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.InvoiceListResponse{
		Invoices:   invoices,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// AddLineItem adds a line item to an invoice
func (s *InvoiceService) AddLineItem(ctx context.Context, invoiceID uuid.UUID, lineItem *models.InvoiceLine) (*models.InvoiceLine, error) {
	lineItem.InvoiceID = invoiceID

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// invoice_lines has no tenant_id: the (tenant-scoped) invoice lookup
		// is what authorises adding a line to it.
		invoice, err := lockInvoice(tx, invoiceID)
		if err != nil {
			return err
		}
		if invoice.Status == models.InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		if !invoice.Status.IsEditable() {
			return ErrInvoiceClosed
		}
		var lines []models.InvoiceLine
		if err := tx.Where("invoice_id = ?", invoiceID).Order("sort_order ASC").Find(&lines).Error; err != nil {
			return err
		}
		rate, err := resolveTaxRate(tx, nil, invoice.TaxRateID, lines)
		if err != nil {
			return err
		}
		lineItem.SortOrder = len(lines)
		lines = append(lines, *lineItem)
		totals, err := calculateInvoiceTotals(lines, invoice.DiscountAmount, rate)
		if err != nil {
			return err
		}
		*lineItem = lines[len(lines)-1]
		if lineItem.ID == uuid.Nil {
			lineItem.ID = uuid.New()
		}
		if err := tx.Create(lineItem).Error; err != nil {
			return fmt.Errorf("failed to add line item: %w", err)
		}
		if err := saveLineTaxes(tx, lines[:len(lines)-1]); err != nil {
			return err
		}
		return tx.Model(&models.Invoice{}).Where("id = ?", invoiceID).Updates(map[string]interface{}{
			"subtotal": totals.Subtotal, "discount_amount": totals.Discount,
			"tax_amount": totals.Tax, "total_amount": totals.Total,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return lineItem, nil
}

// RecordPayment records a manual payment (cash, check, ACH, ...) against an
// invoice.
//
// In one transaction it locks the invoice row (SELECT ... FOR UPDATE),
// re-checks status and balance under the lock, inserts the payment and
// updates amount_paid, status and paid_date — so two concurrent payments can
// never overpay the invoice (correctness audit 7.C1, 7.M15).
func (s *InvoiceService) RecordPayment(ctx context.Context, payment *models.InvoicePayment) (*models.InvoicePayment, error) {
	if payment.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("payment amount must be greater than zero")
	}
	payment.Amount = payment.Amount.Round(2)
	if payment.PaymentDate.IsZero() {
		payment.PaymentDate = time.Now()
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, payment.InvoiceID)
		if err != nil {
			return err
		}
		if invoice.Status == models.InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		if !invoice.Status.AcceptsPayments() {
			return ErrPaymentNotAllowed
		}
		if payment.Amount.GreaterThan(invoice.BalanceDue()) {
			return ErrOverpayment
		}

		if err := tx.Create(payment).Error; err != nil {
			return fmt.Errorf("failed to record payment: %w", err)
		}
		return applyPayment(tx, invoice, payment.Amount, payment.PaymentDate, nil)
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// lockInvoice loads an invoice with SELECT ... FOR UPDATE inside tx. The
// tenancy plugin scopes the lookup, so another tenant's invoice is not found.
func lockInvoice(tx *gorm.DB, id uuid.UUID) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	if afterInvoiceLockHook != nil {
		afterInvoiceLockHook()
	}
	return &invoice, nil
}

// afterInvoiceLockHook lets concurrency tests widen the window between the
// locked read and the write. Always nil outside tests.
var afterInvoiceLockHook func()

// applyPayment adds amount to a locked invoice's amount_paid and moves it to
// partially_paid or paid. Closed invoices (cancelled, refunded) keep their
// status: money received for them is still recorded, and needs a refund.
func applyPayment(tx *gorm.DB, invoice *models.Invoice, amount decimal.Decimal, paidAt time.Time, extra map[string]interface{}) error {
	newAmountPaid := invoice.AmountPaid.Add(amount)
	updates := map[string]interface{}{"amount_paid": newAmountPaid}
	for k, v := range extra {
		updates[k] = v
	}

	switch invoice.Status {
	case models.InvoiceStatusCancelled, models.InvoiceStatusRefunded:
		// keep status
	default:
		if newAmountPaid.GreaterThanOrEqual(invoice.TotalAmount) {
			updates["status"] = models.InvoiceStatusPaid
			updates["paid_date"] = paidAt
		} else {
			updates["status"] = models.InvoiceStatusPartiallyPaid
		}
	}

	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoice.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update invoice balance: %w", err)
	}
	if to, ok := updates["status"].(models.InvoiceStatus); ok {
		if err := recordInvoiceTransition(tx, invoice.ID, invoice.Status, to, "payment", nil); err != nil {
			return err
		}
		if to == models.InvoiceStatusPaid {
			return moveInvoiceJob(tx, invoice, models.JobStatusInvoiced, models.JobStatusPaid,
				fmt.Sprintf("Invoice %s paid", invoice.InvoiceNumber), invoice.CreatedBy)
		}
	}
	return nil
}

// moveInvoiceJob moves the job an invoice was created from along the job
// state machine (completed→invoiced on invoicing, invoiced→paid when the
// invoice is paid, back again on cancel/refund), with a history row in the
// same transaction (T124/T225, audit 7.M5). Jobs in any other status are
// left alone (e.g. a deposit invoice for a job still in progress).
func moveInvoiceJob(tx *gorm.DB, invoice *models.Invoice, from, to models.JobStatus, note string, by uuid.UUID) error {
	if invoice.JobID == nil {
		return nil
	}
	var job models.Job
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", *invoice.JobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // job deleted since
		}
		return err
	}
	if job.Status != from || !models.CanTransition(from, to) {
		return nil
	}
	if err := tx.Model(&models.Job{}).Where("id = ?", job.ID).Update("status", to).Error; err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return tx.Create(&models.JobStatusTransition{
		ID: uuid.New(), JobID: job.ID, FromStatus: from, ToStatus: to,
		Reason: models.ReasonOther, Notes: &note, ChangedBy: by, TransitionedAt: epoch.NowEpoch(),
	}).Error
}

// nonEmpty returns nil for "" so optional reference columns stay NULL.
func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// hasPayments reports whether money has been recorded against a (locked)
// invoice.
func hasPayments(tx *gorm.DB, invoice *models.Invoice) (bool, error) {
	if invoice.AmountPaid.GreaterThan(decimal.Zero) {
		return true, nil
	}
	var n int64
	if err := tx.Model(&models.InvoicePayment{}).Where("invoice_id = ?", invoice.ID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// SendInvoice marks an invoice as sent and generates a payment token
func (s *InvoiceService) SendInvoice(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	// Generate new payment token
	token, err := s.generatePaymentToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment token: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}

		// A draft is issued (draft→sent); an open invoice with money still
		// owed can be re-sent with a fresh pay link, keeping its status.
		switch {
		case invoice.Status == models.InvoiceStatusDraft:
			var lineCount int64
			if err := tx.Model(&models.InvoiceLine{}).Where("invoice_id = ?", id).Count(&lineCount).Error; err != nil {
				return err
			}
			if err := s.validateForSending(invoice, lineCount); err != nil {
				return err
			}
		case invoice.Status.AcceptsPayments():
		default:
			return ErrInvalidInvoiceTransition
		}

		now := time.Now()
		updates := map[string]interface{}{
			"sent_date":                now,
			"payment_token":            token,
			"payment_token_expires_at": now.Add(PaymentTokenExpiry),
			"payment_token_voided_at":  nil, // Clear any previous voided timestamp
		}
		if invoice.Status == models.InvoiceStatusDraft {
			updates["status"] = models.InvoiceStatusSent
		}
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to send invoice: %w", err)
		}
		if invoice.Status != models.InvoiceStatusDraft {
			return nil // re-send: status unchanged
		}
		return recordInvoiceTransition(tx, id, invoice.Status, models.InvoiceStatusSent, "sent", nil)
	})
	if err != nil {
		return nil, err
	}

	// Reload invoice
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	// Send email asynchronously if email client is configured
	if s.emailClient != nil && invoice.Customer != nil {
		go s.sendInvoiceEmail(invoice)
	}

	return invoice, nil
}

// CancelInvoice cancels an invoice
func (s *InvoiceService) CancelInvoice(ctx context.Context, id uuid.UUID, reason string) (*models.Invoice, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}
		if invoice.Status == models.InvoiceStatusPaid {
			return ErrInvoiceAlreadyPaid
		}
		paid, err := hasPayments(tx, invoice)
		if err != nil {
			return err
		}
		if paid {
			return ErrInvoiceHasPayments
		}
		if invoice.Status == models.InvoiceStatusCancelled || !invoice.Status.CanTransitionTo(models.InvoiceStatusCancelled) {
			return ErrInvalidInvoiceTransition
		}

		updates := map[string]interface{}{
			"status": models.InvoiceStatusCancelled,
			"notes":  fmt.Sprintf("%s\n\nCancellation reason: %s", invoice.Notes, reason),
			// The pay link must stop working.
			"payment_token_voided_at": time.Now(),
		}
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to cancel invoice: %w", err)
		}
		if err := recordInvoiceTransition(tx, id, invoice.Status, models.InvoiceStatusCancelled, "cancelled", nil); err != nil {
			return err
		}
		// The job can be invoiced again.
		return moveInvoiceJob(tx, invoice, models.JobStatusInvoiced, models.JobStatusCompleted,
			fmt.Sprintf("Invoice %s cancelled", invoice.InvoiceNumber), invoice.CreatedBy)
	})
	if err != nil {
		return nil, err
	}

	return s.GetInvoice(ctx, id)
}

// GetInvoiceSummary retrieves invoice summary data
func (s *InvoiceService) GetInvoiceSummary(ctx context.Context, id uuid.UUID) (*models.InvoiceSummary, error) {
	var summary models.InvoiceSummary

	err := s.db.WithContext(ctx).
		Raw("SELECT * FROM invoice_summary WHERE id = ?", id).
		Scan(&summary).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get invoice summary: %w", err)
	}

	return &summary, nil
}

// Private helper methods

func (s *InvoiceService) calculateDueDate(ctx context.Context, issueDate time.Time, paymentTermID *uuid.UUID) (time.Time, error) {
	if paymentTermID == nil {
		// Default to 30 days
		return issueDate.AddDate(0, 0, 30), nil
	}

	var paymentTerm models.PaymentTerm
	if err := s.db.WithContext(ctx).First(&paymentTerm, *paymentTermID).Error; err != nil {
		return time.Time{}, fmt.Errorf("failed to get payment term: %w", err)
	}

	return issueDate.AddDate(0, 0, paymentTerm.DaysUntilDue), nil
}

// requireCustomer returns ErrCustomerNotFound unless the customer is visible
// to the caller (tenant scoping hides other tenants' customers).
func (s *InvoiceService) requireCustomer(ctx context.Context, customerID uuid.UUID) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Customer{}).
		Where("id = ? AND deleted_at IS NULL", customerID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

func (s *InvoiceService) validateInvoice(invoice *models.Invoice) error {
	if invoice.CustomerID == uuid.Nil {
		return invalidInvoice("customer_id is required")
	}

	if invoice.IssueDate.IsZero() {
		return invalidInvoice("issue_date is required")
	}

	if invoice.DueDate.IsZero() {
		return invalidInvoice("due_date is required")
	}

	if invoice.DueDate.Before(invoice.IssueDate) {
		return invalidInvoice("due_date must be after issue_date")
	}

	return nil
}

func (s *InvoiceService) validateForSending(invoice *models.Invoice, lineCount int64) error {
	if invoice.Status != models.InvoiceStatusDraft {
		return invalidInvoice("only draft invoices can be sent")
	}

	if lineCount == 0 {
		return invalidInvoice("invoice must have at least one line item")
	}

	if invoice.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return invalidInvoice("invoice total must be greater than zero")
	}

	return nil
}

// recordInvoiceTransition writes a status change to invoice_audit_log in the
// same transaction as the change. by is nil for system changes (webhooks).
func recordInvoiceTransition(tx *gorm.DB, invoiceID uuid.UUID, from, to models.InvoiceStatus, action string, by *uuid.UUID) error {
	if from == to {
		return nil
	}
	changedByType := "system"
	if by != nil {
		changedByType = "user"
	}
	entry := &models.InvoiceAuditLog{
		ID: uuid.New(), InvoiceID: invoiceID, Action: action,
		FromStatus: &from, ToStatus: &to, ChangedBy: by, ChangedByType: changedByType,
	}
	if err := tx.Omit("ip_address").Create(entry).Error; err != nil {
		return fmt.Errorf("failed to record invoice status change: %w", err)
	}
	return nil
}

// GetInvoiceByPaymentToken retrieves an invoice by its payment token
func (s *InvoiceService) GetInvoiceByPaymentToken(ctx context.Context, token string) (*models.Invoice, error) {
	if token == "" {
		return nil, ErrInvalidPaymentToken
	}

	var invoice models.Invoice
	err := s.db.WithContext(ctx).
		Preload("Customer").
		Preload("Lines", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("PaymentTerm").
		Preload("TaxRate").
		First(&invoice, "payment_token = ?", token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidPaymentToken
		}
		return nil, fmt.Errorf("failed to get invoice by payment token: %w", err)
	}

	return &invoice, nil
}

// ValidatePaymentToken validates a payment token and returns the invoice if valid
func (s *InvoiceService) ValidatePaymentToken(ctx context.Context, token string) (*models.Invoice, error) {
	invoice, err := s.GetInvoiceByPaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if token is voided
	if invoice.PaymentTokenVoidedAt != nil {
		return nil, ErrPaymentTokenVoided
	}

	// Check if token is expired
	if invoice.PaymentTokenExpiresAt == nil || invoice.PaymentTokenExpiresAt.Before(time.Now()) {
		return nil, ErrPaymentTokenExpired
	}

	// Payable: issued, not closed, with money still owed. Partially paid
	// invoices can be paid online for the remaining balance (T119).
	if !invoice.Status.AcceptsPayments() || !invoice.BalanceDue().IsPositive() {
		return nil, ErrInvoiceNotPayable
	}

	return invoice, nil
}

// MarkInvoiceAsPaid applies a Stripe Checkout payment to an invoice.
//
// The money has already been taken, so unlike RecordPayment it never
// rejects: it records an invoice_payments row and updates amount_paid and
// status in one transaction under the invoice row lock (T118). Money received
// for a cancelled or refunded invoice is recorded without reviving it.
//
// It is idempotent per checkout session: a session that already has a
// payment row is not credited again, so duplicate webhook deliveries — even
// to two instances at once — can't double-count a payment (F-10). The unique
// index on invoice_payments.stripe_checkout_session_id (migration 000007)
// backs this up.
func (s *InvoiceService) MarkInvoiceAsPaid(ctx context.Context, id uuid.UUID, amountPaid decimal.Decimal, stripeCheckoutSessionID, stripePaymentIntentID string) (*models.Invoice, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}

		if stripeCheckoutSessionID != "" {
			var n int64
			if err := tx.Model(&models.InvoicePayment{}).
				Where("invoice_id = ? AND stripe_checkout_session_id = ?", id, stripeCheckoutSessionID).
				Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return nil // this session was already applied
			}
		}

		if !invoice.Status.AcceptsPayments() || amountPaid.GreaterThan(invoice.BalanceDue()) {
			logging.Warn(ctx, "[INVOICE-SERVICE] Stripe payment received for an invoice that was not expecting it; recorded for manual review", map[string]any{
				"invoice_id": id, "status": invoice.Status, "amount": amountPaid.StringFixed(2), "balance_due": invoice.BalanceDue().StringFixed(2),
			})
		}

		now := time.Now()
		payment := &models.InvoicePayment{
			InvoiceID:               id,
			Amount:                  amountPaid,
			PaymentDate:             now,
			PaymentMethod:           "stripe",
			ReferenceNumber:         stripePaymentIntentID,
			StripeCheckoutSessionID: nonEmpty(stripeCheckoutSessionID),
			StripePaymentIntentID:   nonEmpty(stripePaymentIntentID),
		}
		if err := tx.Create(payment).Error; err != nil {
			return fmt.Errorf("failed to record stripe payment: %w", err)
		}

		extra := map[string]interface{}{}
		if stripeCheckoutSessionID != "" {
			extra["stripe_checkout_session_id"] = stripeCheckoutSessionID
		}
		if stripePaymentIntentID != "" {
			extra["stripe_payment_intent_id"] = stripePaymentIntentID
		}
		return applyPayment(tx, invoice, amountPaid, now, extra)
	})
	if err != nil {
		return nil, err
	}

	return s.GetInvoice(ctx, id)
}

// VoidPaymentToken voids the current payment token for an invoice
func (s *InvoiceService) VoidPaymentToken(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&models.Invoice{}).
		Where("id = ? AND payment_token IS NOT NULL", id).
		Update("payment_token_voided_at", now).Error
}

// generatePaymentToken generates a cryptographically secure random token
func (s *InvoiceService) generatePaymentToken() (string, error) {
	bytes := make([]byte, PaymentTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// sendInvoiceEmail sends the invoice email to the customer with PDF attachment
func (s *InvoiceService) sendInvoiceEmail(invoice *models.Invoice) {
	ctx := context.Background()

	if invoice.Customer == nil || invoice.PaymentToken == nil {
		logging.Warn(ctx, "[INVOICE-SERVICE] Cannot send invoice email: missing customer or payment token", nil)
		return
	}

	// Build payment URL - points to backend API which redirects to Stripe Checkout
	paymentURL := fmt.Sprintf("%s/api/v1/public/invoices/pay/%s", s.backendURL, *invoice.PaymentToken)

	var pdfAttachment *email.Attachment
	var downloadURL string

	// Generate PDF if PDF service is configured
	if s.pdfService != nil {
		pdfResult, err := s.pdfService.GenerateInvoicePDF(ctx, invoice)
		if err != nil {
			logging.Error(ctx, "[INVOICE-SERVICE] Failed to generate PDF for invoice", map[string]any{"invoice_number": invoice.InvoiceNumber, "error": err})
		} else {
			pdfAttachment = &email.Attachment{
				Filename:    pdfResult.FileName,
				Content:     pdfResult.Content,
				ContentType: "application/pdf",
			}

			// Get download URL
			url, _, err := s.pdfService.GetInvoicePDFURL(ctx, invoice.ID)
			if err != nil {
				logging.Error(ctx, "[INVOICE-SERVICE] Failed to get download URL for invoice", map[string]any{"invoice_number": invoice.InvoiceNumber, "error": err})
			} else {
				downloadURL = url
			}
		}
	}

	// Send invoice email
	if err := s.emailClient.SendInvoiceEmail(ctx, invoice.Customer.Email, invoice, paymentURL, pdfAttachment, downloadURL); err != nil {
		logging.Error(ctx, "[INVOICE-SERVICE] Failed to send invoice email", map[string]any{"customer_email": invoice.Customer.Email, "error": err})
	} else {
		logging.Info(ctx, "[INVOICE-SERVICE] Invoice email sent", map[string]any{"customer_email": invoice.Customer.Email, "invoice_number": invoice.InvoiceNumber})
	}
}

// SendReceiptEmail sends a payment receipt email to the customer with PDF attachment
func (s *InvoiceService) SendReceiptEmail(invoice *models.Invoice) {
	ctx := context.Background()

	if s.emailClient == nil || invoice.Customer == nil {
		logging.Warn(ctx, "[INVOICE-SERVICE] Cannot send receipt email: missing email client or customer", nil)
		return
	}

	var pdfAttachment *email.Attachment
	var downloadURL string

	// Generate receipt PDF if PDF service is configured
	if s.pdfService != nil {
		pdfResult, err := s.pdfService.GenerateReceiptPDF(ctx, invoice)
		if err != nil {
			logging.Error(ctx, "[INVOICE-SERVICE] Failed to generate receipt PDF for invoice", map[string]any{"invoice_number": invoice.InvoiceNumber, "error": err})
		} else {
			pdfAttachment = &email.Attachment{
				Filename:    pdfResult.FileName,
				Content:     pdfResult.Content,
				ContentType: "application/pdf",
			}

			// Get download URL
			url, _, err := s.pdfService.GetReceiptPDFURL(ctx, invoice.ID)
			if err != nil {
				logging.Error(ctx, "[INVOICE-SERVICE] Failed to get download URL for receipt", map[string]any{"invoice_number": invoice.InvoiceNumber, "error": err})
			} else {
				downloadURL = url
			}
		}
	}

	// Send receipt email
	if err := s.emailClient.SendPaymentReceiptEmail(ctx, invoice.Customer.Email, invoice, pdfAttachment, downloadURL); err != nil {
		logging.Error(ctx, "[INVOICE-SERVICE] Failed to send receipt email", map[string]any{"customer_email": invoice.Customer.Email, "error": err})
	} else {
		logging.Info(ctx, "[INVOICE-SERVICE] Receipt email sent", map[string]any{"customer_email": invoice.Customer.Email, "invoice_number": invoice.InvoiceNumber})
	}
}

// GetInvoicePDFURL returns a presigned URL for downloading the invoice PDF
func (s *InvoiceService) GetInvoicePDFURL(ctx context.Context, id uuid.UUID) (string, time.Time, error) {
	// Verify invoice exists
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return "", time.Time{}, err
	}

	if s.pdfService == nil {
		return "", time.Time{}, errors.New("PDF service not configured")
	}

	// Check if PDF exists, generate if not
	url, expiresAt, err := s.pdfService.GetInvoicePDFURL(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		_, err = s.pdfService.GenerateInvoicePDF(ctx, invoice)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to generate invoice PDF: %w", err)
		}

		// Try to get URL again
		url, expiresAt, err = s.pdfService.GetInvoicePDFURL(ctx, id)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to get invoice PDF URL: %w", err)
		}
	}

	return url, expiresAt, nil
}

// DownloadInvoicePDF returns the invoice PDF content directly
func (s *InvoiceService) DownloadInvoicePDF(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	// Verify invoice exists
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, "", err
	}

	if s.pdfService == nil {
		return nil, "", errors.New("PDF service not configured")
	}

	// Try to download existing PDF
	content, filename, err := s.pdfService.DownloadInvoicePDF(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		result, genErr := s.pdfService.GenerateInvoicePDF(ctx, invoice)
		if genErr != nil {
			return nil, "", fmt.Errorf("failed to generate invoice PDF: %w", genErr)
		}
		return result.Content, result.FileName, nil
	}

	return content, filename, nil
}

// GetReceiptPDFURL returns a presigned URL for downloading the receipt PDF
func (s *InvoiceService) GetReceiptPDFURL(ctx context.Context, id uuid.UUID) (string, time.Time, error) {
	// Verify invoice exists
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return "", time.Time{}, err
	}

	// Only allow receipt download for paid invoices
	if invoice.Status != models.InvoiceStatusPaid {
		return "", time.Time{}, errors.New("receipt only available for paid invoices")
	}

	if s.pdfService == nil {
		return "", time.Time{}, errors.New("PDF service not configured")
	}

	// Check if PDF exists, generate if not
	url, expiresAt, err := s.pdfService.GetReceiptPDFURL(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		_, err = s.pdfService.GenerateReceiptPDF(ctx, invoice)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to generate receipt PDF: %w", err)
		}

		// Try to get URL again
		url, expiresAt, err = s.pdfService.GetReceiptPDFURL(ctx, id)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to get receipt PDF URL: %w", err)
		}
	}

	return url, expiresAt, nil
}

// DownloadReceiptPDF returns the receipt PDF content directly
func (s *InvoiceService) DownloadReceiptPDF(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	// Verify invoice exists
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, "", err
	}

	// Only allow receipt download for paid invoices
	if invoice.Status != models.InvoiceStatusPaid {
		return nil, "", ErrReceiptNotAvailable
	}

	if s.pdfService == nil {
		return nil, "", errors.New("PDF service not configured")
	}

	// Try to download existing PDF
	content, filename, err := s.pdfService.DownloadReceiptPDF(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		result, genErr := s.pdfService.GenerateReceiptPDF(ctx, invoice)
		if genErr != nil {
			return nil, "", fmt.Errorf("failed to generate receipt PDF: %w", genErr)
		}
		return result.Content, result.FileName, nil
	}

	return content, filename, nil
}
