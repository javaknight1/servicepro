package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType represents the type of payment notification
type NotificationType string

const (
	NotificationPaymentDueSoon         NotificationType = "payment_due_soon"
	NotificationPaymentOverdue         NotificationType = "payment_overdue"
	NotificationPaymentInGracePeriod   NotificationType = "payment_in_grace_period"
	NotificationEarlyPaymentDiscount   NotificationType = "early_payment_discount_available"
	NotificationLateFeeApplied         NotificationType = "late_fee_applied"
	NotificationPaymentReceived        NotificationType = "payment_received"
	NotificationPartialPaymentReceived NotificationType = "partial_payment_received"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusDismissed NotificationStatus = "dismissed"
)

// NotificationChannel represents the delivery channel for notifications
type NotificationChannel string

const (
	NotificationChannelEmail   NotificationChannel = "email"
	NotificationChannelSMS     NotificationChannel = "sms"
	NotificationChannelPush    NotificationChannel = "push"
	NotificationChannelInApp   NotificationChannel = "in_app"
	NotificationChannelWebhook NotificationChannel = "webhook"
)

// PaymentNotification represents a payment notification
type PaymentNotification struct {
	ID               uuid.UUID              `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	InvoiceID        uuid.UUID              `gorm:"type:uuid;not null" json:"invoice_id"`
	NotificationType NotificationType       `gorm:"type:notification_type;not null" json:"notification_type"`
	Status           NotificationStatus     `gorm:"type:notification_status;not null;default:pending" json:"status"`
	Channel          NotificationChannel    `gorm:"type:notification_channel;not null;default:email" json:"channel"`
	RecipientID      *uuid.UUID             `gorm:"type:uuid" json:"recipient_id,omitempty"`
	RecipientEmail   string                 `gorm:"type:varchar(255)" json:"recipient_email,omitempty"`
	RecipientPhone   string                 `gorm:"type:varchar(50)" json:"recipient_phone,omitempty"`
	Subject          string                 `gorm:"type:varchar(500)" json:"subject,omitempty"`
	Message          string                 `gorm:"type:text;not null" json:"message"`
	Data             map[string]interface{} `gorm:"type:jsonb" json:"data,omitempty"`
	ScheduledAt      time.Time              `gorm:"not null;default:NOW()" json:"scheduled_at"`
	SentAt           *time.Time             `json:"sent_at,omitempty"`
	RetryCount       int                    `gorm:"default:0" json:"retry_count"`
	MaxRetries       int                    `gorm:"default:3" json:"max_retries"`
	LastError        string                 `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt        time.Time              `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time              `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (PaymentNotification) TableName() string {
	return "payment_notifications"
}

// PaymentNotificationRule represents a notification rule configuration
type PaymentNotificationRule struct {
	ID               uuid.UUID             `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name             string                `gorm:"type:varchar(200);not null" json:"name"`
	Description      string                `gorm:"type:text" json:"description,omitempty"`
	NotificationType NotificationType      `gorm:"type:notification_type;not null" json:"notification_type"`
	DaysBeforeDue    *int                  `json:"days_before_due,omitempty"`
	DaysAfterDue     *int                  `json:"days_after_due,omitempty"`
	IsActive         bool                  `gorm:"default:true" json:"is_active"`
	Channels         []NotificationChannel `gorm:"type:notification_channel[];not null;default:ARRAY['email']::notification_channel[]" json:"channels"`
	EmailTemplateID  *uuid.UUID            `gorm:"type:uuid" json:"email_template_id,omitempty"`
	SMSTemplateID    *uuid.UUID            `gorm:"type:uuid" json:"sms_template_id,omitempty"`
	CreatedAt        time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (PaymentNotificationRule) TableName() string {
	return "payment_notification_rules"
}

// PaymentNotificationHistory represents the delivery history of a notification
type PaymentNotificationHistory struct {
	ID             uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NotificationID uuid.UUID           `gorm:"type:uuid;not null" json:"notification_id"`
	Status         NotificationStatus  `gorm:"type:notification_status;not null" json:"status"`
	Channel        NotificationChannel `gorm:"type:notification_channel;not null" json:"channel"`
	SentAt         *time.Time          `json:"sent_at,omitempty"`
	DeliveredAt    *time.Time          `json:"delivered_at,omitempty"`
	OpenedAt       *time.Time          `json:"opened_at,omitempty"`
	ClickedAt      *time.Time          `json:"clicked_at,omitempty"`
	ErrorMessage   string              `gorm:"type:text" json:"error_message,omitempty"`
	ResponseCode   string              `gorm:"type:varchar(50)" json:"response_code,omitempty"`
	ResponseBody   string              `gorm:"type:text" json:"response_body,omitempty"`
	CreatedAt      time.Time           `gorm:"autoCreateTime" json:"created_at"`
}

// TableName overrides the table name
func (PaymentNotificationHistory) TableName() string {
	return "payment_notification_history"
}

// PaymentNotificationService handles notification operations
type PaymentNotificationService struct {
	db *gorm.DB
}

// NewPaymentNotificationService creates a new notification service
func NewPaymentNotificationService(db *gorm.DB) *PaymentNotificationService {
	return &PaymentNotificationService{db: db}
}

// CreateNotification creates a new payment notification
func (s *PaymentNotificationService) CreateNotification(ctx context.Context, notification *PaymentNotification) error {
	if notification.InvoiceID == uuid.Nil {
		return errors.New("invoice_id is required")
	}

	if notification.Message == "" {
		return errors.New("message is required")
	}

	if err := s.db.WithContext(ctx).Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// GetPendingNotifications retrieves pending notifications for processing
func (s *PaymentNotificationService) GetPendingNotifications(ctx context.Context, batchSize int) ([]PaymentNotification, error) {
	var notifications []PaymentNotification

	// Use raw SQL to leverage database-level locking
	err := s.db.WithContext(ctx).Raw(`
		SELECT * FROM get_pending_notifications(?)
	`, batchSize).Scan(&notifications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}

	return notifications, nil
}

// MarkNotificationSent marks a notification as successfully sent
func (s *PaymentNotificationService) MarkNotificationSent(ctx context.Context, notificationID uuid.UUID) error {
	_, err := s.db.WithContext(ctx).Exec(`SELECT mark_notification_sent(?)`, notificationID).Rows()
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}
	return nil
}

// MarkNotificationFailed marks a notification as failed
func (s *PaymentNotificationService) MarkNotificationFailed(ctx context.Context, notificationID uuid.UUID, errorMessage string) error {
	_, err := s.db.WithContext(ctx).Exec(`SELECT mark_notification_failed(?, ?)`, notificationID, errorMessage).Rows()
	if err != nil {
		return fmt.Errorf("failed to mark notification as failed: %w", err)
	}
	return nil
}

// GetNotificationsByInvoice retrieves all notifications for an invoice
func (s *PaymentNotificationService) GetNotificationsByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]PaymentNotification, error) {
	var notifications []PaymentNotification

	if err := s.db.WithContext(ctx).
		Where("invoice_id = ?", invoiceID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	return notifications, nil
}

// GetNotificationHistory retrieves the delivery history for a notification
func (s *PaymentNotificationService) GetNotificationHistory(ctx context.Context, notificationID uuid.UUID) ([]PaymentNotificationHistory, error) {
	var history []PaymentNotificationHistory

	if err := s.db.WithContext(ctx).
		Where("notification_id = ?", notificationID).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}

	return history, nil
}

// CheckAllInvoicesForNotifications triggers notification checks for all invoices
func (s *PaymentNotificationService) CheckAllInvoicesForNotifications(ctx context.Context) (int, error) {
	var count int
	err := s.db.WithContext(ctx).Raw(`SELECT check_all_invoices_for_notifications()`).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to check invoices for notifications: %w", err)
	}
	return count, nil
}

// GetNotificationRules retrieves all notification rules
func (s *PaymentNotificationService) GetNotificationRules(ctx context.Context, activeOnly bool) ([]PaymentNotificationRule, error) {
	var rules []PaymentNotificationRule

	query := s.db.WithContext(ctx)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification rules: %w", err)
	}

	return rules, nil
}

// CreateNotificationRule creates a new notification rule
func (s *PaymentNotificationService) CreateNotificationRule(ctx context.Context, rule *PaymentNotificationRule) error {
	if rule.Name == "" {
		return errors.New("rule name is required")
	}

	if len(rule.Channels) == 0 {
		return errors.New("at least one notification channel is required")
	}

	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("failed to create notification rule: %w", err)
	}

	return nil
}

// UpdateNotificationRule updates an existing notification rule
func (s *PaymentNotificationService) UpdateNotificationRule(ctx context.Context, id uuid.UUID, updates *PaymentNotificationRule) error {
	if err := s.db.WithContext(ctx).Model(&PaymentNotificationRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update notification rule: %w", err)
	}
	return nil
}

// DeleteNotificationRule deletes a notification rule
func (s *PaymentNotificationService) DeleteNotificationRule(ctx context.Context, id uuid.UUID) error {
	if err := s.db.WithContext(ctx).Delete(&PaymentNotificationRule{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete notification rule: %w", err)
	}
	return nil
}

// NotificationProcessor processes and sends notifications
type NotificationProcessor struct {
	db              *gorm.DB
	notificationSvc *PaymentNotificationService
	emailSender     EmailSender
	smsSender       SMSSender
}

// EmailSender interface for sending emails
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMSSender interface for sending SMS
type SMSSender interface {
	Send(ctx context.Context, to, message string) error
}

// NewNotificationProcessor creates a new notification processor
func NewNotificationProcessor(db *gorm.DB, emailSender EmailSender, smsSender SMSSender) *NotificationProcessor {
	return &NotificationProcessor{
		db:              db,
		notificationSvc: NewPaymentNotificationService(db),
		emailSender:     emailSender,
		smsSender:       smsSender,
	}
}

// ProcessBatch processes a batch of pending notifications
func (p *NotificationProcessor) ProcessBatch(ctx context.Context, batchSize int) error {
	notifications, err := p.notificationSvc.GetPendingNotifications(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}

	for _, notification := range notifications {
		if err := p.ProcessNotification(ctx, &notification); err != nil {
			// Log error but continue processing other notifications
			fmt.Printf("Error processing notification %s: %v\n", notification.ID, err)
		}
	}

	return nil
}

// ProcessNotification processes a single notification
func (p *NotificationProcessor) ProcessNotification(ctx context.Context, notification *PaymentNotification) error {
	var err error

	switch notification.Channel {
	case NotificationChannelEmail:
		err = p.sendEmailNotification(ctx, notification)
	case NotificationChannelSMS:
		err = p.sendSMSNotification(ctx, notification)
	case NotificationChannelPush, NotificationChannelInApp:
		// These would be implemented with push notification service
		err = fmt.Errorf("channel %s not yet implemented", notification.Channel)
	case NotificationChannelWebhook:
		// This would be implemented with webhook delivery
		err = fmt.Errorf("channel %s not yet implemented", notification.Channel)
	default:
		err = fmt.Errorf("unknown notification channel: %s", notification.Channel)
	}

	if err != nil {
		return p.notificationSvc.MarkNotificationFailed(ctx, notification.ID, err.Error())
	}

	return p.notificationSvc.MarkNotificationSent(ctx, notification.ID)
}

// sendEmailNotification sends an email notification
func (p *NotificationProcessor) sendEmailNotification(ctx context.Context, notification *PaymentNotification) error {
	if p.emailSender == nil {
		return errors.New("email sender not configured")
	}

	if notification.RecipientEmail == "" {
		return errors.New("recipient email is required")
	}

	subject := notification.Subject
	if subject == "" {
		subject = fmt.Sprintf("Payment Notification - %s", notification.NotificationType)
	}

	// Format message with data if available
	message := notification.Message
	if len(notification.Data) > 0 {
		dataJSON, _ := json.MarshalIndent(notification.Data, "", "  ")
		message = fmt.Sprintf("%s\n\nDetails:\n%s", message, string(dataJSON))
	}

	return p.emailSender.Send(ctx, notification.RecipientEmail, subject, message)
}

// sendSMSNotification sends an SMS notification
func (p *NotificationProcessor) sendSMSNotification(ctx context.Context, notification *PaymentNotification) error {
	if p.smsSender == nil {
		return errors.New("SMS sender not configured")
	}

	if notification.RecipientPhone == "" {
		return errors.New("recipient phone is required")
	}

	// Truncate message for SMS (typically 160 characters)
	message := notification.Message
	if len(message) > 160 {
		message = message[:157] + "..."
	}

	return p.smsSender.Send(ctx, notification.RecipientPhone, message)
}

// ScheduleNotification schedules a notification to be sent at a specific time
func (s *PaymentNotificationService) ScheduleNotification(ctx context.Context, notification *PaymentNotification, scheduledAt time.Time) error {
	notification.ScheduledAt = scheduledAt
	return s.CreateNotification(ctx, notification)
}

// DismissNotification marks a notification as dismissed
func (s *PaymentNotificationService) DismissNotification(ctx context.Context, notificationID uuid.UUID) error {
	if err := s.db.WithContext(ctx).
		Model(&PaymentNotification{}).
		Where("id = ?", notificationID).
		Update("status", NotificationStatusDismissed).Error; err != nil {
		return fmt.Errorf("failed to dismiss notification: %w", err)
	}
	return nil
}

// GetNotificationStats retrieves notification statistics
func (s *PaymentNotificationService) GetNotificationStats(ctx context.Context, from, to time.Time) (map[string]interface{}, error) {
	var stats struct {
		Total     int64 `json:"total"`
		Pending   int64 `json:"pending"`
		Sent      int64 `json:"sent"`
		Failed    int64 `json:"failed"`
		Dismissed int64 `json:"dismissed"`
	}

	baseQuery := s.db.WithContext(ctx).Model(&PaymentNotification{}).
		Where("created_at BETWEEN ? AND ?", from, to)

	if err := baseQuery.Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	baseQuery.Where("status = ?", NotificationStatusPending).Count(&stats.Pending)
	baseQuery.Where("status = ?", NotificationStatusSent).Count(&stats.Sent)
	baseQuery.Where("status = ?", NotificationStatusFailed).Count(&stats.Failed)
	baseQuery.Where("status = ?", NotificationStatusDismissed).Count(&stats.Dismissed)

	// Get counts by type
	var typeCounts []struct {
		NotificationType NotificationType `json:"type"`
		Count            int64            `json:"count"`
	}

	s.db.WithContext(ctx).
		Model(&PaymentNotification{}).
		Select("notification_type, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", from, to).
		Group("notification_type").
		Scan(&typeCounts)

	result := map[string]interface{}{
		"total":     stats.Total,
		"pending":   stats.Pending,
		"sent":      stats.Sent,
		"failed":    stats.Failed,
		"dismissed": stats.Dismissed,
		"by_type":   typeCounts,
	}

	return result, nil
}
