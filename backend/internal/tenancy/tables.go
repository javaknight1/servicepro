package tenancy

import "gorm.io/gorm"

// DefaultTables lists every tenant-owned table the plugin scopes.
//
// Child tables (invoice_lines, invoice_payments, quote_items, job_notes, ...)
// have no tenant_id: they are only reachable through their tenant-scoped
// parent, and their repositories must look the parent up first.
//
// Tables that are already explicitly scoped and are also written by Stripe
// webhooks (tenant_subscriptions, payment_methods, billing_events) are not
// registered.
func DefaultTables() map[string]Mode {
	return map[string]Mode{
		"customers":             Owned,
		"jobs":                  Owned,
		"quotes":                Owned,
		"invoices":              Owned,
		"payments":              Owned,
		"schedules":             Owned,
		"recurring_schedules":   Owned,
		"export_jobs":           Owned,
		"import_jobs":           Owned,
		"payment_reminders":     Owned,
		"revenue_transactions":  Owned,
		"revenue_summary_cache": Owned,
		"payment_terms":         SharedRead,
		"tax_rates":             SharedRead,
		"invoice_templates":     SharedRead,
	}
}

// Register installs the tenancy plugin with DefaultTables on db.
func Register(db *gorm.DB) error {
	return db.Use(New(DefaultTables()))
}
