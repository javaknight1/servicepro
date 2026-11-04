// Package quickbooks is a small client for the QuickBooks Online
// Accounting API (Intuit, T192): OAuth 2.0 authorization-code flow, token
// refresh and revoke, and the customer / invoice / payment / item / account /
// tax code calls ServicePro's one-way sync needs.
//
// Client is the interface the sync uses; HTTPClient talks to Intuit and Fake
// is an in-memory QuickBooks for tests and local runs (QUICKBOOKS_FAKE).
//
// API facts this package relies on (developer.intuit.com):
//   - Authorize: https://appcenter.intuit.com/connect/oauth2, scope
//     com.intuit.quickbooks.accounting; the redirect carries code, state and
//     realmId (the company ID).
//   - Tokens: POST https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer
//     (HTTP Basic client_id:client_secret). Access tokens live 1 hour;
//     refresh tokens ~100 days and roll: every refresh may return a new one,
//     which must replace the old one.
//   - Revoke: POST https://developer.api.intuit.com/v2/oauth2/tokens/revoke.
//   - Data: {base}/v3/company/{realmId}/{entity}?minorversion=75; minor
//     versions below 75 are ignored since 2025-08-01. requestid=<unique id>
//     makes a write idempotent: Intuit answers a repeated requestid with
//     the original result instead of writing twice.
//   - Updates need the entity's current SyncToken (optimistic locking;
//     a stale one fails with error 5010). sparse=true updates only the
//     fields sent.
//   - Throttling: 500 requests/minute and 10 concurrent requests per
//     company; over the limit answers HTTP 429.
package quickbooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// MinorVersion is the API minor version sent on every data call.
const MinorVersion = "75"

// Scope is the OAuth scope for the Accounting API.
const Scope = "com.intuit.quickbooks.accounting"

// Tokens is the result of a code exchange or refresh.
type Tokens struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// Session addresses one company with a valid access token.
type Session struct {
	RealmID     string
	AccessToken string
}

// Ref is a QuickBooks reference ({"value": id, "name": name}).
type Ref struct {
	Value string `json:"value"`
	Name  string `json:"name,omitempty"`
}

// Entity is the identity of a QuickBooks record after a write or lookup.
type Entity struct {
	ID        string
	SyncToken string
	Name      string
}

// EmailAddr is a QuickBooks email address.
type EmailAddr struct {
	Address string `json:"Address"`
}

// Phone is a QuickBooks phone number.
type Phone struct {
	FreeFormNumber string `json:"FreeFormNumber"`
}

// Address is a QuickBooks physical address.
type Address struct {
	Line1                  string `json:"Line1,omitempty"`
	City                   string `json:"City,omitempty"`
	CountrySubDivisionCode string `json:"CountrySubDivisionCode,omitempty"`
	PostalCode             string `json:"PostalCode,omitempty"`
}

// Customer is the subset of the QuickBooks Customer the sync writes.
// DisplayName must be unique across customers, vendors and employees.
type Customer struct {
	DisplayName      string     `json:"DisplayName"`
	GivenName        string     `json:"GivenName,omitempty"`
	FamilyName       string     `json:"FamilyName,omitempty"`
	CompanyName      string     `json:"CompanyName,omitempty"`
	PrimaryEmailAddr *EmailAddr `json:"PrimaryEmailAddr,omitempty"`
	PrimaryPhone     *Phone     `json:"PrimaryPhone,omitempty"`
	BillAddr         *Address   `json:"BillAddr,omitempty"`
}

// SalesItemLineDetail is an invoice line's item, quantity and tax code.
type SalesItemLineDetail struct {
	ItemRef    Ref          `json:"ItemRef"`
	Qty        *json.Number `json:"Qty,omitempty"`
	UnitPrice  *json.Number `json:"UnitPrice,omitempty"`
	TaxCodeRef *Ref         `json:"TaxCodeRef,omitempty"`
}

// DiscountLineDetail is an invoice-level discount.
type DiscountLineDetail struct {
	PercentBased bool `json:"PercentBased"`
}

// InvoiceLine is one invoice line (SalesItemLineDetail or
// DiscountLineDetail).
type InvoiceLine struct {
	DetailType          string               `json:"DetailType"`
	Amount              json.Number          `json:"Amount"`
	Description         string               `json:"Description,omitempty"`
	SalesItemLineDetail *SalesItemLineDetail `json:"SalesItemLineDetail,omitempty"`
	DiscountLineDetail  *DiscountLineDetail  `json:"DiscountLineDetail,omitempty"`
}

