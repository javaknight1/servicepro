package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T028 on real Postgres (migration 000016): an accepted quote converts to
// exactly one job.

// seedPGQuote inserts a quote for the tenant's customer with two items.
func seedPGQuote(t *testing.T, db *gorm.DB, p *pgTenant, status models.QuoteStatus) *models.Quote {
	t.Helper()
	notes := "Use the service entrance."
	q := &models.Quote{
		ID: uuid.New(), CustomerID: p.customerID, Status: status,
		ValidUntil: time.Now().AddDate(0, 0, 30), Notes: &notes, CreatedBy: p.userID,
		Items: []models.QuoteItem{
			{Description: "Carpet extraction, lobby", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.RequireFromString("150.00"), SortOrder: 0},
			{Description: "Spot treatment", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("45.50"), SortOrder: 1},
		},
	}
	for i := range q.Items {
		q.Items[i].CalculateItemTotal()
	}
	q.CalculateTotals()
	require.NoError(t, db.WithContext(p.ctx).Create(q).Error)
	return q
}

func TestPostgres_ConvertQuoteToJob(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	svc := NewQuoteConversionService(db)
	q := seedPGQuote(t, db, p, models.QuoteStatusAccepted)

	job, err := svc.ConvertToJob(p.ctx, q.ID, nil, p.userID)
	require.NoError(t, err)
	assert.Equal(t, p.customerID, job.CustomerID)
	require.NotNil(t, job.QuoteID)
	assert.Equal(t, q.ID, *job.QuoteID)
	assert.Equal(t, models.JobStatusNew, job.Status)
	assert.Equal(t, models.JobTypeMaintenance, job.JobType)
	assert.Equal(t, "Carpet extraction, lobby", job.Title)
	assert.Contains(t, job.Description, "From quote "+q.QuoteNumber)
	assert.Contains(t, job.Description, "2 × Carpet extraction, lobby @ 150.00 = 300.00")
	assert.Contains(t, job.Description, "1 × Spot treatment @ 45.50 = 45.50")
	assert.Contains(t, job.Description, "Notes: Use the service entrance.")
	assert.InDelta(t, 345.50, job.EstimatedCost, 0.001)
	assert.Equal(t, "1 Main St", job.ServiceAddress.Street, "customer's billing address")

	var history int64
	require.NoError(t, db.WithContext(p.ctx).Model(&models.JobStatusTransition{}).Where("job_id = ?", job.ID).Count(&history).Error)
	assert.Equal(t, int64(1), history)

	// The quote now names its job ("View job").
	got, err := NewQuoteService(repository.NewQuoteRepository(db)).WithContext(p.ctx).GetQuote(q.ID)
	require.NoError(t, err)
	require.NotNil(t, got.JobID)
	assert.Equal(t, job.ID, *got.JobID)

	// A quote converts once; the 409 names the job.
	_, err = svc.ConvertToJob(p.ctx, q.ID, nil, p.userID)
	require.ErrorIs(t, err, ErrQuoteAlreadyConverted)
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, de.Status)
	assert.Equal(t, job.ID.String(), de.ResourceID)

	// Deleting the job frees the quote.
	require.NoError(t, db.WithContext(p.ctx).Delete(&models.Job{}, "id = ?", job.ID).Error)
	again, err := svc.ConvertToJob(p.ctx, q.ID, nil, p.userID)
	require.NoError(t, err)
	assert.NotEqual(t, job.ID, again.ID)
}

func TestPostgres_ConvertQuoteToJob_Options(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	svc := NewQuoteConversionService(db)
	q := seedPGQuote(t, db, p, models.QuoteStatusAccepted)

	access := "Key at the front desk"
	site := &models.Site{CustomerID: p.customerID, Name: "Harborview Lofts", AddressStreet: "220 W Spring St",
		AddressCity: "Chicago", AddressState: "IL", AccessNotes: &access, IsActive: true}
	require.NoError(t, repository.NewSiteRepository(db).Create(p.ctx, p.tenantID, site))

	title := "  Lobby carpets  "
	job, err := svc.ConvertToJob(p.ctx, q.ID, &models.ConvertQuoteToJobRequest{
		Title: &title, JobType: models.JobTypeRepair, SiteID: &site.ID,
	}, p.userID)
	require.NoError(t, err)
	assert.Equal(t, "Lobby carpets", job.Title)
	assert.Equal(t, models.JobTypeRepair, job.JobType)
	require.NotNil(t, job.SiteID)
	assert.Equal(t, site.ID, *job.SiteID)
	assert.Equal(t, "220 W Spring St", job.ServiceAddress.Street)
	require.NotNil(t, job.SpecialInstructions)
	assert.Equal(t, access, *job.SpecialInstructions)
}

