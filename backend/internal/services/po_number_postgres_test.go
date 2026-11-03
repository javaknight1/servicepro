package services

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T187 on real Postgres (migration 000018): PO numbers on quotes and
// agreements flow to their jobs, and invoices use the most specific PO.

func TestPostgres_PONumber_QuoteToJobToInvoice(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50") // customer default PO-7731
	quotes := NewQuoteService(repository.NewQuoteRepository(db)).WithContext(f.p.ctx)

	po := "  Q-PO-4410 "
	created, err := quotes.CreateQuote(&models.QuoteRequest{
		CustomerID: f.p.customerID, ValidUntil: time.Now().AddDate(0, 0, 30), PONumber: &po,
		Items: []models.QuoteItemRequest{{Description: "Carpet extraction", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}},
	}, f.p.userID)
	require.NoError(t, err)
	require.NotNil(t, created.PONumber)
	assert.Equal(t, "Q-PO-4410", *created.PONumber, "trimmed")

	// Updating without a PO clears it; with one sets it.
	blank := " "
	updated, err := quotes.UpdateQuote(created.ID, &models.QuoteRequest{
		CustomerID: f.p.customerID, ValidUntil: time.Now().AddDate(0, 0, 30), PONumber: &blank,
		Items: []models.QuoteItemRequest{{Description: "Carpet extraction", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}},
	}, f.p.userID)
	require.NoError(t, err)
	assert.Nil(t, updated.PONumber)
	po = "Q-PO-4411"
	_, err = quotes.UpdateQuote(created.ID, &models.QuoteRequest{
		CustomerID: f.p.customerID, ValidUntil: time.Now().AddDate(0, 0, 30), PONumber: &po,
		Items: []models.QuoteItemRequest{{Description: "Carpet extraction", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}},
	}, f.p.userID)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Quote{}).Where("id = ?", created.ID).
		UpdateColumn("status", models.QuoteStatusAccepted).Error)

	job, err := NewQuoteConversionService(db).ConvertToJob(f.p.ctx, created.ID, nil, f.p.userID)
	require.NoError(t, err)
	require.NotNil(t, job.PONumber)
	assert.Equal(t, "Q-PO-4411", *job.PONumber, "the job carries the quote's PO")

	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Job{}).Where("id = ?", job.ID).
		Update("status", models.JobStatusCompleted).Error)
	inv, err := f.svc.InvoiceJob(f.p.ctx, job.ID, f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, "Q-PO-4411", inv.PONumber, "the job's PO beats the customer's default")

	// A job without its own PO falls back to the customer's default.
	inv, err = f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, "PO-7731", inv.PONumber)
}

func TestPostgres_PONumber_JobCreateUpdate(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	jobs := NewJobService(repository.NewJobRepository(db), repository.NewCustomerRepository(db), db).WithContext(p.ctx)

	po := "WO-881"
	created, err := jobs.CreateJob(&models.CreateJobRequest{
		CustomerID: p.customerID, Title: "Strip and wax", JobType: models.JobTypeMaintenance, PONumber: &po,
	}, p.userID)
	require.NoError(t, err)
	require.NotNil(t, created.PONumber)
	assert.Equal(t, "WO-881", *created.PONumber)

	none := ""
	updated, err := jobs.UpdateJob(created.ID, &models.UpdateJobRequest{PONumber: &none}, p.userID, models.UserRole("admin"))
	require.NoError(t, err)
	assert.Nil(t, updated.PONumber)
}

func TestPostgres_PONumber_AgreementVisitsAndConsolidation(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db) // customer default 7731-HV
	agreements := NewAgreementService(db, repository.NewAgreementRepository(db))

	po := "LOFTS-55"
	updated, err := agreements.Update(f.p.ctx, f.monthly.ID, &models.UpdateAgreementRequest{PONumber: &po})
	require.NoError(t, err)
	require.NotNil(t, updated.PONumber)
	assert.Equal(t, "LOFTS-55", *updated.PONumber)

	// A generated visit's job carries the agreement's PO.
	a, err := repository.NewAgreementRepository(db).Get(f.p.ctx, f.p.tenantID, f.monthly.ID)
	require.NoError(t, err)
	created, err := generateVisit(f.p.ctx, db, a, f.lofts, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), nyLoc)
	require.NoError(t, err)
	require.True(t, created)
	var visitJob models.Job
	require.NoError(t, db.WithContext(f.p.ctx).Where("agreement_id = ? AND po_number IS NOT NULL", a.ID).First(&visitJob).Error)
	require.NotNil(t, visitJob.PONumber)
	assert.Equal(t, "LOFTS-55", *visitJob.PONumber)

	// Lofts only: the monthly agreement (LOFTS-55) and the approved extra
	// work. The work has no PO of its own, so its PO is the customer's
	// default: the POs differ and the invoice falls back to the default.
	req := f.october()
	req.SiteID = &f.lofts.ID
	prev, err := f.svc.PreviewConsolidatedInvoice(f.p.ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "7731-HV", prev.PONumber)

	// Once the extra work is under the same PO, every line agrees.
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Job{}).Where("id = ?", f.approvedJob).
		Update("po_number", "LOFTS-55").Error)
	prev, err = f.svc.PreviewConsolidatedInvoice(f.p.ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "LOFTS-55", prev.PONumber)
	inv, err := f.svc.ConsolidateInvoice(f.p.ctx, req, f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, "LOFTS-55", inv.PONumber)

	// The whole customer mixes POs: the default.
	prev, err = f.svc.PreviewConsolidatedInvoice(f.p.ctx, f.october())
	require.NoError(t, err)
	assert.Equal(t, "7731-HV", prev.PONumber)

	// No default and mixed POs: none (send then asks for one if required).
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).Exec(
		`UPDATE customers SET default_po_number = NULL WHERE id = ?`, f.p.customerID).Error)
	prev, err = f.svc.PreviewConsolidatedInvoice(f.p.ctx, f.october())
	require.NoError(t, err)
	assert.Empty(t, prev.PONumber)
}
