package tenancy

import (
	"context"

	"github.com/google/uuid"
)

// Per-request access limits inside a tenant (T035). Like the tenant itself
// they are put into the request context only by the tenant middleware, from
// the caller's effective permissions in that tenant, and read by
// repositories and services.

type accessKey int

const (
	assignedJobsKey accessKey = iota
	moneyHiddenKey
)

// WithAssignedJobsOnly limits job queries made with ctx to jobs userID is
// assigned to (callers without jobs.view_all, e.g. technicians).
func WithAssignedJobsOnly(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, assignedJobsKey, userID)
}

// AssignedJobsUser returns the user whose assignments limit job queries made
// with ctx, if any. ok is false for callers who see every job.
func AssignedJobsUser(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	id, ok := ctx.Value(assignedJobsKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// WithMoneyHidden marks ctx as a caller who must not see prices, costs or
// customer billing rules (callers without invoices.read, e.g. technicians).
func WithMoneyHidden(ctx context.Context) context.Context {
	return context.WithValue(ctx, moneyHiddenKey, true)
}

// MoneyHidden reports whether ctx was marked with WithMoneyHidden.
func MoneyHidden(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(moneyHiddenKey).(bool)
	return v
}
