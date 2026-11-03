package models

import (
	"time"

	"github.com/google/uuid"
)

// Stripe v2 capability status values (configuration.merchant.capabilities.*.status).
const (
	CapabilityActive = "active"
)

// StripeConnectedAccount is a tenant's own Stripe account (Connect, Accounts
// v2, direct charges — D9, T169). The Stripe account ID is created by the
// server and never accepted from a client; the statuses are a cache of the v2
// account, refreshed from Stripe on webhooks and on the settings page.
type StripeConnectedAccount struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID           uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex"`
	StripeAccountID    string     `json:"-" gorm:"not null;uniqueIndex"`
	Livemode           bool       `json:"livemode" gorm:"not null;default:false"`
	CardPaymentsStatus string     `json:"card_payments_status" gorm:"not null;default:''"`
	ACHPaymentsStatus  string     `json:"ach_payments_status" gorm:"column:ach_payments_status;not null;default:''"`
	PayoutsStatus      string     `json:"payouts_status" gorm:"not null;default:''"`
	RequirementsStatus string     `json:"requirements_status" gorm:"not null;default:''"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName specifies the table name.
func (StripeConnectedAccount) TableName() string { return "stripe_connected_accounts" }

// CanAcceptCards reports whether the tenant can take card payments (and so
// may show pay links).
func (a *StripeConnectedAccount) CanAcceptCards() bool {
	return a != nil && a.CardPaymentsStatus == CapabilityActive
}

// CanAcceptACH reports whether the tenant can take US bank (ACH) payments.
func (a *StripeConnectedAccount) CanAcceptACH() bool {
	return a != nil && a.ACHPaymentsStatus == CapabilityActive
}

// OnboardingComplete reports whether Stripe needs nothing more right now.
func (a *StripeConnectedAccount) OnboardingComplete() bool {
	return a != nil && a.CanAcceptCards() &&
		a.RequirementsStatus != "currently_due" && a.RequirementsStatus != "past_due"
}

// StripeAccountStatus is the cached part of a v2 account.
type StripeAccountStatus struct {
	Livemode           bool
	CardPaymentsStatus string
	ACHPaymentsStatus  string
	PayoutsStatus      string
	RequirementsStatus string
}

// PaymentsConnectResponse is the onboarding link for POST /tenants/{id}/payments/connect.
type PaymentsConnectResponse struct {
	URL string `json:"url"`
}

// PaymentsStatusResponse is GET /tenants/{id}/payments/status.
type PaymentsStatusResponse struct {
	// Connected is true once a Stripe account exists for the tenant.
	Connected bool `json:"connected"`
	// State summarises onboarding: not_connected, onboarding, action_required, restricted, active.
	State string `json:"state"`
	// OnlinePaymentsEnabled is the platform-wide switch (INVOICE_PAYMENTS_ENABLED).
	OnlinePaymentsEnabled bool `json:"online_payments_enabled"`
	// PayLinksActive is true when invoice emails carry a pay link for this tenant.
	PayLinksActive     bool       `json:"pay_links_active"`
	CardPaymentsStatus string     `json:"card_payments_status"`
	ACHPaymentsStatus  string     `json:"ach_payments_status"`
	PayoutsStatus      string     `json:"payouts_status"`
	RequirementsStatus string     `json:"requirements_status"`
	Livemode           bool       `json:"livemode"`
	SyncedAt           *time.Time `json:"synced_at,omitempty"`
	// Fees are ServicePro's fees on top of Stripe's (D10).
	Fees PaymentFeesResponse `json:"fees"`
}

// PaymentFeesResponse describes ServicePro's application fees (D10).
type PaymentFeesResponse struct {
	CardPercent string `json:"card_percent"`
	ACHPercent  string `json:"ach_percent"`
	ACHCap      string `json:"ach_cap"`
	Currency    string `json:"currency"`
}

// Payment state values for PaymentsStatusResponse.State.
const (
	PaymentsStateNotConnected   = "not_connected"
	PaymentsStateOnboarding     = "onboarding"
	PaymentsStateActionRequired = "action_required"
	PaymentsStateRestricted     = "restricted"
	PaymentsStateActive         = "active"
)
