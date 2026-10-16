package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Quote represents a customer quote
type Quote struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null;index"` // set from the caller's tenant (internal/tenancy)
	CustomerID    uuid.UUID       `json:"customer_id" gorm:"type:uuid;not null;index"`
	Customer      *Customer       `json:"customer,omitempty" gorm:"foreignKey:CustomerID;constraint:OnDelete:CASCADE"`
	QuoteNumber   string          `json:"quote_number" gorm:"type:varchar(50);uniqueIndex;not null"`
	Status        QuoteStatus     `json:"status" gorm:"type:quote_status;not null;default:'draft';index"`
	ValidUntil    time.Time       `json:"valid_until" gorm:"not null;index"`
	Subtotal      decimal.Decimal `json:"subtotal" gorm:"type:decimal(12,2);not null;default:0.00"`
	TaxRate       decimal.Decimal `json:"tax_rate" gorm:"type:decimal(5,4);not null;default:0.0000"`
	TaxAmount     decimal.Decimal `json:"tax_amount" gorm:"type:decimal(12,2);not null;default:0.00"`
	Total         decimal.Decimal `json:"total" gorm:"type:decimal(12,2);not null;default:0.00"`
	Currency      string          `json:"currency" gorm:"type:varchar(3);not null;default:'USD'"` // ISO 4217, from the tenant setting at create time
	Notes         *string         `json:"notes,omitempty" gorm:"type:text"`
	Terms         *string         `json:"terms,omitempty" gorm:"type:text"`
	Items         []QuoteItem     `json:"items,omitempty" gorm:"foreignKey:QuoteID;constraint:OnDelete:CASCADE"`
	CreatedBy     uuid.UUID       `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser *User           `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
	UpdatedBy     *uuid.UUID      `json:"updated_by,omitempty" gorm:"type:uuid"`
	UpdatedByUser *User           `json:"updated_by_user,omitempty" gorm:"foreignKey:UpdatedBy"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     gorm.DeletedAt  `json:"-" gorm:"index"`

	// ApplyDefaultTaxRate is a create-time input only (not stored): the
	// request gave no tax rate, so BeforeCreate applies the tenant's default
	// tax rate (TenantSettings.DefaultTaxRateID, T127) and recomputes totals.
	ApplyDefaultTaxRate bool `json:"-" gorm:"-" swaggerignore:"true"`
}

// QuoteItem represents a line item in a quote
type QuoteItem struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	QuoteID     uuid.UUID       `json:"quote_id" gorm:"type:uuid;not null;index"`
	Description string          `json:"description" gorm:"type:text;not null"`
	Quantity    decimal.Decimal `json:"quantity" gorm:"type:decimal(10,2);not null;default:1.00"`
	UnitPrice   decimal.Decimal `json:"unit_price" gorm:"type:decimal(12,2);not null;default:0.00"`
	Total       decimal.Decimal `json:"total" gorm:"type:decimal(12,2);not null;default:0.00"`
	SortOrder   int             `json:"sort_order" gorm:"not null;default:0;index:idx_quote_items_sort"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`
}

// TableName specifies the table name for Quote model
func (Quote) TableName() string {
	return "quotes"
}

// TableName specifies the table name for QuoteItem model
func (QuoteItem) TableName() string {
	return "quote_items"
}

// BeforeCreate hook for Quote
func (q *Quote) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}

	// Allocate the tenant's next quote number if not set (NextDocumentNumber).
	if q.QuoteNumber == "" {
		n, err := NextDocumentNumber(tx, q.TenantID, CounterQuote)
		if err != nil {
			return err
		}
		q.QuoteNumber = n
	}
	if q.Currency == "" {
		q.Currency = documentCurrency(tx, q.TenantID)
	}
	if q.ApplyDefaultTaxRate {
		if err := q.applyDefaultTaxRate(tx); err != nil {
			return err
		}
	}

	// Validate quote
	if err := q.Validate(); err != nil {
		return err
	}

	return nil
}

// applyDefaultTaxRate sets the tenant's default tax rate (if any) and
// recomputes the totals; without one the quote stays untaxed.
func (q *Quote) applyDefaultTaxRate(tx *gorm.DB) error {
	tenantID := documentTenantID(tx, q.TenantID)
	if tenantID == uuid.Nil {
		return nil
	}
	// A fresh session (it keeps the context, so tenant scoping still applies).
	db := tx.Session(&gorm.Session{NewDB: true})
	rate, err := DefaultTaxRate(db, tenantID, TenantSettingsInTx(db, tenantID))
	if err != nil {
		return err
	}
	if rate != nil {
		q.TaxRate = rate.Rate
		q.CalculateTotals()
	}
	return nil
}

// BeforeUpdate hook for Quote
func (q *Quote) BeforeUpdate(tx *gorm.DB) error {
	return q.Validate()
}

// BeforeCreate hook for QuoteItem
func (qi *QuoteItem) BeforeCreate(tx *gorm.DB) error {
	if qi.ID == uuid.Nil {
		qi.ID = uuid.New()
	}

	return qi.Validate()
}

// BeforeUpdate hook for QuoteItem
func (qi *QuoteItem) BeforeUpdate(tx *gorm.DB) error {
	return qi.Validate()
}

// Validate validates the Quote model
func (q *Quote) Validate() error {
	// Validate customer ID
	if q.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}

	// Validate quote number
	if q.QuoteNumber == "" {
		return errors.New("quote_number is required")
	}

	// Validate status
	if !q.IsValidStatus() {
		return fmt.Errorf("invalid status: %s", q.Status)
	}

	// Validate valid_until is in the future for new quotes
	if q.ValidUntil.Before(time.Now()) && q.Status == QuoteStatusDraft {
		return errors.New("valid_until must be in the future for new quotes")
	}

	// Validate numeric fields are non-negative
	if q.Subtotal.LessThan(decimal.Zero) {
		return errors.New("subtotal must be non-negative")
	}

	if q.TaxRate.LessThan(decimal.Zero) || q.TaxRate.GreaterThan(decimal.NewFromInt(1)) {
		return errors.New("tax_rate must be between 0 and 1")
	}

	if q.TaxAmount.LessThan(decimal.Zero) {
		return errors.New("tax_amount must be non-negative")
	}

	if q.Total.LessThan(decimal.Zero) {
		return errors.New("total must be non-negative")
	}

	// Validate created_by
	if q.CreatedBy == uuid.Nil {
		return errors.New("created_by is required")
	}

	return nil
}

// Validate validates the QuoteItem model
func (qi *QuoteItem) Validate() error {
	// Validate quote ID
	if qi.QuoteID == uuid.Nil {
		return errors.New("quote_id is required")
	}

	// Validate description
	if qi.Description == "" {
		return errors.New("description is required")
	}

	// Validate quantity is positive
	if qi.Quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be greater than 0")
	}

	// Validate unit price is non-negative
	if qi.UnitPrice.LessThan(decimal.Zero) {
		return errors.New("unit_price must be non-negative")
	}

	// Validate total is non-negative
	if qi.Total.LessThan(decimal.Zero) {
		return errors.New("total must be non-negative")
	}

	return nil
}

// IsValidStatus checks if the status is valid
func (q *Quote) IsValidStatus() bool {
	validStatuses := []QuoteStatus{
		QuoteStatusDraft,
		QuoteStatusSent,
		QuoteStatusViewed,
		QuoteStatusAccepted,
		QuoteStatusDeclined,
		QuoteStatusExpired,
	}
	for _, status := range validStatuses {
		if q.Status == status {
			return true
		}
	}
	return false
}

// CanEdit checks if the quote can be edited. Only drafts: a sent quote is
// what the customer saw, so changing it needs an explicit revise (back to
// draft) and a re-send (audit 7.M4).
func (q *Quote) CanEdit() bool {
	return q.Status == QuoteStatusDraft
}

// CanRevise reports whether a quote can be taken back to draft for changes:
// sent, viewed, expired or declined quotes (not accepted ones).
func (q *Quote) CanRevise() bool {
	switch q.Status {
	case QuoteStatusSent, QuoteStatusViewed, QuoteStatusExpired, QuoteStatusDeclined:
		return true
	}
	return false
}

// CanDelete checks if the quote can be deleted
func (q *Quote) CanDelete() bool {
	return q.Status == QuoteStatusDraft
}

// CanSend checks if the quote can be sent
func (q *Quote) CanSend() bool {
	return q.Status == QuoteStatusDraft && !q.IsExpired()
}

// CanAccept checks if the quote can be accepted
func (q *Quote) CanAccept() bool {
	return q.Status == QuoteStatusSent && !q.IsExpired()
}

// CanReject checks if the quote can be rejected
func (q *Quote) CanReject() bool {
	return q.Status == QuoteStatusSent && !q.IsExpired()
}

// IsExpired checks if the quote has expired
func (q *Quote) IsExpired() bool {
	return time.Now().After(q.ValidUntil)
}

// IsFinal checks if the quote is in a final state
func (q *Quote) IsFinal() bool {
	return q.Status == QuoteStatusAccepted || q.Status == QuoteStatusDeclined
}

// CalculateTotals calculates and updates the quote totals
func (q *Quote) CalculateTotals() {
	// Subtotal is the sum of the ROUNDED item totals (what the items show), so
	// the quote's lines always add up to its subtotal (audit 7.L1).
	subtotal := decimal.Zero
	for _, item := range q.Items {
		itemTotal := item.Quantity.Mul(item.UnitPrice).Round(2)
		subtotal = subtotal.Add(itemTotal)
	}

	q.Subtotal = subtotal
	q.TaxAmount = subtotal.Mul(q.TaxRate).Round(2)
	q.Total = q.Subtotal.Add(q.TaxAmount)
}

// CalculateItemTotal calculates the total for a quote item
func (qi *QuoteItem) CalculateItemTotal() {
	qi.Total = qi.Quantity.Mul(qi.UnitPrice).Round(2)
}

// QuoteRequest represents a request to create or update a quote
type QuoteRequest struct {
	CustomerID uuid.UUID `json:"customer_id" binding:"required"`
	ValidUntil time.Time `json:"valid_until" binding:"required"`
	// TaxRate is a fraction (0.0825 = 8.25%). Omitted on create, the
	// tenant's default tax rate applies (T127); omitted on update, the
	// quote keeps its current rate.
	TaxRate *decimal.Decimal   `json:"tax_rate,omitempty"`
	Notes   *string            `json:"notes"`
	Terms   *string            `json:"terms"`
	Items   []QuoteItemRequest `json:"items" binding:"required,min=1,dive"`
}

// QuoteItemRequest represents a request to create or update a quote item
type QuoteItemRequest struct {
	Description string          `json:"description" binding:"required"`
	Quantity    decimal.Decimal `json:"quantity" binding:"required"`
	UnitPrice   decimal.Decimal `json:"unit_price" binding:"required"`
	SortOrder   int             `json:"sort_order"`
}

// QuoteResponse represents a quote response
type QuoteResponse struct {
	ID          uuid.UUID           `json:"id"`
	CustomerID  uuid.UUID           `json:"customer_id"`
	Customer    *UserResponse       `json:"customer,omitempty"`
	QuoteNumber string              `json:"quote_number"`
	Status      QuoteStatus         `json:"status"`
	ValidUntil  time.Time           `json:"valid_until"`
	IsExpired   bool                `json:"is_expired"`
	Subtotal    decimal.Decimal     `json:"subtotal"`
	TaxRate     decimal.Decimal     `json:"tax_rate"`
	TaxAmount   decimal.Decimal     `json:"tax_amount"`
	Total       decimal.Decimal     `json:"total"`
	Currency    string              `json:"currency"`
	Notes       *string             `json:"notes,omitempty"`
	Terms       *string             `json:"terms,omitempty"`
	Items       []QuoteItemResponse `json:"items"`
	CreatedBy   uuid.UUID           `json:"created_by"`
	UpdatedBy   *uuid.UUID          `json:"updated_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// QuoteItemResponse represents a quote item response
