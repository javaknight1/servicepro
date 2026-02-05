package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ARAgingService defines the interface for A/R aging report operations
type ARAgingService interface {
	GenerateReport(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) (*models.ARAgingReportResponse, error)
	GetBucketSummary(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) ([]models.ARAgingBucketSummary, error)
	GetInvoicesByBucket(ctx context.Context, tenantID uuid.UUID, bucket models.AgingBucket, query *models.ARAgingQuery) ([]models.ARAgingInvoice, error)
	GetCustomerSummary(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) ([]models.ARAgingCustomerSummary, error)
}

// arAgingService implements ARAgingService
type arAgingService struct {
	db *gorm.DB
}

// NewARAgingService creates a new A/R aging service
func NewARAgingService(db *gorm.DB) ARAgingService {
	return &arAgingService{db: db}
}

// GenerateReport generates the full A/R aging report
func (s *arAgingService) GenerateReport(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) (*models.ARAgingReportResponse, error) {
	startTime := time.Now()

	// Set default as-of date to today if not specified
	asOfDate := time.Now()
	if query.AsOfDate != nil {
		asOfDate = *query.AsOfDate
	}

	// Get bucket summary
	buckets, err := s.GetBucketSummary(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}

	// Get customer summary
	customerSummary, err := s.GetCustomerSummary(ctx, tenantID, query)
	if err != nil {
		return nil, err
	}

	// Calculate overall summary
	summary := s.calculateSummary(buckets, customerSummary)

	return &models.ARAgingReportResponse{
		Summary:         summary,
		Buckets:         buckets,
		CustomerSummary: customerSummary,
		Metadata: models.ARAgingReportMetadata{
			GeneratedAt:   time.Now(),
			AsOfDate:      asOfDate,
			Query:         *query,
			ExecutionTime: time.Since(startTime).Milliseconds(),
		},
	}, nil
}

// GetBucketSummary returns the summary for each aging bucket
func (s *arAgingService) GetBucketSummary(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) ([]models.ARAgingBucketSummary, error) {
	asOfDate := time.Now()
	if query.AsOfDate != nil {
		asOfDate = *query.AsOfDate
	}

	// Build the query for outstanding invoices
	// Use DATE cast for proper PostgreSQL date arithmetic (date - date = integer days)
	db := s.db.WithContext(ctx).Table("invoices").
		Select(`
			CASE
				WHEN ?::date - due_date <= 0 THEN 'current'
				WHEN ?::date - due_date <= 30 THEN '1-30'
				WHEN ?::date - due_date <= 60 THEN '31-60'
				WHEN ?::date - due_date <= 90 THEN '61-90'
				ELSE '90+'
			END as bucket,
			COUNT(*) as invoice_count,
			COALESCE(SUM(total_amount - amount_paid), 0) as total_amount
		`, asOfDate, asOfDate, asOfDate, asOfDate).
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Where("total_amount - amount_paid > 0") // Only outstanding invoices

	// Apply filters
	if !query.IncludeDraft {
		db = db.Where("status != ?", models.InvoiceStatusDraft)
	}
	if !query.IncludePaid {
		db = db.Where("status NOT IN ?", []models.InvoiceStatus{models.InvoiceStatusPaid, models.InvoiceStatusRefunded})
	}
	if query.CustomerID != nil {
		db = db.Where("customer_id = ?", *query.CustomerID)
	}
	if query.MinAmount != nil {
		db = db.Where("total_amount - amount_paid >= ?", *query.MinAmount)
	}

	db = db.Group("bucket")

	var results []struct {
		Bucket       string          `gorm:"column:bucket"`
		InvoiceCount int             `gorm:"column:invoice_count"`
		TotalAmount  decimal.Decimal `gorm:"column:total_amount"`
	}

	if err := db.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Calculate total for percentages
	var totalAmount decimal.Decimal
	for _, r := range results {
		totalAmount = totalAmount.Add(r.TotalAmount)
	}

	// Build bucket summary with all buckets (even if empty)
	allBuckets := []models.AgingBucket{
		models.AgingBucketCurrent,
		models.AgingBucket1To30,
		models.AgingBucket31To60,
		models.AgingBucket61To90,
		models.AgingBucket90Plus,
	}

	bucketMap := make(map[string]struct {
		InvoiceCount int
		TotalAmount  decimal.Decimal
	})
	for _, r := range results {
		bucketMap[r.Bucket] = struct {
			InvoiceCount int
			TotalAmount  decimal.Decimal
		}{r.InvoiceCount, r.TotalAmount}
	}

	var summaries []models.ARAgingBucketSummary
	for _, bucket := range allBuckets {
		data := bucketMap[string(bucket)]
		percentage := decimal.Zero
		if totalAmount.GreaterThan(decimal.Zero) {
			percentage = data.TotalAmount.Div(totalAmount).Mul(decimal.NewFromInt(100)).Round(2)
		}

		summaries = append(summaries, models.ARAgingBucketSummary{
			Bucket:       bucket,
			DisplayName:  models.GetBucketDisplayName(bucket),
			InvoiceCount: data.InvoiceCount,
			TotalAmount:  data.TotalAmount,
			Percentage:   percentage,
			Color:        models.GetBucketColor(bucket),
		})
	}

	return summaries, nil
}

