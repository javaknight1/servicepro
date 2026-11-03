package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

// Consolidated monthly invoicing (T188): one invoice per customer (or site)
// per period, a line per site per agreement plus each completed extra-work
// job.
//
// Period semantics: period_start..period_end are tenant-local calendar
// dates, both inclusive.
//   - Monthly agreements bill price_per_period once per calendar month the
//     period touches (no proration): Oct 15–Nov 14 bills October and
//     November in full. A month is billed when the agreement ran in it
//     (start/end dates) and is active, or had a completed visit in it.
//   - Per-visit agreements bill completed visits dated inside the period ×
//     price_per_period.
//   - Extra work is every completed job without an agreement that finished
//     on or before the end of the period (earlier unbilled work is swept in
//     too) and has no other invoice.
//
// Double billing (T225): each month, visit and job is claimed in
// invoice_billed_items in the invoice's transaction. A second consolidation
// for an overlapping period skips what is already claimed; when nothing is
// left it is 409 nothing_to_invoice. If a concurrent call claims an item
// first, the later one rolls back with 409 already_invoiced.

// ErrNothingToInvoice: no unbilled work for the customer and period.
var ErrNothingToInvoice = apperrors.Conflict("nothing_to_invoice", "There is no unbilled work for this customer and period")

// ErrConsolidationRace: an item was invoiced by a concurrent request.
var ErrConsolidationRace = apperrors.Conflict("already_invoiced", "Some of this work was just invoiced elsewhere; preview the invoice again")

// ErrConsolidationSiteNotFound: site_id is not one of the customer's sites.
var ErrConsolidationSiteNotFound = apperrors.NotFound("site_not_found", "Site not found")

// maxConsolidationDays bounds the period (a year, plus a leap day).
const maxConsolidationDays = 366

// consolidatedLine is a would-be line plus the claims that bill it.
type consolidatedLine struct {
	models.ConsolidatedInvoiceLine
	month    *time.Time  // agreement_month: the calendar month claimed
	visitIDs []uuid.UUID // visits claimed by this line
	jobIDs   []uuid.UUID // jobs moved to invoiced with this line
}

// consolidation is everything one consolidated invoice would bill.
type consolidation struct {
	customer   *models.Customer
	start, end time.Time // civil dates (midnight UTC)
	lines      []consolidatedLine
}

// parseConsolidationPeriod validates the request's dates.
func parseConsolidationPeriod(req *models.ConsolidateInvoiceRequest) (time.Time, time.Time, error) {
	fields := map[string]string{}
	start, err := time.Parse(models.DateLayout, strings.TrimSpace(req.PeriodStart))
	if err != nil {
		fields["period_start"] = "must be a date (YYYY-MM-DD)"
	}
	end, err := time.Parse(models.DateLayout, strings.TrimSpace(req.PeriodEnd))
	if err != nil {
		fields["period_end"] = "must be a date (YYYY-MM-DD)"
	}
	if len(fields) == 0 {
		switch {
		case end.Before(start):
			fields["period_end"] = "must be on or after period_start"
		case end.Sub(start) > maxConsolidationDays*24*time.Hour:
			fields["period_end"] = "the period can be at most a year"
		}
	}
	if len(fields) > 0 {
		return time.Time{}, time.Time{}, apperrors.Validation("Invalid billing period", fields)
	}
	return start, end, nil
}

// monthStart is the first day of t's month.
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// pluralVisits is "1 visit" / "22 visits".
func pluralVisits(n int) string {
	if n == 1 {
		return "1 visit"
	}
	return fmt.Sprintf("%d visits", n)
}

// consolidationVisit is a completed, unbilled agreement visit.
type consolidationVisit struct {
	ID          uuid.UUID
	AgreementID uuid.UUID
	VisitDate   time.Time
	JobID       uuid.UUID
}

// consolidationJob is a completed, unbilled extra-work job.
type consolidationJob struct {
	ID        uuid.UUID
	JobNumber string
	Title     string
	SiteID    *uuid.UUID
	Amount    decimal.Decimal
}

