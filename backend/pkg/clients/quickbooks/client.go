package quickbooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is everything the sync needs from QuickBooks Online.
type Client interface {
	// AuthorizeURL is the Intuit consent page for a connect request.
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*Tokens, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*Tokens, error)
	// Revoke revokes a refresh (or access) token and with it the
	// connection.
	Revoke(ctx context.Context, token string) error

	CompanyName(ctx context.Context, s Session) (string, error)
	ListAccounts(ctx context.Context, s Session, accountType string) ([]Ref, error)
	ListTaxCodes(ctx context.Context, s Session) ([]Ref, error)
	FindItemByName(ctx context.Context, s Session, name string) (*Entity, error)
	CreateItem(ctx context.Context, s Session, item Item, requestID string) (*Entity, error)

	FindCustomerByDisplayName(ctx context.Context, s Session, name string) (*Entity, error)
	FindInvoiceByDocNumber(ctx context.Context, s Session, docNumber string) (*Entity, error)
	// Get reads an entity's current SyncToken (kind: EntityKind*).
	Get(ctx context.Context, s Session, kind, id string) (*Entity, error)

	CreateCustomer(ctx context.Context, s Session, c Customer, requestID string) (*Entity, error)
	UpdateCustomer(ctx context.Context, s Session, e Entity, c Customer, requestID string) (*Entity, error)
	CreateInvoice(ctx context.Context, s Session, inv Invoice, requestID string) (*Entity, error)
	UpdateInvoice(ctx context.Context, s Session, e Entity, inv Invoice, requestID string) (*Entity, error)
	CreatePayment(ctx context.Context, s Session, p Payment, requestID string) (*Entity, error)
}

// Config configures HTTPClient.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	// Environment: "sandbox" or "production".
	Environment string
	HTTPClient  *http.Client
	// Endpoint overrides (tests).
	AuthorizeEndpoint string
	TokenEndpoint     string
	RevokeEndpoint    string
	APIBase           string
}

// Intuit endpoints.
const (
	authorizeEndpoint = "https://appcenter.intuit.com/connect/oauth2"
	tokenEndpoint     = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer" //nolint:gosec // G101: a public URL, not a credential
	revokeEndpoint    = "https://developer.api.intuit.com/v2/oauth2/tokens/revoke"
	sandboxAPIBase    = "https://sandbox-quickbooks.api.intuit.com"
	productionAPIBase = "https://quickbooks.api.intuit.com"
)

// HTTPClient talks to Intuit.
type HTTPClient struct {
	cfg  Config
	http *http.Client
	now  func() time.Time
}

// NewHTTPClient returns a client for Intuit's live endpoints.
func NewHTTPClient(cfg Config) *HTTPClient {
	if cfg.AuthorizeEndpoint == "" {
		cfg.AuthorizeEndpoint = authorizeEndpoint
	}
	if cfg.TokenEndpoint == "" {
		cfg.TokenEndpoint = tokenEndpoint
	}
	if cfg.RevokeEndpoint == "" {
		cfg.RevokeEndpoint = revokeEndpoint
	}
	if cfg.APIBase == "" {
		cfg.APIBase = sandboxAPIBase
		if cfg.Environment == "production" {
			cfg.APIBase = productionAPIBase
		}
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	return &HTTPClient{cfg: cfg, http: hc, now: time.Now}
}

// AuthorizeURL implements Client.
func (c *HTTPClient) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("scope", Scope)
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("state", state)
	return c.cfg.AuthorizeEndpoint + "?" + q.Encode()
}

type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"x_refresh_token_expires_in"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
}

// ExchangeCode implements Client.
func (c *HTTPClient) ExchangeCode(ctx context.Context, code string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	return c.tokenRequest(ctx, form)
}

// RefreshTokens implements Client.
func (c *HTTPClient) RefreshTokens(ctx context.Context, refreshToken string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return c.tokenRequest(ctx, form)
}

func (c *HTTPClient) tokenRequest(ctx context.Context, form url.Values) (*Tokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quickbooks token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tr tokenResponse
	_ = json.Unmarshal(body, &tr)
	if resp.StatusCode != http.StatusOK {
		if tr.Error == errorCodeInvalidGrantFragment {
			return nil, ErrInvalidGrant
		}
		// Never include the body verbatim: it may echo request values.
		return nil, &APIError{Status: resp.StatusCode, Code: tr.Error, Message: "token request failed", Detail: tr.ErrorDescription}
	}
	if tr.AccessToken == "" || tr.RefreshToken == "" {
		return nil, &APIError{Status: resp.StatusCode, Message: "token response without tokens"}
	}
	now := c.now()
	refreshIn := tr.RefreshTokenExpiresIn
	if refreshIn == 0 {
		refreshIn = 100 * 24 * 3600
	}
	expiresIn := tr.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}
	return &Tokens{
		AccessToken:      tr.AccessToken,
		RefreshToken:     tr.RefreshToken,
		AccessExpiresAt:  now.Add(time.Duration(expiresIn) * time.Second),
		RefreshExpiresAt: now.Add(time.Duration(refreshIn) * time.Second),
	}, nil
}

