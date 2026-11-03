package stripe

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"
)

// T169: readiness comes from the v2 configuration paths, not v1 charges_enabled.
func TestStatusFromV2Account(t *testing.T) {
	raw := `{
	  "id": "acct_123", "object": "v2.core.account", "livemode": false, "created": "2026-11-02T10:00:00.000Z",
	  "applied_configurations": ["merchant", "customer"],
	  "configuration": {"merchant": {"applied": true, "capabilities": {
	    "card_payments": {"status": "active", "status_details": []},
	    "ach_debit_payments": {"status": "pending", "status_details": [{"code": "requirements_pending_verification", "resolution": "provide_info"}]},
	    "stripe_balance": {"payouts": {"status": "restricted", "status_details": []}}
	  }}},
	  "requirements": {"entries": [], "summary": {"minimum_deadline": {"status": "currently_due", "time": "2026-12-01T00:00:00.000Z"}}}
	}`
	var acct stripe.V2CoreAccount
	require.NoError(t, json.Unmarshal([]byte(raw), &acct))
	st := StatusFromV2Account(&acct)
	assert.Equal(t, "acct_123", st.AccountID)
	assert.Equal(t, "active", st.CardPaymentsStatus)
	assert.Equal(t, "pending", st.ACHPaymentsStatus)
	assert.Equal(t, "restricted", st.PayoutsStatus)
	assert.Equal(t, "currently_due", st.RequirementsStatus)

	empty := StatusFromV2Account(&stripe.V2CoreAccount{ID: "acct_new"})
	assert.Equal(t, "", empty.CardPaymentsStatus)
	assert.Nil(t, StatusFromV2Account(nil))
}

func TestWrapConnectError(t *testing.T) {
	blocked := errors.New(`{"code":"non_connect_platform_accounts_v2_access_blocked","status":400}`)
	assert.ErrorIs(t, wrapConnectError("create account", blocked), ErrConnectNotEnabled)
	assert.NotErrorIs(t, wrapConnectError("create account", errors.New("rate limited")), ErrConnectNotEnabled)
}

func TestIdempotencyKeys(t *testing.T) {
	assert.Equal(t, "connect-account:t1", AccountCreateKey("t1"))
	assert.Equal(t, "connect-link:t1:202611021004", OnboardingLinkKey("t1", time.Date(2026, 11, 2, 10, 4, 59, 0, time.UTC)))
}
