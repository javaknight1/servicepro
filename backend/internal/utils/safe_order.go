package utils

import (
	"slices"
	"strings"
)

// SafeOrderDirection validates and returns a safe ORDER BY direction.
// Only "ASC" and "DESC" are allowed. Defaults to "DESC" for invalid input.
func SafeOrderDirection(direction string) string {
	upper := strings.ToUpper(strings.TrimSpace(direction))
	if upper == "ASC" || upper == "DESC" {
		return upper
	}
	return "DESC"
}

// SafeOrderColumn validates a column name against an allowlist.
// Returns the column if it's in the allowlist, otherwise returns the defaultColumn.
// This prevents SQL injection by ensuring only known column names are used.
func SafeOrderColumn(column string, allowedColumns []string, defaultColumn string) string {
	trimmed := strings.TrimSpace(column)
	if trimmed == "" {
		return defaultColumn
	}

	// Check if column is in the allowlist (case-insensitive)
	lower := strings.ToLower(trimmed)
	for _, allowed := range allowedColumns {
		if strings.ToLower(allowed) == lower {
			return allowed // Return the canonical form from allowlist
		}
	}

	return defaultColumn
}

// SafeOrderClause builds a safe ORDER BY clause using validated column and direction.
// The column is validated against the allowlist, and direction is restricted to ASC/DESC.
func SafeOrderClause(column string, direction string, allowedColumns []string, defaultColumn string) string {
	safeColumn := SafeOrderColumn(column, allowedColumns, defaultColumn)
	safeDirection := SafeOrderDirection(direction)
	return safeColumn + " " + safeDirection
}

// Common column allowlists for different entities
var (
	// CustomerAllowedSortColumns contains valid sort columns for customers
	CustomerAllowedSortColumns = []string{
		"created_at",
		"updated_at",
		"first_name",
		"last_name",
		"email",
		"company_name",
		"status",
		"customer_type",
		"billing_address_city",
		"billing_address_state",
	}

	// JobAllowedSortColumns contains valid sort columns for jobs
	JobAllowedSortColumns = []string{
		"created_at",
		"updated_at",
		"job_number",
		"status",
		"priority",
		"job_type",
		"scheduled_start_at",
		"scheduled_end_at",
		"actual_start_at",
		"actual_end_at",
		"estimated_cost",
		"actual_cost",
	}

	// InvoiceAllowedSortColumns contains valid sort columns for invoices
	InvoiceAllowedSortColumns = []string{
		"created_at",
		"updated_at",
		"invoice_number",
		"status",
		"issue_date",
		"due_date",
		"sent_date",
		"total_amount",
		"amount_paid",
		"amount_due",
	}

	// PaymentHistoryAllowedSortColumns contains valid sort columns for payment history
	PaymentHistoryAllowedSortColumns = []string{
		"created_at",
		"updated_at",
		"from_status",
		"to_status",
	}

	// AnalyticsEventAllowedSortColumns contains valid sort columns for analytics events
	AnalyticsEventAllowedSortColumns = []string{
		"timestamp",
		"name",
		"category",
		"created_at",
	}
)

// IsValidColumn checks if a column name is in the allowlist.
// Useful for custom validation logic.
func IsValidColumn(column string, allowedColumns []string) bool {
	lower := strings.ToLower(strings.TrimSpace(column))
	for _, allowed := range allowedColumns {
		if strings.ToLower(allowed) == lower {
			return true
		}
	}
	return false
}

// ValidateColumns checks if all columns in a list are valid.
// Returns true if all columns are valid, false otherwise.
func ValidateColumns(columns []string, allowedColumns []string) bool {
	for _, col := range columns {
		if !IsValidColumn(col, allowedColumns) {
			return false
		}
	}
	return true
}

// FilterValidColumns returns only the columns that are in the allowlist.
func FilterValidColumns(columns []string, allowedColumns []string) []string {
	var valid []string
	for _, col := range columns {
		if IsValidColumn(col, allowedColumns) {
			valid = append(valid, col)
		}
	}
	return slices.Clip(valid)
}