// Revoke implements Client.
func (c *HTTPClient) Revoke(ctx context.Context, token string) error {
	body, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.RevokeEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("quickbooks revoke: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Message: "revoke failed"}
	}
	return nil
}

type fault struct {
	Fault *struct {
		Error []struct {
			Message string `json:"Message"`
			Detail  string `json:"Detail"`
			Code    string `json:"code"`
		} `json:"Error"`
		Type string `json:"type"`
	} `json:"Fault"`
}

// do sends one data API request and decodes the answer into out.
func (c *HTTPClient) do(ctx context.Context, s Session, method, path string, query url.Values, in, out any) error {
	if query == nil {
		query = url.Values{}
	}
	query.Set("minorversion", MinorVersion)
	u := fmt.Sprintf("%s/v3/company/%s/%s?%s", c.cfg.APIBase, url.PathEscape(s.RealmID), path, query.Encode())
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("quickbooks %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("quickbooks %s %s: %w", method, path, err)
	}
	var f fault
	_ = json.Unmarshal(raw, &f)
	if resp.StatusCode >= 300 || (f.Fault != nil && len(f.Fault.Error) > 0) {
		apiErr := &APIError{Status: resp.StatusCode, Message: http.StatusText(resp.StatusCode), IntuitTID: resp.Header.Get("intuit_tid")}
		if f.Fault != nil && len(f.Fault.Error) > 0 {
			apiErr.Code = f.Fault.Error[0].Code
			apiErr.Message = f.Fault.Error[0].Message
			apiErr.Detail = f.Fault.Error[0].Detail
		}
		return apiErr
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("quickbooks %s %s: decode: %w", method, path, err)
	}
	return nil
}

// entityJSON is the identity part of any entity in a response.
type entityJSON struct {
	ID          string `json:"Id"`
	SyncToken   string `json:"SyncToken"`
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`
	DocNumber   string `json:"DocNumber"`
}

func (e entityJSON) entity() *Entity {
	name := e.Name
	if name == "" {
		name = e.DisplayName
	}
	return &Entity{ID: e.ID, SyncToken: e.SyncToken, Name: name}
}

// entityName is the JSON wrapper key of a kind.
var entityName = map[string]string{
	EntityKindCustomer: "Customer",
	EntityKindInvoice:  "Invoice",
	EntityKindPayment:  "Payment",
	EntityKindItem:     "Item",
}

// write POSTs an entity (create, or sparse update when e is set).
func (c *HTTPClient) write(ctx context.Context, s Session, kind string, e *Entity, payload any, requestID string) (*Entity, error) {
	body := map[string]any{}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if e != nil {
		body["Id"] = e.ID
		body["SyncToken"] = e.SyncToken
		body["sparse"] = true
	}
	q := url.Values{}
	if requestID != "" {
		q.Set("requestid", requestID)
	}
	out := map[string]json.RawMessage{}
	if err := c.do(ctx, s, http.MethodPost, kind, q, body, &out); err != nil {
		return nil, err
	}
	return entityFrom(out, kind)
}

// entityFrom decodes the entity of kind from a response wrapper
// ({"Invoice": {...}, "time": "..."}).
func entityFrom(out map[string]json.RawMessage, kind string) (*Entity, error) {
	var got entityJSON
	raw, ok := out[entityName[kind]]
	if ok {
		if err := json.Unmarshal(raw, &got); err != nil {
			return nil, fmt.Errorf("quickbooks %s: decode: %w", kind, err)
		}
	}
	if got.ID == "" {
		return nil, &APIError{Status: 200, Message: "response without " + entityName[kind]}
	}
	return got.entity(), nil
}

// queryEscape quotes a value for a QuickBooks query string literal.
func queryEscape(v string) string {
	return strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `'`, `\'`)
}

func (c *HTTPClient) query(ctx context.Context, s Session, q string) (map[string]json.RawMessage, error) {
	var out struct {
		QueryResponse map[string]json.RawMessage `json:"QueryResponse"`
	}
	if err := c.do(ctx, s, http.MethodGet, "query", url.Values{"query": {q}}, nil, &out); err != nil {
		return nil, err
	}
	return out.QueryResponse, nil
}

