package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// metricsMaxAge is how old a customer_metrics_cache row may get before the
// report endpoints recompute the tenant's metrics.
const metricsMaxAge = time.Hour

// customerActivity is everything the metrics and cohort calculations need
// about one customer, loaded with tenant-scoped queries (internal/tenancy) on
// customers, jobs, invoices and quotes.
//
// T123: this replaces the refresh_customer_metrics_cache() SQL function and
// the v_customer_cohorts view, neither of which ever existed (and neither of
// which would have been tenant-scoped).
type customerActivity struct {
	CustomerID uuid.UUID
	CreatedAt  time.Time
	Jobs       []activityJob
	Invoices   []activityInvoice
	Quotes     []activityQuote
}

type activityJob struct {
	CustomerID  uuid.UUID
	Status      string
	TotalAmount decimal.Decimal
	CreatedAt   time.Time
}

type activityInvoice struct {
	CustomerID uuid.UUID
	Status     string
	AmountPaid decimal.Decimal
	CreatedAt  time.Time
}

type activityQuote struct {
	CustomerID uuid.UUID
	Status     string
	CreatedAt  time.Time
}

// loadCustomerActivity loads the caller's tenant's customers (optionally one
// customer, or customers created since createdSince) with their jobs,
// invoices and quotes. Every query goes through the tenancy plugin.
func (s *customerReportService) loadCustomerActivity(ctx context.Context, customerID *uuid.UUID, createdSince *time.Time) ([]*customerActivity, error) {
	if _, err := tenancy.MustTenantID(ctx); err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)

	var customers []struct {
		ID        uuid.UUID
		CreatedAt time.Time
	}
	cq := db.Model(&models.Customer{}).Select("id, created_at").Where("deleted_at IS NULL")
	if customerID != nil {
		cq = cq.Where("id = ?", *customerID)
	}
	if createdSince != nil {
		cq = cq.Where("created_at >= ?", *createdSince)
	}
	if err := cq.Scan(&customers).Error; err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}
	if customerID != nil && len(customers) == 0 {
		return nil, ErrCustomerNotFound
	}
	if len(customers) == 0 {
		return nil, nil
	}

	byID := make(map[uuid.UUID]*customerActivity, len(customers))
	ids := make([]uuid.UUID, 0, len(customers))
	out := make([]*customerActivity, 0, len(customers))
	for _, c := range customers {
		a := &customerActivity{CustomerID: c.ID, CreatedAt: c.CreatedAt}
		byID[c.ID] = a
		ids = append(ids, c.ID)
		out = append(out, a)
	}

	// Restrict child rows to the loaded customers only when the set is
	// narrowed; otherwise the tenant filter alone is enough (and avoids a
	// huge IN list).
	narrow := func(q *gorm.DB) *gorm.DB {
		if customerID != nil || createdSince != nil {
			return q.Where("customer_id IN ?", ids)
		}
		return q
	}

	var jobs []activityJob
	if err := narrow(db.Model(&models.Job{}).Select("customer_id, status, total_amount, created_at").
		Where("deleted_at IS NULL")).Scan(&jobs).Error; err != nil {
		return nil, fmt.Errorf("load jobs: %w", err)
	}
	for _, j := range jobs {
		if a := byID[j.CustomerID]; a != nil {
			a.Jobs = append(a.Jobs, j)
		}
	}

	var invoices []activityInvoice
	if err := narrow(db.Model(&models.Invoice{}).Select("customer_id, status, amount_paid, created_at").
		Where("deleted_at IS NULL")).Scan(&invoices).Error; err != nil {
		return nil, fmt.Errorf("load invoices: %w", err)
	}
	for _, inv := range invoices {
		if a := byID[inv.CustomerID]; a != nil {
			a.Invoices = append(a.Invoices, inv)
		}
	}

	var quotes []activityQuote
	if err := narrow(db.Model(&models.Quote{}).Select("customer_id, status, created_at").
		Where("deleted_at IS NULL")).Scan(&quotes).Error; err != nil {
		return nil, fmt.Errorf("load quotes: %w", err)
	}
	for _, q := range quotes {
		if a := byID[q.CustomerID]; a != nil {
			a.Quotes = append(a.Quotes, q)
		}
	}

	return out, nil
}

// lastActivity returns the latest job/invoice/quote date, if any.
func (a *customerActivity) lastActivity() *time.Time {
	var last *time.Time
	consider := func(t time.Time) {
		if last == nil || t.After(*last) {
			tt := t
			last = &tt
		}
	}
	for _, j := range a.Jobs {
		consider(j.CreatedAt)
	}
	for _, inv := range a.Invoices {
		consider(inv.CreatedAt)
	}
	for _, q := range a.Quotes {
		consider(q.CreatedAt)
	}
	return last
}

func daysBetween(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from).Hours() / 24)
}