// TxnTaxDetail carries the invoice's total tax (and optionally the tax code
// for companies without automated sales tax).
type TxnTaxDetail struct {
	TxnTaxCodeRef *Ref        `json:"TxnTaxCodeRef,omitempty"`
	TotalTax      json.Number `json:"TotalTax"`
}

// Invoice is the subset of the QuickBooks Invoice the sync writes.
type Invoice struct {
	DocNumber    string        `json:"DocNumber,omitempty"`
	TxnDate      string        `json:"TxnDate,omitempty"`
	DueDate      string        `json:"DueDate,omitempty"`
	CustomerRef  Ref           `json:"CustomerRef"`
	Line         []InvoiceLine `json:"Line"`
	TxnTaxDetail *TxnTaxDetail `json:"TxnTaxDetail,omitempty"`
	PrivateNote  string        `json:"PrivateNote,omitempty"`
}

// LinkedTxn links a payment line to the invoice it pays.
type LinkedTxn struct {
	TxnID   string `json:"TxnId"`
	TxnType string `json:"TxnType"`
}

// PaymentLine applies part of a payment to one invoice.
type PaymentLine struct {
	Amount    json.Number `json:"Amount"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn"`
}

// Payment is the subset of the QuickBooks (ReceivePayment) Payment the sync
// writes.
type Payment struct {
	CustomerRef         Ref           `json:"CustomerRef"`
	TotalAmt            json.Number   `json:"TotalAmt"`
	TxnDate             string        `json:"TxnDate,omitempty"`
	PaymentRefNum       string        `json:"PaymentRefNum,omitempty"`
	DepositToAccountRef *Ref          `json:"DepositToAccountRef,omitempty"`
	PrivateNote         string        `json:"PrivateNote,omitempty"`
	Line                []PaymentLine `json:"Line"`
}

// Item is a QuickBooks product/service (type Service) the invoice lines use.
type Item struct {
	Name             string `json:"Name"`
	Type             string `json:"Type"`
	IncomeAccountRef Ref    `json:"IncomeAccountRef"`
}

// Account types the mapping settings list.
const (
	AccountTypeIncome             = "Income"
	AccountTypeBank               = "Bank"
	AccountTypeOtherCurrentAsset  = "Other Current Asset"
	EntityKindCustomer            = "customer"
	EntityKindInvoice             = "invoice"
	EntityKindPayment             = "payment"
	EntityKindItem                = "item"
	errorCodeStaleObject          = "5010"
	errorCodeDuplicateName        = "6240"
	errorCodeInvalidGrantFragment = "invalid_grant"
)

// ErrInvalidGrant means the refresh token (or authorization code) is no
// longer valid: the tenant must reconnect.
var ErrInvalidGrant = errors.New("quickbooks: authorization expired or revoked")

// APIError is an error answer from QuickBooks (a Fault, or an HTTP error).
type APIError struct {
	Status  int
	Code    string
	Message string
	Detail  string
	// IntuitTID is Intuit's transaction ID for support requests.
	IntuitTID string
}

func (e *APIError) Error() string {
	msg := e.Message
	if e.Detail != "" && e.Detail != e.Message {
		msg += ": " + e.Detail
	}
	if e.Code != "" {
		return fmt.Sprintf("QuickBooks error %s (HTTP %d): %s", e.Code, e.Status, msg)
	}
	return fmt.Sprintf("QuickBooks HTTP %d: %s", e.Status, msg)
}

func asAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := errors.As(err, &apiErr)
	return apiErr, ok
}

// IsUnauthorized reports a 401 (expired or revoked access token).
func IsUnauthorized(err error) bool {
	e, ok := asAPIError(err)
	return ok && e.Status == 401
}

// IsRateLimited reports a 429 (throttled).
func IsRateLimited(err error) bool {
	e, ok := asAPIError(err)
	return ok && e.Status == 429
}

// IsStale reports a stale SyncToken (the record changed in QuickBooks).
func IsStale(err error) bool {
	e, ok := asAPIError(err)
	return ok && e.Code == errorCodeStaleObject
}

// IsDuplicateName reports a DisplayName already used by another customer,
// vendor or employee.
func IsDuplicateName(err error) bool {
	e, ok := asAPIError(err)
	return ok && e.Code == errorCodeDuplicateName
}

// IsTransient reports errors worth retrying later without counting against
// the record (throttling, Intuit outages, network failures).
func IsTransient(err error) bool {
	e, ok := asAPIError(err)
	if !ok {
		return !errors.Is(err, ErrInvalidGrant)
	}
	return e.Status == 429 || e.Status >= 500
}