func (c *HTTPClient) queryEntities(ctx context.Context, s Session, key, q string) ([]entityJSON, error) {
	res, err := c.query(ctx, s, q)
	if err != nil {
		return nil, err
	}
	var list []entityJSON
	if raw, ok := res[key]; ok {
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (c *HTTPClient) queryFirst(ctx context.Context, s Session, key, q string) (*Entity, error) {
	list, err := c.queryEntities(ctx, s, key, q)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0].entity(), nil
}

// CompanyName implements Client.
func (c *HTTPClient) CompanyName(ctx context.Context, s Session) (string, error) {
	var out struct {
		CompanyInfo struct {
			CompanyName string `json:"CompanyName"`
		} `json:"CompanyInfo"`
	}
	if err := c.do(ctx, s, http.MethodGet, "companyinfo/"+url.PathEscape(s.RealmID), nil, nil, &out); err != nil {
		return "", err
	}
	return out.CompanyInfo.CompanyName, nil
}

// ListAccounts implements Client.
func (c *HTTPClient) ListAccounts(ctx context.Context, s Session, accountType string) ([]Ref, error) {
	list, err := c.queryEntities(ctx, s, "Account",
		fmt.Sprintf("select Id, Name from Account where AccountType = '%s' and Active = true maxresults 1000", queryEscape(accountType)))
	if err != nil {
		return nil, err
	}
	refs := make([]Ref, 0, len(list))
	for _, a := range list {
		refs = append(refs, Ref{Value: a.ID, Name: a.Name})
	}
	return refs, nil
}

// ListTaxCodes implements Client.
func (c *HTTPClient) ListTaxCodes(ctx context.Context, s Session) ([]Ref, error) {
	list, err := c.queryEntities(ctx, s, "TaxCode", "select Id, Name from TaxCode where Active = true maxresults 1000")
	if err != nil {
		return nil, err
	}
	refs := make([]Ref, 0, len(list))
	for _, t := range list {
		refs = append(refs, Ref{Value: t.ID, Name: t.Name})
	}
	return refs, nil
}

// FindItemByName implements Client.
func (c *HTTPClient) FindItemByName(ctx context.Context, s Session, name string) (*Entity, error) {
	return c.queryFirst(ctx, s, "Item", fmt.Sprintf("select * from Item where Name = '%s'", queryEscape(name)))
}

// CreateItem implements Client.
func (c *HTTPClient) CreateItem(ctx context.Context, s Session, item Item, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindItem, nil, item, requestID)
}

// FindCustomerByDisplayName implements Client.
func (c *HTTPClient) FindCustomerByDisplayName(ctx context.Context, s Session, name string) (*Entity, error) {
	return c.queryFirst(ctx, s, "Customer", fmt.Sprintf("select * from Customer where DisplayName = '%s'", queryEscape(name)))
}

// FindInvoiceByDocNumber implements Client.
func (c *HTTPClient) FindInvoiceByDocNumber(ctx context.Context, s Session, docNumber string) (*Entity, error) {
	return c.queryFirst(ctx, s, "Invoice", fmt.Sprintf("select * from Invoice where DocNumber = '%s'", queryEscape(docNumber)))
}

// Get implements Client.
func (c *HTTPClient) Get(ctx context.Context, s Session, kind, id string) (*Entity, error) {
	out := map[string]json.RawMessage{}
	if err := c.do(ctx, s, http.MethodGet, kind+"/"+url.PathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return entityFrom(out, kind)
}

// CreateCustomer implements Client.
func (c *HTTPClient) CreateCustomer(ctx context.Context, s Session, cust Customer, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindCustomer, nil, cust, requestID)
}

// UpdateCustomer implements Client.
func (c *HTTPClient) UpdateCustomer(ctx context.Context, s Session, e Entity, cust Customer, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindCustomer, &e, cust, requestID)
}

// CreateInvoice implements Client.
func (c *HTTPClient) CreateInvoice(ctx context.Context, s Session, inv Invoice, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindInvoice, nil, inv, requestID)
}

// UpdateInvoice implements Client.
func (c *HTTPClient) UpdateInvoice(ctx context.Context, s Session, e Entity, inv Invoice, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindInvoice, &e, inv, requestID)
}

// CreatePayment implements Client.
func (c *HTTPClient) CreatePayment(ctx context.Context, s Session, p Payment, requestID string) (*Entity, error) {
	return c.write(ctx, s, EntityKindPayment, nil, p, requestID)
}
