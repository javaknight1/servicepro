package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Quote domain errors (T223): typed, so the API returns their status, code
// and message as-is. A status that doesn't allow the action is a 409
// conflict, not a 403; a database failure is never reported as one of these.
var (
	// ErrQuoteNotFound is returned when a quote is not found (or belongs to another tenant)
	ErrQuoteNotFound = apperrors.NotFound("not_found", "Quote not found")
	// ErrInvalidQuoteData is returned when quote data is invalid
	ErrInvalidQuoteData = apperrors.BadRequest("invalid_data", "Invalid quote data")
	// ErrQuoteCannotBeEdited is returned when trying to edit a non-editable quote
	ErrQuoteCannotBeEdited = apperrors.Conflict("cannot_edit", "Only draft quotes can be edited")
	// ErrQuoteCannotBeDeleted is returned when trying to delete a non-deletable quote
	ErrQuoteCannotBeDeleted = apperrors.Conflict("cannot_delete", "Only draft quotes can be deleted")
	// ErrQuoteCannotBeSent is returned when trying to send a non-sendable quote
	ErrQuoteCannotBeSent = apperrors.Conflict("cannot_send", "Only draft quotes whose valid-until date hasn't passed can be sent")
	// ErrQuoteCannotBeAccepted is returned when trying to accept a non-acceptable quote
	ErrQuoteCannotBeAccepted = apperrors.Conflict("cannot_accept", "Only sent or viewed quotes that haven't expired can be accepted")
	// ErrQuoteCannotBeRejected is returned when trying to reject a non-rejectable quote
	ErrQuoteCannotBeRejected = apperrors.Conflict("cannot_reject", "Only sent or viewed quotes can be declined")
	// ErrQuoteCannotBeRevised is returned when revising a draft or accepted quote
	ErrQuoteCannotBeRevised = apperrors.Conflict("cannot_revise", "Only sent, viewed, expired or declined quotes can be revised")
)

// quoteLookupError maps a repository lookup failure: a missing quote is
// ErrQuoteNotFound, anything else is an infrastructure error (500).
func quoteLookupError(err error) error {
	if err.Error() == "quote not found" {
		return ErrQuoteNotFound
	}
	return fmt.Errorf("failed to get quote: %w", err)
}

// quoteTransitionError maps a failed conditional status change: not found,
// a status that no longer allows it (conflict), or an infrastructure error.
func quoteTransitionError(err error, conflict *apperrors.Error) error {
	if errors.Is(err, repository.ErrQuoteStatusConflict) {
		return conflict
	}
	return quoteLookupError(err)
}

// QuoteServiceInterface defines the interface for quote service
type QuoteServiceInterface interface {
	// WithContext returns a service bound to ctx (the request context), so
	// every query is scoped to the caller's tenant.
	WithContext(ctx context.Context) QuoteServiceInterface
	// CreateQuote creates a new quote
	CreateQuote(req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error)
	// GetQuote retrieves a quote by ID
	GetQuote(id uuid.UUID) (*models.QuoteResponse, error)
	// GetQuoteByNumber retrieves a quote by quote number
	GetQuoteByNumber(quoteNumber string) (*models.QuoteResponse, error)
	// ListQuotes lists quotes with pagination and filters
	ListQuotes(params *models.QuoteQueryParams) (*models.QuoteListResponse, error)
	// UpdateQuote updates a quote
	UpdateQuote(id uuid.UUID, req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error)
	// DeleteQuote soft deletes a quote
	DeleteQuote(id uuid.UUID) error
	// SendQuote marks a quote as sent and sends email with PDF
	SendQuote(id uuid.UUID) error
	// AcceptQuote marks a quote as accepted
	AcceptQuote(id uuid.UUID) error
	// RejectQuote marks a quote as rejected
	RejectQuote(id uuid.UUID) error
	// ReviseQuote takes a sent/viewed/expired/declined quote back to draft
	ReviseQuote(id uuid.UUID) error
	// GetQuotesByCustomer retrieves all quotes for a customer
	GetQuotesByCustomer(customerID uuid.UUID) ([]models.QuoteResponse, error)
	// GetQuoteStats retrieves quote statistics
	GetQuoteStats() (*QuoteStats, error)
	// GetQuotePDFURL returns a presigned URL for downloading the quote PDF
	GetQuotePDFURL(id uuid.UUID) (string, time.Time, error)
	// DownloadQuotePDF returns the quote PDF content directly
	DownloadQuotePDF(id uuid.UUID) ([]byte, string, error)
}

