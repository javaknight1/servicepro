package resend

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Retry policy for Resend API calls (T159). Transient failures — 429, 5xx
// and network errors — are retried with capped exponential backoff and full
// jitter; Resend's Retry-After wins when present. Anything else (4xx: bad
// address, unverified domain, invalid key) fails fast. Sends carry an
// Idempotency-Key, so a retry after a lost response can't send twice.
const (
	retryMaxAttempts   = 4
	retryBaseDelay     = 500 * time.Millisecond
	retryMaxDelay      = 8 * time.Second
	retryMaxRetryAfter = 30 * time.Second
)

// retryTransport wraps an http.RoundTripper with the retry policy above.
type retryTransport struct {
	next  http.RoundTripper
	sleep func(context.Context, time.Duration) error // replaced in tests
}

func newRetryTransport(next http.RoundTripper) *retryTransport {
	if next == nil {
		next = http.DefaultTransport
	}
	return &retryTransport{next: next, sleep: sleepCtx}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// resend-go sets req.Body without GetBody, so buffer the (small, JSON)
	// body once to replay it on each attempt.
	if req.Body != nil && req.GetBody == nil {
		buf, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(buf))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(buf)), nil
		}
	}
	canReplay := req.Body == nil || req.GetBody != nil
	for attempt := 1; ; attempt++ {
		if attempt > 1 && req.Body != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		resp, err := t.next.RoundTrip(req)
		if !canReplay || attempt >= retryMaxAttempts || !retryable(resp, err) || req.Context().Err() != nil {
			return resp, err
		}

		delay := backoff(attempt)
		if resp != nil {
			if ra, ok := retryAfter(resp); ok {
				delay = ra
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		if serr := t.sleep(req.Context(), delay); serr != nil {
			return nil, serr
		}
	}
}

func retryable(resp *http.Response, err error) bool {
	if err != nil {
		return true // connection reset, timeout, DNS: transient
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// backoff is full-jitter exponential: a random delay in [0, min(max, base*2^(n-1))].
func backoff(attempt int) time.Duration {
	ceiling := retryBaseDelay << (attempt - 1)
	if ceiling > retryMaxDelay || ceiling <= 0 {
		ceiling = retryMaxDelay
	}
	return time.Duration(rand.Int64N(int64(ceiling) + 1)) //nolint:gosec // jitter, not security
}

func retryAfter(resp *http.Response) (time.Duration, bool) {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0, false
	}
	d := time.Duration(secs) * time.Second
	if d > retryMaxRetryAfter {
		d = retryMaxRetryAfter
	}
	return d, true
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