// buildConsolidation collects the customer's unbilled work for the period.
// db carries the caller's tenant (request context or a transaction).
func (s *InvoiceService) buildConsolidation(db *gorm.DB, tenantID uuid.UUID, req *models.ConsolidateInvoiceRequest, loc *time.Location) (*consolidation, error) {
	start, end, err := parseConsolidationPeriod(req)
	if err != nil {
		return nil, err
	}
	customer, err := s.invoiceCustomer(db, req.CustomerID)
	if err != nil {
		return nil, err
	}
	c := &consolidation{customer: customer, start: start, end: end}

	// Site names, including sites deleted since (old work still names them).
	var sites []models.Site
	if err := db.Unscoped().Where("customer_id = ?", customer.ID).Find(&sites).Error; err != nil {
		return nil, fmt.Errorf("load sites: %w", err)
	}
	siteNames := map[uuid.UUID]string{}
	for _, st := range sites {
		siteNames[st.ID] = st.Name
	}
	if req.SiteID != nil {
		found := false
		for _, st := range sites {
			if st.ID == *req.SiteID && !st.DeletedAt.Valid {
				found = true
			}
		}
		if !found {
			return nil, ErrConsolidationSiteNotFound
		}
	}

	if err := s.addAgreementLines(db, tenantID, c, req.SiteID, siteNames); err != nil {
		return nil, err
	}
	if err := s.addExtraWorkLines(db, tenantID, c, req.SiteID, siteNames, loc); err != nil {
		return nil, err
	}

	sort.SliceStable(c.lines, func(i, j int) bool {
		a, b := c.lines[i], c.lines[j]
		if (a.SiteName == "") != (b.SiteName == "") {
			return b.SiteName == "" // work without a site last
		}
		if a.SiteName != b.SiteName {
			return a.SiteName < b.SiteName
		}
		if (a.Kind == models.ConsolidatedLineExtraWork) != (b.Kind == models.ConsolidatedLineExtraWork) {
			return b.Kind == models.ConsolidatedLineExtraWork // agreements first
		}
		if a.Month != b.Month {
			return a.Month < b.Month
		}
		if a.JobNumber != b.JobNumber {
			return a.JobNumber < b.JobNumber
		}
		return a.Description < b.Description
	})
	return c, nil
}