// QuoteStats represents quote statistics
type QuoteStats struct {
	TotalQuotes    int64 `json:"total_quotes"`
	DraftQuotes    int64 `json:"draft_quotes"`
	SentQuotes     int64 `json:"sent_quotes"`
	AcceptedQuotes int64 `json:"accepted_quotes"`
	DeclinedQuotes int64 `json:"declined_quotes"`
	ExpiredQuotes  int64 `json:"expired_quotes"`
}

// QuoteService implements QuoteServiceInterface
type QuoteService struct {
	quoteRepo   repository.QuoteRepositoryInterface
	pdfService  *DocumentPDFService
	emailClient email.Client
}

// QuoteServiceConfig holds configuration for the quote service
type QuoteServiceConfig struct {
	PDFService  *DocumentPDFService
	EmailClient email.Client
}

// WithContext returns a copy of the service whose repository runs with ctx.
func (s *QuoteService) WithContext(ctx context.Context) QuoteServiceInterface {
	cp := *s
	cp.quoteRepo = s.quoteRepo.WithContext(ctx)
	return &cp
}

// NewQuoteService creates a new quote service
func NewQuoteService(quoteRepo repository.QuoteRepositoryInterface) *QuoteService {
	return &QuoteService{
		quoteRepo: quoteRepo,
	}
}

// NewQuoteServiceWithConfig creates a new quote service with configuration
func NewQuoteServiceWithConfig(quoteRepo repository.QuoteRepositoryInterface, cfg *QuoteServiceConfig) *QuoteService {
	svc := &QuoteService{quoteRepo: quoteRepo}
	if cfg != nil {
		svc.pdfService = cfg.PDFService
		svc.emailClient = cfg.EmailClient
	}
	return svc
}

// CreateQuote creates a new quote
func (s *QuoteService) CreateQuote(req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error) {
	// Validate request
	if err := s.validateQuoteRequest(req); err != nil {
		return nil, ErrInvalidQuoteData.WithMessage("Invalid quote data: " + err.Error())
	}

	// Create quote model from request
	quote := &models.Quote{
		CustomerID: req.CustomerID,
		ValidUntil: req.ValidUntil,
		Notes:      req.Notes,
		Terms:      req.Terms,
		CreatedBy:  userID,
		Status:     models.QuoteStatusDraft,
	}
	// No tax rate in the request: the tenant's default applies (T127).
	if req.TaxRate != nil {
		quote.TaxRate = *req.TaxRate
	} else {
		quote.ApplyDefaultTaxRate = true
	}

	// Create quote items
	quote.Items = make([]models.QuoteItem, len(req.Items))
	for i, itemReq := range req.Items {
		item := models.QuoteItem{
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			SortOrder:   itemReq.SortOrder,
		}
		// Calculate item total
		item.CalculateItemTotal()
		quote.Items[i] = item
	}

	// Calculate quote totals
	quote.CalculateTotals()

	// Create quote in database
	if err := s.quoteRepo.Create(quote); err != nil {
		return nil, fmt.Errorf("failed to create quote: %w", err)
	}

	// Fetch the created quote with all relationships
	createdQuote, err := s.quoteRepo.GetByID(quote.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created quote: %w", err)
	}

	response := createdQuote.ToResponse()
	return &response, nil
}

// GetQuote retrieves a quote by ID
func (s *QuoteService) GetQuote(id uuid.UUID) (*models.QuoteResponse, error) {
	quote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return nil, quoteLookupError(err)
	}

	response := quote.ToResponse()
	return &response, nil
}

// GetQuoteByNumber retrieves a quote by quote number
func (s *QuoteService) GetQuoteByNumber(quoteNumber string) (*models.QuoteResponse, error) {
	quote, err := s.quoteRepo.GetByQuoteNumber(quoteNumber)
	if err != nil {
		return nil, quoteLookupError(err)
	}

	response := quote.ToResponse()
	return &response, nil
}

