package models

import (
	"time"

	"github.com/google/uuid"
)

// QuickBooks connection statuses (T192).
const (
	QuickBooksConnected      = "connected"
	QuickBooksReauthRequired = "reauth_required"
	QuickBooksDisconnected   = "disconnected"
)

// QuickBooks sync entities and record statuses (T192).
const (
	QuickBooksEntityCustomer = "customer"
	QuickBooksEntityInvoice  = "invoice"
	QuickBooksEntityPayment  = "payment"

	QuickBooksSyncPending = "pending"
	QuickBooksSyncSynced  = "synced"
	QuickBooksSyncError   = "error"
)

// QuickBooksConnection is a tenant's link to its QuickBooks Online company
// (migration 000023). Tokens are sealed with pkg/secretbox; never serialise
// this struct to a client.
type QuickBooksConnection struct {
	ID                    uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID              uuid.UUID  `gorm:"type:uuid;not null"`
	Status                string     `gorm:"type:varchar(20);not null"`
	Environment           string     `gorm:"type:varchar(20);not null"`
	RealmID               *string    `gorm:"type:varchar(50)"`
	CompanyName           *string    `gorm:"type:varchar(255)"`
	AccessTokenEnc        *string    `gorm:"type:text"`
	AccessTokenExpiresAt  *time.Time `gorm:"type:timestamptz"`
	RefreshTokenEnc       *string    `gorm:"type:text"`
	RefreshTokenExpiresAt *time.Time `gorm:"type:timestamptz"`
	ConnectedBy           *uuid.UUID `gorm:"type:uuid"`
	ConnectedAt           *time.Time `gorm:"type:timestamptz"`
	DisconnectedAt        *time.Time `gorm:"type:timestamptz"`
	SyncFrom              *time.Time `gorm:"type:date"`
	OAuthStateHash        *string    `gorm:"column:oauth_state_hash;type:varchar(64)"`
	OAuthStateExpiresAt   *time.Time `gorm:"column:oauth_state_expires_at;type:timestamptz"`
	IncomeAccountID       *string    `gorm:"type:varchar(50)"`
	IncomeAccountName     *string    `gorm:"type:varchar(255)"`
	ItemID                *string    `gorm:"type:varchar(50)"`
	ItemName              *string    `gorm:"type:varchar(255)"`
	TaxCodeID             *string    `gorm:"type:varchar(50)"`
	TaxCodeName           *string    `gorm:"type:varchar(255)"`
	DepositAccountID      *string    `gorm:"type:varchar(50)"`
	DepositAccountName    *string    `gorm:"type:varchar(255)"`
	SyncLockedUntil       *time.Time `gorm:"type:timestamptz"`
	LastSyncAt            *time.Time `gorm:"type:timestamptz"`
	LastSyncError         *string    `gorm:"type:text"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

// TableName overrides the table name.
func (QuickBooksConnection) TableName() string { return "quickbooks_connections" }

// QuickBooksSyncRecord maps one local record to its QuickBooks entity and is
// its work-queue entry (migration 000023).
type QuickBooksSyncRecord struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null"`
	Entity         string     `gorm:"type:varchar(20);not null"`
	LocalID        uuid.UUID  `gorm:"type:uuid;not null"`
	QBOID          *string    `gorm:"column:qbo_id;type:varchar(50)"`
	SyncToken      *string    `gorm:"type:varchar(20)"`
	Status         string     `gorm:"type:varchar(20);not null"`
	Attempts       int        `gorm:"not null"`
	NextAttemptAt  time.Time  `gorm:"type:timestamptz;not null"`
	RequestID      *string    `gorm:"type:varchar(50)"`
	PayloadHash    *string    `gorm:"type:varchar(64)"`
	LocalUpdatedAt *time.Time `gorm:"type:timestamptz"`
	LastSyncedAt   *time.Time `gorm:"type:timestamptz"`
	LastError      *string    `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime"`
}

// TableName overrides the table name.
func (QuickBooksSyncRecord) TableName() string { return "quickbooks_sync_map" }

