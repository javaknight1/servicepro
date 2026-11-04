// Package quickbooks syncs a tenant's customers, invoices and payments one
// way, ServicePro → QuickBooks Online (T192).
//
// A tenant connects its QuickBooks company with Intuit's OAuth 2.0
// authorization-code flow (oauth.go). Tokens are sealed at rest with
// pkg/secretbox and refreshed under a row lock so concurrent refreshes don't
// clobber each other's rotated refresh token.
//
// quickbooks_sync_map maps each local record to its QuickBooks entity and is
// the work queue (sync.go). Records are enqueued cheaply when an invoice is
// sent or a payment recorded, and a sweep in the sync pass catches anything
// else (customer/invoice edits, Stripe payments, consolidated invoices).
// POST /internal/jobs/quickbooks-sync pushes pending records in time-boxed
// batches; every write is idempotent (the map row, Intuit requestid reused
// on retries, lookups before re-creating), so retries never duplicate.
package quickbooks

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
	"github.com/javaknight1/servicepro/backend/pkg/secretbox"
)

// Domain errors.
var (
	ErrNotConfigured = apperrors.New(http.StatusServiceUnavailable, "quickbooks_not_configured",
		"QuickBooks isn't available yet")
	ErrNotConnected = apperrors.Conflict("quickbooks_not_connected",
		"QuickBooks isn't connected")
	ErrReauthRequired = apperrors.Conflict("quickbooks_reauth_required",
		"QuickBooks needs to be reconnected: the authorization expired or was revoked")
	ErrInvalidState = apperrors.BadRequest("quickbooks_invalid_state",
		"This QuickBooks connect link expired or was already used. Start again from Settings → Integrations.")
	ErrRecordNotFound = apperrors.NotFound("quickbooks_record_not_found", "Sync record not found")
	ErrInvalidSetting = apperrors.BadRequest("quickbooks_invalid_setting",
		"That account or tax code isn't in the connected QuickBooks company")
	ErrQuickBooksUnavailable = apperrors.New(http.StatusBadGateway, "quickbooks_unavailable",
		"QuickBooks didn't answer. Try again in a minute.")
)

// PermissionChecker re-checks, at the OAuth callback, that the user who
// started the connect still may manage integrations in the tenant.
type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID, tenantID uuid.UUID, permissionName string) (bool, error)
}

// Config configures the service.
type Config struct {
	DB *gorm.DB
	// Client is nil when the integration is not configured (no Intuit keys).
	Client qbo.Client
	// Box seals tokens; required when Client is set.
	Box *secretbox.Box
	// Environment is "sandbox" or "production" (recorded on connections).
	Environment string
	Perms       PermissionChecker
	Now         func() time.Time
}

// Service is the QuickBooks integration.
type Service struct {
	db          *gorm.DB
	client      qbo.Client
	box         *secretbox.Box
	environment string
	perms       PermissionChecker
	now         func() time.Time
	newID       func() string
}

// NewService returns the service. With a nil Client it only answers
// "not configured".
func NewService(cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	env := cfg.Environment
	if env == "" {
		env = "sandbox"
	}
	return &Service{
		db: cfg.DB, client: cfg.Client, box: cfg.Box, environment: env, perms: cfg.Perms, now: now,
		newID: uuid.NewString,
	}
}

// Configured reports whether the server can offer QuickBooks.
func (s *Service) Configured() bool { return s != nil && s.client != nil && s.box != nil }

// tenantCtx scopes ctx to tenantID (tenancy plugin + RLS).
func tenantCtx(ctx context.Context, tenantID uuid.UUID) context.Context {
	return tenancy.WithTenant(ctx, tenantID)
}

// loadConnection returns the tenant's connection row, or nil.
func (s *Service) loadConnection(ctx context.Context, tenantID uuid.UUID) (*models.QuickBooksConnection, error) {
	var conn models.QuickBooksConnection
	err := s.db.WithContext(tenantCtx(ctx, tenantID)).Where("tenant_id = ?", tenantID).First(&conn).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// connected returns the tenant's connection when it is connected.
func (s *Service) connected(ctx context.Context, tenantID uuid.UUID) (*models.QuickBooksConnection, error) {
	if !s.Configured() {
		return nil, ErrNotConfigured
	}
	conn, err := s.loadConnection(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if conn == nil || conn.Status == models.QuickBooksDisconnected {
		return nil, ErrNotConnected
	}
	if conn.Status == models.QuickBooksReauthRequired {
		return nil, ErrReauthRequired
	}
	return conn, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ref(id, name *string) *models.QuickBooksRef {
	if id == nil || *id == "" {
		return nil
	}
	return &models.QuickBooksRef{ID: *id, Name: deref(name)}
}
