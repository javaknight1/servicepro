package stripe

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v87"
)

// Stripe Connect on Accounts v2 with direct charges (D9, T169). See
// docs/architecture/stripe-connect.md for the decisions and the spike.

// AccountStatus is the part of a v2 account ServicePro caches.
type AccountStatus struct {
	AccountID          string
	Livemode           bool
	CardPaymentsStatus string // configuration.merchant.capabilities.card_payments.status
	ACHPaymentsStatus  string // configuration.merchant.capabilities.ach_debit_payments.status
	PayoutsStatus      string // configuration.merchant.capabilities.stripe_balance.payouts.status
	RequirementsStatus string // requirements.summary.minimum_deadline.status ("" = nothing due)
}

// CreateAccountInput is what ServicePro knows about a tenant when it creates
// its connected account. The tenant fills in everything else in onboarding.
type CreateAccountInput struct {
	TenantID       string
	DisplayName    string
	ContactEmail   string
	IdempotencyKey string
}

// ConnectAPI is the Connect surface the services use (faked in tests).
type ConnectAPI interface {
	CreateAccount(ctx context.Context, in CreateAccountInput) (*AccountStatus, error)
	CreateOnboardingLink(ctx context.Context, accountID, returnURL, refreshURL, idempotencyKey string) (string, error)
	RetrieveAccount(ctx context.Context, accountID string) (*AccountStatus, error)
}

// ErrConnectNotEnabled means the platform has not finished Connect setup in
// the Stripe Dashboard (Settings → Connect → Platform setup).
var ErrConnectNotEnabled = errors.New("stripe connect is not enabled on the platform account")

// ConnectClient calls the Accounts v2 API.
type ConnectClient struct {
	api *stripe.Client
}

// NewConnectClient creates a Connect client for the platform's secret key.
func NewConnectClient(secretKey string) *ConnectClient {
	return &ConnectClient{api: stripe.NewClient(secretKey)}
}

var _ ConnectAPI = (*ConnectClient)(nil)

var accountIncludes = []string{"configuration.merchant", "requirements"}

// CreateAccount creates the tenant's v2 account: merchant configuration with
// card and ACH payments (direct charges), a customer configuration for
// platform billing (T210), Stripe collecting its fees from the tenant and
// owning negative balances, and the full Stripe Dashboard (SaaS defaults).
func (c *ConnectClient) CreateAccount(ctx context.Context, in CreateAccountInput) (*AccountStatus, error) {
	params := &stripe.V2CoreAccountCreateParams{
		Dashboard: stripe.String("full"),
		Identity: &stripe.V2CoreAccountCreateIdentityParams{
			Country: stripe.String("us"),
		},
		Configuration: &stripe.V2CoreAccountCreateConfigurationParams{
			Merchant: &stripe.V2CoreAccountCreateConfigurationMerchantParams{
				Capabilities: &stripe.V2CoreAccountCreateConfigurationMerchantCapabilitiesParams{
					CardPayments:     &stripe.V2CoreAccountCreateConfigurationMerchantCapabilitiesCardPaymentsParams{Requested: stripe.Bool(true)},
					ACHDebitPayments: &stripe.V2CoreAccountCreateConfigurationMerchantCapabilitiesACHDebitPaymentsParams{Requested: stripe.Bool(true)},
				},
			},
			Customer: &stripe.V2CoreAccountCreateConfigurationCustomerParams{},
		},
		Defaults: &stripe.V2CoreAccountCreateDefaultsParams{
			Currency: stripe.String("usd"),
			Locales:  []*string{stripe.String("en-US")},
			Responsibilities: &stripe.V2CoreAccountCreateDefaultsResponsibilitiesParams{
				FeesCollector:   stripe.String("stripe"),
				LossesCollector: stripe.String("stripe"),
			},
		},
		Include:  stripe.StringSlice(accountIncludes),
		Metadata: map[string]string{"tenant_id": in.TenantID},
	}
	if in.DisplayName != "" {
		params.DisplayName = stripe.String(in.DisplayName)
	}
	if in.ContactEmail != "" {
		params.ContactEmail = stripe.String(in.ContactEmail)
	}
	params.SetIdempotencyKey(in.IdempotencyKey)

	acct, err := c.api.V2CoreAccounts.Create(ctx, params)
	if err != nil {
		return nil, wrapConnectError("create account", err)
	}
	return StatusFromV2Account(acct), nil
}

