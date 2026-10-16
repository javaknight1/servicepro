package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/javaknight1/servicepro/backend/internal/tenancy"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

const logPrefix = "[PAYMENT-REMINDER]"

// PaymentReminderService handles payment reminder business logic
type PaymentReminderService struct {
	db          *gorm.DB
	emailClient emailclient.Client
	smsClient   smsclient.Client
	frontendURL string
	backendURL  string
}

// PaymentReminderServiceConfig holds configuration for the service
type PaymentReminderServiceConfig struct {
	EmailClient emailclient.Client
	SMSClient   smsclient.Client
	FrontendURL string
	BackendURL  string
}

// NewPaymentReminderService creates a new payment reminder service

// Manual reminder domain errors (T223).
var (
	// ErrReminderNotAllowed: the invoice's status doesn't take reminders (409)
	ErrReminderNotAllowed = apperrors.Conflict("reminder_not_allowed", "Reminders can't be sent for this invoice's status")
	// ErrReminderNoCustomer: the invoice has no customer to remind (409)
	ErrReminderNoCustomer = apperrors.Conflict("reminder_no_customer", "This invoice has no customer to send a reminder to")
)

func NewPaymentReminderService(db *gorm.DB, cfg *PaymentReminderServiceConfig) *PaymentReminderService {
	svc := &PaymentReminderService{db: db}
	if cfg != nil {
		svc.emailClient = cfg.EmailClient
		svc.smsClient = cfg.SMSClient
		svc.frontendURL = cfg.FrontendURL
		svc.backendURL = cfg.BackendURL
	}
	return svc
}

// ErrReminderAlreadySent is returned by SendReminder when a reminder with the
// same (invoice_id, reminder_number) already exists — e.g. a concurrent run
// (in-process worker + scheduled job endpoint, or two Cloud Run instances)
// claimed it first. Nothing is sent in that case.
var ErrReminderAlreadySent = errors.New("payment reminder already sent")

// ReminderRunResult summarizes one automatic reminder run. Runs are
// idempotent: re-running immediately sends nothing new (Sent == 0).
type ReminderRunResult struct {
	TenantsProcessed int `json:"tenants_processed" example:"3"`
	Sent             int `json:"sent" example:"5"`
	Skipped          int `json:"skipped" example:"2"`
	Failed           int `json:"failed" example:"0"`
}

func (r *ReminderRunResult) add(o ReminderRunResult) {
	r.TenantsProcessed += o.TenantsProcessed
	r.Sent += o.Sent
	r.Skipped += o.Skipped
	r.Failed += o.Failed
}

// ProcessAllTenants checks all tenants for overdue invoices and sends reminders
func (s *PaymentReminderService) ProcessAllTenants(ctx context.Context) (ReminderRunResult, error) {
	var result ReminderRunResult
	var tenants []models.Tenant
	if err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&tenants).Error; err != nil {
		logging.Error(ctx, fmt.Sprintf("%s Failed to query tenants", logPrefix), map[string]any{"error": err.Error()})
		return result, fmt.Errorf("query tenants: %w", err)
	}

	for i := range tenants {
		if tenants[i].Settings.PaymentRemindersEnabled {
			// Scope each tenant's run to that tenant (internal/tenancy).
			result.add(s.ProcessTenantReminders(tenancy.WithTenant(ctx, tenants[i].ID), &tenants[i]))
		}
	}

	logging.Info(ctx, fmt.Sprintf("%s Run complete", logPrefix), map[string]any{
		"tenants": result.TenantsProcessed, "sent": result.Sent, "skipped": result.Skipped, "failed": result.Failed,
	})
	return result, nil
}