// QuickBooksRef is a QuickBooks entity picked in the mapping settings.
type QuickBooksRef struct {
	ID   string `json:"id" example:"79"`
	Name string `json:"name" example:"Cleaning Services"`
}

// QuickBooksSyncCounts counts synced records per entity.
type QuickBooksSyncCounts struct {
	Customers int64 `json:"customers" example:"23"`
	Invoices  int64 `json:"invoices" example:"118"`
	Payments  int64 `json:"payments" example:"96"`
	Pending   int64 `json:"pending" example:"2"`
	Errors    int64 `json:"errors" example:"0"`
}

// QuickBooksSyncRecordResponse is one record in the sync activity or error
// list.
type QuickBooksSyncRecordResponse struct {
	ID           uuid.UUID  `json:"id"`
	Entity       string     `json:"entity" example:"invoice"`
	LocalID      uuid.UUID  `json:"local_id"`
	Label        string     `json:"label" example:"Invoice INV-1042 · Harborview Property Management"`
	Status       string     `json:"status" example:"error"`
	Attempts     int        `json:"attempts" example:"2"`
	LastError    string     `json:"last_error,omitempty" example:"Choose an income account in Settings → Integrations"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// QuickBooksStatusResponse is the Integrations settings view of QuickBooks.
type QuickBooksStatusResponse struct {
	// Configured is false when the server has no Intuit app keys: the UI
	// shows "Coming soon".
	Configured  bool       `json:"configured" example:"true"`
	Status      string     `json:"status" example:"connected" enums:"connected,reauth_required,disconnected"`
	Environment string     `json:"environment,omitempty" example:"sandbox"`
	CompanyName string     `json:"company_name,omitempty" example:"Brightline Facility Services LLC"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	SyncFrom    *string    `json:"sync_from,omitempty" example:"2026-11-04"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	// LastSyncError is the last connection-level problem (e.g. the
	// QuickBooks authorization expired), not a per-record error.
	LastSyncError  string                         `json:"last_sync_error,omitempty"`
	IncomeAccount  *QuickBooksRef                 `json:"income_account,omitempty"`
	Item           *QuickBooksRef                 `json:"item,omitempty"`
	TaxCode        *QuickBooksRef                 `json:"tax_code,omitempty"`
	DepositAccount *QuickBooksRef                 `json:"deposit_account,omitempty"`
	Counts         QuickBooksSyncCounts           `json:"counts"`
	Errors         []QuickBooksSyncRecordResponse `json:"errors"`
	Recent         []QuickBooksSyncRecordResponse `json:"recent"`
}

// QuickBooksConnectResponse carries the Intuit consent URL to open.
type QuickBooksConnectResponse struct {
	URL string `json:"url" example:"https://appcenter.intuit.com/connect/oauth2?client_id=...&state=..."`
}

// QuickBooksOptionsResponse lists what the mapping settings can pick from
// the connected company.
type QuickBooksOptionsResponse struct {
	IncomeAccounts  []QuickBooksRef `json:"income_accounts"`
	TaxCodes        []QuickBooksRef `json:"tax_codes"`
	DepositAccounts []QuickBooksRef `json:"deposit_accounts"`
}

// QuickBooksSettingsRequest updates the mapping settings. Empty IDs clear
// the optional mappings (tax code, deposit account).
type QuickBooksSettingsRequest struct {
	IncomeAccountID  string `json:"income_account_id" binding:"max=50" example:"79"`
	TaxCodeID        string `json:"tax_code_id" binding:"max=50" example:"3"`
	DepositAccountID string `json:"deposit_account_id" binding:"max=50" example:"35"`
}

// QuickBooksSyncResponse reports one sync pass.
type QuickBooksSyncResponse struct {
	Enqueued int  `json:"enqueued" example:"3"`
	Synced   int  `json:"synced" example:"3"`
	Failed   int  `json:"failed" example:"0"`
	Skipped  int  `json:"skipped" example:"0"`
	More     bool `json:"more" example:"false"`
}