// addAgreementLines adds a line per monthly agreement per calendar month and
// per per-visit agreement.
func (s *InvoiceService) addAgreementLines(db *gorm.DB, tenantID uuid.UUID, c *consolidation, siteID *uuid.UUID, siteNames map[uuid.UUID]string) error {
	q := db.Where("customer_id = ?", c.customer.ID)
	if siteID != nil {
		q = q.Where("site_id = ?", *siteID)
	}
	var agreements []models.ServiceAgreement
	if err := q.Order("title, id").Find(&agreements).Error; err != nil {
		return fmt.Errorf("load agreements: %w", err)
	}
	if len(agreements) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(agreements))
	for i := range agreements {
		ids[i] = agreements[i].ID
	}

	firstMonth := monthStart(c.start)
	lastMonthEnd := monthStart(c.end).AddDate(0, 1, -1)

	// Completed visits not billed yet. Raw SQL is not rewritten by the
	// tenancy plugin: tenant_id is filtered explicitly (and RLS backs it).
	var visits []consolidationVisit
	if err := db.Raw(`
		SELECT v.id, v.agreement_id, v.visit_date, v.job_id
		FROM agreement_visits v
		JOIN jobs j ON j.id = v.job_id AND j.tenant_id = v.tenant_id
		WHERE v.tenant_id = ? AND v.agreement_id IN ? AND v.visit_date BETWEEN ? AND ?
		  AND j.status = ? AND j.deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM invoice_billed_items b WHERE b.agreement_visit_id = v.id)
		ORDER BY v.visit_date, v.id`,
		tenantID, ids, firstMonth, lastMonthEnd, models.JobStatusCompleted).Scan(&visits).Error; err != nil {
		return fmt.Errorf("load visits: %w", err)
	}
	byAgreement := map[uuid.UUID][]consolidationVisit{}
	for _, v := range visits {
		byAgreement[v.AgreementID] = append(byAgreement[v.AgreementID], v)
	}

	// Months of monthly agreements already billed.
	var billed []struct {
		AgreementID  uuid.UUID
		BillingMonth time.Time
	}
	if err := db.Model(&models.InvoiceBilledItem{}).Select("agreement_id, billing_month").
		Where("agreement_id IN ? AND billing_month BETWEEN ? AND ?", ids, firstMonth, lastMonthEnd).
		Scan(&billed).Error; err != nil {
		return fmt.Errorf("load billed months: %w", err)
	}
	billedMonth := map[string]bool{}
	for _, b := range billed {
		billedMonth[b.AgreementID.String()+b.BillingMonth.UTC().Format("2006-01")] = true
	}

	multiMonth := !firstMonth.Equal(monthStart(c.end))
	for i := range agreements {
		a := &agreements[i]
		agreementID, site := a.ID, a.SiteID
		base := models.ConsolidatedInvoiceLine{
			SiteID: &site, SiteName: siteNames[a.SiteID], AgreementID: &agreementID, UnitPrice: a.PricePerPeriod.Round(2),
		}
		switch a.BillingPeriod {
		case models.BillingPeriodVisit:
			var line consolidatedLine
			line.ConsolidatedInvoiceLine = base
			for _, v := range byAgreement[a.ID] {
				if v.VisitDate.Before(c.start) || v.VisitDate.After(c.end) {
					continue
				}
				line.visitIDs = append(line.visitIDs, v.ID)
				line.jobIDs = append(line.jobIDs, v.JobID)
			}
			if len(line.visitIDs) == 0 {
				continue
			}
			line.Kind = models.ConsolidatedLineAgreementVisits
			line.Visits = len(line.visitIDs)
			line.Quantity = decimal.NewFromInt(int64(line.Visits))
			line.Amount = line.Quantity.Mul(line.UnitPrice).Round(2)
			line.Description = fmt.Sprintf("%s · %s", a.Title, pluralVisits(line.Visits))
			c.lines = append(c.lines, line)
		default: // month
			for m := firstMonth; !m.After(c.end); m = m.AddDate(0, 1, 0) {
				mEnd := m.AddDate(0, 1, -1)
				if a.StartDate.After(mEnd) || (a.EndDate != nil && a.EndDate.Before(m)) {
					continue // not running that month
				}
				var line consolidatedLine
				line.ConsolidatedInvoiceLine = base
				for _, v := range byAgreement[a.ID] {
					if !v.VisitDate.Before(m) && !v.VisitDate.After(mEnd) {
						line.visitIDs = append(line.visitIDs, v.ID)
						line.jobIDs = append(line.jobIDs, v.JobID)
					}
				}
				month := m
				label := m.Format("2006-01")
				if billedMonth[a.ID.String()+label] {
					// Billed already. Visits completed after that stay
					// completed: the month's price already covers them.
					continue
				}
				if a.Status != models.AgreementStatusActive && len(line.visitIDs) == 0 {
					continue // paused with no work that month
				}
				line.Kind = models.ConsolidatedLineAgreementMonth
				line.month = &month
				line.Month = label
				line.Visits = len(line.visitIDs)
				line.Quantity = decimal.NewFromInt(1)
				line.Amount = line.UnitPrice
				desc := a.Title
				if multiMonth {
					desc += " · " + m.Format("January 2006")
				}
				if line.Visits > 0 {
					desc += " · " + pluralVisits(line.Visits)
				}
				line.Description = desc
				c.lines = append(c.lines, line)
			}
		}
	}
	return nil
}

