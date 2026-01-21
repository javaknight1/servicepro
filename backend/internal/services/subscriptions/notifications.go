package subscriptions

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// Notification Types
// =============================================================================

// NotificationType represents the type of subscription notification
type NotificationType string

const (
	NotificationTypeTrialStarted        NotificationType = "trial_started"
	NotificationTypeTrialEnding         NotificationType = "trial_ending"
	NotificationTypeTrialExpired        NotificationType = "trial_expired"
	NotificationTypeSubscriptionCreated NotificationType = "subscription_created"
	NotificationTypePlanChanged         NotificationType = "plan_changed"
	NotificationTypePaymentSucceeded    NotificationType = "payment_succeeded"
	NotificationTypePaymentFailed       NotificationType = "payment_failed"
	NotificationTypeCancelled           NotificationType = "cancelled"
	NotificationTypeCancelScheduled     NotificationType = "cancel_scheduled"
	NotificationTypeRenewalReminder     NotificationType = "renewal_reminder"
	NotificationTypeInvoiceCreated      NotificationType = "invoice_created"
	NotificationTypeCardExpiring        NotificationType = "card_expiring"
)

// =============================================================================
// Notification Models
// =============================================================================

// NotificationLog records sent notifications
type NotificationLog struct {
	ID             uuid.UUID        `gorm:"type:uuid;primary_key" json:"id"`
	SubscriptionID uuid.UUID        `gorm:"type:uuid;index" json:"subscription_id"`
	CustomerID     uuid.UUID        `gorm:"type:uuid;not null;index" json:"customer_id"`
	Type           NotificationType `gorm:"type:varchar(50);not null;index" json:"type"`
	Recipient      string           `gorm:"type:varchar(255);not null" json:"recipient"`
	Subject        string           `gorm:"type:varchar(500)" json:"subject"`
	TemplateID     string           `gorm:"type:varchar(100)" json:"template_id,omitempty"`
	Status         string           `gorm:"type:varchar(50);not null" json:"status"`
	ErrorMessage   *string          `gorm:"type:text" json:"error_message,omitempty"`
	Metadata       JSONB            `gorm:"type:jsonb" json:"metadata,omitempty"`
	SentAt         *time.Time       `json:"sent_at,omitempty"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for NotificationLog
func (NotificationLog) TableName() string {
	return "subscription_notifications"
}

// BeforeCreate sets the UUID before creating a new record
func (nl *NotificationLog) BeforeCreate(tx *gorm.DB) error {
	if nl.ID == uuid.Nil {
		nl.ID = uuid.New()
	}
	return nil
}

// =============================================================================
// Email Sender Interface
// =============================================================================

// EmailSender interface for sending emails
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
	SendTemplateEmail(ctx context.Context, to, templateID string, data map[string]interface{}) error
}

// =============================================================================
// Notification Service
// =============================================================================

// NotificationServiceConfig contains configuration for the notification service
type NotificationServiceConfig struct {
	Enabled             bool
	FromEmail           string
	FromName            string
	ReplyToEmail        string
	BaseURL             string
	CompanyName         string
	SupportEmail        string
	TrialEndingDays     []int // Days before trial ends to send reminders
	RenewalReminderDays []int // Days before renewal to send reminders
	CardExpiringDays    int   // Days before card expiry to notify
	UseTemplates        bool  // Use email service templates vs inline
}

// DefaultNotificationServiceConfig returns default configuration
func DefaultNotificationServiceConfig() *NotificationServiceConfig {
	return &NotificationServiceConfig{
		Enabled:             true,
		FromEmail:           "noreply@example.com",
		FromName:            "ServicePro",
		BaseURL:             "https://app.example.com",
		CompanyName:         "ServicePro",
		SupportEmail:        "support@example.com",
		TrialEndingDays:     []int{3, 1},
		RenewalReminderDays: []int{7, 1},
		CardExpiringDays:    14,
		UseTemplates:        false,
	}
}

// NotificationService handles subscription notifications
type NotificationService struct {
	config      *NotificationServiceConfig
	db          *gorm.DB
	emailSender EmailSender
	logger      Logger
	templates   map[NotificationType]*template.Template
}

// NewNotificationService creates a new notification service
func NewNotificationService(config *NotificationServiceConfig, db *gorm.DB, emailSender EmailSender, logger Logger) *NotificationService {
	if config == nil {
		config = DefaultNotificationServiceConfig()
	}

	svc := &NotificationService{
		config:      config,
		db:          db,
		emailSender: emailSender,
		logger:      logger,
		templates:   make(map[NotificationType]*template.Template),
	}

	// Initialize templates
	svc.initTemplates()

	return svc
}

// =============================================================================
// Notification Senders
// =============================================================================

// SendTrialStarted sends a trial started notification
func (s *NotificationService) SendTrialStarted(ctx context.Context, sub *Subscription) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	// Get customer email (would normally fetch from customer service)
	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["TrialDays"] = sub.TrialDaysRemaining()
	if sub.TrialEnd != nil {
		data["TrialEndDate"] = sub.TrialEnd.Format("January 2, 2006")
	}

	subject := fmt.Sprintf("Welcome to %s - Your trial has started!", s.config.CompanyName)

	return s.sendNotification(ctx, sub, NotificationTypeTrialStarted, email, subject, data)
}

// SendTrialEnding sends a trial ending reminder
func (s *NotificationService) SendTrialEnding(ctx context.Context, sub *Subscription, daysRemaining int) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["DaysRemaining"] = daysRemaining
	if sub.TrialEnd != nil {
		data["TrialEndDate"] = sub.TrialEnd.Format("January 2, 2006")
	}
	data["UpgradeURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	var subject string
	if daysRemaining == 1 {
		subject = "Your trial ends tomorrow - upgrade now to keep your access"
	} else {
		subject = fmt.Sprintf("Your trial ends in %d days", daysRemaining)
	}

	return s.sendNotification(ctx, sub, NotificationTypeTrialEnding, email, subject, data)
}

// SendTrialExpired sends a trial expired notification
func (s *NotificationService) SendTrialExpired(ctx context.Context, sub *Subscription) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["UpgradeURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	subject := "Your trial has expired"

	return s.sendNotification(ctx, sub, NotificationTypeTrialExpired, email, subject, data)
}

// SendSubscriptionCreated sends a subscription created notification
func (s *NotificationService) SendSubscriptionCreated(ctx context.Context, sub *Subscription) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["ManageURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	subject := fmt.Sprintf("Welcome to %s %s!", s.config.CompanyName, sub.Plan.Name)

	return s.sendNotification(ctx, sub, NotificationTypeSubscriptionCreated, email, subject, data)
}

// SendPlanChanged sends a plan changed notification
func (s *NotificationService) SendPlanChanged(ctx context.Context, sub *Subscription, isUpgrade bool) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["IsUpgrade"] = isUpgrade
	data["ManageURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	var subject string
	if isUpgrade {
		subject = fmt.Sprintf("You've upgraded to %s!", sub.Plan.Name)
	} else {
		subject = fmt.Sprintf("Your plan has been changed to %s", sub.Plan.Name)
	}

	return s.sendNotification(ctx, sub, NotificationTypePlanChanged, email, subject, data)
}

// SendPaymentSucceeded sends a payment succeeded notification
func (s *NotificationService) SendPaymentSucceeded(ctx context.Context, sub *Subscription) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["InvoicesURL"] = fmt.Sprintf("%s/settings/billing", s.config.BaseURL)

	subject := fmt.Sprintf("Payment received for %s", sub.Plan.Name)

	return s.sendNotification(ctx, sub, NotificationTypePaymentSucceeded, email, subject, data)
}

// SendPaymentFailed sends a payment failed notification
func (s *NotificationService) SendPaymentFailed(ctx context.Context, sub *Subscription, attemptCount int) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["AttemptCount"] = attemptCount
	data["UpdatePaymentURL"] = fmt.Sprintf("%s/settings/billing/payment-methods", s.config.BaseURL)

	subject := "Payment failed - please update your payment method"

	return s.sendNotification(ctx, sub, NotificationTypePaymentFailed, email, subject, data)
}

// SendSubscriptionCancelled sends a cancellation notification
func (s *NotificationService) SendSubscriptionCancelled(ctx context.Context, sub *Subscription, immediate bool) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["Immediate"] = immediate
	data["EndDate"] = sub.CurrentPeriodEnd.Format("January 2, 2006")
	data["ReactivateURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	var notifType NotificationType
	var subject string
	if immediate {
		notifType = NotificationTypeCancelled
		subject = "Your subscription has been cancelled"
	} else {
		notifType = NotificationTypeCancelScheduled
		subject = "Your subscription will be cancelled"
	}

	return s.sendNotification(ctx, sub, notifType, email, subject, data)
}

// SendRenewalReminder sends a renewal reminder notification
func (s *NotificationService) SendRenewalReminder(ctx context.Context, sub *Subscription, daysUntilRenewal int) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, sub.CustomerID)
	if email == "" {
		return nil
	}

	data := s.buildBaseData(sub)
	data["DaysUntilRenewal"] = daysUntilRenewal
	data["RenewalDate"] = sub.CurrentPeriodEnd.Format("January 2, 2006")
	data["ManageURL"] = fmt.Sprintf("%s/settings/subscription", s.config.BaseURL)

	subject := fmt.Sprintf("Your subscription renews in %d days", daysUntilRenewal)

	return s.sendNotification(ctx, sub, NotificationTypeRenewalReminder, email, subject, data)
}

// SendCardExpiring sends a card expiring notification
func (s *NotificationService) SendCardExpiring(ctx context.Context, customerID uuid.UUID, cardLast4 string, expiryMonth, expiryYear int) error {
	if !s.config.Enabled || s.emailSender == nil {
		return nil
	}

	email := s.getCustomerEmail(ctx, customerID)
	if email == "" {
		return nil
	}

	data := map[string]interface{}{
		"CompanyName":      s.config.CompanyName,
		"SupportEmail":     s.config.SupportEmail,
		"BaseURL":          s.config.BaseURL,
		"CardLast4":        cardLast4,
		"ExpiryMonth":      expiryMonth,
		"ExpiryYear":       expiryYear,
		"UpdatePaymentURL": fmt.Sprintf("%s/settings/billing/payment-methods", s.config.BaseURL),
	}

	subject := "Your card is expiring soon - please update your payment method"

	return s.sendNotificationWithoutSub(ctx, NotificationTypeCardExpiring, customerID, email, subject, data)
}

// =============================================================================
// Scheduled Notifications
// =============================================================================

// ProcessScheduledNotifications processes scheduled notifications (call from cron)
func (s *NotificationService) ProcessScheduledNotifications(ctx context.Context) error {
	if err := s.processTrialEndingNotifications(ctx); err != nil {
		s.logError("Failed to process trial ending notifications", err)
	}

	if err := s.processRenewalReminders(ctx); err != nil {
		s.logError("Failed to process renewal reminders", err)
	}

	return nil
}

func (s *NotificationService) processTrialEndingNotifications(ctx context.Context) error {
	for _, days := range s.config.TrialEndingDays {
		targetDate := time.Now().AddDate(0, 0, days)
		startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var subs []*Subscription
		if err := s.db.WithContext(ctx).
			Preload("Plan").
			Where("status = ? AND trial_end >= ? AND trial_end < ?", SubscriptionStatusTrialing, startOfDay, endOfDay).
			Find(&subs).Error; err != nil {
			return err
		}

		for _, sub := range subs {
			// Check if notification already sent
			if s.hasRecentNotification(ctx, sub.ID, NotificationTypeTrialEnding, 24*time.Hour) {
				continue
			}
			s.SendTrialEnding(ctx, sub, days)
		}
	}

	return nil
}

func (s *NotificationService) processRenewalReminders(ctx context.Context) error {
	for _, days := range s.config.RenewalReminderDays {
		targetDate := time.Now().AddDate(0, 0, days)
		startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var subs []*Subscription
		if err := s.db.WithContext(ctx).
			Preload("Plan").
			Where("status = ? AND cancel_at_period_end = ? AND current_period_end >= ? AND current_period_end < ?",
				SubscriptionStatusActive, false, startOfDay, endOfDay).
			Find(&subs).Error; err != nil {
			return err
		}

		for _, sub := range subs {
			if s.hasRecentNotification(ctx, sub.ID, NotificationTypeRenewalReminder, 24*time.Hour) {
				continue
			}
			s.SendRenewalReminder(ctx, sub, days)
		}
	}

	return nil
}

func (s *NotificationService) hasRecentNotification(ctx context.Context, subID uuid.UUID, notifType NotificationType, duration time.Duration) bool {
	var count int64
	cutoff := time.Now().Add(-duration)
	s.db.WithContext(ctx).Model(&NotificationLog{}).
		Where("subscription_id = ? AND type = ? AND created_at > ?", subID, notifType, cutoff).
		Count(&count)
	return count > 0
}

// =============================================================================
// Helper Methods
// =============================================================================

func (s *NotificationService) buildBaseData(sub *Subscription) map[string]interface{} {
	data := map[string]interface{}{
		"CompanyName":  s.config.CompanyName,
		"SupportEmail": s.config.SupportEmail,
		"BaseURL":      s.config.BaseURL,
	}

	if sub != nil {
		data["SubscriptionID"] = sub.ID.String()
		data["BillingCycle"] = string(sub.BillingCycle)
		data["Status"] = string(sub.Status)
		data["CurrentPeriodEnd"] = sub.CurrentPeriodEnd.Format("January 2, 2006")

		if sub.Plan != nil {
			data["PlanName"] = sub.Plan.Name
			data["PlanTier"] = string(sub.Plan.Tier)
		}
	}

	return data
}

func (s *NotificationService) getCustomerEmail(ctx context.Context, customerID uuid.UUID) string {
	// In a real implementation, this would fetch from a customer service
	// For now, return empty string to indicate lookup needed
	type customer struct {
		Email string
	}
	var c customer
	s.db.WithContext(ctx).Table("customers").Select("email").Where("id = ?", customerID).First(&c)
	return c.Email
}

func (s *NotificationService) sendNotification(ctx context.Context, sub *Subscription, notifType NotificationType, email, subject string, data map[string]interface{}) error {
	// Log the notification
	log := &NotificationLog{
		SubscriptionID: sub.ID,
		CustomerID:     sub.CustomerID,
		Type:           notifType,
		Recipient:      email,
		Subject:        subject,
		Status:         "pending",
	}

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		s.logError("Failed to create notification log", err)
	}

	// Send the email
	var err error
	if s.config.UseTemplates {
		templateID := string(notifType)
		err = s.emailSender.SendTemplateEmail(ctx, email, templateID, data)
	} else {
		htmlBody, textBody := s.renderTemplate(notifType, data)
		err = s.emailSender.SendEmail(ctx, email, subject, htmlBody, textBody)
	}

	// Update log status
	if err != nil {
		errMsg := err.Error()
		log.Status = "failed"
		log.ErrorMessage = &errMsg
	} else {
		now := time.Now()
		log.Status = "sent"
		log.SentAt = &now
	}
	s.db.WithContext(ctx).Save(log)

	return err
}

func (s *NotificationService) sendNotificationWithoutSub(ctx context.Context, notifType NotificationType, customerID uuid.UUID, email, subject string, data map[string]interface{}) error {
	log := &NotificationLog{
		CustomerID: customerID,
		Type:       notifType,
		Recipient:  email,
		Subject:    subject,
		Status:     "pending",
	}

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		s.logError("Failed to create notification log", err)
	}

	var err error
	if s.config.UseTemplates {
		templateID := string(notifType)
		err = s.emailSender.SendTemplateEmail(ctx, email, templateID, data)
	} else {
		htmlBody, textBody := s.renderTemplate(notifType, data)
		err = s.emailSender.SendEmail(ctx, email, subject, htmlBody, textBody)
	}

	if err != nil {
		errMsg := err.Error()
		log.Status = "failed"
		log.ErrorMessage = &errMsg
	} else {
		now := time.Now()
		log.Status = "sent"
		log.SentAt = &now
	}
	s.db.WithContext(ctx).Save(log)

	return err
}

func (s *NotificationService) renderTemplate(notifType NotificationType, data map[string]interface{}) (string, string) {
	tmpl, ok := s.templates[notifType]
	if !ok {
		return s.renderGenericTemplate(data)
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		s.logError("Failed to render template", err)
		return s.renderGenericTemplate(data)
	}

	// Generate plain text version (simplified)
	textBody := fmt.Sprintf("%s\n\nVisit %s to manage your subscription.\n\nIf you have questions, contact %s",
		data["Subject"], data["BaseURL"], data["SupportEmail"])

	return htmlBuf.String(), textBody
}

func (s *NotificationService) renderGenericTemplate(data map[string]interface{}) (string, string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333;}.container{max-width:600px;margin:0 auto;padding:20px;}.button{display:inline-block;padding:10px 20px;background:#4F46E5;color:white;text-decoration:none;border-radius:5px;}</style></head>
<body>
<div class="container">
<h2>%s</h2>
<p>Thank you for being a valued customer.</p>
<p>If you have any questions, please contact us at %s</p>
<p>Best regards,<br>The %s Team</p>
</div>
</body>
</html>`, data["CompanyName"], data["SupportEmail"], data["CompanyName"])

	text := fmt.Sprintf("Thank you for being a valued customer of %s.\n\nIf you have questions, contact %s",
		data["CompanyName"], data["SupportEmail"])

	return html, text
}

