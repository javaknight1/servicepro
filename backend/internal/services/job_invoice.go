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
	"gorm.io/gorm/clause"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// One-click invoice from a completed job (T029), pre-filled from what was
// recorded on the job (T030).
var (
	// ErrJobNotInvoiceable: only completed jobs are invoiced.
	ErrJobNotInvoiceable = apperrors.Conflict("job_not_completed", "Only completed jobs can be invoiced")
	// ErrJobAlreadyInvoiced: the job has an invoice (its own or a
	// consolidated one); resource_id is that invoice.
	ErrJobAlreadyInvoiced = apperrors.Conflict("already_invoiced", "This job has already been invoiced")
	// ErrAgreementVisitInvoice: agreement visits are billed by the
	// agreement (consolidated invoice, T188), never one by one.
	ErrAgreementVisitInvoice = apperrors.Conflict("agreement_visit", "Agreement visits are billed on the customer's consolidated invoice")
)

var secondsPerHour = decimal.NewFromInt(3600)

// InvoiceJob creates a draft invoice for a completed job and moves the job to
// invoiced, all in one transaction (T029). The job row is locked first, so
// concurrent calls serialise: the first creates the invoice, the others get
// ErrJobAlreadyInvoiced naming it. Lines are built server-side (T030):
//
//   - a job converted from a quote (T028) bills the quote's lines at the
//     quoted prices and links the quote;
//   - otherwise labor from the actual time on site × the tenant's laborRate
//     setting, or — no rate or no recorded time — one line for the job's
//     amount (total, else actual cost, else estimate);
//   - then each billable material at its unit cost.
//
// Every line carries the job's site name. The job's completion notes and
// customer-visible notes go into the invoice notes; the customer's standing
// PO, the tenant's default payment term and tax rate apply. Totals are
// computed by insertInvoice like any other invoice. The job is looked up in
// the caller's tenant (another tenant's job is ErrJobNotFound).
func (s *InvoiceService) InvoiceJob(ctx context.Context, jobID, userID uuid.UUID) (*models.Invoice, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var invoiceID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", jobID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrJobNotFound
			}
			return fmt.Errorf("load job: %w", err)
		}
		existing, err := jobInvoiceID(tx, &job)
		if err != nil {
			return err
		}
		if existing != nil {
			return ErrJobAlreadyInvoiced.WithResource(existing.String())
		}
		switch job.Status {
		case models.JobStatusCompleted:
		case models.JobStatusInvoiced, models.JobStatusPaid:
			return ErrJobAlreadyInvoiced
		default:
			return ErrJobNotInvoiceable
		}
		if job.AgreementID != nil {
			return ErrAgreementVisitInvoice
		}

		invoice, err := s.jobInvoiceDraft(ctx, tx, tenantID, &job)
		if err != nil {
			return err
		}
		invoice.CreatedBy = userID
		if err := insertInvoice(tx, invoice, userID); err != nil {
			return err
		}
		invoiceID = invoice.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, invoiceID)
}

