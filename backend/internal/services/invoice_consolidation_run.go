package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// ConsolidatedInvoicesResult is what one monthly-invoicing run did.
type ConsolidatedInvoicesResult struct {
	// Customers with consolidated billing in tenants whose local date is the 1st.
	Customers int `json:"customers" example:"12"`
	// Created drafts.
	Created int `json:"created" example:"9"`
	// Empty: claimed, but no unbilled work for the month.
	Empty int `json:"empty" example:"2"`
	// Skipped: already handled by an earlier or concurrent run.
	Skipped int `json:"skipped" example:"1"`
	Failed  int `json:"failed" example:"0"`
}

// GenerateConsolidatedInvoices creates a draft consolidated invoice for the
// previous month for every customer with consolidated billing, in tenants
// whose local date (tenant settings time zone; UTC when unset) is the 1st
// (T188). Drafts are never sent automatically.
//
// Idempotent and safe to run hourly, late or concurrently: each (customer,
// period) is claimed in consolidated_invoice_runs with ON CONFLICT DO
// NOTHING, and the invoice is created in the same transaction only when the
// claim was won. A month with no unbilled work keeps its claim with no
// invoice; a failed customer rolls back and is retried on the next run that
// day.
func (s *InvoiceService) GenerateConsolidatedInvoices(ctx context.Context, now time.Time) (*ConsolidatedInvoicesResult, error) {
	sys := tenancy.WithSystem(ctx)
	var customers []struct {
		ID       uuid.UUID
		TenantID uuid.UUID
	}
	if err := s.db.WithContext(sys).Model(&models.Customer{}).Select("id, tenant_id").
		Where("consolidated_billing = ?", true).Order("tenant_id, id").Scan(&customers).Error; err != nil {
		return nil, fmt.Errorf("list consolidated-billing customers: %w", err)
	}

	type tenantInfo struct {
		due        bool
		start, end time.Time
		owner      uuid.UUID
		err        error
	}
	tenants := map[uuid.UUID]*tenantInfo{}
	res := &ConsolidatedInvoicesResult{}
	var errs []error
	for _, c := range customers {
		ti, ok := tenants[c.TenantID]
		if !ok {
			ti = &tenantInfo{}
			tenants[c.TenantID] = ti
			loc, _, err := tenantLocation(sys, s.db, c.TenantID)
			if err == nil {
				today := models.CivilDate(now, loc)
				if today.Day() == 1 {
					ti.due = true
					ti.start = today.AddDate(0, -1, 0)
					ti.end = today.AddDate(0, 0, -1)
					var tenant models.Tenant
					err = s.db.WithContext(sys).Select("id", "owner_id").First(&tenant, "id = ?", c.TenantID).Error
					ti.owner = tenant.OwnerID
				}
			}
			ti.err = err
		}
		if ti.err != nil {
			res.Failed++
			errs = append(errs, fmt.Errorf("customer %s: tenant: %w", c.ID, ti.err))
			continue
		}
		if !ti.due {
			continue
		}
		res.Customers++
		outcome, err := s.runConsolidationFor(tenancy.WithTenant(ctx, c.TenantID), c.TenantID, c.ID, ti.start, ti.end, ti.owner)
		switch {
		case err != nil:
			res.Failed++
			errs = append(errs, fmt.Errorf("customer %s: %w", c.ID, err))
		case outcome == runCreated:
			res.Created++
		case outcome == runEmpty:
			res.Empty++
		default:
			res.Skipped++
		}
	}
	if len(errs) > 0 {
		logging.Error(ctx, "[CONSOLIDATED-INVOICES] some customers failed", map[string]any{"failed": res.Failed, "created": res.Created})
		return res, errors.Join(errs...)
	}
	return res, nil
}

type runOutcome int

const (
	runSkipped runOutcome = iota
	runCreated
	runEmpty
)

// runConsolidationFor claims (customer, period) and creates its draft in one
// transaction.
func (s *InvoiceService) runConsolidationFor(ctx context.Context, tenantID, customerID uuid.UUID, start, end time.Time, owner uuid.UUID) (runOutcome, error) {
	outcome := runSkipped
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		runID := uuid.New()
		var claimed []uuid.UUID
		// Raw SQL: tenant_id is set explicitly and checked by RLS.
		if err := tx.Raw(`INSERT INTO consolidated_invoice_runs (id, tenant_id, customer_id, period_start, period_end)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (tenant_id, customer_id, period_start, period_end) DO NOTHING
			RETURNING id`, runID, tenantID, customerID, start, end).Scan(&claimed).Error; err != nil {
			return fmt.Errorf("claim run: %w", err)
		}
		if len(claimed) == 0 {
			return nil // another run handled this customer and month
		}
		inv, err := s.consolidateInTx(ctx, tx, tenantID, &models.ConsolidateInvoiceRequest{
			CustomerID: customerID, PeriodStart: start.Format(models.DateLayout), PeriodEnd: end.Format(models.DateLayout),
		}, owner)
		if errors.Is(err, ErrNothingToInvoice) {
			outcome = runEmpty
			return nil
		}
		if err != nil {
			return err
		}
		outcome = runCreated
		return tx.Model(&models.ConsolidatedInvoiceRun{}).Where("id = ?", runID).Update("invoice_id", inv.ID).Error
	})
	if err != nil {
		return runSkipped, err
	}
	return outcome, nil
}
