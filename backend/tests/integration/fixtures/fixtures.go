//go:build integration

package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =============================================================================
// Fixture Manager
// =============================================================================

// FixtureManager manages test data fixtures
type FixtureManager struct {
	db *gorm.DB
}

// NewFixtureManager creates a new fixture manager
func NewFixtureManager(db *gorm.DB) *FixtureManager {
	return &FixtureManager{db: db}
}

// =============================================================================
// Customer Fixtures
// =============================================================================

// Customer represents a test customer
type Customer struct {
	ID          string
	Email       string
	FirstName   string
	LastName    string
	Phone       string
	CompanyName string
	Address     string
	City        string
	State       string
	ZipCode     string
	Country     string
	Notes       string
	CompanyID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateCustomer creates a test customer
func (f *FixtureManager) CreateCustomer(t *testing.T, c *Customer) *Customer {
	t.Helper()

	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.Email == "" {
		c.Email = "customer-" + uuid.New().String()[:8] + "@test.com"
	}
	if c.FirstName == "" {
		c.FirstName = "Test"
	}
	if c.LastName == "" {
		c.LastName = "Customer"
	}
	if c.Phone == "" {
		c.Phone = "555-0100"
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	err := f.db.Exec(`
		INSERT INTO customers (id, email, first_name, last_name, phone, company_name, address, city, state, zip_code, country, notes, company_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Email, c.FirstName, c.LastName, c.Phone, c.CompanyName, c.Address, c.City, c.State, c.ZipCode, c.Country, c.Notes, c.CompanyID, c.CreatedAt, c.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test customer")

	return c
}

// CreateCustomers creates multiple test customers
func (f *FixtureManager) CreateCustomers(t *testing.T, count int) []*Customer {
	t.Helper()

	customers := make([]*Customer, count)
	for i := 0; i < count; i++ {
		customers[i] = f.CreateCustomer(t, &Customer{
			FirstName: "Customer",
			LastName:  string(rune('A' + i)),
		})
	}
	return customers
}

// =============================================================================
// Job Fixtures
// =============================================================================

// Job represents a test job
type Job struct {
	ID              string
	Title           string
	Description     string
	CustomerID      string
	Status          string
	Priority        string
	ScheduledStart  time.Time
	ScheduledEnd    time.Time
	ActualStart     *time.Time
	ActualEnd       *time.Time
	AssignedTechIDs []string
	EstimatedCost   decimal.Decimal
	ActualCost      decimal.Decimal
	Notes           string
	CompanyID       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateJob creates a test job
func (f *FixtureManager) CreateJob(t *testing.T, j *Job) *Job {
	t.Helper()

	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	if j.Title == "" {
		j.Title = "Test Job " + uuid.New().String()[:8]
	}
	if j.Status == "" {
		j.Status = "scheduled"
	}
	if j.Priority == "" {
		j.Priority = "normal"
	}
	if j.ScheduledStart.IsZero() {
		j.ScheduledStart = time.Now().Add(24 * time.Hour)
	}
	if j.ScheduledEnd.IsZero() {
		j.ScheduledEnd = j.ScheduledStart.Add(2 * time.Hour)
	}
	if j.EstimatedCost.IsZero() {
		j.EstimatedCost = decimal.NewFromFloat(100.00)
	}
	j.CreatedAt = time.Now()
	j.UpdatedAt = time.Now()

	err := f.db.Exec(`
		INSERT INTO jobs (id, title, description, customer_id, status, priority, scheduled_start, scheduled_end, estimated_cost, notes, company_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, j.ID, j.Title, j.Description, j.CustomerID, j.Status, j.Priority, j.ScheduledStart, j.ScheduledEnd, j.EstimatedCost, j.Notes, j.CompanyID, j.CreatedAt, j.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test job")

	return j
}

// CreateJobWithStatus creates a job with a specific status
func (f *FixtureManager) CreateJobWithStatus(t *testing.T, customerID, status string) *Job {
	t.Helper()
	return f.CreateJob(t, &Job{
		CustomerID: customerID,
		Status:     status,
	})
}

// =============================================================================
// Invoice Fixtures
// =============================================================================

// Invoice represents a test invoice
type Invoice struct {
	ID         string
	Number     string
	CustomerID string
	JobID      string
	Status     string
	IssueDate  time.Time
	DueDate    time.Time
	Subtotal   decimal.Decimal
	TaxRate    decimal.Decimal
	TaxAmount  decimal.Decimal
	Total      decimal.Decimal
	PaidAmount decimal.Decimal
	Notes      string
	CompanyID  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LineItems  []*InvoiceLineItem
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ID          string
	InvoiceID   string
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	Total       decimal.Decimal
}

// CreateInvoice creates a test invoice
func (f *FixtureManager) CreateInvoice(t *testing.T, inv *Invoice) *Invoice {
	t.Helper()

	if inv.ID == "" {
		inv.ID = uuid.New().String()
	}
	if inv.Number == "" {
		inv.Number = "INV-" + uuid.New().String()[:8]
	}
	if inv.Status == "" {
		inv.Status = "draft"
	}
	if inv.IssueDate.IsZero() {
		inv.IssueDate = time.Now()
	}
	if inv.DueDate.IsZero() {
		inv.DueDate = inv.IssueDate.Add(30 * 24 * time.Hour) // Net 30
	}
	if inv.TaxRate.IsZero() {
		inv.TaxRate = decimal.NewFromFloat(0.08) // 8%
	}
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = time.Now()

	// Calculate totals from line items
	inv.Subtotal = decimal.Zero
	for _, item := range inv.LineItems {
		item.Total = item.Quantity.Mul(item.UnitPrice)
		inv.Subtotal = inv.Subtotal.Add(item.Total)
	}
	inv.TaxAmount = inv.Subtotal.Mul(inv.TaxRate)
	inv.Total = inv.Subtotal.Add(inv.TaxAmount)

	err := f.db.Exec(`
		INSERT INTO invoices (id, number, customer_id, job_id, status, issue_date, due_date, subtotal, tax_rate, tax_amount, total, paid_amount, notes, company_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.ID, inv.Number, inv.CustomerID, inv.JobID, inv.Status, inv.IssueDate, inv.DueDate, inv.Subtotal, inv.TaxRate, inv.TaxAmount, inv.Total, inv.PaidAmount, inv.Notes, inv.CompanyID, inv.CreatedAt, inv.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test invoice")

	// Create line items
	for _, item := range inv.LineItems {
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.InvoiceID = inv.ID

		err = f.db.Exec(`
			INSERT INTO invoice_line_items (id, invoice_id, description, quantity, unit_price, total)
			VALUES (?, ?, ?, ?, ?, ?)
		`, item.ID, item.InvoiceID, item.Description, item.Quantity, item.UnitPrice, item.Total).Error
		require.NoError(t, err, "Failed to create test line item")
	}

	return inv
}

// CreateInvoiceWithLineItems creates an invoice with sample line items
func (f *FixtureManager) CreateInvoiceWithLineItems(t *testing.T, customerID string) *Invoice {
	t.Helper()

	return f.CreateInvoice(t, &Invoice{
		CustomerID: customerID,
		LineItems: []*InvoiceLineItem{
			{Description: "Service Call", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(75.00)},
			{Description: "Labor (hours)", Quantity: decimal.NewFromFloat(2.5), UnitPrice: decimal.NewFromFloat(50.00)},
			{Description: "Parts", Quantity: decimal.NewFromInt(3), UnitPrice: decimal.NewFromFloat(25.00)},
		},
	})
}

// =============================================================================
// Quote Fixtures
// =============================================================================

// Quote represents a test quote
type Quote struct {
	ID         string
	Number     string
	CustomerID string
	Status     string
	ValidUntil time.Time
	Subtotal   decimal.Decimal
	TaxRate    decimal.Decimal
	TaxAmount  decimal.Decimal
	Total      decimal.Decimal
	Notes      string
	CompanyID  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateQuote creates a test quote
func (f *FixtureManager) CreateQuote(t *testing.T, q *Quote) *Quote {
	t.Helper()

	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.Number == "" {
		q.Number = "QT-" + uuid.New().String()[:8]
	}
	if q.Status == "" {
		q.Status = "draft"
	}
	if q.ValidUntil.IsZero() {
		q.ValidUntil = time.Now().Add(30 * 24 * time.Hour)
	}
	if q.TaxRate.IsZero() {
		q.TaxRate = decimal.NewFromFloat(0.08)
	}
	if q.Subtotal.IsZero() {
		q.Subtotal = decimal.NewFromFloat(500.00)
	}
	q.TaxAmount = q.Subtotal.Mul(q.TaxRate)
	q.Total = q.Subtotal.Add(q.TaxAmount)
	q.CreatedAt = time.Now()
	q.UpdatedAt = time.Now()

	err := f.db.Exec(`
		INSERT INTO quotes (id, number, customer_id, status, valid_until, subtotal, tax_rate, tax_amount, total, notes, company_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, q.ID, q.Number, q.CustomerID, q.Status, q.ValidUntil, q.Subtotal, q.TaxRate, q.TaxAmount, q.Total, q.Notes, q.CompanyID, q.CreatedAt, q.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test quote")

	return q
}

// =============================================================================
// Payment Fixtures
// =============================================================================

// Payment represents a test payment
type Payment struct {
	ID              string
	InvoiceID       string
	Amount          decimal.Decimal
	Method          string
	Status          string
	StripePaymentID string
	StripeChargeID  string
	ProcessedAt     *time.Time
	FailureReason   string
	RefundAmount    decimal.Decimal
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreatePayment creates a test payment
func (f *FixtureManager) CreatePayment(t *testing.T, p *Payment) *Payment {
	t.Helper()

	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Method == "" {
		p.Method = "card"
	}
	if p.Status == "" {
		p.Status = "pending"
	}
	if p.Amount.IsZero() {
		p.Amount = decimal.NewFromFloat(100.00)
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	err := f.db.Exec(`
		INSERT INTO payments (id, invoice_id, amount, method, status, stripe_payment_id, stripe_charge_id, processed_at, failure_reason, refund_amount, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.InvoiceID, p.Amount, p.Method, p.Status, p.StripePaymentID, p.StripeChargeID, p.ProcessedAt, p.FailureReason, p.RefundAmount, p.Notes, p.CreatedAt, p.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test payment")

	return p
}

// CreateSuccessfulPayment creates a successful payment
func (f *FixtureManager) CreateSuccessfulPayment(t *testing.T, invoiceID string, amount decimal.Decimal) *Payment {
	t.Helper()
	now := time.Now()
	return f.CreatePayment(t, &Payment{
		InvoiceID:       invoiceID,
		Amount:          amount,
		Status:          "succeeded",
		StripePaymentID: "pi_test_" + uuid.New().String()[:8],
		StripeChargeID:  "ch_test_" + uuid.New().String()[:8],
		ProcessedAt:     &now,
	})
}

// =============================================================================
// Schedule Fixtures
// =============================================================================

// Schedule represents a test schedule entry
type Schedule struct {
	ID           string
	JobID        string
	TechnicianID string
	StartTime    time.Time
	EndTime      time.Time
	Status       string
	Notes        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateSchedule creates a test schedule
func (f *FixtureManager) CreateSchedule(t *testing.T, s *Schedule) *Schedule {
	t.Helper()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.Status == "" {
		s.Status = "scheduled"
	}
	if s.StartTime.IsZero() {
		s.StartTime = time.Now().Add(24 * time.Hour)
	}
	if s.EndTime.IsZero() {
		s.EndTime = s.StartTime.Add(2 * time.Hour)
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	err := f.db.Exec(`
		INSERT INTO schedules (id, job_id, technician_id, start_time, end_time, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.JobID, s.TechnicianID, s.StartTime, s.EndTime, s.Status, s.Notes, s.CreatedAt, s.UpdatedAt).Error
	require.NoError(t, err, "Failed to create test schedule")

	return s
}

// =============================================================================
// Bulk Data Creation
// =============================================================================

// SeedTestData creates a standard set of test data
func (f *FixtureManager) SeedTestData(t *testing.T) *TestDataSet {
	t.Helper()

	data := &TestDataSet{
		Customers: f.CreateCustomers(t, 5),
	}

	// Create jobs for each customer
	for _, customer := range data.Customers {
		job := f.CreateJob(t, &Job{CustomerID: customer.ID})
		data.Jobs = append(data.Jobs, job)

		// Create invoice for each job
		invoice := f.CreateInvoiceWithLineItems(t, customer.ID)
		invoice.JobID = job.ID
		data.Invoices = append(data.Invoices, invoice)
	}

	return data
}

// TestDataSet contains all seeded test data
type TestDataSet struct {
	Customers []*Customer
	Jobs      []*Job
	Invoices  []*Invoice
	Quotes    []*Quote
	Payments  []*Payment
}