// jobInvoiceDraft builds the (unsaved) draft invoice for a locked job.
func (s *InvoiceService) jobInvoiceDraft(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID, job *models.Job) (*models.Invoice, error) {
	customer, err := s.invoiceCustomer(tx, job.CustomerID)
	if err != nil {
		return nil, err
	}
	loc, err := s.tenantLoc(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	jobID := job.ID
	invoice := &models.Invoice{
		CustomerID: customer.ID, JobID: &jobID, Status: models.InvoiceStatusDraft,
		IssueDate: models.CivilDate(time.Now(), loc),
	}
	if customer.DefaultPONumber != nil {
		invoice.PONumber = *customer.DefaultPONumber
	}
	if err := s.applyTenantInvoiceDefaults(ctx, invoice); err != nil {
		return nil, err
	}
	due, err := s.calculateDueDate(ctx, invoice.IssueDate, invoice.PaymentTermID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate due date: %w", err)
	}
	invoice.DueDate = due

	lines, quoteID, err := quoteLinesForJob(tx, job)
	if err != nil {
		return nil, err
	}
	invoice.QuoteID = quoteID
	if quoteID == nil {
		settings := models.TenantSettingsInTx(tx, tenantID)
		work, err := jobWorkLine(tx, job, settings.LaborRate)
		if err != nil {
			return nil, err
		}
		lines = append(lines, work...)
	}
	materials, err := jobMaterialLines(tx, job.ID)
	if err != nil {
		return nil, err
	}
	lines = append(lines, materials...)
	if len(lines) == 0 {
		// Nothing priced on the job: a zero line the office fills in.
		lines = []models.InvoiceLine{{Description: jobLineDescription(job), Quantity: decimal.NewFromInt(1), UnitPrice: decimal.Zero}}
	}

	siteName, err := jobSiteName(tx, job)
	if err != nil {
		return nil, err
	}
	for i := range lines {
		lines[i].Taxable = true
		lines[i].SortOrder = i
		lines[i].SiteName = nonEmpty(siteName)
		if lines[i].UnitOfMeasure == "" {
			lines[i].UnitOfMeasure = "each"
		}
	}
	invoice.Lines = lines

	notes, err := jobInvoiceNotes(tx, job)
	if err != nil {
		return nil, err
	}
	invoice.Notes = notes
	return invoice, nil
}

// quoteLinesForJob returns the accepted quote's lines for a job converted
// from it (T028), and the quote's ID; nil, nil when the job has no quote.
func quoteLinesForJob(tx *gorm.DB, job *models.Job) ([]models.InvoiceLine, *uuid.UUID, error) {
	if job.QuoteID == nil {
		return nil, nil, nil
	}
	var quote models.Quote
	if err := tx.First(&quote, "id = ?", *job.QuoteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil // quote deleted since: bill the job itself
		}
		return nil, nil, fmt.Errorf("load quote: %w", err)
	}
	if quote.Status != models.QuoteStatusAccepted {
		return nil, nil, nil
	}
	var items []models.QuoteItem
	if err := tx.Where("quote_id = ?", quote.ID).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("load quote items: %w", err)
	}
	lines := make([]models.InvoiceLine, 0, len(items))
	for _, it := range items {
		if !it.Quantity.IsPositive() || it.UnitPrice.IsNegative() {
			continue
		}
		lines = append(lines, models.InvoiceLine{Description: it.Description, Quantity: it.Quantity, UnitPrice: it.UnitPrice})
	}
	id := quote.ID
	return lines, &id, nil
}

// jobWorkLine is the line for the work itself: labor (hours on site × the
// hourly rate) when both are known, else the job's amount, else nothing.
func jobWorkLine(tx *gorm.DB, job *models.Job, rate *decimal.Decimal) ([]models.InvoiceLine, error) {
	hours := laborHours(job)
	if rate != nil && rate.IsPositive() && hours.IsPositive() {
		return []models.InvoiceLine{{
			Description:   fmt.Sprintf("Labor · %s · %s h", jobLineDescription(job), hours.StringFixed(2)),
			Quantity:      hours,
			UnitPrice:     rate.Round(2),
			UnitOfMeasure: "hour",
		}}, nil
	}
	// The job's money columns are float64 in Go (T166 open); read them as
	// numeric straight from the database instead.
	var row struct{ Amount decimal.Decimal }
	if err := tx.Model(&models.Job{}).
		Select("COALESCE(NULLIF(total_amount, 0), NULLIF(actual_cost, 0), estimated_cost, 0) AS amount").
		Where("id = ?", job.ID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("load job amount: %w", err)
	}
	if !row.Amount.IsPositive() {
		return nil, nil
	}
	return []models.InvoiceLine{{Description: jobLineDescription(job), Quantity: decimal.NewFromInt(1), UnitPrice: row.Amount.Round(2)}}, nil
}

// laborHours is the actual time on site in hours (2 decimals): from the
// actual start/end timestamps, else the recorded actual duration.
func laborHours(job *models.Job) decimal.Decimal {
	if job.ActualStartAt != nil && job.ActualEndAt != nil && *job.ActualEndAt > *job.ActualStartAt {
		return decimal.NewFromInt(*job.ActualEndAt - *job.ActualStartAt).Div(secondsPerHour).Round(2)
	}
	if job.ActualDuration > 0 {
		return decimal.NewFromInt(int64(job.ActualDuration)).Div(decimal.NewFromInt(60)).Round(2)
	}
	return decimal.Zero
}

// jobMaterialRow is a job material read as numeric (job_materials is float64
// in Go).
type jobMaterialRow struct {
	Name        string
	Description *string
	SKU         *string
	Quantity    decimal.Decimal
	Unit        string
	UnitCost    decimal.Decimal
}