type QuoteItemResponse struct {
	ID          uuid.UUID       `json:"id"`
	QuoteID     uuid.UUID       `json:"quote_id"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Total       decimal.Decimal `json:"total"`
	SortOrder   int             `json:"sort_order"`
}

// UserResponse is a simplified user response (to avoid circular dependency)
type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name,omitempty"`
}

// ToResponse converts Quote to QuoteResponse
func (q *Quote) ToResponse() QuoteResponse {
	resp := QuoteResponse{
		ID:          q.ID,
		CustomerID:  q.CustomerID,
		QuoteNumber: q.QuoteNumber,
		Status:      q.Status,
		ValidUntil:  q.ValidUntil,
		IsExpired:   q.IsExpired(),
		Subtotal:    q.Subtotal,
		TaxRate:     q.TaxRate,
		TaxAmount:   q.TaxAmount,
		Total:       q.Total,
		Currency:    q.Currency,
		Notes:       q.Notes,
		Terms:       q.Terms,
		CreatedBy:   q.CreatedBy,
		UpdatedBy:   q.UpdatedBy,
		CreatedAt:   q.CreatedAt,
		UpdatedAt:   q.UpdatedAt,
	}

	// Add customer info if loaded
	if q.Customer != nil {
		resp.Customer = &UserResponse{
			ID:    q.Customer.ID,
			Email: q.Customer.Email,
		}
	}

	// Add items
	resp.Items = make([]QuoteItemResponse, len(q.Items))
	for i, item := range q.Items {
		resp.Items[i] = QuoteItemResponse{
			ID:          item.ID,
			QuoteID:     item.QuoteID,
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Total:       item.Total,
			SortOrder:   item.SortOrder,
		}
	}

	return resp
}

// QuoteListResponse represents a paginated list of quotes
type QuoteListResponse struct {
	Quotes   []QuoteResponse `json:"quotes"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// QuoteQueryParams represents query parameters for listing quotes
type QuoteQueryParams struct {
	CustomerID *uuid.UUID   `form:"customer_id"`
	Status     *QuoteStatus `form:"status"`
	// Q matches quote number and customer first/last/full/company name
	// (case-insensitive substring).
	Q string `form:"q" binding:"max=100"`
	// SortBy must be one of utils.QuoteAllowedSortColumns (kept in sync with
	// this oneof); empty means created_at.
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=created_at updated_at quote_number status total valid_until"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc ASC DESC"`
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
}
