package services

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// MarkOverdueInvoices moves every sent or viewed invoice whose due date is
// before today (UTC) to overdue, across all tenants (nothing ever set the
// overdue status, so lists and filters showed late invoices as "sent").
// Partially paid invoices keep their status; their lateness shows through
// is_overdue. It is one conditional UPDATE, so it is idempotent and safe to
// run late, twice or concurrently: an invoice paid or cancelled in the
// meantime no longer matches. Returns the number of invoices marked.
func MarkOverdueInvoices(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	today := now.UTC().Truncate(24 * time.Hour)
	res := db.WithContext(tenancy.WithSystem(ctx)).
		Session(&gorm.Session{SkipHooks: true}).
		Model(&models.Invoice{}).
		Where("status IN ? AND due_date < ?", []models.InvoiceStatus{models.InvoiceStatusSent, models.InvoiceStatusViewed}, today).
		Update("status", models.InvoiceStatusOverdue)
	if res.Error != nil {
		return 0, fmt.Errorf("mark overdue invoices: %w", res.Error)
	}
	return res.RowsAffected, nil
}