// computeCustomerMetrics builds the cache row for one customer.
func computeCustomerMetrics(a *customerActivity, now time.Time) *models.CustomerMetricsCache {
	m := &models.CustomerMetricsCache{
		ID:          uuid.New(),
		CustomerID:  a.CustomerID,
		LastUpdated: now,
	}

	jobDates := make([]time.Time, 0, len(a.Jobs))
	for _, j := range a.Jobs {
		m.TotalJobs++
		jobDates = append(jobDates, j.CreatedAt)
		switch models.JobStatus(j.Status) {
		case models.JobStatusCompleted, models.JobStatusInvoiced, models.JobStatusPaid:
			m.CompletedJobs++
			m.TotalJobsRevenue = m.TotalJobsRevenue.Add(j.TotalAmount)
		case models.JobStatusCancelled:
			m.CancelledJobs++
		default:
			m.PendingJobs++
		}
	}
	if m.CompletedJobs > 0 {
		m.AverageJobValue = m.TotalJobsRevenue.Div(decimal.NewFromInt(int64(m.CompletedJobs))).Round(2)
	}
	if len(jobDates) >= 2 {
		sort.Slice(jobDates, func(i, j int) bool { return jobDates[i].Before(jobDates[j]) })
		span := decimal.NewFromInt(int64(daysBetween(jobDates[0], jobDates[len(jobDates)-1])))
		m.AverageDaysBetweenJobs = span.Div(decimal.NewFromInt(int64(len(jobDates) - 1))).Round(2)
	}

	for _, inv := range a.Invoices {
		switch models.InvoiceStatus(inv.Status) {
		case models.InvoiceStatusCancelled, models.InvoiceStatusDraft:
			continue
		}
		m.TotalRevenue = m.TotalRevenue.Add(inv.AmountPaid)
	}

	for _, q := range a.Quotes {
		m.TotalQuotes++
		switch models.QuoteStatus(q.Status) {
		case models.QuoteStatusAccepted:
			m.AcceptedQuotes++
		case models.QuoteStatusDeclined, models.QuoteStatusExpired:
			m.RejectedQuotes++
		default:
			m.PendingQuotes++
		}
	}
	if decided := m.AcceptedQuotes + m.RejectedQuotes; decided > 0 {
		m.QuoteAcceptanceRate = decimal.NewFromInt(int64(m.AcceptedQuotes * 100)).
			Div(decimal.NewFromInt(int64(decided))).Round(2)
	}

	first := a.CreatedAt
	m.FirstActivityDate = &first
	m.LastActivityDate = a.lastActivity()
	m.CustomerLifetimeDays = daysBetween(a.CreatedAt, now)
	if m.LastActivityDate != nil {
		m.DaysSinceLastActivity = daysBetween(*m.LastActivityDate, now)
	} else {
		m.DaysSinceLastActivity = m.CustomerLifetimeDays
	}
	m.TotalInteractions = len(a.Jobs) + len(a.Invoices) + len(a.Quotes)
	m.Segment = models.CalculateCustomerSegment(m.TotalRevenue, m.TotalJobs, m.DaysSinceLastActivity, m.CustomerLifetimeDays)
	return m
}

// RefreshMetricsCache recomputes customer_metrics_cache for the caller's
// tenant (or one of its customers). customer_metrics_cache has no tenant_id;
// rows are only written for customers the tenant-scoped load returned.
func (s *customerReportService) RefreshMetricsCache(ctx context.Context, customerID *uuid.UUID) error {
	activity, err := s.loadCustomerActivity(ctx, customerID, nil)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return err
		}
		return fmt.Errorf("failed to refresh metrics cache: %w", err)
	}
	if len(activity) == 0 {
		return nil
	}

	now := time.Now()
	rows := make([]*models.CustomerMetricsCache, 0, len(activity))
	for _, a := range activity {
		rows = append(rows, computeCustomerMetrics(a, now))
	}

	err = s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "customer_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"total_revenue", "total_jobs_revenue", "average_job_value",
				"total_jobs", "completed_jobs", "cancelled_jobs", "pending_jobs",
				"total_quotes", "accepted_quotes", "rejected_quotes", "pending_quotes", "quote_acceptance_rate",
				"first_activity_date", "last_activity_date", "days_since_last_activity", "total_interactions",
				"customer_lifetime_days", "average_days_between_jobs", "segment", "segment_score",
				"last_updated", "is_stale",
			}),
		}).
		CreateInBatches(rows, 200).Error
	if err != nil {
		return fmt.Errorf("failed to refresh metrics cache: %w", err)
	}
	return nil
}

// ensureMetricsFresh recomputes the tenant's metrics when any live customer
// has no cache row, a stale one, or one older than metricsMaxAge, so the
// segment, top-customer and at-risk reports are never silently empty.
func (s *customerReportService) ensureMetricsFresh(ctx context.Context) error {
	var outdated int64
	err := s.db.WithContext(ctx).Table("customers c").
		Joins("LEFT JOIN customer_metrics_cache cmc ON cmc.customer_id = c.id").
		Where("c.deleted_at IS NULL").
		Where("cmc.id IS NULL OR cmc.is_stale = ? OR cmc.last_updated < ?", true, time.Now().Add(-metricsMaxAge)).
		Count(&outdated).Error
	if err != nil {
		return fmt.Errorf("check metrics freshness: %w", err)
	}
	if outdated == 0 {
		return nil
	}
	return s.RefreshMetricsCache(ctx, nil)
}