// ProcessTenantReminders processes reminders for a single tenant
func (s *PaymentReminderService) ProcessTenantReminders(ctx context.Context, tenant *models.Tenant) ReminderRunResult {
	result := ReminderRunResult{TenantsProcessed: 1}
	settings := tenant.Settings
	reminderDays := settings.ReminderDaysAfterDue
	if len(reminderDays) == 0 {
		reminderDays = []int{7, 14, 30}
	}
	maxReminders := settings.MaxRemindersPerInvoice
	if maxReminders <= 0 {
		maxReminders = 3
	}

	// Sort reminder days ascending
	sort.Ints(reminderDays)

	// Cap to maxReminders
	if len(reminderDays) > maxReminders {
		reminderDays = reminderDays[:maxReminders]
	}

	// Query overdue invoices for this tenant
	var invoices []models.Invoice
	if err := s.db.WithContext(ctx).
		Preload("Customer").
		Where("invoices.deleted_at IS NULL").
		Where("invoices.status IN ?", []models.InvoiceStatus{
			models.InvoiceStatusSent,
			models.InvoiceStatusViewed,
			models.InvoiceStatusOverdue,
			models.InvoiceStatusPartiallyPaid,
		}).
		Where("invoices.due_date < ?", time.Now()).
		Joins("JOIN customers ON customers.id = invoices.customer_id AND customers.deleted_at IS NULL").
		Where("customers.tenant_id = ?", tenant.ID).
		Find(&invoices).Error; err != nil {
		logging.Error(ctx, fmt.Sprintf("%s Failed to query overdue invoices for tenant %s", logPrefix, tenant.ID), map[string]any{"error": err.Error()})
		result.Failed++
		return result
	}

	for i := range invoices {
		invoice := &invoices[i]
		if invoice.Customer == nil {
			continue
		}

		daysOverdue := tenantDaysOverdue(tenant, invoice)

		// Only the latest reminder whose threshold has passed is due. Earlier
		// slots that were never sent (the invoice became overdue before the
		// first run, or reminders were just enabled) are skipped rather than
		// sent all at once.
		reminderNumber := 0
		for idx, dayThreshold := range reminderDays {
			if daysOverdue >= dayThreshold {
				reminderNumber = idx + 1
			}
		}
		if reminderNumber == 0 {
			continue
		}

		// Determine channel and send
		channel, recipient := s.resolveChannel(invoice.Customer)
		if channel == "" {
			logging.Warn(ctx, fmt.Sprintf("%s Skipping reminder for invoice %s: customer has all contact methods disabled",
				logPrefix, invoice.InvoiceNumber), map[string]any{"invoice_id": invoice.ID.String()})
			result.Skipped++
			continue
		}

		tone := determineTone(daysOverdue)
		_, err := s.sendReminder(ctx, tenant, invoice, invoice.Customer, reminderNumber, false, channel, recipient, tone)
		switch {
		case errors.Is(err, ErrReminderAlreadySent):
			// Already sent, or another run holds the claim.
			result.Skipped++
		case err != nil:
			result.Failed++
			logging.Error(ctx, fmt.Sprintf("%s Failed to send reminder for invoice %s", logPrefix, invoice.InvoiceNumber),
				map[string]any{"error": err.Error(), "reminder_number": reminderNumber})
		default:
			result.Sent++
		}
	}
	return result
}

// tenantDaysOverdue counts whole calendar days past the invoice's due date in
// the tenant's time zone (T127; was UTC hours / 24).
func tenantDaysOverdue(tenant *models.Tenant, invoice *models.Invoice) int {
	days := calendarDaysBetween(invoice.DueDate, time.Now(), loadLocationOrUTC(tenant.Settings.Timezone))
	if days < 0 {
		return 0
	}
	return days
}

// staleReminderClaim is how long a "pending" reminder may sit before another
// run may take it over (the run that claimed it crashed before sending).
const staleReminderClaim = time.Hour

// SendReminder sends automatic reminder #reminderNumber for an invoice.
func (s *PaymentReminderService) SendReminder(ctx context.Context, tenant *models.Tenant, invoice *models.Invoice, customer *models.Customer, reminderNumber int, channel, recipient, tone string) error {
	_, err := s.sendReminder(ctx, tenant, invoice, customer, reminderNumber, false, channel, recipient, tone)
	return err
}