// GetInvoicesByBucket returns invoices for a specific aging bucket
func (s *arAgingService) GetInvoicesByBucket(ctx context.Context, tenantID uuid.UUID, bucket models.AgingBucket, query *models.ARAgingQuery) ([]models.ARAgingInvoice, error) {
	asOfDate := time.Now()
	if query.AsOfDate != nil {
		asOfDate = *query.AsOfDate
	}

	// Determine date range for bucket
	var minDays, maxDays int
	switch bucket {
	case models.AgingBucketCurrent:
		minDays, maxDays = -999999, 0
	case models.AgingBucket1To30:
		minDays, maxDays = 1, 30
	case models.AgingBucket31To60:
		minDays, maxDays = 31, 60
	case models.AgingBucket61To90:
		minDays, maxDays = 61, 90
	case models.AgingBucket90Plus:
		minDays, maxDays = 91, 999999
	}

	db := s.db.WithContext(ctx).Table("invoices").
		Select(`
			invoices.id as invoice_id,
			invoices.invoice_number,
			invoices.customer_id,
			CONCAT(customers.first_name, ' ', customers.last_name) as customer_name,
			customers.company_name,
			invoices.issue_date,
			invoices.due_date,
			invoices.total_amount,
			invoices.amount_paid,
			invoices.total_amount - invoices.amount_paid as amount_due,
			GREATEST(0, ?::date - invoices.due_date) as days_overdue,
			invoices.status
		`, asOfDate).
		Joins("LEFT JOIN customers ON customers.id = invoices.customer_id").
		Where("invoices.tenant_id = ?", tenantID).
		Where("invoices.deleted_at IS NULL").
		Where("invoices.total_amount - invoices.amount_paid > 0").
		Where("?::date - invoices.due_date >= ? AND ?::date - invoices.due_date <= ?", asOfDate, minDays, asOfDate, maxDays)

	// Apply filters
	if !query.IncludeDraft {
		db = db.Where("invoices.status != ?", models.InvoiceStatusDraft)
	}
	if !query.IncludePaid {
		db = db.Where("invoices.status NOT IN ?", []models.InvoiceStatus{models.InvoiceStatusPaid, models.InvoiceStatusRefunded})
	}
	if query.CustomerID != nil {
		db = db.Where("invoices.customer_id = ?", *query.CustomerID)
	}
	if query.MinAmount != nil {
		db = db.Where("invoices.total_amount - invoices.amount_paid >= ?", *query.MinAmount)
	}

	db = db.Order("days_overdue DESC, invoices.total_amount DESC")

	var invoices []models.ARAgingInvoice
	if err := db.Scan(&invoices).Error; err != nil {
		return nil, err
	}

	// Set the bucket for each invoice
	for i := range invoices {
		invoices[i].Bucket = bucket
	}

	return invoices, nil
}

