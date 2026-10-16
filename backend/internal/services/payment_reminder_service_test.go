package services

import (
	"context"
	"errors"
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
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	emailmock "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
	smsmock "github.com/javaknight1/servicepro/backend/pkg/clients/sms/mock"
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
	db         *gorm.DB
	email      *flakyEmail
	svc        *PaymentReminderService
	tenant     *models.Tenant
	invoiceID  uuid.UUID
	customerID uuid.UUID
}

// flakyEmail is the mock email client with a switch to make sends fail.
type flakyEmail struct {
	*emailmock.Client
	fail bool
}

func (f *flakyEmail) Send(ctx context.Context, msg *emailclient.EmailMessage) (*emailclient.SendResult, error) {
	if f.fail {
		return nil, errors.New("provider unavailable")
	}
	return f.Client.Send(ctx, msg)
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
	createSQLiteTableForModel(t, db, &models.PaymentReminder{}, "UNIQUE (invoice_id, is_manual, reminder_number)")

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

	email := &flakyEmail{Client: emailmock.NewClient(nil)}
	svc := NewPaymentReminderService(db, &PaymentReminderServiceConfig{EmailClient: email})
	return &reminderFixture{db: db, email: email, svc: svc, tenant: tenant, invoiceID: invoiceID, customerID: customerID}
}

func TestProcessTenantReminders_IsIdempotent(t *testing.T) {
	f := setupReminderFixture(t)
	ctx := context.Background()

	// Also a regression test for the overdue-invoice query: customers has its
	// own "status" column, so an unqualified status filter was ambiguous and
	// the automatic run never found any invoice.
	first := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 1, first.Sent, "only the latest due reminder (#2) is sent for a 20-day overdue invoice")
	assert.Equal(t, 0, first.Failed)
	assert.Len(t, f.email.GetSentEmails(), 1)

	// A second run (e.g. scheduler retry, or the job endpoint overlapping the
	// in-process worker) must not send anything again.
	second := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 0, second.Sent)
	assert.Len(t, f.email.GetSentEmails(), 1, "no duplicate reminder emails")

	var reminders []models.PaymentReminder
	require.NoError(t, f.db.Where("invoice_id = ?", f.invoiceID).Find(&reminders).Error)
	require.Len(t, reminders, 1)
	assert.Equal(t, 2, reminders[0].ReminderNumber)
	assert.Equal(t, "sent", reminders[0].Status)
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

// T121: a failed send is retried on the next run instead of blocking the slot.
func TestProcessTenantReminders_RetriesFailedSend(t *testing.T) {
	f := setupReminderFixture(t)
	ctx := context.Background()

	f.email.fail = true
	first := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 1, first.Failed)
	assert.Equal(t, 0, first.Sent)

	f.email.fail = false
	second := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 1, second.Sent, "failed reminder must be retried")
	assert.Len(t, f.email.GetSentEmails(), 1)

	var reminders []models.PaymentReminder
	require.NoError(t, f.db.Where("invoice_id = ?", f.invoiceID).Find(&reminders).Error)
	require.Len(t, reminders, 1, "the retry reuses the claimed row")
	assert.Equal(t, "sent", reminders[0].Status)
	assert.Nil(t, reminders[0].ErrorMessage)
}

// T121: a fresh "pending" claim (another run is sending) is not taken over.
func TestSendReminder_FreshPendingClaimIsNotRetaken(t *testing.T) {
	f := setupReminderFixture(t)
	ctx := context.Background()
	require.NoError(t, f.db.Exec(
		`INSERT INTO payment_reminders (id, tenant_id, invoice_id, customer_id, reminder_number, is_manual, channel, status, tone, recipient, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 2, false, 'email', 'pending', 'firm', 'pat@example.com', ?, ?)`,
		uuid.New().String(), f.tenant.ID.String(), f.invoiceID.String(), f.customerID.String(), time.Now(), time.Now()).Error)

	res := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 0, res.Sent)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, f.email.GetSentEmails())
}

// T121: manual reminders are numbered on their own and do not consume the
// automatic schedule's slots (MAX+1 numbering used to skip automatic #1).
func TestSendManualReminder_DoesNotTakeAutomaticSlot(t *testing.T) {
	f := setupReminderFixture(t)
	createSQLiteTableForModel(t, f.db, &models.Tenant{}, "")
	require.NoError(t, f.db.Exec(`INSERT INTO tenants (id, name, slug, created_at, updated_at) VALUES (?, 'Acme HVAC', 'acme', ?, ?)`,
		f.tenant.ID.String(), time.Now(), time.Now()).Error)
	ctx := context.Background()

	// Move the due date so only automatic #1 (7 days) is due.
	require.NoError(t, f.db.Exec(`UPDATE invoices SET due_date = ? WHERE id = ?`, time.Now().AddDate(0, 0, -8), f.invoiceID.String()).Error)

	m1, err := f.svc.SendManualReminder(ctx, f.tenant.ID, f.invoiceID)
	require.NoError(t, err)
	m2, err := f.svc.SendManualReminder(ctx, f.tenant.ID, f.invoiceID)
	require.NoError(t, err)
	assert.True(t, m1.IsManual)
	assert.Equal(t, 1, m1.ReminderNumber)
	assert.Equal(t, 2, m2.ReminderNumber)

	res := f.svc.ProcessTenantReminders(ctx, f.tenant)
	assert.Equal(t, 1, res.Sent, "automatic reminder #1 still goes out after manual reminders")
	assert.Len(t, f.email.GetSentEmails(), 3)
}

// T121: customers who prefer SMS get an SMS when an SMS client is configured.
func TestProcessTenantReminders_SendsSMSWhenPreferred(t *testing.T) {
	f := setupReminderFixture(t)
	require.NoError(t, f.db.Exec(`UPDATE customers SET preferred_contact_method = 'sms' WHERE id = ?`, f.customerID.String()).Error)
	sms := smsmock.NewClient(nil)
	svc := NewPaymentReminderService(f.db, &PaymentReminderServiceConfig{EmailClient: f.email, SMSClient: sms})

	res := svc.ProcessTenantReminders(context.Background(), f.tenant)
	assert.Equal(t, 1, res.Sent)
	assert.Len(t, sms.GetSentMessages(), 1)
	assert.Empty(t, f.email.GetSentEmails())
}

func TestPaymentReminderWorker_StopIsIdempotent(t *testing.T) {
	f := setupReminderFixture(t)
	w := NewPaymentReminderWorker(f.svc, time.Hour)
	w.Start(context.Background())
	w.Stop()
	assert.NotPanics(t, w.Stop)
}
