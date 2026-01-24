package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// MembershipStatus represents the status of a subscription
type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusCancelled MembershipStatus = "cancelled"
	MembershipStatusExpired   MembershipStatus = "expired"
	MembershipStatusPending   MembershipStatus = "pending"
)

// BillingCycle represents the billing frequency
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleAnnual  BillingCycle = "annual"
)

// TierFeatures represents the features available in a membership tier
type TierFeatures struct {
	TeamMembers       int  `json:"team_members"`     // -1 for unlimited
	Customers         int  `json:"customers"`        // -1 for unlimited
	JobsPerMonth      int  `json:"jobs_per_month"`   // -1 for unlimited
	QuotesPerMonth    int  `json:"quotes_per_month"` // -1 for unlimited
	StorageGB         int  `json:"storage_gb"`
	AdvancedReporting bool `json:"advanced_reporting"`
	APIAccess         bool `json:"api_access"`
	Webhooks          bool `json:"webhooks"`
	CustomBranding    bool `json:"custom_branding"`
}

// Value implements driver.Valuer for database serialization
func (f TierFeatures) Value() (driver.Value, error) {
	return json.Marshal(f)
}

// Scan implements sql.Scanner for database deserialization
func (f *TierFeatures) Scan(value any) error {
	if value == nil {
		*f = TierFeatures{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan TierFeatures: invalid type")
	}

	if len(bytes) == 0 {
		*f = TierFeatures{}
		return nil
	}

	return json.Unmarshal(bytes, f)
}

// MembershipTier represents a membership tier definition
type MembershipTier struct {
	ID                uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name              string       `json:"name" gorm:"uniqueIndex;not null;size:50"`
	DisplayName       string       `json:"display_name" gorm:"not null;size:100"`
	Description       string       `json:"description,omitempty" gorm:"type:text"`
	MonthlyPriceCents int          `json:"monthly_price_cents" gorm:"not null;default:0"`
	AnnualPriceCents  int          `json:"annual_price_cents" gorm:"not null;default:0"`
	Features          TierFeatures `json:"features" gorm:"type:jsonb;not null;default:'{}'"`
	Enabled           bool         `json:"enabled" gorm:"not null;default:true"` // FALSE for legacy tiers
	SortOrder         int          `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// TableName specifies the table name for MembershipTier
func (MembershipTier) TableName() string {
	return "membership_tiers"
}

// MembershipLimitation represents a resource limit for a tier
type MembershipLimitation struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TierID       uuid.UUID `json:"tier_id" gorm:"type:uuid;not null"`
	ResourceType string    `json:"resource_type" gorm:"not null;size:50"`
	LimitValue   int       `json:"limit_value" gorm:"not null"` // -1 for unlimited
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Tier *MembershipTier `json:"tier,omitempty" gorm:"foreignKey:TierID"`
}

// TableName specifies the table name for MembershipLimitation
func (MembershipLimitation) TableName() string {
	return "membership_limitations"
}

// TenantSubscription represents a tenant's subscription to a membership tier
type TenantSubscription struct {
	ID                 uuid.UUID        `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID           uuid.UUID        `json:"tenant_id" gorm:"type:uuid;not null"`
	TierID             uuid.UUID        `json:"tier_id" gorm:"type:uuid;not null"`
	Status             MembershipStatus `json:"status" gorm:"type:membership_status;not null;default:'active'"`
	BillingCycle       BillingCycle     `json:"billing_cycle" gorm:"type:billing_cycle;not null;default:'monthly'"`
	PriceCents         int              `json:"price_cents" gorm:"not null;default:0"` // Locked-in price at subscription time
	StartedAt          time.Time        `json:"started_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	EndedAt            *time.Time       `json:"ended_at,omitempty"`
	CurrentPeriodStart time.Time        `json:"current_period_start" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CurrentPeriodEnd   *time.Time       `json:"current_period_end,omitempty"`
	CancelledAt        *time.Time       `json:"cancelled_at,omitempty"`
	ChangedBy          *uuid.UUID       `json:"changed_by,omitempty" gorm:"type:uuid"`
	ChangeReason       string           `json:"change_reason,omitempty" gorm:"type:text"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`

	// Relations
	Tenant *Tenant         `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	Tier   *MembershipTier `json:"tier,omitempty" gorm:"foreignKey:TierID"`
	User   *User           `json:"changed_by_user,omitempty" gorm:"foreignKey:ChangedBy"`
}

// TableName specifies the table name for TenantSubscription
func (TenantSubscription) TableName() string {
	return "tenant_subscriptions"
}

// ============================================================================
// Request/Response types
// ============================================================================

// GetMembershipResponse represents the response for getting tenant membership
type GetMembershipResponse struct {
	Subscription *TenantSubscriptionResponse `json:"subscription"`
	Tier         *MembershipTierResponse     `json:"tier"`
}

// MembershipTierResponse represents the API response for a membership tier
type MembershipTierResponse struct {
	ID                uuid.UUID    `json:"id"`
	Name              string       `json:"name"`
	DisplayName       string       `json:"display_name"`
	Description       string       `json:"description,omitempty"`
	MonthlyPriceCents int          `json:"monthly_price_cents"`
	AnnualPriceCents  int          `json:"annual_price_cents"`
	Features          TierFeatures `json:"features"`
	Enabled           bool         `json:"enabled"`
	SortOrder         int          `json:"sort_order"`
}

// ToResponse converts a MembershipTier model to MembershipTierResponse
func (t *MembershipTier) ToResponse() MembershipTierResponse {
	return MembershipTierResponse{
		ID:                t.ID,
		Name:              t.Name,
		DisplayName:       t.DisplayName,
		Description:       t.Description,
		MonthlyPriceCents: t.MonthlyPriceCents,
		AnnualPriceCents:  t.AnnualPriceCents,
		Features:          t.Features,
		Enabled:           t.Enabled,
		SortOrder:         t.SortOrder,
	}
}

// TenantSubscriptionResponse represents the API response for a subscription
type TenantSubscriptionResponse struct {
	ID                 uuid.UUID        `json:"id"`
	TenantID           uuid.UUID        `json:"tenant_id"`
	TierID             uuid.UUID        `json:"tier_id"`
	TierName           string           `json:"tier_name"`
	TierDisplayName    string           `json:"tier_display_name"`
	Status             MembershipStatus `json:"status"`
	BillingCycle       BillingCycle     `json:"billing_cycle"`
	PriceCents         int              `json:"price_cents"`
	IsLegacy           bool             `json:"is_legacy"` // True if tier is disabled (grandfathered)
	StartedAt          time.Time        `json:"started_at"`
	EndedAt            *time.Time       `json:"ended_at,omitempty"`
	CurrentPeriodStart time.Time        `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time       `json:"current_period_end,omitempty"`
}

// ToResponse converts a TenantSubscription model to TenantSubscriptionResponse
func (s *TenantSubscription) ToResponse() TenantSubscriptionResponse {
	response := TenantSubscriptionResponse{
		ID:                 s.ID,
		TenantID:           s.TenantID,
		TierID:             s.TierID,
		Status:             s.Status,
		BillingCycle:       s.BillingCycle,
		PriceCents:         s.PriceCents,
		StartedAt:          s.StartedAt,
		EndedAt:            s.EndedAt,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
	}

	if s.Tier != nil {
		response.TierName = s.Tier.Name
		response.TierDisplayName = s.Tier.DisplayName
		response.IsLegacy = !s.Tier.Enabled
	}

	return response
}

// UpdateMembershipRequest represents the request to update a tenant's membership
type UpdateMembershipRequest struct {
	TierID       uuid.UUID    `json:"tier_id" binding:"required"`
	BillingCycle BillingCycle `json:"billing_cycle" binding:"required,oneof=monthly annual"`
	ChangeReason string       `json:"change_reason,omitempty"`
}

// ListTiersResponse represents the response for listing all tiers
type ListTiersResponse struct {
	Tiers []MembershipTierResponse `json:"tiers"`
}