// jobMaterialLines is a line per billable material at its unit cost.
func jobMaterialLines(tx *gorm.DB, jobID uuid.UUID) ([]models.InvoiceLine, error) {
	var rows []jobMaterialRow
	if err := tx.Model(&models.JobMaterial{}).
		Select("name, description, sku, quantity, unit, unit_cost").
		Where("job_id = ? AND billable", jobID).
		Order("created_at ASC, id ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load job materials: %w", err)
	}
	lines := make([]models.InvoiceLine, 0, len(rows))
	for _, m := range rows {
		if !m.Quantity.IsPositive() || m.UnitCost.IsNegative() {
			continue
		}
		desc := strings.TrimSpace(m.Name)
		if m.SKU != nil && strings.TrimSpace(*m.SKU) != "" {
			desc += " (" + strings.TrimSpace(*m.SKU) + ")"
		}
		if m.Description != nil && strings.TrimSpace(*m.Description) != "" {
			desc += " — " + strings.TrimSpace(*m.Description)
		}
		lines = append(lines, models.InvoiceLine{
			Description: desc, Quantity: m.Quantity, UnitPrice: m.UnitCost.Round(2),
			UnitOfMeasure: strings.TrimSpace(m.Unit),
		})
	}
	return lines, nil
}

// jobLineDescription is "<title> · <job number>".
func jobLineDescription(job *models.Job) string {
	return fmt.Sprintf("%s · %s", job.Title, job.JobNumber)
}

// jobSiteName is the job's site name, or "".
func jobSiteName(tx *gorm.DB, job *models.Job) (string, error) {
	if job.SiteID == nil {
		return "", nil
	}
	var names []string
	if err := tx.Model(&models.Site{}).Where("id = ?", *job.SiteID).Limit(1).Pluck("name", &names).Error; err != nil {
		return "", fmt.Errorf("load site: %w", err)
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[0], nil
}

// jobInvoiceNotes is what the crew recorded for the customer: the job's
// completion notes, then its customer-visible notes (internal notes never
// reach an invoice).
func jobInvoiceNotes(tx *gorm.DB, job *models.Job) (string, error) {
	parts := []string{fmt.Sprintf("Job %s: %s", job.JobNumber, job.Title)}
	if job.CompletionNotes != nil && strings.TrimSpace(*job.CompletionNotes) != "" {
		parts = append(parts, strings.TrimSpace(*job.CompletionNotes))
	}
	var notes []string
	if err := tx.Model(&models.JobNote{}).
		Where("job_id = ? AND NOT is_internal", job.ID).
		Order("created_at ASC, id ASC").Pluck("note", &notes).Error; err != nil {
		return "", fmt.Errorf("load job notes: %w", err)
	}
	for _, n := range notes {
		if n = strings.TrimSpace(n); n != "" {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, "\n\n"), nil
}

// jobInvoiceID is the job's invoice: a non-cancelled invoice for it, else
// the consolidated invoice that billed it (directly or as an agreement
// visit). Nil when the job hasn't been invoiced.
func jobInvoiceID(tx *gorm.DB, job *models.Job) (*uuid.UUID, error) {
	id, err := firstID(tx.Model(&models.Invoice{}).
		Where("job_id = ? AND status <> ?", job.ID, models.InvoiceStatusCancelled).
		Order("created_at DESC"))
	if err != nil || id != nil {
		return id, err
	}
	return consolidatedInvoiceForJob(tx, job)
}

// consolidatedInvoiceForJob is the consolidated invoice whose claims
// (invoice_billed_items, T188) cover the job — as extra work, or as an
// agreement visit — or nil.
func consolidatedInvoiceForJob(tx *gorm.DB, job *models.Job) (*uuid.UUID, error) {
	q := tx.Model(&models.InvoiceBilledItem{})
	if job.AgreementID != nil {
		q = q.Where("(job_id = ? OR agreement_visit_id IN (SELECT v.id FROM agreement_visits v WHERE v.job_id = ?))", job.ID, job.ID)
	} else {
		q = q.Where("job_id = ?", job.ID)
	}
	var ids []uuid.UUID
	if err := q.Limit(1).Pluck("invoice_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load billed items: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &ids[0], nil
}