func TestPostgres_ConvertQuoteToJob_OnlyAccepted(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	svc := NewQuoteConversionService(db)

	for _, st := range []models.QuoteStatus{models.QuoteStatusDraft, models.QuoteStatusSent, models.QuoteStatusDeclined, models.QuoteStatusExpired} {
		q := seedPGQuote(t, db, p, st)
		_, err := svc.ConvertToJob(p.ctx, q.ID, nil, p.userID)
		assert.ErrorIs(t, err, ErrQuoteNotConvertible, "status %s", st)
	}
	var n int64
	require.NoError(t, db.WithContext(p.ctx).Model(&models.Job{}).Count(&n).Error)
	assert.Zero(t, n)
}

func TestPostgres_ConvertQuoteToJob_AlreadyInvoiced(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	q := seedPGQuote(t, db, p, models.QuoteStatusAccepted)
	inv, err := NewInvoiceService(db).CreateInvoice(p.ctx, &models.Invoice{
		CustomerID: p.customerID, QuoteID: &q.ID, DueDate: time.Now().AddDate(0, 0, 30),
		Lines: []models.InvoiceLine{{Description: "Quoted work", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(345)}},
	}, p.userID)
	require.NoError(t, err)

	_, err = NewQuoteConversionService(db).ConvertToJob(p.ctx, q.ID, nil, p.userID)
	require.ErrorIs(t, err, ErrQuoteAlreadyInvoiced)
	de, _ := apperrors.As(err)
	assert.Equal(t, inv.ID.String(), de.ResourceID)
}

func TestPostgres_ConvertQuoteToJob_Concurrent(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	svc := NewQuoteConversionService(db)
	q := seedPGQuote(t, db, p, models.QuoteStatusAccepted)

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() {
			_, errs[i] = svc.ConvertToJob(p.ctx, q.ID, nil, p.userID)
		})
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
			continue
		}
		assert.True(t, errors.Is(err, ErrQuoteAlreadyConverted), "unexpected error: %v", err)
	}
	assert.Equal(t, 1, ok)
	var jobs int64
	require.NoError(t, db.WithContext(p.ctx).Model(&models.Job{}).Where("quote_id = ?", q.ID).Count(&jobs).Error)
	assert.Equal(t, int64(1), jobs)
}

func TestPostgres_ConvertQuoteToJob_CrossTenant(t *testing.T) {
	db := setupPostgres(t)
	a := seedPGTenant(t, db)
	b := seedPGTenant(t, db)
	svc := NewQuoteConversionService(db)
	q := seedPGQuote(t, db, a, models.QuoteStatusAccepted)

	// Tenant B can't convert (or learn about) tenant A's quote.
	_, err := svc.ConvertToJob(b.ctx, q.ID, nil, b.userID)
	require.ErrorIs(t, err, ErrQuoteNotFound)

	// Nor put A's job on one of B's sites.
	siteB := &models.Site{CustomerID: b.customerID, Name: "B site", AddressStreet: "9 Elm St", IsActive: true}
	require.NoError(t, repository.NewSiteRepository(db).Create(b.ctx, b.tenantID, siteB))
	_, err = svc.ConvertToJob(a.ctx, q.ID, &models.ConvertQuoteToJobRequest{SiteID: &siteB.ID}, a.userID)
	require.ErrorIs(t, err, ErrSiteCustomerMismatch)

	var n int64
	require.NoError(t, db.WithContext(context.Background()).Raw(`SELECT COUNT(*) FROM jobs WHERE quote_id = ?`, q.ID).Scan(&n).Error)
	assert.Zero(t, n)

	// B's quote listing doesn't leak A's job id either.
	_, err = NewQuoteService(repository.NewQuoteRepository(db)).WithContext(b.ctx).GetQuote(q.ID)
	assert.ErrorIs(t, err, ErrQuoteNotFound)
}
