package resend

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/resend/resend-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

// fakeResend answers POST /emails with the scripted statuses in order and
// records each request's Idempotency-Key and body.
type fakeResend struct {
	mu       sync.Mutex
	statuses []int
	keys     []string
	bodies   []string
}

func (f *fakeResend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, _ := io.ReadAll(r.Body)
	f.keys = append(f.keys, r.Header.Get("Idempotency-Key"))
	f.bodies = append(f.bodies, string(b))
	status := http.StatusOK
	if n := len(f.keys) - 1; n < len(f.statuses) {
		status = f.statuses[n]
	}
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "0")
	}
	w.WriteHeader(status)
	if status == http.StatusOK {
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
		return
	}
	_, _ = w.Write([]byte(`{"message":"scripted failure"}`))
}

func newTestClient(t *testing.T, srv *httptest.Server) (*Client, *[]time.Duration) {
	t.Helper()
	slept := &[]time.Duration{}
	rt := newRetryTransport(http.DefaultTransport)
	rt.sleep = func(_ context.Context, d time.Duration) error { *slept = append(*slept, d); return nil }
	rc := resend.NewCustomClient(&http.Client{Transport: rt, Timeout: 5 * time.Second}, "re_test")
	base, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	rc.BaseURL = base
	return &Client{client: rc, config: &Config{FromEmail: "noreply@example.test"}, fromAddr: "noreply@example.test"}, slept
}

func send(t *testing.T, c *Client) (*email.SendResult, error) {
	t.Helper()
	return c.Send(context.Background(), email.NewEmailMessage("to@example.test", "Hi", "<p>Hi</p>"))
}

func TestSend_RetriesTransientFailuresWithOneIdempotencyKey(t *testing.T) {
	f := &fakeResend{statuses: []int{503, 429, 200}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c, slept := newTestClient(t, srv)

	res, err := send(t, c)
	require.NoError(t, err)
	assert.Equal(t, "email_123", res.MessageID)
	require.Len(t, f.keys, 3, "503 and 429 are retried")
	assert.NotEmpty(t, f.keys[0])
	assert.Equal(t, f.keys[0], f.keys[1], "every attempt reuses the idempotency key")
	assert.Equal(t, f.keys[0], f.keys[2])
	assert.Equal(t, f.bodies[0], f.bodies[2], "the body is replayed intact")
	require.Len(t, *slept, 2)
	assert.Equal(t, time.Duration(0), (*slept)[1], "Retry-After: 0 is honoured")
}

func TestSend_DoesNotRetryClientErrors(t *testing.T) {
	f := &fakeResend{statuses: []int{422}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c, slept := newTestClient(t, srv)

	_, err := send(t, c)
	require.Error(t, err)
	assert.Len(t, f.keys, 1, "a 4xx fails fast")
	assert.Empty(t, *slept)
}

func TestSend_GivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeResend{statuses: []int{500, 500, 500, 500, 500}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c, _ := newTestClient(t, srv)

	_, err := send(t, c)
	require.Error(t, err)
	assert.Len(t, f.keys, retryMaxAttempts)
}

func TestBackoff_IsCapped(t *testing.T) {
	for attempt := 1; attempt <= 10; attempt++ {
		d := backoff(attempt)
		assert.GreaterOrEqual(t, d, time.Duration(0))
		assert.LessOrEqual(t, d, retryMaxDelay)
	}
}
