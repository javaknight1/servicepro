package stripe

// T169 spike: Stripe Connect on Accounts v2 with direct charges, against the
// Stripe TEST API. Skipped unless STRIPE_CONNECT_SPIKE=1; reads the test key
// from STRIPE_SECRET_KEY or backend/.env and refuses anything but sk_test_/rk_test_.
// Findings: docs/architecture/stripe-connect.md.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v87"
)

func spikeClient(t *testing.T) *stripe.Client {
	t.Helper()
	if os.Getenv("STRIPE_CONNECT_SPIKE") != "1" {
		t.Skip("set STRIPE_CONNECT_SPIKE=1 to run the Stripe Connect spike against the test API")
	}
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		env, err := godotenv.Read("../../../.env")
		if err != nil {
			t.Fatalf("no STRIPE_SECRET_KEY and no backend/.env: %v", err)
		}
		key = env["STRIPE_SECRET_KEY"]
	}
	if !strings.HasPrefix(key, "sk_test_") && !strings.HasPrefix(key, "rk_test_") {
		t.Fatal("the spike only runs with a TEST key")
	}
	return stripe.NewClient(key)
}

func TestConnectSpike(t *testing.T) {
	sc := spikeClient(t)
	ctx := context.Background()
	run := time.Now().UTC().Format("20060102T150405")

	// 1. Accounts v2: SaaS merchant (direct charges), Stripe collects fees and
	// owns losses, full Dashboard; customer configuration for platform billing.
	create := &stripe.V2CoreAccountCreateParams{
		DisplayName:  stripe.String("T169 Spike Cleaning Co"),
		ContactEmail: stripe.String("t169-spike@example.com"),
		Dashboard:    stripe.String("full"),
		Identity: &stripe.V2CoreAccountCreateIdentityParams{
			Country:    stripe.String("us"),
			EntityType: stripe.String("company"),
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
			Responsibilities: &stripe.V2CoreAccountCreateDefaultsResponsibilitiesParams{
				FeesCollector:   stripe.String("stripe"),
				LossesCollector: stripe.String("stripe"),
			},
			Locales: []*string{stripe.String("en-US")},
		},
		Include:  stripe.StringSlice([]string{"configuration.merchant", "configuration.customer", "requirements"}),
		Metadata: map[string]string{"spike": "T169"},
	}
	create.SetIdempotencyKey("t169-spike-account-" + run)
	acct, err := sc.V2CoreAccounts.Create(ctx, create)
	if err != nil {
		// v2 errors are not *stripe.Error; match the code in the message.
		if strings.Contains(err.Error(), "non_connect_platform_accounts_v2_access_blocked") {
			t.Log("BLOCKER: Connect / Accounts v2 is not enabled on this platform account (non_connect_platform_accounts_v2_access_blocked)")
			spikePlatformCheckoutShape(t, sc, run)
			t.Skip("Connect platform setup needed in the Dashboard before the rest of the spike can run")
		}
		t.Fatalf("create v2 account: %v", err)
	}
	t.Logf("account %s dashboard=%s applied=%v", acct.ID, acct.Dashboard, acct.AppliedConfigurations)
	if acct.Configuration != nil && acct.Configuration.Merchant != nil && acct.Configuration.Merchant.Capabilities != nil {
		caps := acct.Configuration.Merchant.Capabilities
		if caps.CardPayments != nil {
			t.Logf("card_payments=%s", caps.CardPayments.Status)
		}
		if caps.ACHDebitPayments != nil {
			t.Logf("ach_debit_payments=%s", caps.ACHDebitPayments.Status)
		}
	}
	if acct.Requirements != nil && acct.Requirements.Summary != nil && acct.Requirements.Summary.MinimumDeadline != nil {
		t.Logf("requirements minimum_deadline=%s entries=%d", acct.Requirements.Summary.MinimumDeadline.Status, len(acct.Requirements.Entries))
	}

	// 2. v2 Account Link (Stripe-hosted onboarding).
	link := &stripe.V2CoreAccountLinkCreateParams{
		Account: stripe.String(acct.ID),
		UseCase: &stripe.V2CoreAccountLinkCreateUseCaseParams{
			Type: stripe.String("account_onboarding"),
			AccountOnboarding: &stripe.V2CoreAccountLinkCreateUseCaseAccountOnboardingParams{
				CollectionOptions: &stripe.V2CoreAccountLinkCreateUseCaseAccountOnboardingCollectionOptionsParams{
					Fields: stripe.String("eventually_due"),
				},
				RefreshURL: stripe.String("http://localhost:5173/settings/payments?refresh=1"),
				ReturnURL:  stripe.String("http://localhost:5173/settings/payments?return=1"),
			},
		},
	}
	link.SetIdempotencyKey("t169-spike-link-" + run)
	al, err := sc.V2CoreAccountLinks.Create(ctx, link)
	if err != nil {
		t.Errorf("create account link: %v", err)
	} else {
		t.Logf("account link created, url host=%s expires=%s", hostOf(al.URL), al.ExpiresAt)
	}

	// 3. Direct charge: Checkout Session ON the connected account, bank only,
	// with an application fee (1% capped at $10 on $5,000 = $10).
	cs := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String("http://localhost:5173/invoices/payment-success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("http://localhost:5173/invoices/payment-cancelled"),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:    stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{Name: stripe.String("Invoice SPIKE-1")},
				UnitAmount:  stripe.Int64(500000),
			},
			Quantity: stripe.Int64(1),
		}},
		AllowedPaymentMethodTypes: stripe.StringSlice([]string{"us_bank_account"}),
		PaymentIntentData: &stripe.CheckoutSessionCreatePaymentIntentDataParams{
			ApplicationFeeAmount: stripe.Int64(1000),
			Metadata:             map[string]string{"spike": "T169"},
		},
	}
	cs.SetStripeAccount(acct.ID)
	cs.SetIdempotencyKey("t169-spike-checkout-bank-" + run)
	sess, err := sc.V1CheckoutSessions.Create(ctx, cs)
	if err != nil {
		t.Logf("checkout (bank) on connected account before onboarding: %v", err)
	} else {
		t.Logf("checkout (bank) session %s url host=%s", sess.ID, hostOf(sess.URL))
	}

	// 4. Direct charge via PaymentIntent with application_fee_amount (card).
	pi := &stripe.PaymentIntentCreateParams{
		Amount:               stripe.Int64(96000),
		Currency:             stripe.String("usd"),
		ApplicationFeeAmount: stripe.Int64(384), // 0.4% of $960
		Metadata:             map[string]string{"spike": "T169"},
	}
	pi.SetStripeAccount(acct.ID)
	pi.SetIdempotencyKey("t169-spike-pi-" + run)
	p, err := sc.V1PaymentIntents.Create(ctx, pi)
	if err != nil {
		t.Logf("payment intent on connected account before onboarding: %v", err)
	} else {
		t.Logf("payment intent %s status=%s methods=%v fee=%d", p.ID, p.Status, p.PaymentMethodTypes, p.ApplicationFeeAmount)
	}

	// 5. Retrieve (what GET /payments/status and webhooks will do).
	got, err := sc.V2CoreAccounts.Retrieve(ctx, acct.ID, &stripe.V2CoreAccountRetrieveParams{
		Include: stripe.StringSlice([]string{"configuration.merchant", "requirements"}),
	})
	if err != nil {
		t.Fatalf("retrieve v2 account: %v", err)
	}
	t.Logf("retrieved %s closed=%v", got.ID, got.Closed)
	fmt.Fprintln(os.Stderr, "spike account:", got.ID)
}

