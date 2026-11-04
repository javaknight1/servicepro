package quickbooks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Fake is an in-memory QuickBooks Online company store for tests and local
// runs (QUICKBOOKS_FAKE=true in development). It honours requestid (a
// repeated requestid returns the first result without writing again),
// SyncToken (a stale token fails with error 5010), unique customer
// DisplayName (6240) and token expiry/rotation.
type Fake struct {
	mu sync.Mutex
	// RedirectURI is where AuthorizeURL sends the browser straight back
	// (no Intuit consent page).
	RedirectURI string
	// RealmID is the company ID the fake consent returns.
	RealmID string
	// AccessTTL is the lifetime of issued access tokens (default 1 hour).
	AccessTTL time.Duration
	Now       func() time.Time

	codes    map[string]bool
	access   map[string]time.Time
	refresh  map[string]bool
	revoked  []string
	requests map[string]*Entity
	seq      int

	Customers map[string]*FakeRecord
	Invoices  map[string]*FakeRecord
	Payments  map[string]*FakeRecord
	Items     map[string]*FakeRecord
	Accounts  []FakeAccount
	TaxCodes  []Ref

	// Calls counts calls per operation ("CreateInvoice", "RefreshTokens", ...).
	Calls map[string]int
	// failures are errors to return from the next calls of an operation.
	failures map[string][]error
	// lose counts calls of an operation that succeed in QuickBooks but whose
	// answer is lost (the caller sees a 503).
	lose map[string]int
	// Hook runs (unlocked) at the start of every call, e.g. to slow a
	// refresh down in a concurrency test.
	Hook func(op string)
}

// FakeRecord is a stored entity with its last written payload.
type FakeRecord struct {
	ID        string
	SyncToken int
	Name      string
	Payload   any
	RequestID string
}

// FakeAccount is a chart-of-accounts entry.
type FakeAccount struct {
	Ref
	Type string
}

// NewFake returns a fake company with a few accounts, tax codes and the
// default "Services" item.
func NewFake(redirectURI string) *Fake {
	f := &Fake{
		RedirectURI: redirectURI,
		RealmID:     "9130350000000000",
		AccessTTL:   time.Hour,
		Now:         time.Now,
		codes:       map[string]bool{},
		access:      map[string]time.Time{},
		refresh:     map[string]bool{},
		requests:    map[string]*Entity{},
		Customers:   map[string]*FakeRecord{},
		Invoices:    map[string]*FakeRecord{},
		Payments:    map[string]*FakeRecord{},
		Items:       map[string]*FakeRecord{},
		Calls:       map[string]int{},
		failures:    map[string][]error{},
		lose:        map[string]int{},
		Accounts: []FakeAccount{
			{Ref{Value: "79", Name: "Cleaning Services"}, AccountTypeIncome},
			{Ref{Value: "80", Name: "Maintenance Income"}, AccountTypeIncome},
			{Ref{Value: "35", Name: "Checking"}, AccountTypeBank},
			{Ref{Value: "4", Name: "Undeposited Funds"}, AccountTypeOtherCurrentAsset},
		},
		TaxCodes: []Ref{{Value: "TAX", Name: "TAX"}, {Value: "NON", Name: "NON"}, {Value: "3", Name: "OH 8.0%"}},
	}
	f.Items["1"] = &FakeRecord{ID: "1", Name: "Services"}
	f.seq = 100
	return f
}

// FailNext makes the next len(errs) calls of op return errs in order.
func (f *Fake) FailNext(op string, errs ...error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures[op] = append(f.failures[op], errs...)
}

// LoseNextResponse makes the next call of op succeed in QuickBooks but
// answer 503, as when the response is lost on the way back.
func (f *Fake) LoseNextResponse(op string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lose[op]++
}

// Revoked lists revoked tokens.
func (f *Fake) Revoked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.revoked...)
}

// CallCount returns how many times op was called.
func (f *Fake) CallCount(op string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Calls[op]
}

// ExpireAccessTokens makes every issued access token invalid (the next
// data call answers 401).
func (f *Fake) ExpireAccessTokens() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.access = map[string]time.Time{}
}

// begin records a call and returns a queued failure, if any. It returns
// with f.mu held.
func (f *Fake) begin(op string) error {
	if f.Hook != nil {
		f.Hook(op)
	}
	f.mu.Lock()
	f.Calls[op]++
	if q := f.failures[op]; len(q) > 0 {
		f.failures[op] = q[1:]
		return q[0]
	}
	return nil
}

func randomToken(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func (f *Fake) nextID() string {
	f.seq++
	return strconv.Itoa(f.seq)
}

// IssueCode returns a valid authorization code (as the consent page would).
func (f *Fake) IssueCode() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	code := randomToken("code-")
	f.codes[code] = true
	return code
}

// AuthorizeURL implements Client: it skips the consent page and redirects
// straight to the callback with a fresh code.
func (f *Fake) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("code", f.IssueCode())
	q.Set("state", state)
	q.Set("realmId", f.RealmID)
	return f.RedirectURI + "?" + q.Encode()
}

