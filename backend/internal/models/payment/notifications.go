package payment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "email"
	NotificationTypeSMS     NotificationType = "sms"
	NotificationTypePush    NotificationType = "push"
	NotificationTypeWebhook NotificationType = "webhook"
	NotificationTypeInApp   NotificationType = "in_app"
	NotificationTypeSlack   NotificationType = "slack"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusQueued    NotificationStatus = "queued"
	NotificationStatusSending   NotificationStatus = "sending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusBounced   NotificationStatus = "bounced"
	NotificationStatusSkipped   NotificationStatus = "skipped"
)

// NotificationPriority represents the priority of a notification
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// PaymentStatusNotification represents a notification about a status change
type PaymentStatusNotification struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_payment_notifications" json:"payment_id"`
	StatusHistoryID *uuid.UUID `gorm:"type:uuid;index" json:"status_history_id,omitempty"`

	// Status information
	FromStatus *PaymentStatus `gorm:"type:varchar(50)" json:"from_status,omitempty"`
	ToStatus   PaymentStatus  `gorm:"type:varchar(50);not null" json:"to_status"`

	// Notification details
	NotificationType  NotificationType `gorm:"type:varchar(20);not null" json:"notification_type"`
	RecipientID       *uuid.UUID       `gorm:"type:uuid;index" json:"recipient_id,omitempty"`
	RecipientType     string           `gorm:"type:varchar(20)" json:"recipient_type"` // user, admin, system
	RecipientEmail    string           `gorm:"type:varchar(255)" json:"recipient_email,omitempty"`
	RecipientPhone    string           `gorm:"type:varchar(50)" json:"recipient_phone,omitempty"`
	RecipientEndpoint string           `gorm:"type:text" json:"recipient_endpoint,omitempty"` // webhook URL, etc.

	// Content
	Subject      string                 `gorm:"type:varchar(500)" json:"subject"`
	Body         string                 `gorm:"type:text" json:"body"`
	BodyHTML     string                 `gorm:"type:text" json:"body_html,omitempty"`
	TemplateID   string                 `gorm:"type:varchar(100)" json:"template_id,omitempty"`
	TemplateData map[string]interface{} `gorm:"type:jsonb" json:"template_data,omitempty"`

	// Status and delivery
	Status       NotificationStatus   `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Priority     NotificationPriority `gorm:"type:varchar(20);not null;default:'normal'" json:"priority"`
	ScheduledFor *time.Time           `gorm:"index" json:"scheduled_for,omitempty"`
	SentAt       *time.Time           `json:"sent_at,omitempty"`
	DeliveredAt  *time.Time           `json:"delivered_at,omitempty"`
	FailedAt     *time.Time           `json:"failed_at,omitempty"`

	// Delivery tracking
	AttemptCount int        `gorm:"not null;default:0" json:"attempt_count"`
	MaxAttempts  int        `gorm:"not null;default:3" json:"max_attempts"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`

	// Provider details
	ProviderID       string                 `gorm:"type:varchar(100)" json:"provider_id,omitempty"` // External provider's ID
	ProviderResponse map[string]interface{} `gorm:"type:jsonb" json:"provider_response,omitempty"`

	// Metadata
	Metadata map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`
	Tags     []string          `gorm:"type:text[]" json:"tags,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (PaymentStatusNotification) TableName() string {
	return "payment_status_notifications"
}

// BeforeCreate hook
func (psn *PaymentStatusNotification) BeforeCreate(tx *gorm.DB) error {
	if psn.ID == uuid.Nil {
		psn.ID = uuid.New()
	}
	if psn.Status == "" {
		psn.Status = NotificationStatusPending
	}
	if psn.Priority == "" {
		psn.Priority = PriorityNormal
	}
	return nil
}

// CanRetry checks if the notification can be retried
func (psn *PaymentStatusNotification) CanRetry() bool {
	return psn.AttemptCount < psn.MaxAttempts &&
		psn.Status != NotificationStatusSent &&
		psn.Status != NotificationStatusDelivered &&
		psn.Status != NotificationStatusSkipped
}

// ShouldRetry checks if it's time to retry the notification
func (psn *PaymentStatusNotification) ShouldRetry() bool {
	if !psn.CanRetry() {
		return false
	}

	if psn.NextRetryAt == nil {
		return true
	}

	return time.Now().After(*psn.NextRetryAt)
}

// NotificationRule represents rules for when to send notifications
type NotificationRule struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(200);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`

	// Trigger conditions
	TriggerStatuses   []PaymentStatus  `gorm:"type:text[]" json:"trigger_statuses"`
	TriggerCategories []StatusCategory `gorm:"type:text[]" json:"trigger_categories,omitempty"`
	FromStatuses      []PaymentStatus  `gorm:"type:text[]" json:"from_statuses,omitempty"`

	// Notification settings
	NotificationType NotificationType     `gorm:"type:varchar(20);not null" json:"notification_type"`
	RecipientType    string               `gorm:"type:varchar(20);not null" json:"recipient_type"` // user, admin, custom
	RecipientList    []string             `gorm:"type:text[]" json:"recipient_list,omitempty"`     // For custom recipients
	Priority         NotificationPriority `gorm:"type:varchar(20);not null;default:'normal'" json:"priority"`

	// Template
	TemplateID      string `gorm:"type:varchar(100)" json:"template_id,omitempty"`
	SubjectTemplate string `gorm:"type:varchar(500)" json:"subject_template,omitempty"`
	BodyTemplate    string `gorm:"type:text" json:"body_template,omitempty"`

	// Conditions
	Conditions      map[string]interface{} `gorm:"type:jsonb" json:"conditions,omitempty"`
	TimeConstraints *TimeConstraints       `gorm:"type:jsonb" json:"time_constraints,omitempty"`

	// Settings
	IsActive         bool `gorm:"not null;default:true" json:"is_active"`
	MaxAttempts      int  `gorm:"not null;default:3" json:"max_attempts"`
	RetryDelay       int  `gorm:"not null;default:300" json:"retry_delay"`        // seconds
	DeduplicationTTL int  `gorm:"not null;default:3600" json:"deduplication_ttl"` // seconds

	// Metadata
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (NotificationRule) TableName() string {
	return "payment_notification_rules"
}