// sendReminder claims the (invoice, kind, number) slot and sends the reminder.
//
// The claim is the idempotency guard: a new row is inserted with ON CONFLICT
// DO NOTHING; an existing row is only taken over when its previous send
// failed (retry) or its claim is stale. Anything else returns
// ErrReminderAlreadySent and sends nothing.
func (s *PaymentReminderService) sendReminder(ctx context.Context, tenant *models.Tenant, invoice *models.Invoice, customer *models.Customer, reminderNumber int, manual bool, channel, recipient, tone string) (*models.PaymentReminder, error) {
	daysOverdue := tenantDaysOverdue(tenant, invoice)
	subject := buildSubject(invoice.InvoiceNumber, tone, daysOverdue)

	reminder := &models.PaymentReminder{
		ID:             uuid.New(),
		TenantID:       tenant.ID,
		InvoiceID:      invoice.ID,
		CustomerID:     customer.ID,
		ReminderNumber: reminderNumber,
		IsManual:       manual,
		Channel:        channel,
		Status:         "pending",
		Tone:           tone,
		Recipient:      recipient,
		Subject:        subject,
	}

	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(reminder)
	if res.Error != nil {
		return nil, fmt.Errorf("failed to create reminder record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// The slot exists. Retake it only if the earlier send failed or its
		// claim is stale (Unscoped: a soft-deleted row still holds the slot).
		var existing models.PaymentReminder
		if err := s.db.WithContext(ctx).Unscoped().
			Where("invoice_id = ? AND is_manual = ? AND reminder_number = ?", invoice.ID, manual, reminderNumber).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrReminderAlreadySent
			}
			return nil, fmt.Errorf("failed to load reminder record: %w", err)
		}
		retake := s.db.WithContext(ctx).Unscoped().Model(&models.PaymentReminder{}).
			Where("id = ? AND deleted_at IS NULL AND (status = ? OR (status = ? AND updated_at < ?))",
				existing.ID, "failed", "pending", time.Now().Add(-staleReminderClaim)).
			Updates(map[string]interface{}{
				"status": "pending", "channel": channel, "recipient": recipient,
				"tone": tone, "subject": subject, "error_message": nil, "updated_at": time.Now(),
			})
		if retake.Error != nil {
			return nil, fmt.Errorf("failed to reclaim reminder record: %w", retake.Error)
		}
		if retake.RowsAffected == 0 {
			return nil, ErrReminderAlreadySent
		}
		reminder.ID = existing.ID
		reminder.CreatedAt = existing.CreatedAt
	}

	// Send via appropriate channel
	var sendErr error
	switch channel {
	case "email":
		sendErr = s.sendEmailReminder(ctx, tenant, invoice, customer, tone, subject, recipient)
	case "sms":
		sendErr = s.sendSMSReminder(ctx, invoice, customer, tone, recipient, daysOverdue)
	default:
		sendErr = fmt.Errorf("unsupported reminder channel %q", channel)
	}

	// Record the outcome. A failed row is retried by the next run.
	if sendErr != nil {
		errMsg := sendErr.Error()
		if err := s.db.WithContext(ctx).Model(reminder).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errMsg,
		}).Error; err != nil {
			logging.Error(ctx, fmt.Sprintf("%s Failed to record reminder failure", logPrefix), map[string]any{"error": err.Error(), "reminder_id": reminder.ID.String()})
		}
		return nil, fmt.Errorf("failed to send reminder: %w", sendErr)
	}

	sentAt := time.Now().Unix()
	if err := s.db.WithContext(ctx).Model(reminder).Updates(map[string]interface{}{
		"status":  "sent",
		"sent_at": sentAt,
	}).Error; err != nil {
		// The message went out; a later run would see a stale "pending" claim
		// and could resend, so surface this loudly.
		logging.Error(ctx, fmt.Sprintf("%s Reminder sent but status update failed", logPrefix), map[string]any{"error": err.Error(), "reminder_id": reminder.ID.String()})
	}
	reminder.Status = "sent"
	reminder.SentAt = &sentAt

	logging.Info(ctx, fmt.Sprintf("%s Reminder sent for invoice %s (#%d via %s)", logPrefix, invoice.InvoiceNumber, reminderNumber, channel),
		map[string]any{"invoice_id": invoice.ID.String(), "manual": manual, "tone": tone})

	return reminder, nil
}