func (f *Fake) issueTokens() *Tokens {
	now := f.Now()
	t := &Tokens{
		AccessToken:      randomToken("access-"),
		RefreshToken:     randomToken("refresh-"),
		AccessExpiresAt:  now.Add(f.AccessTTL),
		RefreshExpiresAt: now.Add(100 * 24 * time.Hour),
	}
	f.access[t.AccessToken] = t.AccessExpiresAt
	f.refresh[t.RefreshToken] = true
	return t
}

// ExchangeCode implements Client.
func (f *Fake) ExchangeCode(_ context.Context, code string) (*Tokens, error) {
	if err := f.begin("ExchangeCode"); err != nil {
		f.mu.Unlock()
		return nil, err
	}
	defer f.mu.Unlock()
	if !f.codes[code] {
		return nil, ErrInvalidGrant
	}
	delete(f.codes, code)
	return f.issueTokens(), nil
}

// RefreshTokens implements Client. Refresh tokens rotate: the old one stops
// working.
func (f *Fake) RefreshTokens(_ context.Context, refreshToken string) (*Tokens, error) {
	if err := f.begin("RefreshTokens"); err != nil {
		f.mu.Unlock()
		return nil, err
	}
	defer f.mu.Unlock()
	if !f.refresh[refreshToken] {
		return nil, ErrInvalidGrant
	}
	delete(f.refresh, refreshToken)
	return f.issueTokens(), nil
}

// Revoke implements Client.
func (f *Fake) Revoke(_ context.Context, token string) error {
	if err := f.begin("Revoke"); err != nil {
		f.mu.Unlock()
		return err
	}
	defer f.mu.Unlock()
	f.revoked = append(f.revoked, token)
	delete(f.refresh, token)
	delete(f.access, token)
	return nil
}

// authorize checks the session's access token. Call with f.mu held.
func (f *Fake) authorize(s Session) error {
	exp, ok := f.access[s.AccessToken]
	if !ok || !f.Now().Before(exp) {
		return &APIError{Status: 401, Code: "3200", Message: "AuthenticationFailed"}
	}
	return nil
}

// call wraps a data operation: queued failure, auth, then fn under the lock.
func (f *Fake) call(op string, s Session, fn func() (*Entity, error)) (*Entity, error) {
	if err := f.begin(op); err != nil {
		f.mu.Unlock()
		return nil, err
	}
	defer f.mu.Unlock()
	if err := f.authorize(s); err != nil {
		return nil, err
	}
	e, err := fn()
	if err == nil && f.lose[op] > 0 {
		f.lose[op]--
		return nil, &APIError{Status: 503, Message: "Service Unavailable"}
	}
	return e, err
}

// CompanyName implements Client.
func (f *Fake) CompanyName(_ context.Context, s Session) (string, error) {
	_, err := f.call("CompanyName", s, func() (*Entity, error) { return nil, nil })
	if err != nil {
		return "", err
	}
	return "Sandbox Company_US_1", nil
}

// ListAccounts implements Client.
func (f *Fake) ListAccounts(_ context.Context, s Session, accountType string) ([]Ref, error) {
	var refs []Ref
	_, err := f.call("ListAccounts", s, func() (*Entity, error) {
		for _, a := range f.Accounts {
			if a.Type == accountType {
				refs = append(refs, a.Ref)
			}
		}
		return nil, nil
	})
	return refs, err
}

// ListTaxCodes implements Client.
func (f *Fake) ListTaxCodes(_ context.Context, s Session) ([]Ref, error) {
	var refs []Ref
	_, err := f.call("ListTaxCodes", s, func() (*Entity, error) {
		refs = append(refs, f.TaxCodes...)
		return nil, nil
	})
	return refs, err
}

func findByName(m map[string]*FakeRecord, name string) *Entity {
	for _, r := range m {
		if r.Name == name {
			return &Entity{ID: r.ID, SyncToken: strconv.Itoa(r.SyncToken), Name: r.Name}
		}
	}
	return nil
}

// FindItemByName implements Client.
func (f *Fake) FindItemByName(_ context.Context, s Session, name string) (*Entity, error) {
	return f.call("FindItemByName", s, func() (*Entity, error) { return findByName(f.Items, name), nil })
}

// create stores a new record unless requestID was seen (idempotent replay).
func (f *Fake) create(m map[string]*FakeRecord, name string, payload any, requestID string) *Entity {
	if requestID != "" {
		if e, ok := f.requests[requestID]; ok {
			return e
		}
	}
	id := f.nextID()
	m[id] = &FakeRecord{ID: id, Name: name, Payload: payload, RequestID: requestID}
	e := &Entity{ID: id, SyncToken: "0", Name: name}
	if requestID != "" {
		f.requests[requestID] = e
	}
	return e
}