// BeforeCreate hook
func (nr *NotificationRule) BeforeCreate(tx *gorm.DB) error {
	if nr.ID == uuid.Nil {
		nr.ID = uuid.New()
	}
	return nil
}

// TimeConstraints represents time-based constraints for notifications
type TimeConstraints struct {
	AllowedDays                  []time.Weekday `json:"allowed_days,omitempty"`        // Only send on these days
	AllowedHoursStart            int            `json:"allowed_hours_start,omitempty"` // 0-23
	AllowedHoursEnd              int            `json:"allowed_hours_end,omitempty"`   // 0-23
	Timezone                     string         `json:"timezone,omitempty"`            // IANA timezone
	MinTimeSinceLastNotification time.Duration  `json:"min_time_since_last,omitempty"`
}

// NotificationTemplate represents a reusable notification template
type NotificationTemplate struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name             string           `gorm:"type:varchar(200);not null;unique" json:"name"`
	Description      string           `gorm:"type:text" json:"description,omitempty"`
	NotificationType NotificationType `gorm:"type:varchar(20);not null" json:"notification_type"`

	// Content
	SubjectTemplate  string `gorm:"type:varchar(500)" json:"subject_template"`
	BodyTemplate     string `gorm:"type:text;not null" json:"body_template"`
	BodyHTMLTemplate string `gorm:"type:text" json:"body_html_template,omitempty"`

	// Variables available in this template
	AvailableVariables []string               `gorm:"type:text[]" json:"available_variables,omitempty"`
	SampleData         map[string]interface{} `gorm:"type:jsonb" json:"sample_data,omitempty"`

	// Settings
	IsActive bool `gorm:"not null;default:true" json:"is_active"`
	Version  int  `gorm:"not null;default:1" json:"version"`

	// Metadata
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (NotificationTemplate) TableName() string {
	return "payment_notification_templates"
}

// BeforeCreate hook
func (nt *NotificationTemplate) BeforeCreate(tx *gorm.DB) error {
	if nt.ID == uuid.Nil {
		nt.ID = uuid.New()
	}
	return nil
}

// NotificationPreference represents user preferences for notifications
type NotificationPreference struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;unique" json:"user_id"`

	// Channel preferences
	EmailEnabled bool `gorm:"not null;default:true" json:"email_enabled"`
	SMSEnabled   bool `gorm:"not null;default:false" json:"sms_enabled"`
	PushEnabled  bool `gorm:"not null;default:true" json:"push_enabled"`
	InAppEnabled bool `gorm:"not null;default:true" json:"in_app_enabled"`

	// Status-specific preferences
	StatusPreferences map[PaymentStatus]ChannelPreferences `gorm:"type:jsonb" json:"status_preferences,omitempty"`

	// Quiet hours
	QuietHoursStart *int   `json:"quiet_hours_start,omitempty"` // 0-23
	QuietHoursEnd   *int   `json:"quiet_hours_end,omitempty"`   // 0-23
	Timezone        string `gorm:"type:varchar(100)" json:"timezone,omitempty"`

	// Frequency limits
	MaxNotificationsPerDay      int `gorm:"not null;default:20" json:"max_notifications_per_day"`
	MinTimeBetweenNotifications int `gorm:"not null;default:60" json:"min_time_between_notifications"` // seconds

	// Metadata
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (NotificationPreference) TableName() string {
	return "payment_notification_preferences"
}

// BeforeCreate hook
func (np *NotificationPreference) BeforeCreate(tx *gorm.DB) error {
	if np.ID == uuid.Nil {
		np.ID = uuid.New()
	}
	return nil
}

