package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Errors returned by the Connect service.
var (
	// ErrPaymentsNotConfigured: Stripe isn't configured on this server.
	ErrPaymentsNotConfigured = apperrors.New(http.StatusServiceUnavailable, "payments_not_configured",
		"Online payments aren't available on this server yet")
	// ErrConnectUnavailable: the platform's Stripe Connect setup is not done.
	ErrConnectUnavailable = apperrors.New(http.StatusServiceUnavailable, "connect_unavailable",
		"Connecting to Stripe isn't available yet. Please try again later")
	// ErrStripeUnavailable: Stripe call failed.
	ErrStripeUnavailable = apperrors.New(http.StatusBadGateway, "stripe_unavailable",
		"Stripe couldn't be reached. Please try again")
)

// PaymentsConnectService connects a tenant's own Stripe account (Accounts v2,
// direct charges, D9/T169) and caches its readiness. The Stripe account ID
// only ever comes from Stripe's response and our own table.
type PaymentsConnectService struct {
	db              *gorm.DB
	repo            repository.ConnectedAccountRepositoryInterface
	api             stripeClient.ConnectAPI // nil when Stripe isn't configured
	frontendURL     string
	paymentsEnabled bool // INVOICE_PAYMENTS_ENABLED master switch (T261)
	now             func() time.Time
}

// PaymentsConnectServiceConfig configures the service.
type PaymentsConnectServiceConfig struct {
	DB              *gorm.DB
	Repo            repository.ConnectedAccountRepositoryInterface
	API             stripeClient.ConnectAPI
	FrontendURL     string
	PaymentsEnabled bool
}

// NewPaymentsConnectService creates the service.
func NewPaymentsConnectService(cfg PaymentsConnectServiceConfig) *PaymentsConnectService {
	return &PaymentsConnectService{
		db:              cfg.DB,
		repo:            cfg.Repo,
		api:             cfg.API,
		frontendURL:     cfg.FrontendURL,
		paymentsEnabled: cfg.PaymentsEnabled,
		now:             time.Now,
	}
}

// settingsURL is where Stripe sends the user back after onboarding.
func (s *PaymentsConnectService) settingsURL(param string) string {
	return fmt.Sprintf("%s/settings/payments?onboarding=%s", s.frontendURL, param)
}

// StartOnboarding creates the tenant's connected account on first use and
// returns a fresh Stripe-hosted onboarding link (links are single-use).
func (s *PaymentsConnectService) StartOnboarding(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if s.api == nil {
		return "", ErrPaymentsNotConfigured
	}
	acct, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if acct == nil {
		if acct, err = s.createAccount(ctx, tenantID); err != nil {
			return "", err
		}
	}
	url, err := s.api.CreateOnboardingLink(ctx, acct.StripeAccountID,
		s.settingsURL("return"), s.settingsURL("refresh"),
		stripeClient.OnboardingLinkKey(tenantID.String(), s.now()))
	if err != nil {
		return "", s.stripeError(ctx, "create onboarding link", err)
	}
	return url, nil
}

func (s *PaymentsConnectService) createAccount(ctx context.Context, tenantID uuid.UUID) (*models.StripeConnectedAccount, error) {
	var tenant models.Tenant
	if err := s.db.WithContext(ctx).Preload("Owner").First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, fmt.Errorf("load tenant: %w", err)
	}
	email := tenant.Email
	if email == "" && tenant.Owner != nil {
		email = tenant.Owner.Email
	}
	status, err := s.api.CreateAccount(ctx, stripeClient.CreateAccountInput{
		TenantID:       tenantID.String(),
		DisplayName:    tenant.Name,
		ContactEmail:   email,
		IdempotencyKey: stripeClient.AccountCreateKey(tenantID.String()),
	})
	if err != nil {
		return nil, s.stripeError(ctx, "create connected account", err)
	}
	logging.Info(ctx, "[PAYMENTS-CONNECT] Connected account created", map[string]any{"tenant_id": tenantID, "account_id": status.AccountID})
	return s.repo.CreateIfAbsent(ctx, tenantID, status.AccountID, toModelStatus(status))
}

func (s *PaymentsConnectService) stripeError(ctx context.Context, op string, err error) error {
	logging.Error(ctx, "[PAYMENTS-CONNECT] Stripe call failed", map[string]any{"op": op, "error": err})
	if errors.Is(err, stripeClient.ErrConnectNotEnabled) {
		return ErrConnectUnavailable.WithCause(err)
	}
	return ErrStripeUnavailable.WithCause(err)
}

// Status returns the tenant's payment readiness. While onboarding isn't
// finished (the user may have just come back from Stripe) it re-reads the
// account from Stripe first; a failed refresh falls back to the cache.
func (s *PaymentsConnectService) Status(ctx context.Context, tenantID uuid.UUID) (*models.PaymentsStatusResponse, error) {
	acct, err := s.repo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if acct != nil && s.api != nil && !acct.OnboardingComplete() {
		if fresh, err := s.api.RetrieveAccount(ctx, acct.StripeAccountID); err != nil {
			logging.Warn(ctx, "[PAYMENTS-CONNECT] Account refresh failed; showing cached status", map[string]any{"tenant_id": tenantID, "error": err})
		} else if err := s.repo.UpdateStatus(ctx, tenantID, toModelStatus(fresh)); err == nil {
			if acct, err = s.repo.GetByTenant(ctx, tenantID); err != nil {
				return nil, err
			}
		}
	}
	return s.statusResponse(acct), nil
}