// ListQuotes lists quotes with pagination and filters
func (s *QuoteService) ListQuotes(params *models.QuoteQueryParams) (*models.QuoteListResponse, error) {
	// Set default pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 50
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	quotes, total, err := s.quoteRepo.List(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list quotes: %w", err)
	}

	// Convert to response
	quoteResponses := make([]models.QuoteResponse, len(quotes))
	for i, quote := range quotes {
		quoteResponses[i] = quote.ToResponse()
	}

	return &models.QuoteListResponse{
		Quotes:   quoteResponses,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// UpdateQuote updates a quote
func (s *QuoteService) UpdateQuote(id uuid.UUID, req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error) {
	// Validate request
	if err := s.validateQuoteRequest(req); err != nil {
		return nil, ErrInvalidQuoteData.WithMessage("Invalid quote data: " + err.Error())
	}

	// Fetch existing quote
	existingQuote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return nil, quoteLookupError(err)
	}

	// Check if quote can be edited
	if !existingQuote.CanEdit() {
		return nil, ErrQuoteCannotBeEdited
	}

	// Update quote fields
	existingQuote.CustomerID = req.CustomerID
	existingQuote.ValidUntil = req.ValidUntil
	if req.TaxRate != nil {
		existingQuote.TaxRate = *req.TaxRate
	}
	existingQuote.Notes = req.Notes
	existingQuote.Terms = req.Terms
	existingQuote.UpdatedBy = &userID

	// Update quote items
	existingQuote.Items = make([]models.QuoteItem, len(req.Items))
	for i, itemReq := range req.Items {
		item := models.QuoteItem{
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			SortOrder:   itemReq.SortOrder,
		}
		// Calculate item total
		item.CalculateItemTotal()
		existingQuote.Items[i] = item
	}

	// Recalculate quote totals
	existingQuote.CalculateTotals()

	// Update quote in database
	if err := s.quoteRepo.Update(existingQuote); err != nil {
		return nil, fmt.Errorf("failed to update quote: %w", err)
	}

	// Fetch updated quote
	updatedQuote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated quote: %w", err)
	}

	response := updatedQuote.ToResponse()
	return &response, nil
}

// DeleteQuote soft deletes a quote
func (s *QuoteService) DeleteQuote(id uuid.UUID) error {
	// Fetch quote to check if it can be deleted
	quote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return quoteLookupError(err)
	}

	// Check if quote can be deleted
	if !quote.CanDelete() {
		return ErrQuoteCannotBeDeleted
	}

	// Delete quote
	if err := s.quoteRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete quote: %w", err)
	}

	return nil
}

// SendQuote marks a quote as sent and sends email with PDF
func (s *QuoteService) SendQuote(id uuid.UUID) error {
	// Fetch the quote first
	quote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return quoteLookupError(err)
	}

	// Check if quote can be sent (must be draft)
	if quote.Status != models.QuoteStatusDraft {
		return ErrQuoteCannotBeSent
	}

	// Mark as sent
	if err := s.quoteRepo.MarkAsSent(id); err != nil {
		return quoteTransitionError(err, ErrQuoteCannotBeSent)
	}

	// Send email with PDF asynchronously
	if s.pdfService != nil && s.emailClient != nil && quote.Customer != nil && quote.Customer.Email != "" {
		go func() {
			ctx := context.Background()

			// Generate PDF
			pdfResult, err := s.pdfService.GenerateQuotePDF(ctx, quote)
			if err != nil {
				logging.Error(ctx, "[QUOTE-SERVICE] Failed to generate PDF for quote", map[string]any{"quote_number": quote.QuoteNumber, "error": err})
				return
			}

			// Get download URL
			downloadURL, _, err := s.pdfService.GetQuotePDFURL(ctx, id)
			if errors.Is(err, ErrPDFStorageUnavailable) {
				downloadURL = "" // no storage (T256): PDF attached, no link
			} else if err != nil {
				logging.Error(ctx, "[QUOTE-SERVICE] Failed to get download URL for quote", map[string]any{"quote_number": quote.QuoteNumber, "error": err})
				// Continue without download URL
				downloadURL = ""
			}

			// Create attachment
			attachment := &email.Attachment{
				Filename:    pdfResult.FileName,
				Content:     pdfResult.Content,
				ContentType: "application/pdf",
			}

			// Send email
			if err := s.emailClient.SendQuoteEmail(ctx, quote.Customer.Email, quote, attachment, downloadURL); err != nil {
				logging.Error(ctx, "[QUOTE-SERVICE] Failed to send quote email", map[string]any{"quote_number": quote.QuoteNumber, "error": err})
				return
			}

			logging.Info(ctx, "[QUOTE-SERVICE] Successfully sent quote email", map[string]any{"quote_number": quote.QuoteNumber, "customer_email": quote.Customer.Email})
		}()
	}

	return nil
}