// CreateOnboardingLink returns a single-use Stripe-hosted onboarding URL
// collecting everything that will eventually be due (one visit, fewer holds).
func (c *ConnectClient) CreateOnboardingLink(ctx context.Context, accountID, returnURL, refreshURL, idempotencyKey string) (string, error) {
	params := &stripe.V2CoreAccountLinkCreateParams{
		Account: stripe.String(accountID),
		UseCase: &stripe.V2CoreAccountLinkCreateUseCaseParams{
			Type: stripe.String("account_onboarding"),
			AccountOnboarding: &stripe.V2CoreAccountLinkCreateUseCaseAccountOnboardingParams{
				CollectionOptions: &stripe.V2CoreAccountLinkCreateUseCaseAccountOnboardingCollectionOptionsParams{
					Fields: stripe.String("eventually_due"),
				},
				ReturnURL:  stripe.String(returnURL),
				RefreshURL: stripe.String(refreshURL),
			},
		},
	}
	params.SetIdempotencyKey(idempotencyKey)
	link, err := c.api.V2CoreAccountLinks.Create(ctx, params)
	if err != nil {
		return "", wrapConnectError("create account link", err)
	}
	return link.URL, nil
}

// RetrieveAccount reads the account's current capability and requirement
// statuses (read-only: no idempotency key).
func (c *ConnectClient) RetrieveAccount(ctx context.Context, accountID string) (*AccountStatus, error) {
	acct, err := c.api.V2CoreAccounts.Retrieve(ctx, accountID, &stripe.V2CoreAccountRetrieveParams{
		Include: stripe.StringSlice(accountIncludes),
	})
	if err != nil {
		return nil, wrapConnectError("retrieve account", err)
	}
	return StatusFromV2Account(acct), nil
}

// StatusFromV2Account extracts the cached statuses from a v2 account. The v1
// charges_enabled flag is deliberately not used (deprecated for v2).
func StatusFromV2Account(acct *stripe.V2CoreAccount) *AccountStatus {
	if acct == nil {
		return nil
	}
	s := &AccountStatus{AccountID: acct.ID, Livemode: acct.Livemode}
	if acct.Configuration != nil && acct.Configuration.Merchant != nil && acct.Configuration.Merchant.Capabilities != nil {
		caps := acct.Configuration.Merchant.Capabilities
		if caps.CardPayments != nil {
			s.CardPaymentsStatus = string(caps.CardPayments.Status)
		}
		if caps.ACHDebitPayments != nil {
			s.ACHPaymentsStatus = string(caps.ACHDebitPayments.Status)
		}
		if caps.StripeBalance != nil && caps.StripeBalance.Payouts != nil {
			s.PayoutsStatus = string(caps.StripeBalance.Payouts.Status)
		}
	}
	if acct.Requirements != nil && acct.Requirements.Summary != nil && acct.Requirements.Summary.MinimumDeadline != nil {
		s.RequirementsStatus = string(acct.Requirements.Summary.MinimumDeadline.Status)
	}
	return s
}

// wrapConnectError maps "Connect isn't set up on the platform" to
// ErrConnectNotEnabled (v2 errors are not *stripe.Error, so match the code).
func wrapConnectError(op string, err error) error {
	if msg := err.Error(); strings.Contains(msg, "non_connect_platform_accounts_v2_access_blocked") ||
		strings.Contains(msg, "platform_account_required") {
		return fmt.Errorf("%s: %w: %w", op, ErrConnectNotEnabled, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// OnboardingLinkKey is the idempotency key for an onboarding link: links are
// single-use, so a new one per tenant per minute (a double click reuses one).
func OnboardingLinkKey(tenantID string, now time.Time) string {
	return fmt.Sprintf("connect-link:%s:%s", tenantID, now.UTC().Format("200601021504"))
}

// AccountCreateKey is the idempotency key for a tenant's account: one per
// tenant, ever, so retries and double clicks can't create two accounts.
func AccountCreateKey(tenantID string) string {
	return "connect-account:" + tenantID
}