// addExtraWorkLines adds each completed extra-work job (no agreement) that
// finished by the end of the period and has no invoice yet.
func (s *InvoiceService) addExtraWorkLines(db *gorm.DB, tenantID uuid.UUID, c *consolidation, siteID *uuid.UUID, siteNames map[uuid.UUID]string, loc *time.Location) error {
	// End of the period's last day in the tenant's time zone.
	cutoff := time.Date(c.end.Year(), c.end.Month(), c.end.Day()+1, 0, 0, 0, 0, loc).Unix()
	sql := `
		SELECT j.id, j.job_number, j.title, j.site_id,
		       COALESCE(NULLIF(j.total_amount, 0), NULLIF(j.actual_cost, 0), j.estimated_cost, 0) AS amount
		FROM jobs j
		WHERE j.tenant_id = ? AND j.customer_id = ? AND j.agreement_id IS NULL
		  AND j.status = ? AND j.deleted_at IS NULL
		  AND COALESCE(j.actual_end_at, j.scheduled_end_at, j.scheduled_start_at,
		               EXTRACT(EPOCH FROM j.created_at)::bigint) < ?
		  AND NOT EXISTS (SELECT 1 FROM invoice_billed_items b WHERE b.job_id = j.id)
		  AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.job_id = j.id AND i.tenant_id = j.tenant_id
		                  AND i.status <> 'cancelled' AND i.deleted_at IS NULL)`
	args := []any{tenantID, c.customer.ID, models.JobStatusCompleted, cutoff}
	if siteID != nil {
		sql += ` AND j.site_id = ?`
		args = append(args, *siteID)
	}
	var jobs []consolidationJob
	if err := db.Raw(sql+` ORDER BY j.job_number`, args...).Scan(&jobs).Error; err != nil {
		return fmt.Errorf("load extra work: %w", err)
	}
	if len(jobs) == 0 {
		return nil
	}

	jobIDs := make([]uuid.UUID, len(jobs))
	for i := range jobs {
		jobIDs[i] = jobs[i].ID
	}
	var approvals []models.JobApproval
	if err := db.Where("job_id IN ?", jobIDs).Order("approved_at DESC, created_at DESC").Find(&approvals).Error; err != nil {
		return fmt.Errorf("load approvals: %w", err)
	}
	latest := map[uuid.UUID]*models.JobApproval{}
	for i := range approvals {
		if _, ok := latest[approvals[i].JobID]; !ok {
			latest[approvals[i].JobID] = &approvals[i]
		}
	}

	for _, j := range jobs {
		jobID := j.ID
		amount := j.Amount.Round(2)
		line := consolidatedLine{
			ConsolidatedInvoiceLine: models.ConsolidatedInvoiceLine{
				Kind: models.ConsolidatedLineExtraWork, SiteID: j.SiteID, JobID: &jobID, JobNumber: j.JobNumber,
				Description: fmt.Sprintf("%s · %s", j.Title, j.JobNumber),
				Quantity:    decimal.NewFromInt(1), UnitPrice: amount, Amount: amount,
			},
			jobIDs: []uuid.UUID{j.ID},
		}
		if j.SiteID != nil {
			line.SiteName = siteNames[*j.SiteID]
		}
		if ap := latest[j.ID]; ap != nil {
			name := ap.ApproverEmail
			if ap.ApproverName != nil && strings.TrimSpace(*ap.ApproverName) != "" {
				name = strings.TrimSpace(*ap.ApproverName)
			}
			approved := ap.Amount.Round(2)
			line.ApproverName = name
			line.ApprovalAmount = &approved
			line.ApprovalNote = fmt.Sprintf("Approved by %s for %s", name, approved.StringFixed(2))
		} else if nte := c.customer.NTELimit; nte != nil && amount.LessThanOrEqual(*nte) {
			line.ApprovalNote = fmt.Sprintf("Under %s NTE", nte.StringFixed(2))
		}
		c.lines = append(c.lines, line)
	}
	return nil
}

// invoiceLines turns the consolidation into invoice lines (all taxable).
func (c *consolidation) invoiceLines() []models.InvoiceLine {
	lines := make([]models.InvoiceLine, len(c.lines))
	for i, l := range c.lines {
		lines[i] = models.InvoiceLine{
			ID: uuid.New(), Description: l.Description, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			UnitOfMeasure: "each", Taxable: true, SortOrder: i,
			SiteName: nonEmpty(l.SiteName), ApprovalNote: nonEmpty(l.ApprovalNote),
		}
		if l.Kind != models.ConsolidatedLineExtraWork {
			lines[i].UnitOfMeasure = "visit"
			if l.Kind == models.ConsolidatedLineAgreementMonth {
				lines[i].UnitOfMeasure = "month"
			}
		}
	}
	return lines
}

// consolidationDefaults returns the tenant's payment term and tax rate for
// a new consolidated invoice, and the tax rate's value.
func (s *InvoiceService) consolidationDefaults(ctx context.Context, db *gorm.DB) (*models.Invoice, decimal.Decimal, error) {
	inv := &models.Invoice{}
	if err := s.applyTenantInvoiceDefaults(ctx, inv); err != nil {
		return nil, decimal.Zero, err
	}
	rate, err := resolveTaxRate(db, nil, inv.TaxRateID, nil)
	if err != nil {
		return nil, decimal.Zero, err
	}
	return inv, rate, nil
}

// tenantLoc is the caller's tenant time zone (UTC when unset).
func (s *InvoiceService) tenantLoc(ctx context.Context, tenantID uuid.UUID) (*time.Location, error) {
	loc, _, err := tenantLocation(tenancy.WithSystem(ctx), s.db, tenantID)
	return loc, err
}

