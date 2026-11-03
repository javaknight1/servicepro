package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Quote → job conversion (T028).
var (
	// ErrQuoteNotConvertible: only accepted quotes become jobs.
	ErrQuoteNotConvertible = apperrors.Conflict("quote_not_accepted", "Only accepted quotes can be converted to a job")
	// ErrQuoteAlreadyConverted: the quote already has a job; resource_id is that job.
	ErrQuoteAlreadyConverted = apperrors.Conflict("already_converted", "This quote has already been converted to a job")
	// ErrQuoteAlreadyInvoiced: the quote was invoiced directly, so a job for
	// it could never be invoiced; resource_id is the invoice.
	ErrQuoteAlreadyInvoiced = apperrors.Conflict("already_invoiced", "This quote has already been invoiced")
)

// maxJobTitle is jobs.title's length.
const maxJobTitle = 200

// QuoteConversionService turns accepted quotes into jobs.
type QuoteConversionService struct {
	db *gorm.DB
}

// NewQuoteConversionService creates the service.
func NewQuoteConversionService(db *gorm.DB) *QuoteConversionService {
	return &QuoteConversionService{db: db}
}

// ConvertToJob creates a job from an accepted quote in one transaction: the
// quote's customer, the given site (if any), a title, the quote's line items
// and notes in the description, and the quote subtotal as the estimate. The
// job links back through jobs.quote_id, whose partial unique index lets a
// quote convert once; the quote row is locked so concurrent conversions
// serialise and the loser gets ErrQuoteAlreadyConverted with the job's ID.
// The quote is looked up in the caller's tenant (another tenant's quote is
// ErrQuoteNotFound).
func (s *QuoteConversionService) ConvertToJob(ctx context.Context, quoteID uuid.UUID, req *models.ConvertQuoteToJobRequest, userID uuid.UUID) (*models.JobResponse, error) {
	if req == nil {
		req = &models.ConvertQuoteToJobRequest{}
	}
	db := s.db.WithContext(ctx)
	var jobID uuid.UUID
	err := db.Transaction(func(tx *gorm.DB) error {
		var quote models.Quote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&quote, "id = ?", quoteID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrQuoteNotFound
			}
			return fmt.Errorf("load quote: %w", err)
		}
		if id, err := firstID(tx.Model(&models.Job{}).Where("quote_id = ?", quote.ID)); err != nil {
			return err
		} else if id != nil {
			return ErrQuoteAlreadyConverted.WithResource(id.String())
		}
		if quote.Status != models.QuoteStatusAccepted {
			return ErrQuoteNotConvertible
		}
		if id, err := firstID(tx.Model(&models.Invoice{}).
			Where("quote_id = ? AND status <> ?", quote.ID, models.InvoiceStatusCancelled)); err != nil {
			return err
		} else if id != nil {
			return ErrQuoteAlreadyInvoiced.WithResource(id.String())
		}

		var items []models.QuoteItem
		if err := tx.Where("quote_id = ?", quote.ID).Order("sort_order ASC").Find(&items).Error; err != nil {
			return fmt.Errorf("load quote items: %w", err)
		}
		var customer models.Customer
		if err := tx.First(&customer, "id = ?", quote.CustomerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCustomerNotFound
			}
			return fmt.Errorf("load customer: %w", err)
		}

		address := models.ServiceAddress{
			Street: customer.BillingAddressStreet, City: customer.BillingAddressCity,
			State: customer.BillingAddressState, Zip: customer.BillingAddressZip,
		}
		var specialInstructions *string
		if req.SiteID != nil {
			site, err := resolveJobSite(ctx, tx, customer.ID, *req.SiteID)
			if err != nil {
				return err
			}
			address = site.ServiceAddress()
			specialInstructions = site.AccessNotes
		}

		title := quoteJobTitle(&quote, items)
		if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
			title = strings.TrimSpace(*req.Title)
		}
		jobType := req.JobType
		if jobType == "" {
			jobType = models.JobTypeMaintenance
		}

		qid := quote.ID
		job := &models.Job{
			ID:                  uuid.New(),
			CustomerID:          customer.ID,
			SiteID:              req.SiteID,
			QuoteID:             &qid,
			PONumber:            quote.PONumber, // the quote's PO goes to the job and its invoice (T187)
			Title:               title,
			Description:         quoteJobDescription(&quote, items),
			JobType:             jobType,
			Status:              models.JobStatusNew,
			Priority:            models.JobPriorityNormal,
			ServiceAddress:      address,
			SpecialInstructions: specialInstructions,
			// The estimate is the quoted price before tax (jobs keep float
			// money until T166; the invoice re-reads the quote's decimals).
			EstimatedCost: quote.Subtotal.InexactFloat64(),
			CreatedBy:     userID,
		}
		if err := tx.Create(job).Error; err != nil {
			if uniqueViolation(err) {
				return ErrQuoteAlreadyConverted
			}
			return fmt.Errorf("create job: %w", err)
		}
		note := fmt.Sprintf("Converted from quote %s", quote.QuoteNumber)
		if err := tx.Create(newTransition(job.ID, models.JobStatusNew, models.JobStatusNew, models.ReasonOther, &note, userID)).Error; err != nil {
			return fmt.Errorf("record job history: %w", err)
		}
		jobID = job.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	var job models.Job
	if err := db.Preload("Customer").First(&job, "id = ?", jobID).Error; err != nil {
		return nil, fmt.Errorf("load job: %w", err)
	}
	resp := job.ToResponse()
	return &resp, nil
}

// uniqueViolation reports a Postgres unique-constraint error (23505).
func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// firstID returns the id of the first row q matches, or nil.
func firstID(q *gorm.DB) (*uuid.UUID, error) {
	var ids []uuid.UUID
	if err := q.Limit(1).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &ids[0], nil
}

// quoteJobTitle is the first line item's description (its first line,
// shortened to fit), or "Quote <number>" for a quote without items.
func quoteJobTitle(q *models.Quote, items []models.QuoteItem) string {
	for _, it := range items {
		t := strings.TrimSpace(strings.SplitN(it.Description, "\n", 2)[0])
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > maxJobTitle {
			t = string([]rune(t)[:maxJobTitle-1]) + "…"
		}
		return t
	}
	return "Quote " + q.QuoteNumber
}

// quoteJobDescription lists what was quoted, so the crew sees the scope:
//
//	From quote Q-000012 (accepted):
//	- 2 × Carpet extraction, lobby @ 150.00 = 300.00
//	Notes: Use the service entrance.
func quoteJobDescription(q *models.Quote, items []models.QuoteItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From quote %s (accepted):", q.QuoteNumber)
	for _, it := range items {
		fmt.Fprintf(&b, "\n- %s × %s @ %s = %s", it.Quantity.String(), strings.TrimSpace(it.Description),
			it.UnitPrice.StringFixed(2), it.Total.StringFixed(2))
	}
	if q.Notes != nil && strings.TrimSpace(*q.Notes) != "" {
		fmt.Fprintf(&b, "\nNotes: %s", strings.TrimSpace(*q.Notes))
	}
	return b.String()
}
