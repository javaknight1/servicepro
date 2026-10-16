package services

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func invoiceFor(f *isolationFixture, jobID, quoteID *uuid.UUID) *models.Invoice {
	now := time.Now()
	return &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-" + uuid.NewString()[:8], IssueDate: now, DueDate: now.Add(24 * time.Hour),
		JobID: jobID, QuoteID: quoteID, Lines: oneLine(100),
	}
}

func seedJob(t *testing.T, f *isolationFixture, status models.JobStatus) *models.Job {
	t.Helper()
	job := &models.Job{ID: uuid.New(), JobNumber: "JOB-" + uuid.NewString()[:8], CustomerID: f.custA.ID, Title: "Fix AC",
		JobType: models.JobTypeRepair, Status: status, Priority: models.JobPriorityNormal, CreatedBy: f.userA}
	require.NoError(t, f.db.WithContext(f.ctxA).Create(job).Error)
	return job
}

func seedQuote(t *testing.T, f *isolationFixture, status models.QuoteStatus) *models.Quote {
	t.Helper()
	q := &models.Quote{ID: uuid.New(), QuoteNumber: "Q-" + uuid.NewString()[:8], CustomerID: f.custA.ID, Status: status,
		ValidUntil: time.Now().Add(72 * time.Hour), CreatedBy: f.userA}
	require.NoError(t, f.db.WithContext(f.ctxA).Create(q).Error)
	return q
}

// Audit 7.M5: a job can't be invoiced twice; a cancelled invoice frees it.
func TestCreateInvoice_RejectsDoubleInvoicingAJob(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	job := seedJob(t, f, models.JobStatusCompleted)

	first, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, &job.ID, nil), f.userA)
	require.NoError(t, err)
	_, err = svc.CreateInvoice(f.ctxA, invoiceFor(f, &job.ID, nil), f.userA)
	assert.ErrorIs(t, err, ErrAlreadyInvoiced)

	_, err = svc.CancelInvoice(f.ctxA, first.ID, "wrong amount")
	require.NoError(t, err)
	_, err = svc.CreateInvoice(f.ctxA, invoiceFor(f, &job.ID, nil), f.userA)
	assert.NoError(t, err, "a cancelled invoice doesn't block re-invoicing")

	cancelled := seedJob(t, f, models.JobStatusCancelled)
	_, err = svc.CreateInvoice(f.ctxA, invoiceFor(f, &cancelled.ID, nil), f.userA)
	assert.ErrorIs(t, err, ErrInvoiceSourceStatus)

	_, err = svc.CreateInvoice(f.ctxB, &models.Invoice{CustomerID: f.custB.ID, JobID: &job.ID, Lines: oneLine(1),
		IssueDate: time.Now(), DueDate: time.Now().Add(time.Hour), InvoiceNumber: "INV-B-X"}, f.userA)
	assert.ErrorIs(t, err, ErrInvoiceSourceNotFound, "another tenant's job is not found")
}

// Concurrent "create invoice from job" requests: the job row lock lets only
// one through (without it, every request sees no invoice yet).
func TestPostgres_CreateInvoice_ConcurrentSameJobInvoicedOnce(t *testing.T) {
	db := setupPostgres(t)
	tn := seedPGTenant(t, db)
	svc := NewInvoiceService(db)
	job := &models.Job{ID: uuid.New(), JobNumber: "JOB-RACE-1", CustomerID: tn.customerID, Title: "Fix AC",
		JobType: models.JobTypeRepair, Status: models.JobStatusCompleted, Priority: models.JobPriorityNormal, CreatedBy: tn.userID}
	require.NoError(t, db.WithContext(tn.ctx).Create(job).Error)

	const workers = 6
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			now := time.Now()
			_, errs[i] = svc.CreateInvoice(tn.ctx, &models.Invoice{
				CustomerID: tn.customerID, InvoiceNumber: "INV-JOB-" + uuid.NewString()[:8], IssueDate: now, DueDate: now.Add(time.Hour),
				JobID: &job.ID, Lines: oneLine(100),
			}, tn.userID)
		}(i)
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			assert.ErrorIs(t, err, ErrAlreadyInvoiced)
		}
	}
	assert.Equal(t, 1, ok)
}

// Only accepted quotes can be invoiced, once.
func TestCreateInvoice_QuoteMustBeAcceptedAndInvoicedOnce(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	for _, st := range []models.QuoteStatus{models.QuoteStatusDraft, models.QuoteStatusSent, models.QuoteStatusDeclined, models.QuoteStatusExpired} {
		q := seedQuote(t, f, st)
		_, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, nil, &q.ID), f.userA)
		assert.ErrorIs(t, err, ErrQuoteNotAccepted, "status %s", st)
	}

	q := seedQuote(t, f, models.QuoteStatusAccepted)
	_, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, nil, &q.ID), f.userA)
	require.NoError(t, err)
	_, err = svc.CreateInvoice(f.ctxA, invoiceFor(f, nil, &q.ID), f.userA)
	assert.ErrorIs(t, err, ErrAlreadyInvoiced)
}
