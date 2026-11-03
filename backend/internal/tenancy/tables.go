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
		// Webhooks resolve it with WithSystem by Stripe account ID, then
		// switch to the owning tenant (T169).
		"stripe_connected_accounts": Owned,
		"service_locations":         Owned, // T186, RLS policy in migration 000010
		"job_approvals":             Owned, // T187, RLS policy in migration 000011
		"service_agreements":        Owned, // T167, RLS policy in migration 000012
		"agreement_visits":          Owned, // T167, RLS policy in migration 000012
		"invoice_billed_items":      Owned, // T188, RLS policy in migration 000015
		"consolidated_invoice_runs": Owned, // T188, RLS policy in migration 000015
		"customer_purchase_orders":  Owned, // T187, RLS policy in migration 000018
		"payment_terms":             SharedRead,
		"tax_rates":                 SharedRead,
		"invoice_templates":         SharedRead,
	}
}

// Register installs the tenancy plugin with DefaultTables on db.
func Register(db *gorm.DB) error {
	return db.Use(New(DefaultTables()))
}