// PreviewConsolidatedInvoice returns the lines and totals POST
// /invoices/consolidate would create, without writing anything.
func (s *InvoiceService) PreviewConsolidatedInvoice(ctx context.Context, req *models.ConsolidateInvoiceRequest) (*models.ConsolidatedInvoicePreview, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	loc, err := s.tenantLoc(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	c, err := s.buildConsolidation(db, tenantID, req, loc)
	if err != nil {
		return nil, err
	}
	_, rate, err := s.consolidationDefaults(ctx, db)
	if err != nil {
		return nil, err
	}
	totals, err := calculateInvoiceTotals(c.invoiceLines(), decimal.Zero, rate)
	if err != nil {
		return nil, err
	}

	p := &models.ConsolidatedInvoicePreview{
		CustomerID: c.customer.ID, PeriodStart: c.start.Format(models.DateLayout), PeriodEnd: c.end.Format(models.DateLayout),
		Lines: make([]models.ConsolidatedInvoiceLine, len(c.lines)), Subtotal: totals.Subtotal,
		TaxRate: rate, TaxAmount: totals.Tax, Total: totals.Total,
	}
	if c.customer.DefaultPONumber != nil {
		p.PONumber = *c.customer.DefaultPONumber
	}
	sites := map[string]bool{}
	for i, l := range c.lines {
		p.Lines[i] = l.ConsolidatedInvoiceLine
		if l.SiteName != "" {
			sites[l.SiteName] = true
		}
		p.Visits += l.Visits
		if l.Kind == models.ConsolidatedLineExtraWork {
			p.ExtraWork++
		}
	}
	p.Sites = len(sites)
	return p, nil
}

// ConsolidateInvoice creates one draft invoice for the customer's unbilled
// work in the period and moves the included jobs to invoiced, all in one
// transaction.
func (s *InvoiceService) ConsolidateInvoice(ctx context.Context, req *models.ConsolidateInvoiceRequest, userID uuid.UUID) (*models.Invoice, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var invoiceID uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inv, err := s.consolidateInTx(ctx, tx, tenantID, req, userID)
		if err != nil {
			return err
		}
		invoiceID = inv.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, invoiceID)
}

// consolidateInTx builds and writes a consolidated draft inside tx:
// invoice, lines, claims and job moves. ErrNothingToInvoice when there is
// no unbilled work; ErrConsolidationRace when a claim was taken first.
func (s *InvoiceService) consolidateInTx(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID, req *models.ConsolidateInvoiceRequest, userID uuid.UUID) (*models.Invoice, error) {
	loc, err := s.tenantLoc(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	c, err := s.buildConsolidation(tx, tenantID, req, loc)
	if err != nil {
		return nil, err
	}
	if len(c.lines) == 0 {
		return nil, ErrNothingToInvoice
	}
	defaults, rate, err := s.consolidationDefaults(ctx, tx)
	if err != nil {
		return nil, err
	}

	issue := models.CivilDate(time.Now(), loc)
	start, end := c.start, c.end
	invoice := &models.Invoice{
		CustomerID: c.customer.ID, Status: models.InvoiceStatusDraft, IssueDate: issue,
		PaymentTermID: defaults.PaymentTermID, TaxRateID: defaults.TaxRateID,
		BillingPeriodStart: &start, BillingPeriodEnd: &end,
		Lines: c.invoiceLines(), CreatedBy: userID,
	}
	if c.customer.DefaultPONumber != nil {
		invoice.PONumber = *c.customer.DefaultPONumber
	}
	due, err := s.calculateDueDate(ctx, issue, invoice.PaymentTermID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate due date: %w", err)
	}
	invoice.DueDate = due
	totals, err := calculateInvoiceTotals(invoice.Lines, decimal.Zero, rate)
	if err != nil {
		return nil, err
	}
	setInvoiceTotals(invoice, totals)
	if err := tx.Create(invoice).Error; err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	for i, l := range c.lines {
		lineID := invoice.Lines[i].ID
		if l.month != nil {
			if err := claimBilledItem(tx, tenantID, invoice.ID, lineID, "agreement_id, billing_month",
				"(agreement_id, billing_month) WHERE billing_month IS NOT NULL", *l.AgreementID, *l.month); err != nil {
				return nil, err
			}
		}
		for _, v := range l.visitIDs {
			if err := claimBilledItem(tx, tenantID, invoice.ID, lineID, "agreement_id, agreement_visit_id",
				"(agreement_visit_id) WHERE agreement_visit_id IS NOT NULL", *l.AgreementID, v); err != nil {
				return nil, err
			}
		}
		if l.Kind == models.ConsolidatedLineExtraWork {
			if err := claimBilledItem(tx, tenantID, invoice.ID, lineID, "job_id",
				"(job_id) WHERE job_id IS NOT NULL", *l.JobID); err != nil {
				return nil, err
			}
		}
		note := fmt.Sprintf("Invoice %s created", invoice.InvoiceNumber)
		for _, jobID := range l.jobIDs {
			if err := invoiceConsolidatedJob(tx, jobID, note, userID); err != nil {
				return nil, err
			}
		}
	}
	return invoice, nil
}

// claimBilledItem inserts one claim; a conflict (claimed by someone else
// since the lines were read) is ErrConsolidationRace. Raw SQL: tenant_id is
// set explicitly and checked by the RLS policy.
func claimBilledItem(tx *gorm.DB, tenantID, invoiceID, lineID uuid.UUID, columns, conflict string, values ...any) error {
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(values)), ", ")
	args := append([]any{uuid.New(), tenantID, invoiceID, lineID}, values...)
	var claimed []uuid.UUID
	if err := tx.Raw(`INSERT INTO invoice_billed_items (id, tenant_id, invoice_id, invoice_line_id, `+columns+`)
		VALUES (?, ?, ?, ?, `+placeholders+`)
		ON CONFLICT `+conflict+` DO NOTHING
		RETURNING id`, args...).Scan(&claimed).Error; err != nil {
		return fmt.Errorf("claim billed item: %w", err)
	}
	if len(claimed) == 0 {
		return ErrConsolidationRace
	}
	return nil
}

