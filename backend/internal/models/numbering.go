package models

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Document number kinds, one counter per tenant each (table tenant_counters,
// migration 000008).
const (
	CounterInvoice = "invoice"
	CounterQuote   = "quote"
	CounterJob     = "job"
)

// Default number prefixes, used when the tenant has not configured its own
// (TenantSettings.InvoicePrefix / QuotePrefix / JobPrefix, T127).
var counterPrefixes = map[string]string{
	CounterInvoice: "INV-",
	CounterQuote:   "QUO-",
	CounterJob:     "JOB-",
}

// ErrNoTenantForNumber is returned when a document is created without a
// tenant (numbers are per tenant).
var ErrNoTenantForNumber = errors.New("cannot number a document without a tenant")

// NextDocumentNumber allocates the next invoice/quote/job number for a
// tenant (T120 / audit 7.C3, 7.C4).
//
// The counter row is incremented with a single upsert, which takes a row
// lock held until the surrounding transaction commits: concurrent creates in
// one tenant get distinct numbers, a rolled-back create releases its number,
// and soft-deleting a document never makes a number reusable (the old
// MAX(number)+1 / COUNT(today)+1 schemes read through the soft-delete scope
// while the unique constraint did not, so one delete wedged every later
// create). Numbers left over from older schemes are skipped.
//
// tx must be the transaction the document is inserted in (GORM hooks run
// inside the create transaction).
func NextDocumentNumber(tx *gorm.DB, tenantID uuid.UUID, kind string) (string, error) {
	if tenantID == uuid.Nil {
		if id, ok := tenancy.TenantID(tx.Statement.Context); ok {
			tenantID = id
		}
	}
	if tenantID == uuid.Nil {
		return "", ErrNoTenantForNumber
	}
	if _, ok := counterPrefixes[kind]; !ok {
		return "", fmt.Errorf("unknown document counter %q", kind)
	}
	table, column := counterTable(kind)

	// A fresh session so the caller's statement (model, clauses) doesn't leak in.
	db := tx.Session(&gorm.Session{NewDB: true})
	prefix := CounterPrefix(TenantSettingsInTx(db, tenantID), kind)
	for attempt := 0; attempt < 1000; attempt++ {
		var n int64
		if err := db.Raw(`INSERT INTO tenant_counters (tenant_id, kind, next_value) VALUES (?, ?, 2)
			ON CONFLICT (tenant_id, kind) DO UPDATE SET next_value = tenant_counters.next_value + 1
			RETURNING next_value - 1`, tenantID, kind).Scan(&n).Error; err != nil {
			return "", fmt.Errorf("allocate %s number: %w", kind, err)
		}
		number := fmt.Sprintf("%s%06d", prefix, n)

		// Skip a number an older scheme already used (deleted rows included).
		var taken int64
		if err := db.Table(table).
			Where("tenant_id = ? AND "+column+" = ?", tenantID, number).
			Count(&taken).Error; err != nil {
			return "", err
		}
		if taken == 0 {
			return number, nil
		}
	}
	return "", fmt.Errorf("allocate %s number: too many numbers already taken", kind)
}

// maxNumberPrefixLen keeps prefix + 6-digit counter well inside the
// varchar(50) number columns.
const maxNumberPrefixLen = 20

// CounterPrefix returns the tenant's configured number prefix for kind, or
// the default (INV-/QUO-/JOB-) when it is unset or unusable (T127). The
// settings endpoint does not validate prefixes, so they are cleaned here:
// surrounding space is trimmed, and a prefix with control characters or
// longer than 20 characters falls back to the default.
func CounterPrefix(settings TenantSettings, kind string) string {
	var configured string
	switch kind {
	case CounterInvoice:
		configured = settings.InvoicePrefix
	case CounterQuote:
		configured = settings.QuotePrefix
	case CounterJob:
		configured = settings.JobPrefix
	}
	configured = strings.TrimSpace(configured)
	if configured == "" || utf8.RuneCountInString(configured) > maxNumberPrefixLen ||
		strings.IndexFunc(configured, unicode.IsControl) >= 0 {
		return counterPrefixes[kind]
	}
	return configured
}

func counterTable(kind string) (table, column string) {
	switch kind {
	case CounterInvoice:
		return "invoices", "invoice_number"
	case CounterQuote:
		return "quotes", "quote_number"
	default:
		return "jobs", "job_number"
	}
}
