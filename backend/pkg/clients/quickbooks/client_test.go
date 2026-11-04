package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHTTPClient(t *testing.T, h http.HandlerFunc) *HTTPClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewHTTPClient(Config{
		ClientID: "cid", ClientSecret: "csecret", RedirectURI: "https://app.example/api/v1/integrations/quickbooks/callback",
		TokenEndpoint: srv.URL + "/token", RevokeEndpoint: srv.URL + "/revoke", APIBase: srv.URL,
	})
}

func TestAuthorizeURL(t *testing.T) {
	c := NewHTTPClient(Config{ClientID: "cid", RedirectURI: "https://app.example/cb"})
	u, err := url.Parse(c.AuthorizeURL("st4te"))
	require.NoError(t, err)
	assert.Equal(t, "appcenter.intuit.com", u.Host)
	q := u.Query()
	assert.Equal(t, "cid", q.Get("client_id"))
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, Scope, q.Get("scope"))
	assert.Equal(t, "https://app.example/cb", q.Get("redirect_uri"))
	assert.Equal(t, "st4te", q.Get("state"))
}

func TestExchangeAndRefresh(t *testing.T) {
	var forms []url.Values
	c := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "cid", user)
		assert.Equal(t, "csecret", pass)
		require.NoError(t, r.ParseForm())
		forms = append(forms, r.PostForm)
		if r.PostForm.Get("refresh_token") == "dead" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"at","refresh_token":"rt2","expires_in":3600,"x_refresh_token_expires_in":8726400}`)
	})
	now := time.Now()
	tok, err := c.ExchangeCode(context.Background(), "the-code")
	require.NoError(t, err)
	assert.Equal(t, "at", tok.AccessToken)
	assert.Equal(t, "rt2", tok.RefreshToken)
	assert.WithinDuration(t, now.Add(time.Hour), tok.AccessExpiresAt, 5*time.Second)
	assert.WithinDuration(t, now.Add(8726400*time.Second), tok.RefreshExpiresAt, 5*time.Second)
	assert.Equal(t, "authorization_code", forms[0].Get("grant_type"))
	assert.Equal(t, "the-code", forms[0].Get("code"))
	assert.Equal(t, "https://app.example/api/v1/integrations/quickbooks/callback", forms[0].Get("redirect_uri"))

	_, err = c.RefreshTokens(context.Background(), "rt1")
	require.NoError(t, err)
	assert.Equal(t, "refresh_token", forms[1].Get("grant_type"))

	_, err = c.RefreshTokens(context.Background(), "dead")
	assert.ErrorIs(t, err, ErrInvalidGrant)
}

func TestDataCallsSendMinorVersionRequestIDAndSparseUpdates(t *testing.T) {
	var got []*http.Request
	var bodies []map[string]any
	c := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"Invoice":{"Id":"130","SyncToken":"1","DocNumber":"INV-1"},"time":"2026-11-04"}`)
	})
	s := Session{RealmID: "123", AccessToken: "tok"}
	e, err := c.UpdateInvoice(context.Background(), s, Entity{ID: "130", SyncToken: "0"}, Invoice{DocNumber: "INV-1", CustomerRef: Ref{Value: "5"}}, "req-1")
	require.NoError(t, err)
	assert.Equal(t, &Entity{ID: "130", SyncToken: "1", Name: ""}, &Entity{ID: e.ID, SyncToken: e.SyncToken})

	require.Len(t, got, 1)
	assert.Equal(t, "/v3/company/123/invoice", got[0].URL.Path)
	assert.Equal(t, MinorVersion, got[0].URL.Query().Get("minorversion"))
	assert.Equal(t, "req-1", got[0].URL.Query().Get("requestid"))
	assert.Equal(t, true, bodies[0]["sparse"])
	assert.Equal(t, "130", bodies[0]["Id"])
	assert.Equal(t, "0", bodies[0]["SyncToken"])
}

func TestFaultsBecomeTypedErrors(t *testing.T) {
	status, body := 400, `{"Fault":{"Error":[{"Message":"Stale Object Error","Detail":"stale","code":"5010"}],"type":"ValidationFault"}}`
	c := testHTTPClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
	s := Session{RealmID: "1", AccessToken: "t"}
	_, err := c.CreateCustomer(context.Background(), s, Customer{DisplayName: "x"}, "r")
	assert.True(t, IsStale(err), err)
	assert.False(t, IsTransient(err))

	status, body = 429, `{}`
	_, err = c.CreateCustomer(context.Background(), s, Customer{DisplayName: "x"}, "r")
	assert.True(t, IsRateLimited(err))
	assert.True(t, IsTransient(err))

	status, body = 401, `{"Fault":{"Error":[{"Message":"AuthenticationFailed","code":"3200"}]}}`
	_, err = c.CreateCustomer(context.Background(), s, Customer{DisplayName: "x"}, "r")
	assert.True(t, IsUnauthorized(err))
}

func TestQueryEscapesQuotes(t *testing.T) {
	var q string
	c := testHTTPClient(t, func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query().Get("query")
		_, _ = io.WriteString(w, `{"QueryResponse":{"Customer":[{"Id":"7","SyncToken":"2","DisplayName":"O'Brien Facilities"}]}}`)
	})
	e, err := c.FindCustomerByDisplayName(context.Background(), Session{RealmID: "1", AccessToken: "t"}, "O'Brien Facilities")
	require.NoError(t, err)
	assert.Equal(t, "7", e.ID)
	assert.Equal(t, "2", e.SyncToken)
	assert.True(t, strings.Contains(q, `O\'Brien`), q)
}

func TestFakeRequestIDAndSyncToken(t *testing.T) {
	f := NewFake("http://localhost/cb")
	tok, err := f.ExchangeCode(context.Background(), f.IssueCode())
	require.NoError(t, err)
	s := Session{RealmID: f.RealmID, AccessToken: tok.AccessToken}

	a, err := f.CreateInvoice(context.Background(), s, Invoice{DocNumber: "INV-1"}, "req-a")
	require.NoError(t, err)
	b, err := f.CreateInvoice(context.Background(), s, Invoice{DocNumber: "INV-1"}, "req-a")
	require.NoError(t, err)
	assert.Equal(t, a.ID, b.ID, "same requestid, same record")
	assert.Len(t, f.Snapshot(EntityKindInvoice), 1)

	_, err = f.UpdateInvoice(context.Background(), s, Entity{ID: a.ID, SyncToken: "7"}, Invoice{}, "req-b")
	assert.True(t, IsStale(err))

	f.ExpireAccessTokens()
	_, err = f.CreateInvoice(context.Background(), s, Invoice{DocNumber: "INV-2"}, "req-c")
	assert.True(t, IsUnauthorized(err))

	_, err = f.RefreshTokens(context.Background(), tok.RefreshToken)
	require.NoError(t, err)
	_, err = f.RefreshTokens(context.Background(), tok.RefreshToken)
	assert.ErrorIs(t, err, ErrInvalidGrant, "refresh tokens rotate")
}