func (s *PaymentsConnectService) statusResponse(acct *models.StripeConnectedAccount) *models.PaymentsStatusResponse {
	resp := &models.PaymentsStatusResponse{
		State:                 models.PaymentsStateNotConnected,
		OnlinePaymentsEnabled: s.paymentsEnabled && s.api != nil,
		Fees: models.PaymentFeesResponse{
			CardPercent: stripeClient.CardFeeRate.Shift(2).String(),
			ACHPercent:  stripeClient.ACHFeeRate.Shift(2).String(),
			ACHCap:      stripeClient.ACHFeeCapAmount.StringFixed(2),
			Currency:    "USD",
		},
	}
	if acct == nil {
		return resp
	}
	resp.Connected = true
	resp.CardPaymentsStatus = acct.CardPaymentsStatus
	resp.ACHPaymentsStatus = acct.ACHPaymentsStatus
	resp.PayoutsStatus = acct.PayoutsStatus
	resp.RequirementsStatus = acct.RequirementsStatus
	resp.Livemode = acct.Livemode
	resp.SyncedAt = acct.SyncedAt
	resp.PayLinksActive = resp.OnlinePaymentsEnabled && acct.CanAcceptCards()
	switch {
	case acct.CardPaymentsStatus == "rejected" || acct.RequirementsStatus == "past_due":
		resp.State = models.PaymentsStateRestricted
	case acct.CanAcceptCards() && acct.RequirementsStatus == "currently_due":
		resp.State = models.PaymentsStateActionRequired
	case acct.CanAcceptCards():
		resp.State = models.PaymentsStateActive
	default:
		resp.State = models.PaymentsStateOnboarding
	}
	return resp
}

// SyncAccount re-reads a connected account from Stripe and caches its
// statuses (webhooks: account.updated, v2.core.account[...]). Accounts that
// aren't ours are ignored. Runs without a tenant session.
func (s *PaymentsConnectService) SyncAccount(ctx context.Context, stripeAccountID string) error {
	if s.api == nil {
		return nil
	}
	sys := tenancy.WithSystem(context.WithoutCancel(ctx))
	acct, err := s.repo.GetByStripeAccountID(sys, stripeAccountID)
	if err != nil {
		return err
	}
	if acct == nil {
		logging.Info(ctx, "[PAYMENTS-CONNECT] Event for an unknown connected account ignored", map[string]any{"account_id": stripeAccountID})
		return nil
	}
	fresh, err := s.api.RetrieveAccount(ctx, stripeAccountID)
	if err != nil {
		return fmt.Errorf("retrieve connected account: %w", err)
	}
	return s.repo.UpdateStatus(tenancy.WithTenant(ctx, acct.TenantID), acct.TenantID, toModelStatus(fresh))
}

// Account returns the tenant's connected account (nil when none).
func (s *PaymentsConnectService) Account(ctx context.Context, tenantID uuid.UUID) (*models.StripeConnectedAccount, error) {
	return s.repo.GetByTenant(tenancy.WithTenant(ctx, tenantID), tenantID)
}

// TenantForStripeAccount resolves a webhook's connected account to its
// tenant; ok is false when the account isn't ours.
func (s *PaymentsConnectService) TenantForStripeAccount(ctx context.Context, stripeAccountID string) (uuid.UUID, bool, error) {
	acct, err := s.repo.GetByStripeAccountID(tenancy.WithSystem(context.WithoutCancel(ctx)), stripeAccountID)
	if err != nil || acct == nil {
		return uuid.Nil, false, err
	}
	return acct.TenantID, true, nil
}

// CanAcceptPayments reports whether invoices of this tenant may carry a pay
// link: the master switch is on and the tenant can take cards.
func (s *PaymentsConnectService) CanAcceptPayments(ctx context.Context, tenantID uuid.UUID) bool {
	if !s.paymentsEnabled || s.api == nil {
		return false
	}
	acct, err := s.Account(ctx, tenantID)
	if err != nil {
		logging.Warn(ctx, "[PAYMENTS-CONNECT] Could not read connected account; no pay link", map[string]any{"tenant_id": tenantID, "error": err})
		return false
	}
	return acct.CanAcceptCards()
}

// accountSyncer is what the Connect webhook handlers need (faked in tests).
type accountSyncer interface {
	SyncAccount(ctx context.Context, stripeAccountID string) error
}

// RegisterConnectWebhookHandlers registers account.updated from the Connect
// endpoint (T169): the account is re-read from Stripe and its statuses cached.
// The payload is never trusted for the status itself.
func RegisterConnectWebhookHandlers(p *stripeClient.EventProcessor, accounts accountSyncer) {
	p.RegisterHandler(stripeClient.EventAccountUpdated, func(ctx context.Context, event *stripe.Event) error {
		accountID := event.Account
		if accountID == "" && event.Data != nil {
			var obj struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(event.Data.Raw, &obj); err == nil {
				accountID = obj.ID
			}
		}
		if accountID == "" {
			return nil
		}
		return accounts.SyncAccount(ctx, accountID)
	})
}

func toModelStatus(s *stripeClient.AccountStatus) models.StripeAccountStatus {
	if s == nil {
		return models.StripeAccountStatus{}
	}
	return models.StripeAccountStatus{
		Livemode:           s.Livemode,
		CardPaymentsStatus: s.CardPaymentsStatus,
		ACHPaymentsStatus:  s.ACHPaymentsStatus,
		PayoutsStatus:      s.PayoutsStatus,
		RequirementsStatus: s.RequirementsStatus,
	}
}
