package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/javaknight1/servicepro/backend/internal/models"
	emailmock "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
)

// createSQLiteTableForModel creates a SQLite table with a column for every
// field GORM maps on model (AutoMigrate cannot be used: the models carry
// Postgres-only defaults). Time and bool columns get declared types so the
// sqlite driver converts them on scan.
func createSQLiteTableForModel(t *testing.T, db *gorm.DB, model any, extra string) {
	t.Helper()
	s, err := schema.Parse(model, &sync.Map{}, db.NamingStrategy)
	require.NoError(t, err)

	timeType := reflect.TypeOf(time.Time{})
	cols := make([]string, 0, len(s.DBNames))
	for _, name := range s.DBNames {
		f := s.FieldsByDBName[name]
		ft := f.FieldType
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		colType := ""
		switch {
		case ft == timeType || ft == reflect.TypeOf(gorm.DeletedAt{}):
			colType = "DATETIME"
		case ft.Kind() == reflect.Bool:
			colType = "BOOLEAN"
		case ft == reflect.TypeOf(decimal.Decimal{}):
			colType = "REAL"
		}
		col := name + " " + colType
		if f.PrimaryKey {
			col += " PRIMARY KEY"
		}
		cols = append(cols, col)
	}
	if extra != "" {
		cols = append(cols, extra)
	}
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE TABLE %s (%s)", s.Table, strings.Join(cols, ", "))).Error)
}

type reminderFixture struct {
	db        *gorm.DB
	email     *emailmock.Client
	svc       *PaymentReminderService
	tenant    *models.Tenant
	invoiceID uuid.UUID
}

// setupReminderFixture creates one tenant with one customer whose invoice is
// 20 days overdue, so reminders #1 (7 days) and #2 (14 days) are due.
func setupReminderFixture(t *testing.T) *reminderFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // one shared in-memory database

	createSQLiteTableForModel(t, db, &models.Customer{}, "")
	createSQLiteTableForModel(t, db, &models.Invoice{}, "")
	createSQLiteTableForModel(t, db, &models.PaymentReminder{}, "UNIQUE (invoice_id, reminder_number)")

	tenant := &models.Tenant{ID: uuid.New(), Name: "Acme HVAC"}
	tenant.Settings.PaymentRemindersEnabled = true
	tenant.Settings.ReminderDaysAfterDue = []int{7, 14, 30}
	tenant.Settings.MaxRemindersPerInvoice = 3

	customerID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary, preferred_contact_method, do_not_email, do_not_sms, created_at, updated_at)
		 VALUES (?, ?, 'Pat', 'Doe', 'pat@example.com', '5555550100', 'email', false, false, ?, ?)`,
		customerID.String(), tenant.ID.String(), time.Now(), time.Now()).Error)

	invoiceID := uuid.New()
	due := time.Now().AddDate(0, 0, -20)
	require.NoError(t, db.Exec(
		`INSERT INTO invoices (id, invoice_number, customer_id, status, issue_date, due_date, total_amount, amount_paid, amount_due, created_by, created_at, updated_at)
		 VALUES (?, 'INV-0001', ?, ?, ?, ?, 100, 0, 100, ?, ?, ?)`,
		invoiceID.String(), customerID.String(), string(models.InvoiceStatusSent), due.AddDate(0, 0, -30), due, uuid.New().String(), time.Now(), time.Now()).Error)

	email := emailmock.NewClient(nil)
	svc := NewPaymentReminderService(db, &PaymentReminderServiceConfig{EmailClient: email})
	return &reminderFixture{db: db, email: email, svc: svc, tenant: tenant, invoiceID: invoiceID}
}

func TestProcessTenantReminders_IsIdempotent(t *testing.T) {
	f := setupReminderFixture(t)
	ctx := context.Background()

	// Also a regression test for the overdue-invoice query: customers has its
	// own "status" column, so an unqualified status filter was ambiguous and
	// the automatic run never found any invoice.
	first := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 2, first.Sent, "reminders #1 and #2 are due for a 20-day overdue invoice")
	assert.Equal(t, 0, first.Failed)
	assert.Len(t, f.email.GetSentEmails(), 2)

	// A second run (e.g. scheduler retry, or the job endpoint overlapping the
	// in-process worker) must not send anything again.
	second := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 0, second.Sent)
	assert.Len(t, f.email.GetSentEmails(), 2, "no duplicate reminder emails")

	var count int64
	require.NoError(t, f.db.Model(&models.PaymentReminder{}).Where("invoice_id = ?", f.invoiceID).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestSendReminder_ConcurrentClaimDoesNotSendTwice(t *testing.T) {
	f := setupReminderFixture(t)
	ctx := context.Background()

	var invoice models.Invoice
	require.NoError(t, f.db.Preload("Customer").First(&invoice, "id = ?", f.invoiceID).Error)

	// Simulate a concurrent run that claimed reminder #1 between this run's
	// "already sent?" check and its insert.
	require.NoError(t, f.svc.SendReminder(ctx, f.tenant, &invoice, invoice.Customer, 1, "email", "pat@example.com", "friendly"))
	err := f.svc.SendReminder(ctx, f.tenant, &invoice, invoice.Customer, 1, "email", "pat@example.com", "friendly")

	assert.ErrorIs(t, err, ErrReminderAlreadySent)
	assert.Len(t, f.email.GetSentEmails(), 1, "the losing run must not send")
}