// spikePlatformCheckoutShape checks, without Connect, that a bank-only
// Checkout Session (allowed_payment_method_types=[us_bank_account]) is
// accepted and ACH is available — the shape used for "Pay by bank".
func spikePlatformCheckoutShape(t *testing.T, sc *stripe.Client, run string) {
	t.Helper()
	cs := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String("http://localhost:5173/invoices/payment-success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("http://localhost:5173/invoices/payment-cancelled"),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:    stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{Name: stripe.String("Invoice SPIKE-1")},
				UnitAmount:  stripe.Int64(500000),
			},
			Quantity: stripe.Int64(1),
		}},
		AllowedPaymentMethodTypes: stripe.StringSlice([]string{"us_bank_account"}),
		PaymentIntentData:         &stripe.CheckoutSessionCreatePaymentIntentDataParams{Metadata: map[string]string{"spike": "T169"}},
	}
	cs.SetIdempotencyKey("t169-spike-platform-bank-" + run)
	sess, err := sc.V1CheckoutSessions.Create(context.Background(), cs)
	if err != nil {
		t.Errorf("platform bank-only checkout: %v", err)
		return
	}
	t.Logf("platform bank-only checkout %s methods=%v", sess.ID, sess.PaymentMethodTypes)
	cs2 := &stripe.CheckoutSessionCreateParams{
		Mode:                       cs.Mode,
		SuccessURL:                 cs.SuccessURL,
		CancelURL:                  cs.CancelURL,
		LineItems:                  cs.LineItems,
		ExcludedPaymentMethodTypes: stripe.StringSlice([]string{"us_bank_account"}),
	}
	cs2.SetIdempotencyKey("t169-spike-platform-card-" + run)
	sess2, err := sc.V1CheckoutSessions.Create(context.Background(), cs2)
	if err != nil {
		t.Errorf("platform card checkout (bank excluded): %v", err)
		return
	}
	t.Logf("platform card checkout %s methods=%v", sess2.ID, sess2.PaymentMethodTypes)
}

func hostOf(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexByte(u, '/'); i >= 0 {
		return u[:i]
	}
	return u
}