// SendManualReminder sends a manual reminder for an invoice
func (s *PaymentReminderService) SendManualReminder(ctx context.Context, tenantID, invoiceID uuid.UUID) (*models.PaymentReminderResponse, error) {
	// Load invoice with customer
	var invoice models.Invoice
	if err := s.db.WithContext(ctx).
		Preload("Customer").
		First(&invoice, "id = ?", invoiceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to load invoice: %w", err)
	}

	if invoice.Customer == nil {
		return nil, ErrReminderNoCustomer
	}

	// Check invoice is in a remindable status
	switch invoice.Status {
	case models.InvoiceStatusPaid, models.InvoiceStatusCancelled, models.InvoiceStatusRefunded, models.InvoiceStatusDraft:
		return nil, ErrReminderNotAllowed.WithMessage(fmt.Sprintf("Reminders can't be sent for %s invoices", invoice.Status))
	}

	// Load tenant
	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, fmt.Errorf("failed to load tenant: %w", err)
	}

	// Manual reminders are numbered separately from the automatic schedule
	// so they never take an automatic slot.
	var maxNumber int
	if err := s.db.WithContext(ctx).Unscoped().Model(&models.PaymentReminder{}).
		Where("invoice_id = ? AND is_manual = ?", invoiceID, true).
		Select("COALESCE(MAX(reminder_number), 0)").
		Scan(&maxNumber).Error; err != nil {
		return nil, fmt.Errorf("failed to number reminder: %w", err)
	}
	reminderNumber := maxNumber + 1

	// Determine channel
	channel, recipient := s.resolveChannel(invoice.Customer)
	if channel == "" {
		return nil, fmt.Errorf("customer has all contact methods disabled")
	}

	daysOverdue := 0
	if invoice.DueDate.Before(time.Now()) {
		daysOverdue = tenantDaysOverdue(&tenant, &invoice)
	}
	tone := determineTone(daysOverdue)

	reminder, err := s.sendReminder(ctx, &tenant, &invoice, invoice.Customer, reminderNumber, true, channel, recipient, tone)
	if err != nil {
		return nil, err
	}
	reminder.Invoice = &invoice
	reminder.Customer = invoice.Customer

	resp := reminder.ToResponse()
	return &resp, nil
}

// GetReminderHistory returns the reminder history for an invoice
func (s *PaymentReminderService) GetReminderHistory(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]models.PaymentReminderResponse, error) {
	var reminders []models.PaymentReminder
	if err := s.db.WithContext(ctx).
		Preload("Invoice").
		Preload("Customer").
		Where("tenant_id = ? AND invoice_id = ?", tenantID, invoiceID).
		Order("created_at DESC").
		Find(&reminders).Error; err != nil {
		return nil, fmt.Errorf("failed to load reminder history: %w", err)
	}

	responses := make([]models.PaymentReminderResponse, len(reminders))
	for i := range reminders {
		responses[i] = reminders[i].ToResponse()
	}
	return responses, nil
}

// GetReminderSettings returns the payment reminder settings for a tenant
func (s *PaymentReminderService) GetReminderSettings(ctx context.Context, tenantID uuid.UUID) (*models.PaymentReminderSettingsResponse, error) {
	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, fmt.Errorf("failed to load tenant: %w", err)
	}

	days := tenant.Settings.ReminderDaysAfterDue
	if len(days) == 0 {
		days = []int{7, 14, 30}
	}
	maxReminders := tenant.Settings.MaxRemindersPerInvoice
	if maxReminders <= 0 {
		maxReminders = 3
	}

	return &models.PaymentReminderSettingsResponse{
		PaymentRemindersEnabled: tenant.Settings.PaymentRemindersEnabled,
		ReminderDaysAfterDue:    days,
		MaxRemindersPerInvoice:  maxReminders,
	}, nil
}

// UpdateReminderSettings updates the payment reminder settings for a tenant
func (s *PaymentReminderService) UpdateReminderSettings(ctx context.Context, tenantID uuid.UUID, req *models.PaymentReminderSettingsRequest) error {
	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, "id = ?", tenantID).Error; err != nil {
		return fmt.Errorf("failed to load tenant: %w", err)
	}

	tenant.Settings.PaymentRemindersEnabled = req.PaymentRemindersEnabled
	tenant.Settings.ReminderDaysAfterDue = req.ReminderDaysAfterDue
	tenant.Settings.MaxRemindersPerInvoice = req.MaxRemindersPerInvoice

	if err := s.db.WithContext(ctx).Model(&tenant).Update("settings", tenant.Settings).Error; err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}

	return nil
}