// GetQuotePDFURL returns a presigned URL for downloading the quote PDF
func (s *QuoteService) GetQuotePDFURL(id uuid.UUID) (string, time.Time, error) {
	// Verify quote exists
	quote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return "", time.Time{}, quoteLookupError(err)
	}

	if s.pdfService == nil {
		return "", time.Time{}, errors.New("PDF service not configured")
	}

	ctx := context.Background()

	// Check if PDF exists, generate if not
	url, expiresAt, err := s.pdfService.GetQuotePDFURL(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		_, err = s.pdfService.GenerateQuotePDF(ctx, quote)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to generate quote PDF: %w", err)
		}

		// Try to get URL again
		url, expiresAt, err = s.pdfService.GetQuotePDFURL(ctx, id)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to get quote PDF URL: %w", err)
		}
	}

	return url, expiresAt, nil
}

// DownloadQuotePDF returns the quote PDF content directly
func (s *QuoteService) DownloadQuotePDF(id uuid.UUID) ([]byte, string, error) {
	// Verify quote exists
	quote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return nil, "", quoteLookupError(err)
	}

	if s.pdfService == nil {
		return nil, "", errors.New("PDF service not configured")
	}

	ctx := context.Background()

	// Try to download existing PDF
	content, filename, err := s.pdfService.DownloadQuotePDF(ctx, id)
	if err != nil {
		// PDF doesn't exist, generate it first
		result, genErr := s.pdfService.GenerateQuotePDF(ctx, quote)
		if genErr != nil {
			return nil, "", fmt.Errorf("failed to generate quote PDF: %w", genErr)
		}
		return result.Content, result.FileName, nil
	}

	return content, filename, nil
}

// AcceptQuote marks a quote as accepted
func (s *QuoteService) AcceptQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsAccepted(id); err != nil {
		return quoteTransitionError(err, ErrQuoteCannotBeAccepted)
	}
	return nil
}

// RejectQuote marks a quote as rejected
func (s *QuoteService) RejectQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsRejected(id); err != nil {
		return quoteTransitionError(err, ErrQuoteCannotBeRejected)
	}
	return nil
}

// ReviseQuote takes a sent, viewed, expired or declined quote back to draft
// so it can be edited (only drafts are editable) and sent again.
func (s *QuoteService) ReviseQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsRevised(id); err != nil {
		return quoteTransitionError(err, ErrQuoteCannotBeRevised)
	}
	return nil
}

// GetQuotesByCustomer retrieves all quotes for a customer
func (s *QuoteService) GetQuotesByCustomer(customerID uuid.UUID) ([]models.QuoteResponse, error) {
	quotes, err := s.quoteRepo.GetByCustomer(customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer quotes: %w", err)
	}

	responses := make([]models.QuoteResponse, len(quotes))
	for i, quote := range quotes {
		responses[i] = quote.ToResponse()
	}

	return responses, nil
}

// GetQuoteStats retrieves quote statistics
func (s *QuoteService) GetQuoteStats() (*QuoteStats, error) {
	total, err := s.quoteRepo.GetQuoteCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	draft, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusDraft)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft count: %w", err)
	}

	sent, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusSent)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent count: %w", err)
	}

	accepted, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusAccepted)
	if err != nil {
		return nil, fmt.Errorf("failed to get accepted count: %w", err)
	}

	declined, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusDeclined)
	if err != nil {
		return nil, fmt.Errorf("failed to get declined count: %w", err)
	}

	expired, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusExpired)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired count: %w", err)
	}

	return &QuoteStats{
		TotalQuotes:    total,
		DraftQuotes:    draft,
		SentQuotes:     sent,
		AcceptedQuotes: accepted,
		DeclinedQuotes: declined,
		ExpiredQuotes:  expired,
	}, nil
}

// validateQuoteRequest validates a quote request
func (s *QuoteService) validateQuoteRequest(req *models.QuoteRequest) error {
	if req.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}

	if req.ValidUntil.IsZero() {
		return errors.New("valid_until is required")
	}

	if len(req.Items) == 0 {
		return errors.New("at least one item is required")
	}

	// Validate each item
	for i, item := range req.Items {
		if item.Description == "" {
			return fmt.Errorf("item %d: description is required", i+1)
		}
		if item.Quantity.IsZero() || item.Quantity.IsNegative() {
			return fmt.Errorf("item %d: quantity must be greater than 0", i+1)
		}
		if item.UnitPrice.IsNegative() {
			return fmt.Errorf("item %d: unit_price cannot be negative", i+1)
		}
	}

	return nil
}
