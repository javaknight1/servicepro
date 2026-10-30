package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// EmailSuppressionRepository stores addresses that must not be emailed again
// (permanent bounce or spam complaint, T159). The table is platform-wide,
// not tenant-owned, so the tenancy plugin leaves it alone.
type EmailSuppressionRepository struct {
	db *gorm.DB
}

// NewEmailSuppressionRepository creates the repository.
func NewEmailSuppressionRepository(db *gorm.DB) *EmailSuppressionRepository {
	return &EmailSuppressionRepository{db: db}
}

// Suppress records an address; a second report for it is a no-op.
func (r *EmailSuppressionRepository) Suppress(ctx context.Context, email, reason, eventID string) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO email_suppressions (email, reason, event_id) VALUES (?, ?, ?)
		 ON CONFLICT ((lower(email))) DO NOTHING`,
		strings.TrimSpace(email), reason, eventID,
	).Error
}

// IsSuppressed reports whether the address may not be emailed.
func (r *EmailSuppressionRepository) IsSuppressed(ctx context.Context, email string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT count(*) FROM email_suppressions WHERE lower(email) = lower(?)`,
		strings.TrimSpace(email),
	).Scan(&n).Error
	return n > 0, err
}