// ChannelPreferences represents notification preferences for specific channels
type ChannelPreferences struct {
	Email   bool `json:"email"`
	SMS     bool `json:"sms"`
	Push    bool `json:"push"`
	InApp   bool `json:"in_app"`
	Webhook bool `json:"webhook"`
}

// NotificationQuery represents a query for notifications
type NotificationQuery struct {
	PaymentIDs      []uuid.UUID            `json:"payment_ids,omitempty"`
	RecipientIDs    []uuid.UUID            `json:"recipient_ids,omitempty"`
	Types           []NotificationType     `json:"types,omitempty"`
	Statuses        []NotificationStatus   `json:"statuses,omitempty"`
	Priorities      []NotificationPriority `json:"priorities,omitempty"`
	StartDate       *time.Time             `json:"start_date,omitempty"`
	EndDate         *time.Time             `json:"end_date,omitempty"`
	PendingOnly     bool                   `json:"pending_only"`
	FailedOnly      bool                   `json:"failed_only"`
	ScheduledBefore *time.Time             `json:"scheduled_before,omitempty"`
	Limit           int                    `json:"limit"`
	Offset          int                    `json:"offset"`
}

// NotificationStatistics represents statistics about notifications
type NotificationStatistics struct {
	TotalNotifications  int64                        `json:"total_notifications"`
	SentCount           int64                        `json:"sent_count"`
	DeliveredCount      int64                        `json:"delivered_count"`
	FailedCount         int64                        `json:"failed_count"`
	PendingCount        int64                        `json:"pending_count"`
	TypeDistribution    map[NotificationType]int64   `json:"type_distribution"`
	StatusDistribution  map[NotificationStatus]int64 `json:"status_distribution"`
	AverageDeliveryTime time.Duration                `json:"average_delivery_time"`
	DeliveryRate        float64                      `json:"delivery_rate"`
	FailureRate         float64                      `json:"failure_rate"`
	Last24Hours         int64                        `json:"last_24_hours"`
	Last7Days           int64                        `json:"last_7_days"`
	GeneratedAt         time.Time                    `json:"generated_at"`
}

// NotificationBatch represents a batch of notifications to be sent
type NotificationBatch struct {
	ID               uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name             string               `gorm:"type:varchar(200)" json:"name,omitempty"`
	Description      string               `gorm:"type:text" json:"description,omitempty"`
	NotificationType NotificationType     `gorm:"type:varchar(20);not null" json:"notification_type"`
	Status           string               `gorm:"type:varchar(20);not null;default:'pending'" json:"status"` // pending, processing, completed, failed
	TotalCount       int                  `gorm:"not null;default:0" json:"total_count"`
	SentCount        int                  `gorm:"not null;default:0" json:"sent_count"`
	FailedCount      int                  `gorm:"not null;default:0" json:"failed_count"`
	Priority         NotificationPriority `gorm:"type:varchar(20);not null;default:'normal'" json:"priority"`
	ScheduledFor     *time.Time           `json:"scheduled_for,omitempty"`
	StartedAt        *time.Time           `json:"started_at,omitempty"`
	CompletedAt      *time.Time           `json:"completed_at,omitempty"`
	CreatedBy        *uuid.UUID           `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt        time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (NotificationBatch) TableName() string {
	return "payment_notification_batches"
}

// BeforeCreate hook
func (nb *NotificationBatch) BeforeCreate(tx *gorm.DB) error {
	if nb.ID == uuid.Nil {
		nb.ID = uuid.New()
	}
	return nil
}

// NotificationDeduplication tracks sent notifications to prevent duplicates
type NotificationDeduplication struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID        uuid.UUID        `gorm:"type:uuid;not null;index:idx_dedup" json:"payment_id"`
	RecipientID      uuid.UUID        `gorm:"type:uuid;not null;index:idx_dedup" json:"recipient_id"`
	NotificationType NotificationType `gorm:"type:varchar(20);not null;index:idx_dedup" json:"notification_type"`
	StatusChange     string           `gorm:"type:varchar(100);not null;index:idx_dedup" json:"status_change"` // "from_status->to_status"
	Hash             string           `gorm:"type:varchar(64);not null;unique" json:"hash"`                    // SHA256 hash of key fields
	SentAt           time.Time        `gorm:"not null;index" json:"sent_at"`
	ExpiresAt        time.Time        `gorm:"not null;index" json:"expires_at"`
}

// TableName overrides the table name
func (NotificationDeduplication) TableName() string {
	return "payment_notification_deduplication"
}

// BeforeCreate hook
func (nd *NotificationDeduplication) BeforeCreate(tx *gorm.DB) error {
	if nd.ID == uuid.Nil {
		nd.ID = uuid.New()
	}
	return nil
}

// IsExpired checks if the deduplication record has expired
func (nd *NotificationDeduplication) IsExpired() bool {
	return time.Now().After(nd.ExpiresAt)
}
