package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils"
)

var (
	// ErrInvoiceNotFound is returned when an invoice is not found
	ErrInvoiceNotFound = errors.New("invoice not found")
	// ErrInvalidInvoiceData is returned when invoice data is invalid
	ErrInvalidInvoiceData = errors.New("invalid invoice data")
	// ErrInvoiceAlreadyPaid is returned when trying to modify a paid invoice
	ErrInvoiceAlreadyPaid = errors.New("cannot modify paid invoice")
)

// InvoiceService handles invoice business logic
type InvoiceService struct {
	db *gorm.DB
}

// NewInvoiceService creates a new invoice service
func NewInvoiceService(db *gorm.DB) *InvoiceService {
	return &InvoiceService{db: db}
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

	// Calculate due date based on payment terms
	if invoice.DueDate.IsZero() {
		dueDate, err := s.calculateDueDate(ctx, invoice.IssueDate, invoice.PaymentTermID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate due date: %w", err)
		}
		invoice.DueDate = dueDate
	}

	// Generate invoice number if not provided (fallback for databases without triggers)
	if invoice.InvoiceNumber == "" {
		invoiceNumber, err := s.generateInvoiceNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate invoice number: %w", err)
		}
		invoice.InvoiceNumber = invoiceNumber
	}

	// Validate invoice
	if err := s.validateInvoice(invoice); err != nil {
		return nil, err
	}

	// Create invoice in transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create invoice
		if err := tx.Create(invoice).Error; err != nil {
			return fmt.Errorf("failed to create invoice: %w", err)
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

// UpdateInvoice updates an existing invoice
func (s *InvoiceService) UpdateInvoice(ctx context.Context, id uuid.UUID, updates *models.Invoice, userID uuid.UUID) (*models.Invoice, error) {
	// Get existing invoice
	existing, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if invoice can be modified
	if existing.Status == models.InvoiceStatusPaid {
		return nil, ErrInvoiceAlreadyPaid
	}

	// Set updated by
	updates.UpdatedBy = &userID

	// Validate updates
	if err := s.validateInvoiceUpdates(existing, updates); err != nil {
		return nil, err
	}

	// Update in transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update invoice
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		// If line items are provided, update them
		if len(updates.Lines) > 0 {
			// Delete existing line items
			if err := tx.Where("invoice_id = ?", id).Delete(&models.InvoiceLine{}).Error; err != nil {
				return fmt.Errorf("failed to delete old line items: %w", err)
			}

			// Create new line items
			for i := range updates.Lines {
				updates.Lines[i].InvoiceID = id
				if err := tx.Create(&updates.Lines[i]).Error; err != nil {
					return fmt.Errorf("failed to create line item: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reload invoice with relationships
	return s.GetInvoice(ctx, id)
}

// DeleteInvoice soft deletes an invoice
func (s *InvoiceService) DeleteInvoice(ctx context.Context, id uuid.UUID) error {
	// Get invoice to check status
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return err
	}

	// Don't allow deletion of paid invoices
	if invoice.Status == models.InvoiceStatusPaid {
		return errors.New("cannot delete paid invoice")
	}

	// Soft delete
	result := s.db.WithContext(ctx).Delete(&models.Invoice{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete invoice: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrInvoiceNotFound
	}

	return nil
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

	// Calculate tax if applicable
	if lineItem.Taxable {
		invoice, err := s.GetInvoice(ctx, invoiceID)
		if err != nil {
			return nil, err
		}

		if invoice.TaxRate != nil {
			lineItem.TaxRate = invoice.TaxRate.Rate
			lineTotal := lineItem.Quantity.Mul(lineItem.UnitPrice).Sub(lineItem.DiscountAmount)
			lineItem.TaxAmount = lineTotal.Mul(lineItem.TaxRate).Round(2)
		}
	}

	if err := s.db.WithContext(ctx).Create(lineItem).Error; err != nil {
		return nil, fmt.Errorf("failed to add line item: %w", err)
	}

	return lineItem, nil
}

// RecordPayment records a payment for an invoice
func (s *InvoiceService) RecordPayment(ctx context.Context, payment *models.InvoicePayment) (*models.InvoicePayment, error) {
	// Validate payment
	if payment.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("payment amount must be greater than zero")
	}

	// Get invoice
	invoice, err := s.GetInvoice(ctx, payment.InvoiceID)
	if err != nil {
		return nil, err
	}

	// Calculate amount due if database didn't compute it (for SQLite compatibility)
	amountDue := invoice.AmountDue
	if amountDue.IsZero() && invoice.TotalAmount.GreaterThan(decimal.Zero) {
		amountDue = invoice.TotalAmount.Sub(invoice.AmountPaid)
	}

	// Check if payment exceeds outstanding amount
	if payment.Amount.GreaterThan(amountDue) {
		return nil, errors.New("payment amount exceeds outstanding balance")
	}

	// Create payment (triggers will update invoice)
	if err := s.db.WithContext(ctx).Create(payment).Error; err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	return payment, nil
}

// SendInvoice marks an invoice as sent
func (s *InvoiceService) SendInvoice(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate invoice before sending
	if err := s.validateForSending(invoice); err != nil {
		return nil, err
	}

	// Update status
	now := time.Now()
	updates := map[string]interface{}{
		"status":    models.InvoiceStatusSent,
		"sent_date": now,
	}

	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to send invoice: %w", err)
	}

	return s.GetInvoice(ctx, id)
}

// CancelInvoice cancels an invoice
func (s *InvoiceService) CancelInvoice(ctx context.Context, id uuid.UUID, reason string) (*models.Invoice, error) {
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	if invoice.Status == models.InvoiceStatusPaid {
		return nil, errors.New("cannot cancel paid invoice")
	}

	updates := map[string]interface{}{
		"status": models.InvoiceStatusCancelled,
		"notes":  fmt.Sprintf("%s\n\nCancellation reason: %s", invoice.Notes, reason),
	}

	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to cancel invoice: %w", err)
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

// generateInvoiceNumber generates a unique invoice number
func (s *InvoiceService) generateInvoiceNumber(ctx context.Context) (string, error) {
	var maxNumber string
	err := s.db.WithContext(ctx).
		Model(&models.Invoice{}).
		Select("COALESCE(MAX(invoice_number), 'INV-000000')").
		Scan(&maxNumber).Error

	if err != nil {
		return "", err
	}

	// Parse the current max number and increment
	var num int
	if _, err := fmt.Sscanf(maxNumber, "INV-%d", &num); err != nil {
		num = 0
	}

	return fmt.Sprintf("INV-%06d", num+1), nil
}

func (s *InvoiceService) validateInvoice(invoice *models.Invoice) error {
	if invoice.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}

	if invoice.IssueDate.IsZero() {
		return errors.New("issue_date is required")
	}

	if invoice.DueDate.IsZero() {
		return errors.New("due_date is required")
	}

	if invoice.DueDate.Before(invoice.IssueDate) {
		return errors.New("due_date must be after issue_date")
	}

	return nil
}

func (s *InvoiceService) validateInvoiceUpdates(existing, updates *models.Invoice) error {
	// Don't allow changing customer on sent invoices
	if existing.Status != models.InvoiceStatusDraft && updates.CustomerID != uuid.Nil && updates.CustomerID != existing.CustomerID {
		return errors.New("cannot change customer on sent invoice")
	}

	// Validate due date if provided
	if !updates.DueDate.IsZero() && updates.DueDate.Before(existing.IssueDate) {
		return errors.New("due_date must be after issue_date")
	}

	return nil
}

func (s *InvoiceService) validateForSending(invoice *models.Invoice) error {
	if invoice.Status != models.InvoiceStatusDraft {
		return errors.New("only draft invoices can be sent")
	}

	if len(invoice.Lines) == 0 {
		return errors.New("invoice must have at least one line item")
	}

	if invoice.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return errors.New("invoice total must be greater than zero")
	}

	return nil
}

// CalculateTotals manually calculates invoice totals (for verification)
func (s *InvoiceService) CalculateTotals(invoice *models.Invoice) {
	subtotal := decimal.Zero
	taxAmount := decimal.Zero

	for _, line := range invoice.Lines {
		lineTotal := line.Quantity.Mul(line.UnitPrice).Sub(line.DiscountAmount)
		subtotal = subtotal.Add(lineTotal)

		if line.Taxable {
			lineTax := lineTotal.Mul(line.TaxRate)
			taxAmount = taxAmount.Add(lineTax)
		}
	}

	invoice.Subtotal = subtotal
	invoice.TaxAmount = taxAmount
	invoice.TotalAmount = subtotal.Add(taxAmount).Sub(invoice.DiscountAmount)
}