// GetCustomerSummary returns A/R summary grouped by customer
func (s *arAgingService) GetCustomerSummary(ctx context.Context, tenantID uuid.UUID, query *models.ARAgingQuery) ([]models.ARAgingCustomerSummary, error) {
	asOfDate := time.Now()
	if query.AsOfDate != nil {
		asOfDate = *query.AsOfDate
	}

	db := s.db.WithContext(ctx).Table("invoices").
		Select(`
			invoices.customer_id,
			CONCAT(customers.first_name, ' ', customers.last_name) as customer_name,
			customers.company_name,
			customers.email,
			COUNT(DISTINCT invoices.id) as invoice_count,
			COALESCE(SUM(invoices.total_amount - invoices.amount_paid), 0) as total_owed,
			COALESCE(SUM(CASE WHEN ?::date - invoices.due_date <= 0 THEN invoices.total_amount - invoices.amount_paid ELSE 0 END), 0) as current,
			COALESCE(SUM(CASE WHEN ?::date - invoices.due_date BETWEEN 1 AND 30 THEN invoices.total_amount - invoices.amount_paid ELSE 0 END), 0) as days_1_to_30,
			COALESCE(SUM(CASE WHEN ?::date - invoices.due_date BETWEEN 31 AND 60 THEN invoices.total_amount - invoices.amount_paid ELSE 0 END), 0) as days_31_to_60,
			COALESCE(SUM(CASE WHEN ?::date - invoices.due_date BETWEEN 61 AND 90 THEN invoices.total_amount - invoices.amount_paid ELSE 0 END), 0) as days_61_to_90,
			COALESCE(SUM(CASE WHEN ?::date - invoices.due_date > 90 THEN invoices.total_amount - invoices.amount_paid ELSE 0 END), 0) as days_90_plus,
			MIN(invoices.issue_date) as oldest_invoice
		`, asOfDate, asOfDate, asOfDate, asOfDate, asOfDate).
		Joins("LEFT JOIN customers ON customers.id = invoices.customer_id").
		Where("invoices.tenant_id = ?", tenantID).
		Where("invoices.deleted_at IS NULL").
		Where("invoices.total_amount - invoices.amount_paid > 0")

	// Apply filters
	if !query.IncludeDraft {
		db = db.Where("invoices.status != ?", models.InvoiceStatusDraft)
	}
	if !query.IncludePaid {
		db = db.Where("invoices.status NOT IN ?", []models.InvoiceStatus{models.InvoiceStatusPaid, models.InvoiceStatusRefunded})
	}
	if query.CustomerID != nil {
		db = db.Where("invoices.customer_id = ?", *query.CustomerID)
	}
	if query.MinAmount != nil {
		db = db.Where("invoices.total_amount - invoices.amount_paid >= ?", *query.MinAmount)
	}

	db = db.Group("invoices.customer_id, customers.first_name, customers.last_name, customers.company_name, customers.email").
		Order("total_owed DESC")

	var summaries []models.ARAgingCustomerSummary
	if err := db.Scan(&summaries).Error; err != nil {
		return nil, err
	}

	// Get last payment date for each customer
	for i, summary := range summaries {
		var lastPayment *time.Time
		s.db.WithContext(ctx).Table("invoice_payments").
			Select("MAX(payment_date)").
			Joins("INNER JOIN invoices ON invoices.id = invoice_payments.invoice_id").
			Where("invoices.customer_id = ?", summary.CustomerID).
			Where("invoices.tenant_id = ?", tenantID).
			Scan(&lastPayment)
		summaries[i].LastPayment = lastPayment
	}

	return summaries, nil
}

// calculateSummary calculates the overall A/R summary from buckets and customers
func (s *arAgingService) calculateSummary(buckets []models.ARAgingBucketSummary, customers []models.ARAgingCustomerSummary) models.ARAgingReportSummary {
	var totalOutstanding, totalCurrent, totalOverdue decimal.Decimal
	var invoiceCount, overdueCount int
	var totalDaysOverdue int

	for _, bucket := range buckets {
		totalOutstanding = totalOutstanding.Add(bucket.TotalAmount)
		invoiceCount += bucket.InvoiceCount

		if bucket.Bucket == models.AgingBucketCurrent {
			totalCurrent = totalCurrent.Add(bucket.TotalAmount)
		} else {
			totalOverdue = totalOverdue.Add(bucket.TotalAmount)
			overdueCount += bucket.InvoiceCount
		}
	}

	// Calculate average days overdue (simplified - would need actual invoice data for precision)
	for _, bucket := range buckets {
		if bucket.Bucket != models.AgingBucketCurrent && bucket.InvoiceCount > 0 {
			var avgDays int
			switch bucket.Bucket {
			case models.AgingBucket1To30:
				avgDays = 15
			case models.AgingBucket31To60:
				avgDays = 45
			case models.AgingBucket61To90:
				avgDays = 75
			case models.AgingBucket90Plus:
				avgDays = 120
			}
			totalDaysOverdue += avgDays * bucket.InvoiceCount
		}
	}

	avgDaysOverdue := decimal.Zero
	if overdueCount > 0 {
		avgDaysOverdue = decimal.NewFromInt(int64(totalDaysOverdue)).Div(decimal.NewFromInt(int64(overdueCount))).Round(1)
	}

	overduePercentage := decimal.Zero
	if totalOutstanding.GreaterThan(decimal.Zero) {
		overduePercentage = totalOverdue.Div(totalOutstanding).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return models.ARAgingReportSummary{
		TotalOutstanding:   totalOutstanding,
		TotalCurrent:       totalCurrent,
		TotalOverdue:       totalOverdue,
		OverduePercentage:  overduePercentage,
		InvoiceCount:       invoiceCount,
		OverdueCount:       overdueCount,
		CustomerCount:      len(customers),
		AverageDaysOverdue: avgDaysOverdue,
	}
}
