package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
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
)

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

// SendInvoice marks an invoice as sent and generates a payment token
func (s *InvoiceService) SendInvoice(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	// Allow sending draft invoices or resending already-sent invoices
	if invoice.Status != models.InvoiceStatusDraft && invoice.Status != models.InvoiceStatusSent {
		return nil, errors.New("only draft or sent invoices can be sent")
	}

	// Validate invoice before sending (only for draft invoices)
	if invoice.Status == models.InvoiceStatusDraft {
		if err := s.validateForSending(invoice); err != nil {
			return nil, err
		}
	}

	// Generate new payment token
	token, err := s.generatePaymentToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment token: %w", err)
	}

	// Update invoice with new token and status
	now := time.Now()
	expiresAt := now.Add(PaymentTokenExpiry)

	updates := map[string]interface{}{
		"status":                   models.InvoiceStatusSent,
		"sent_date":                now,
		"payment_token":            token,
		"payment_token_expires_at": expiresAt,
		"payment_token_voided_at":  nil, // Clear any previous voided timestamp
	}

	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to send invoice: %w", err)
	}

	// Reload invoice
	invoice, err = s.GetInvoice(ctx, id)
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

	// Check if invoice is in a payable status
	if invoice.Status != models.InvoiceStatusSent {
		return nil, ErrInvoiceNotPayable
	}

	return invoice, nil
}

// MarkInvoiceAsPaid marks an invoice as paid and stores Stripe payment information
func (s *InvoiceService) MarkInvoiceAsPaid(ctx context.Context, id uuid.UUID, amountPaid decimal.Decimal, stripeCheckoutSessionID, stripePaymentIntentID string) (*models.Invoice, error) {
	invoice, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if already paid
	if invoice.Status == models.InvoiceStatusPaid {
		return invoice, nil // Idempotent
	}

	now := time.Now()

	// Determine new status based on amount paid
	newAmountPaid := invoice.AmountPaid.Add(amountPaid)
	var newStatus models.InvoiceStatus
	if newAmountPaid.GreaterThanOrEqual(invoice.TotalAmount) {
		newStatus = models.InvoiceStatusPaid
	} else {
		newStatus = models.InvoiceStatusPartiallyPaid
	}

	updates := map[string]interface{}{
		"status":                     newStatus,
		"amount_paid":                newAmountPaid,
		"paid_date":                  now,
		"stripe_checkout_session_id": stripeCheckoutSessionID,
		"stripe_payment_intent_id":   stripePaymentIntentID,
	}

	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to mark invoice as paid: %w", err)
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
	if invoice.Customer == nil || invoice.PaymentToken == nil {
		log.Printf("[INVOICE-SERVICE] Cannot send invoice email: missing customer or payment token")
		return
	}

	ctx := context.Background()

	// Build payment URL - points to backend API which redirects to Stripe Checkout
	paymentURL := fmt.Sprintf("%s/api/v1/public/invoices/pay/%s", s.backendURL, *invoice.PaymentToken)

	var pdfAttachment *email.Attachment
	var downloadURL string

	// Generate PDF if PDF service is configured
	if s.pdfService != nil {
		pdfResult, err := s.pdfService.GenerateInvoicePDF(ctx, invoice)
		if err != nil {
			log.Printf("[INVOICE-SERVICE] Failed to generate PDF for invoice %s: %v", invoice.InvoiceNumber, err)
		} else {
			pdfAttachment = &email.Attachment{
				Filename:    pdfResult.FileName,
				Content:     pdfResult.Content,
				ContentType: "application/pdf",
			}

			// Get download URL
			url, _, err := s.pdfService.GetInvoicePDFURL(ctx, invoice.ID)
			if err != nil {
				log.Printf("[INVOICE-SERVICE] Failed to get download URL for invoice %s: %v", invoice.InvoiceNumber, err)
			} else {
				downloadURL = url
			}
		}
	}

	// Send invoice email
	if err := s.emailClient.SendInvoiceEmail(ctx, invoice.Customer.Email, invoice, paymentURL, pdfAttachment, downloadURL); err != nil {
		log.Printf("[INVOICE-SERVICE] Failed to send invoice email to %s: %v", invoice.Customer.Email, err)
	} else {
		log.Printf("[INVOICE-SERVICE] Invoice email sent to %s for invoice %s", invoice.Customer.Email, invoice.InvoiceNumber)
	}
}

// SendReceiptEmail sends a payment receipt email to the customer with PDF attachment
func (s *InvoiceService) SendReceiptEmail(invoice *models.Invoice) {
	if s.emailClient == nil || invoice.Customer == nil {
		log.Printf("[INVOICE-SERVICE] Cannot send receipt email: missing email client or customer")
		return
	}

	ctx := context.Background()

	var pdfAttachment *email.Attachment
	var downloadURL string

	// Generate receipt PDF if PDF service is configured
	if s.pdfService != nil {
		pdfResult, err := s.pdfService.GenerateReceiptPDF(ctx, invoice)
		if err != nil {
			log.Printf("[INVOICE-SERVICE] Failed to generate receipt PDF for invoice %s: %v", invoice.InvoiceNumber, err)
		} else {
			pdfAttachment = &email.Attachment{
				Filename:    pdfResult.FileName,
				Content:     pdfResult.Content,
				ContentType: "application/pdf",
			}

			// Get download URL
			url, _, err := s.pdfService.GetReceiptPDFURL(ctx, invoice.ID)
			if err != nil {
				log.Printf("[INVOICE-SERVICE] Failed to get download URL for receipt %s: %v", invoice.InvoiceNumber, err)
			} else {
				downloadURL = url
			}
		}
	}

	// Send receipt email
	if err := s.emailClient.SendPaymentReceiptEmail(ctx, invoice.Customer.Email, invoice, pdfAttachment, downloadURL); err != nil {
		log.Printf("[INVOICE-SERVICE] Failed to send receipt email to %s: %v", invoice.Customer.Email, err)
	} else {
		log.Printf("[INVOICE-SERVICE] Receipt email sent to %s for invoice %s", invoice.Customer.Email, invoice.InvoiceNumber)
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
		return nil, "", errors.New("receipt only available for paid invoices")
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