// update applies a sparse update when the SyncToken is current.
func (f *Fake) update(m map[string]*FakeRecord, e Entity, name string, payload any, requestID string) (*Entity, error) {
	if requestID != "" {
		if prev, ok := f.requests[requestID]; ok {
			return prev, nil
		}
	}
	r, ok := m[e.ID]
	if !ok {
		return nil, &APIError{Status: 400, Code: "610", Message: "Object Not Found"}
	}
	if e.SyncToken != strconv.Itoa(r.SyncToken) {
		return nil, &APIError{Status: 400, Code: errorCodeStaleObject, Message: "Stale Object Error"}
	}
	r.SyncToken++
	r.Payload = payload
	if name != "" {
		r.Name = name
	}
	out := &Entity{ID: r.ID, SyncToken: strconv.Itoa(r.SyncToken), Name: r.Name}
	if requestID != "" {
		f.requests[requestID] = out
	}
	return out, nil
}

// CreateItem implements Client.
func (f *Fake) CreateItem(_ context.Context, s Session, item Item, requestID string) (*Entity, error) {
	return f.call("CreateItem", s, func() (*Entity, error) { return f.create(f.Items, item.Name, item, requestID), nil })
}

// FindCustomerByDisplayName implements Client.
func (f *Fake) FindCustomerByDisplayName(_ context.Context, s Session, name string) (*Entity, error) {
	return f.call("FindCustomerByDisplayName", s, func() (*Entity, error) { return findByName(f.Customers, name), nil })
}

// FindInvoiceByDocNumber implements Client.
func (f *Fake) FindInvoiceByDocNumber(_ context.Context, s Session, docNumber string) (*Entity, error) {
	return f.call("FindInvoiceByDocNumber", s, func() (*Entity, error) { return findByName(f.Invoices, docNumber), nil })
}

// Get implements Client.
func (f *Fake) Get(_ context.Context, s Session, kind, id string) (*Entity, error) {
	return f.call("Get", s, func() (*Entity, error) {
		m := map[string]map[string]*FakeRecord{
			EntityKindCustomer: f.Customers, EntityKindInvoice: f.Invoices,
			EntityKindPayment: f.Payments, EntityKindItem: f.Items,
		}[kind]
		r, ok := m[id]
		if !ok {
			return nil, &APIError{Status: 400, Code: "610", Message: "Object Not Found"}
		}
		return &Entity{ID: r.ID, SyncToken: strconv.Itoa(r.SyncToken), Name: r.Name}, nil
	})
}

// CreateCustomer implements Client.
func (f *Fake) CreateCustomer(_ context.Context, s Session, c Customer, requestID string) (*Entity, error) {
	return f.call("CreateCustomer", s, func() (*Entity, error) {
		if _, seen := f.requests[requestID]; !seen && findByName(f.Customers, c.DisplayName) != nil {
			return nil, &APIError{Status: 400, Code: errorCodeDuplicateName, Message: "Duplicate Name Exists Error"}
		}
		return f.create(f.Customers, c.DisplayName, c, requestID), nil
	})
}

// UpdateCustomer implements Client.
func (f *Fake) UpdateCustomer(_ context.Context, s Session, e Entity, c Customer, requestID string) (*Entity, error) {
	return f.call("UpdateCustomer", s, func() (*Entity, error) {
		return f.update(f.Customers, e, c.DisplayName, c, requestID)
	})
}

// CreateInvoice implements Client.
func (f *Fake) CreateInvoice(_ context.Context, s Session, inv Invoice, requestID string) (*Entity, error) {
	return f.call("CreateInvoice", s, func() (*Entity, error) {
		return f.create(f.Invoices, inv.DocNumber, inv, requestID), nil
	})
}

// UpdateInvoice implements Client.
func (f *Fake) UpdateInvoice(_ context.Context, s Session, e Entity, inv Invoice, requestID string) (*Entity, error) {
	return f.call("UpdateInvoice", s, func() (*Entity, error) {
		return f.update(f.Invoices, e, inv.DocNumber, inv, requestID)
	})
}

// CreatePayment implements Client.
func (f *Fake) CreatePayment(_ context.Context, s Session, p Payment, requestID string) (*Entity, error) {
	return f.call("CreatePayment", s, func() (*Entity, error) {
		return f.create(f.Payments, p.PaymentRefNum, p, requestID), nil
	})
}

// Bump simulates an accountant editing a record in QuickBooks (its
// SyncToken moves on).
func (f *Fake) Bump(kind, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := map[string]map[string]*FakeRecord{EntityKindCustomer: f.Customers, EntityKindInvoice: f.Invoices}[kind]
	if r, ok := m[id]; ok {
		r.SyncToken++
	}
}

// Snapshot returns copies of the stored records of a kind.
func (f *Fake) Snapshot(kind string) []FakeRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := map[string]map[string]*FakeRecord{
		EntityKindCustomer: f.Customers, EntityKindInvoice: f.Invoices,
		EntityKindPayment: f.Payments, EntityKindItem: f.Items,
	}[kind]
	out := make([]FakeRecord, 0, len(m))
	for _, r := range m {
		out = append(out, *r)
	}
	return out
}

var _ Client = (*Fake)(nil)
var _ Client = (*HTTPClient)(nil)