func (s *NotificationService) initTemplates() {
	// Initialize email templates
	s.templates[NotificationTypeTrialStarted] = template.Must(template.New("trial_started").Parse(`
<!DOCTYPE html>
<html>
<head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333;}.container{max-width:600px;margin:0 auto;padding:20px;}.button{display:inline-block;padding:12px 24px;background:#4F46E5;color:white;text-decoration:none;border-radius:5px;margin-top:20px;}</style></head>
<body>
<div class="container">
<h1>Welcome to {{.CompanyName}}!</h1>
<p>Your free trial has started. You have <strong>{{.TrialDays}} days</strong> to explore all our features.</p>
<p>Your trial will end on <strong>{{.TrialEndDate}}</strong>.</p>
<h3>What's included in your trial:</h3>
<ul>
<li>Full access to {{.PlanName}} features</li>
<li>Unlimited projects during trial</li>
<li>Priority support</li>
</ul>
<a href="{{.BaseURL}}" class="button">Get Started</a>
<p style="margin-top:30px;color:#666;font-size:14px;">Questions? Contact us at {{.SupportEmail}}</p>
</div>
</body>
</html>`))

	s.templates[NotificationTypeTrialEnding] = template.Must(template.New("trial_ending").Parse(`
<!DOCTYPE html>
<html>
<head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333;}.container{max-width:600px;margin:0 auto;padding:20px;}.button{display:inline-block;padding:12px 24px;background:#4F46E5;color:white;text-decoration:none;border-radius:5px;margin-top:20px;}.warning{background:#FEF3C7;border:1px solid #F59E0B;padding:15px;border-radius:5px;margin:20px 0;}</style></head>
<body>
<div class="container">
<h1>Your trial is ending soon</h1>
<div class="warning">
<strong>{{.DaysRemaining}} day(s) remaining</strong> - Your trial ends on {{.TrialEndDate}}
</div>
<p>Don't lose access to your data and features! Upgrade now to continue using {{.CompanyName}}.</p>
<a href="{{.UpgradeURL}}" class="button">Upgrade Now</a>
<p style="margin-top:30px;color:#666;font-size:14px;">Questions? Contact us at {{.SupportEmail}}</p>
</div>
</body>
</html>`))

	s.templates[NotificationTypePaymentFailed] = template.Must(template.New("payment_failed").Parse(`
<!DOCTYPE html>
<html>
<head><style>body{font-family:Arial,sans-serif;line-height:1.6;color:#333;}.container{max-width:600px;margin:0 auto;padding:20px;}.button{display:inline-block;padding:12px 24px;background:#DC2626;color:white;text-decoration:none;border-radius:5px;margin-top:20px;}.alert{background:#FEE2E2;border:1px solid #DC2626;padding:15px;border-radius:5px;margin:20px 0;}</style></head>
<body>
<div class="container">
<h1>Payment Failed</h1>
<div class="alert">
<strong>Action Required:</strong> We were unable to process your payment.
</div>
<p>This is attempt #{{.AttemptCount}}. Please update your payment method to avoid service interruption.</p>
<a href="{{.UpdatePaymentURL}}" class="button">Update Payment Method</a>
<p style="margin-top:30px;color:#666;font-size:14px;">If you believe this is an error, contact us at {{.SupportEmail}}</p>
</div>
</body>
</html>`))
}

func (s *NotificationService) logError(message string, err error) {
	if s.logger != nil {
		s.logger.Errorf("%s: %v", message, err)
	}
}

// =============================================================================
// Notification History
// =============================================================================

// GetNotificationHistory retrieves notification history for a subscription
func (s *NotificationService) GetNotificationHistory(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]*NotificationLog, error) {
	var logs []*NotificationLog
	query := s.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// GetCustomerNotificationHistory retrieves notification history for a customer
func (s *NotificationService) GetCustomerNotificationHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]*NotificationLog, error) {
	var logs []*NotificationLog
	query := s.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}