// resolveChannel determines the best channel and recipient based on customer preferences
func (s *PaymentReminderService) resolveChannel(customer *models.Customer) (channel, recipient string) {
	if customer.DoNotEmail && customer.DoNotSMS {
		return "", ""
	}

	switch customer.PreferredContactMethod {
	case models.ContactMethodSMS:
		if !customer.DoNotSMS && s.smsClient != nil {
			return "sms", customer.PhonePrimary
		}
		if !customer.DoNotEmail {
			return "email", customer.Email
		}
	case models.ContactMethodEmail:
		if !customer.DoNotEmail {
			return "email", customer.Email
		}
		if !customer.DoNotSMS && s.smsClient != nil {
			return "sms", customer.PhonePrimary
		}
	default: // "any", "phone", "mail", or empty
		if !customer.DoNotEmail {
			return "email", customer.Email
		}
		if !customer.DoNotSMS && s.smsClient != nil {
			return "sms", customer.PhonePrimary
		}
	}

	return "", ""
}

// sendEmailReminder sends a reminder via email
func (s *PaymentReminderService) sendEmailReminder(ctx context.Context, tenant *models.Tenant, invoice *models.Invoice, customer *models.Customer, tone, subject, recipient string) error {
	if s.emailClient == nil {
		return fmt.Errorf("email client not configured")
	}

	htmlBody, textBody := renderReminderEmail(tenant.Name, customer.GetDisplayName(), invoice, tone, tenantDaysOverdue(tenant, invoice))

	msg := emailclient.NewEmailMessage(recipient, subject, htmlBody).
		WithTextBody(textBody).
		WithTag("type", "payment_reminder").
		WithTag("invoice_id", invoice.ID.String()).
		WithTag("tone", tone)

	_, err := s.emailClient.Send(ctx, msg)
	return err
}

// sendSMSReminder sends a reminder via SMS
func (s *PaymentReminderService) sendSMSReminder(ctx context.Context, invoice *models.Invoice, customer *models.Customer, tone, recipient string, daysOverdue int) error {
	if s.smsClient == nil {
		return fmt.Errorf("SMS client not configured")
	}

	message := buildSMSContent(customer.FirstName, invoice.InvoiceNumber, invoice.AmountDue.StringFixed(2), daysOverdue, tone)

	return s.smsClient.SendNotification(ctx, recipient, message)
}

// determineTone returns the appropriate tone based on days overdue
func determineTone(daysOverdue int) string {
	switch {
	case daysOverdue <= 14:
		return "friendly"
	case daysOverdue <= 30:
		return "firm"
	default:
		return "final"
	}
}

// buildSubject creates the email subject line based on tone
func buildSubject(invoiceNumber, tone string, daysOverdue int) string {
	switch tone {
	case "friendly":
		return fmt.Sprintf("Friendly reminder: Invoice %s is past due", invoiceNumber)
	case "firm":
		return fmt.Sprintf("Payment reminder: Invoice %s is %d days overdue", invoiceNumber, daysOverdue)
	default: // final
		return fmt.Sprintf("Final notice: Invoice %s requires immediate attention", invoiceNumber)
	}
}

// buildSMSContent creates the SMS message content
func buildSMSContent(firstName, invoiceNumber, amountDue string, daysOverdue int, tone string) string {
	switch tone {
	case "friendly":
		return fmt.Sprintf("Hi %s, this is a friendly reminder that invoice %s ($%s) is past due. Please arrange payment at your earliest convenience.", firstName, invoiceNumber, amountDue)
	case "firm":
		return fmt.Sprintf("Payment reminder: Invoice %s ($%s) is %d days overdue. Please make payment promptly to avoid further action.", invoiceNumber, amountDue, daysOverdue)
	default: // final
		return fmt.Sprintf("FINAL NOTICE: Invoice %s ($%s) is %d days overdue and requires immediate attention. Please contact us to arrange payment.", invoiceNumber, amountDue, daysOverdue)
	}
}

