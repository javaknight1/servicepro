package services

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// ExpireQuotes moves every sent or viewed quote whose valid_until has passed
// to expired, across all tenants (T224 / audit 7.M4: nothing ever expired
// quotes). It is one conditional UPDATE, so it is idempotent and safe to run
// late, twice or concurrently: a quote accepted or declined in the meantime
// no longer matches. Returns the number of quotes expired.
func ExpireQuotes(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	res := db.WithContext(tenancy.WithSystem(ctx)).
		Session(&gorm.Session{SkipHooks: true}). // Quote.BeforeUpdate validates the (empty) model
		Model(&models.Quote{}).
		Where("status IN ? AND valid_until < ?", []models.QuoteStatus{models.QuoteStatusSent, models.QuoteStatusViewed}, now).
		Update("status", models.QuoteStatusExpired)
	if res.Error != nil {
		return 0, fmt.Errorf("expire quotes: %w", res.Error)
	}
	return res.RowsAffected, nil
}