// invoiceConsolidatedJob locks a job and moves it completed→invoiced; a job
// that is no longer completed (invoiced on its own meanwhile) is a race.
func invoiceConsolidatedJob(tx *gorm.DB, jobID uuid.UUID, note string, by uuid.UUID) error {
	var job models.Job
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConsolidationRace
		}
		return err
	}
	if job.Status != models.JobStatusCompleted {
		return ErrConsolidationRace
	}
	return moveJob(tx, &job, models.JobStatusInvoiced, note, by)
}

// moveJob updates a locked job's status with a history row.
func moveJob(tx *gorm.DB, job *models.Job, to models.JobStatus, note string, by uuid.UUID) error {
	if err := tx.Model(&models.Job{}).Where("id = ?", job.ID).Update("status", to).Error; err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return tx.Create(newTransition(job.ID, job.Status, to, models.ReasonOther, &note, by)).Error
}

// consolidatedJobIDs lists the jobs a consolidated invoice billed (its
// extra-work jobs and its visits' jobs). Nil for other invoices.
func consolidatedJobIDs(tx *gorm.DB, invoice *models.Invoice) ([]uuid.UUID, error) {
	if invoice.BillingPeriodStart == nil {
		return nil, nil
	}
	var ids []uuid.UUID
	err := tx.Raw(`
		SELECT DISTINCT COALESCE(b.job_id, v.job_id)
		FROM invoice_billed_items b
		LEFT JOIN agreement_visits v ON v.id = b.agreement_visit_id
		WHERE b.tenant_id = ? AND b.invoice_id = ? AND COALESCE(b.job_id, v.job_id) IS NOT NULL`,
		invoice.TenantID, invoice.ID).Scan(&ids).Error
	return ids, err
}

// releaseConsolidatedInvoice moves a cancelled or deleted consolidated
// invoice's jobs back to completed and deletes its claims, so the work can
// be billed again.
func releaseConsolidatedInvoice(tx *gorm.DB, invoice *models.Invoice, note string) error {
	if invoice.BillingPeriodStart == nil {
		return nil
	}
	if err := moveInvoiceJob(tx, invoice, models.JobStatusInvoiced, models.JobStatusCompleted, note, invoice.CreatedBy); err != nil {
		return err
	}
	return tx.Where("invoice_id = ?", invoice.ID).Delete(&models.InvoiceBilledItem{}).Error
}