// renderReminderEmail renders the payment reminder email HTML and text
func renderReminderEmail(companyName, customerName string, invoice *models.Invoice, tone string, daysOverdue int) (string, string) {
	data := map[string]string{
		"CompanyName":   companyName,
		"CustomerName":  customerName,
		"InvoiceNumber": invoice.InvoiceNumber,
		"AmountDue":     invoice.AmountDue.StringFixed(2),
		"DueDate":       invoice.DueDate.Format("January 2, 2006"),
		"DaysOverdue":   fmt.Sprintf("%d", daysOverdue),
		"Tone":          tone,
	}

	htmlBody := renderHTMLTemplate(data)
	textBody := renderTextTemplate(data)
	return htmlBody, textBody
}

func renderHTMLTemplate(data map[string]string) string {
	tmpl := template.Must(template.New("reminder").Parse(reminderHTMLTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Payment reminder for invoice %s", data["InvoiceNumber"])
	}
	return buf.String()
}

func renderTextTemplate(data map[string]string) string {
	tmpl := template.Must(template.New("reminder_text").Parse(reminderTextTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Payment reminder for invoice %s - Amount due: $%s", data["InvoiceNumber"], data["AmountDue"])
	}
	return buf.String()
}

const reminderHTMLTemplate = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
{{if eq .Tone "friendly"}}
<h2 style="color: #2563eb;">Friendly Payment Reminder</h2>
<p>Dear {{.CustomerName}},</p>
<p>We hope this message finds you well. This is a friendly reminder that payment for the following invoice is past due:</p>
{{else if eq .Tone "firm"}}
<h2 style="color: #d97706;">Payment Reminder</h2>
<p>Dear {{.CustomerName}},</p>
<p>We are writing to remind you that the following invoice is now <strong>{{.DaysOverdue}} days overdue</strong>:</p>
{{else}}
<h2 style="color: #dc2626;">Final Notice</h2>
<p>Dear {{.CustomerName}},</p>
<p>This is a final notice regarding the following overdue invoice that is now <strong>{{.DaysOverdue}} days past due</strong>:</p>
{{end}}
<table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
<tr style="background-color: #f3f4f6;">
<td style="padding: 12px; border: 1px solid #e5e7eb;"><strong>Invoice Number</strong></td>
<td style="padding: 12px; border: 1px solid #e5e7eb;">{{.InvoiceNumber}}</td>
</tr>
<tr>
<td style="padding: 12px; border: 1px solid #e5e7eb;"><strong>Amount Due</strong></td>
<td style="padding: 12px; border: 1px solid #e5e7eb;">${{.AmountDue}}</td>
</tr>
<tr style="background-color: #f3f4f6;">
<td style="padding: 12px; border: 1px solid #e5e7eb;"><strong>Due Date</strong></td>
<td style="padding: 12px; border: 1px solid #e5e7eb;">{{.DueDate}}</td>
</tr>
</table>
{{if eq .Tone "friendly"}}
<p>If you have already sent payment, please disregard this notice. If you have any questions about this invoice, please don't hesitate to reach out.</p>
{{else if eq .Tone "firm"}}
<p>Please arrange payment at your earliest convenience. If you have already made payment, please let us know so we can update our records.</p>
{{else}}
<p><strong>Immediate action is required.</strong> Please contact us to arrange payment or discuss a payment plan. Failure to respond may result in additional late fees or collection activity.</p>
{{end}}
<p>Thank you for your attention to this matter.</p>
<p>Best regards,<br/>{{.CompanyName}}</p>
</body>
</html>`

const reminderTextTemplate = `{{if eq .Tone "friendly"}}FRIENDLY PAYMENT REMINDER{{else if eq .Tone "firm"}}PAYMENT REMINDER{{else}}FINAL NOTICE{{end}}

Dear {{.CustomerName}},

{{if eq .Tone "friendly"}}This is a friendly reminder that payment for the following invoice is past due:{{else if eq .Tone "firm"}}We are writing to remind you that the following invoice is now {{.DaysOverdue}} days overdue:{{else}}This is a final notice regarding your overdue invoice that is now {{.DaysOverdue}} days past due:{{end}}

Invoice Number: {{.InvoiceNumber}}
Amount Due: ${{.AmountDue}}
Due Date: {{.DueDate}}

{{if eq .Tone "friendly"}}If you have already sent payment, please disregard this notice.{{else if eq .Tone "firm"}}Please arrange payment at your earliest convenience.{{else}}Immediate action is required. Please contact us to arrange payment.{{end}}

Best regards,
{{.CompanyName}}`
