package main

import (
	"testing"

	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
)

// T126: the binary must link the error-tracking providers it selects;
// otherwise errortracking.NewClient fails and nothing reaches Sentry.
func TestErrorTrackingProvidersRegistered(t *testing.T) {
	registered := map[errortracking.Provider]bool{}
	for _, p := range errortracking.GetRegisteredProviders() {
		registered[p] = true
	}
	for _, want := range []errortracking.Provider{errortracking.ProviderSentry, errortracking.ProviderMock} {
		if !registered[want] {
			t.Errorf("error tracking provider %q is not linked into cmd/main", want)
		}
	}
}
