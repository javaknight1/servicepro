// Package tenancy carries the caller's tenant through context.Context and
// enforces tenant isolation on every GORM query against tenant-owned models.
//
// The tenant is put into the request context ONLY by the tenant middleware
// (middleware.RequireTenant / RequireTenantFromPath) after it has verified
// that the authenticated user is an active member of that tenant. Nothing
// else may call WithTenant with a value taken from the request.
//
// Code that legitimately runs outside a tenant (Stripe webhooks, background
// workers iterating all tenants, public token-based lookups) marks its
// context with WithSystem. Such code is responsible for scoping its own
// queries and should switch to WithTenant(ctx, row.TenantID) as soon as it
// knows which tenant a row belongs to.
package tenancy

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ctxKey int

const (
	tenantKey ctxKey = iota
	systemKey
)

// ErrNoTenant is returned when a tenant-owned model is queried without a
// tenant (or system) context. Failing closed prevents silent cross-tenant
// access when a code path forgets to propagate the request context.
var ErrNoTenant = errors.New("tenancy: no tenant in context for tenant-owned query")

// ErrTenantMismatch is returned when a write tries to set a tenant_id that is
// not the caller's tenant.
var ErrTenantMismatch = errors.New("tenancy: tenant_id does not match the caller's tenant")

// WithTenant returns a context scoped to tenantID.
func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantID returns the tenant in ctx, if any.
func TenantID(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	id, ok := ctx.Value(tenantKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// MustTenantID returns the tenant in ctx or ErrNoTenant.
func MustTenantID(ctx context.Context) (uuid.UUID, error) {
	id, ok := TenantID(ctx)
	if !ok {
		return uuid.Nil, ErrNoTenant
	}
	return id, nil
}

// WithSystem marks ctx as trusted system code that may query across tenants.
// A tenant set later with WithTenant takes precedence.
func WithSystem(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemKey, true)
}

// IsSystem reports whether ctx was marked with WithSystem.
func IsSystem(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(systemKey).(bool)
	return v
}
