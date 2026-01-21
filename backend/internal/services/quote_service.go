package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	// ErrQuoteNotFound is returned when a quote is not found
	ErrQuoteNotFound = errors.New("quote not found")
	// ErrInvalidQuoteData is returned when quote data is invalid
	ErrInvalidQuoteData = errors.New("invalid quote data")
	// ErrQuoteCannotBeEdited is returned when trying to edit a non-editable quote
	ErrQuoteCannotBeEdited = errors.New("quote cannot be edited in current status")
	// ErrQuoteCannotBeDeleted is returned when trying to delete a non-deletable quote
	ErrQuoteCannotBeDeleted = errors.New("quote can only be deleted in draft status")
	// ErrQuoteCannotBeSent is returned when trying to send a non-sendable quote
	ErrQuoteCannotBeSent = errors.New("quote cannot be sent")
	// ErrQuoteCannotBeAccepted is returned when trying to accept a non-acceptable quote
	ErrQuoteCannotBeAccepted = errors.New("quote cannot be accepted")
	// ErrQuoteCannotBeRejected is returned when trying to reject a non-rejectable quote
	ErrQuoteCannotBeRejected = errors.New("quote cannot be rejected")
)

// QuoteServiceInterface defines the interface for quote service
type QuoteServiceInterface interface {
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
	// SendQuote marks a quote as sent
	SendQuote(id uuid.UUID) error
	// AcceptQuote marks a quote as accepted
	AcceptQuote(id uuid.UUID) error
	// RejectQuote marks a quote as rejected
	RejectQuote(id uuid.UUID) error
	// GetQuotesByCustomer retrieves all quotes for a customer
	GetQuotesByCustomer(customerID uuid.UUID) ([]models.QuoteResponse, error)
	// GetQuoteStats retrieves quote statistics
	GetQuoteStats() (*QuoteStats, error)
}

// QuoteStats represents quote statistics
type QuoteStats struct {
	TotalQuotes    int64 `json:"total_quotes"`
	DraftQuotes    int64 `json:"draft_quotes"`
	SentQuotes     int64 `json:"sent_quotes"`
	AcceptedQuotes int64 `json:"accepted_quotes"`
	RejectedQuotes int64 `json:"rejected_quotes"`
	ExpiredQuotes  int64 `json:"expired_quotes"`
}

// QuoteService implements QuoteServiceInterface
type QuoteService struct {
	quoteRepo repository.QuoteRepositoryInterface
}

// NewQuoteService creates a new quote service
func NewQuoteService(quoteRepo repository.QuoteRepositoryInterface) *QuoteService {
	return &QuoteService{
		quoteRepo: quoteRepo,
	}
}

// CreateQuote creates a new quote
func (s *QuoteService) CreateQuote(req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error) {
	// Validate request
	if err := s.validateQuoteRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuoteData, err)
	}

	// Create quote model from request
	quote := &models.Quote{
		CustomerID: req.CustomerID,
		ValidUntil: req.ValidUntil,
		TaxRate:    req.TaxRate,
		Notes:      req.Notes,
		Terms:      req.Terms,
		CreatedBy:  userID,
		Status:     models.QuoteStatusDraft,
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
		return nil, ErrQuoteNotFound
	}

	response := quote.ToResponse()
	return &response, nil
}

// GetQuoteByNumber retrieves a quote by quote number
func (s *QuoteService) GetQuoteByNumber(quoteNumber string) (*models.QuoteResponse, error) {
	quote, err := s.quoteRepo.GetByQuoteNumber(quoteNumber)
	if err != nil {
		return nil, ErrQuoteNotFound
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
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuoteData, err)
	}

	// Fetch existing quote
	existingQuote, err := s.quoteRepo.GetByID(id)
	if err != nil {
		return nil, ErrQuoteNotFound
	}

	// Check if quote can be edited
	if !existingQuote.CanEdit() {
		return nil, ErrQuoteCannotBeEdited
	}

	// Update quote fields
	existingQuote.CustomerID = req.CustomerID
	existingQuote.ValidUntil = req.ValidUntil
	existingQuote.TaxRate = req.TaxRate
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
		return ErrQuoteNotFound
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

// SendQuote marks a quote as sent
func (s *QuoteService) SendQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsSent(id); err != nil {
		if err.Error() == "quote not found" {
			return ErrQuoteNotFound
		}
		return ErrQuoteCannotBeSent
	}
	return nil
}

// AcceptQuote marks a quote as accepted
func (s *QuoteService) AcceptQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsAccepted(id); err != nil {
		if err.Error() == "quote not found" {
			return ErrQuoteNotFound
		}
		return ErrQuoteCannotBeAccepted
	}
	return nil
}

// RejectQuote marks a quote as rejected
func (s *QuoteService) RejectQuote(id uuid.UUID) error {
	if err := s.quoteRepo.MarkAsRejected(id); err != nil {
		if err.Error() == "quote not found" {
			return ErrQuoteNotFound
		}
		return ErrQuoteCannotBeRejected
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

	rejected, err := s.quoteRepo.GetQuoteCountByStatus(models.QuoteStatusDeclined)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected count: %w", err)
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
		RejectedQuotes: rejected,
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
