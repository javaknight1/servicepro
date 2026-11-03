package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerPurchaseOrder is a purchase order a customer issued (T187): the
// amount it authorises and an optional expiry. Invoices reference it by
// number (invoices.po_number, same customer); what they bill counts against
// the amount. Tenant-owned; one row per (customer, po_number).
type CustomerPurchaseOrder struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID   uuid.UUID       `json:"-" gorm:"type:uuid;not null"`
	CustomerID uuid.UUID       `json:"customer_id" gorm:"type:uuid;not null"`
	PONumber   string          `json:"po_number" gorm:"column:po_number;type:varchar(100);not null"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	// ExpiresOn is a calendar date (midnight UTC; DATE column), inclusive.
	ExpiresOn *time.Time `json:"-" gorm:"type:date"`
	Notes     *string    `json:"notes" gorm:"type:text"`
	CreatedBy *uuid.UUID `json:"-" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName specifies the table name for CustomerPurchaseOrder.
func (CustomerPurchaseOrder) TableName() string { return "customer_purchase_orders" }

// CreatePurchaseOrderRequest is POST /customers/{id}/purchase-orders.
type CreatePurchaseOrderRequest struct {
	PONumber string `json:"po_number" binding:"required,max=100" example:"PO-7731-HV"`
	// Amount is the total the PO authorises (more than zero).
	Amount decimal.Decimal `json:"amount" swaggertype:"string" example:"12000.00"`
	// ExpiresOn is YYYY-MM-DD (inclusive); omit for no expiry.
	ExpiresOn *string `json:"expires_on" example:"2027-06-30"`
	Notes     *string `json:"notes" binding:"omitempty,max=2000"`
}

// UpdatePurchaseOrderRequest is PUT /customers/{id}/purchase-orders/{poId};
// omitted fields are left unchanged. Renaming the PO number re-points the
// balance at invoices carrying the new number.
type UpdatePurchaseOrderRequest struct {
	PONumber       *string          `json:"po_number" binding:"omitempty,max=100"`
	Amount         *decimal.Decimal `json:"amount" swaggertype:"string" example:"12000.00"`
	ExpiresOn      *string          `json:"expires_on" example:"2027-06-30"`
	ClearExpiresOn bool             `json:"clear_expires_on"`
	Notes          *string          `json:"notes" binding:"omitempty,max=2000"`
}

// PurchaseOrderResponse is a PO with its balance: Billed is the total of
// the customer's issued invoices (not draft, not cancelled) carrying the PO
// number, Remaining = Amount - Billed (negative when over).
type PurchaseOrderResponse struct {
	ID           uuid.UUID       `json:"id"`
	CustomerID   uuid.UUID       `json:"customer_id"`
	PONumber     string          `json:"po_number"`
	Amount       decimal.Decimal `json:"amount" swaggertype:"string" example:"12000.00"`
	Billed       decimal.Decimal `json:"billed" swaggertype:"string" example:"4800.00"`
	Remaining    decimal.Decimal `json:"remaining" swaggertype:"string" example:"7200.00"`
	InvoiceCount int             `json:"invoice_count"`
	ExpiresOn    *string         `json:"expires_on,omitempty" example:"2027-06-30"`
	// Expired: the expiry date has passed (tenant's time zone).
	Expired   bool      `json:"expired"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PurchaseOrderListResponse is GET /customers/{id}/purchase-orders.
type PurchaseOrderListResponse struct {
	PurchaseOrders []PurchaseOrderResponse `json:"purchase_orders"`
	Total          int                     `json:"total"`
}

// POBalance is what the customer's invoices have billed against one PO
// number.
type POBalance struct {
	Billed       decimal.Decimal
	InvoiceCount int
}

// ToResponse converts a PO and its balance to the API form; today is the
// tenant-local calendar date (midnight UTC).
func (p *CustomerPurchaseOrder) ToResponse(bal POBalance, today time.Time) PurchaseOrderResponse {
	r := PurchaseOrderResponse{
		ID: p.ID, CustomerID: p.CustomerID, PONumber: p.PONumber, Amount: p.Amount,
		Billed: bal.Billed.Round(2), Remaining: p.Amount.Sub(bal.Billed).Round(2), InvoiceCount: bal.InvoiceCount,
		Notes: p.Notes, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if p.ExpiresOn != nil {
		d := normalizeDate(*p.ExpiresOn)
		s := d.Format(DateLayout)
		r.ExpiresOn = &s
		r.Expired = d.Before(today)
	}
	return r
}
