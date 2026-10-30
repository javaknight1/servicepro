package email

import (
	"context"
	"errors"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Suppressor reports addresses that must not be emailed: a permanent bounce
// or a spam complaint recorded from the provider's webhook (T159).
type Suppressor interface {
	IsSuppressed(ctx context.Context, email string) (bool, error)
}

// ErrAllRecipientsSuppressed is returned when every To address is suppressed.
var ErrAllRecipientsSuppressed = errors.New("all recipients are suppressed (bounced or complained)")

// FilterSuppressed drops suppressed addresses. A lookup error keeps the
// address (fail open: a transient DB error must not silently drop mail) and
// is logged.
func FilterSuppressed(ctx context.Context, s Suppressor, addrs []string) []string {
	if s == nil || len(addrs) == 0 {
		return addrs
	}
	kept := addrs[:0:0]
	for _, a := range addrs {
		suppressed, err := s.IsSuppressed(ctx, a)
		if err != nil {
			logging.Error(ctx, "[EMAIL] Suppression lookup failed; sending anyway", map[string]any{"error": err})
		}
		if suppressed {
			logging.Info(ctx, "[EMAIL] Skipping suppressed recipient", nil)
			continue
		}
		kept = append(kept, a)
	}
	return kept
}
