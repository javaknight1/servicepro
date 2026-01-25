package utils

import (
	"testing"
)

func TestSafeOrderDirection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid ASC", "ASC", "ASC"},
		{"valid DESC", "DESC", "DESC"},
		{"lowercase asc", "asc", "ASC"},
		{"lowercase desc", "desc", "DESC"},
		{"mixed case", "Asc", "ASC"},
		{"with spaces", "  DESC  ", "DESC"},
		{"empty string defaults to DESC", "", "DESC"},
		{"invalid value defaults to DESC", "RANDOM", "DESC"},
		// SQL injection attempts
		{"injection attempt 1", "ASC; DROP TABLE users;", "DESC"},
		{"injection attempt 2", "DESC--", "DESC"},
		{"injection attempt 3", "ASC/*comment*/", "DESC"},
		{"injection attempt 4", "1=1", "DESC"},
		{"injection attempt 5", "ASC UNION SELECT", "DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeOrderDirection(tt.input)
			if result != tt.expected {
				t.Errorf("SafeOrderDirection(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSafeOrderColumn(t *testing.T) {
	allowedColumns := []string{"created_at", "updated_at", "name", "email"}
	defaultColumn := "created_at"

	tests := []struct {
		name     string
		column   string
		expected string
	}{
		{"valid column", "name", "name"},
		{"valid column mixed case", "Name", "name"},
		{"valid column uppercase", "EMAIL", "email"},
		{"with spaces", "  name  ", "name"},
		{"empty returns default", "", defaultColumn},
		{"invalid column returns default", "invalid_column", defaultColumn},
		// SQL injection attempts - all should return default
		{"injection: semicolon", "name; DROP TABLE users;", defaultColumn},
		{"injection: comment", "name--", defaultColumn},
		{"injection: union", "name UNION SELECT", defaultColumn},
		{"injection: subquery", "(SELECT password FROM users)", defaultColumn},
		{"injection: OR condition", "1 OR 1=1", defaultColumn},
		{"injection: quotes", "name'", defaultColumn},
		{"injection: double quotes", "name\"", defaultColumn},
		{"injection: null byte", "name\x00", defaultColumn},
		{"injection: newline - trimmed to valid", "name\n", "name"}, // Trimmed to "name" which is valid
		{"injection: backtick", "name`", defaultColumn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeOrderColumn(tt.column, allowedColumns, defaultColumn)
			if result != tt.expected {
				t.Errorf("SafeOrderColumn(%q, %v, %q) = %q, want %q",
					tt.column, allowedColumns, defaultColumn, result, tt.expected)
			}
		})
	}
}

func TestSafeOrderClause(t *testing.T) {
	allowedColumns := []string{"created_at", "updated_at", "name"}
	defaultColumn := "created_at"

	tests := []struct {
		name      string
		column    string
		direction string
		expected  string
	}{
		{"valid column and direction", "name", "ASC", "name ASC"},
		{"valid column default direction", "name", "", "name DESC"},
		{"invalid column uses default", "invalid", "ASC", "created_at ASC"},
		{"empty column uses default", "", "DESC", "created_at DESC"},
		{"both invalid uses defaults", "", "", "created_at DESC"},
		// SQL injection attempts
		{"injection in column", "name; DROP TABLE--", "ASC", "created_at ASC"},
		{"injection in direction", "name", "ASC; DROP TABLE", "name DESC"},
		{"injection in both", "x; DROP TABLE", "y; DROP TABLE", "created_at DESC"},
		{"complex injection", "1=1 UNION SELECT * FROM users--", "ASC", "created_at ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeOrderClause(tt.column, tt.direction, allowedColumns, defaultColumn)
			if result != tt.expected {
				t.Errorf("SafeOrderClause(%q, %q, %v, %q) = %q, want %q",
					tt.column, tt.direction, allowedColumns, defaultColumn, result, tt.expected)
			}
		})
	}
}

func TestIsValidColumn(t *testing.T) {
	allowedColumns := []string{"created_at", "name", "email"}

	tests := []struct {
		name     string
		column   string
		expected bool
	}{
		{"valid column", "name", true},
		{"valid column case insensitive", "NAME", true},
		{"valid column with spaces", "  name  ", true},
		{"invalid column", "invalid", false},
		{"empty string", "", false},
		{"injection attempt", "name; DROP TABLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidColumn(tt.column, allowedColumns)
			if result != tt.expected {
				t.Errorf("IsValidColumn(%q, %v) = %v, want %v",
					tt.column, allowedColumns, result, tt.expected)
			}
		})
	}
}

func TestValidateColumns(t *testing.T) {
	allowedColumns := []string{"created_at", "name", "email"}

	tests := []struct {
		name     string
		columns  []string
		expected bool
	}{
		{"all valid", []string{"name", "email"}, true},
		{"one invalid", []string{"name", "invalid"}, false},
		{"all invalid", []string{"foo", "bar"}, false},
		{"empty slice", []string{}, true},
		{"injection in list", []string{"name", "email; DROP TABLE"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateColumns(tt.columns, allowedColumns)
			if result != tt.expected {
				t.Errorf("ValidateColumns(%v, %v) = %v, want %v",
					tt.columns, allowedColumns, result, tt.expected)
			}
		})
	}
}

func TestFilterValidColumns(t *testing.T) {
	allowedColumns := []string{"created_at", "name", "email"}

	tests := []struct {
		name     string
		columns  []string
		expected []string
	}{
		{"all valid", []string{"name", "email"}, []string{"name", "email"}},
		{"some valid", []string{"name", "invalid", "email"}, []string{"name", "email"}},
		{"none valid", []string{"foo", "bar"}, nil},
		{"empty slice", []string{}, nil},
		{"injection filtered out", []string{"name", "email; DROP TABLE"}, []string{"name"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterValidColumns(tt.columns, allowedColumns)
			if len(result) != len(tt.expected) {
				t.Errorf("FilterValidColumns(%v, %v) = %v, want %v",
					tt.columns, allowedColumns, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("FilterValidColumns(%v, %v)[%d] = %q, want %q",
						tt.columns, allowedColumns, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestPredefinedAllowlists ensures all predefined allowlists have expected columns
func TestPredefinedAllowlists(t *testing.T) {
	// Test CustomerAllowedSortColumns
	if !IsValidColumn("created_at", CustomerAllowedSortColumns) {
		t.Error("CustomerAllowedSortColumns should contain created_at")
	}
	if !IsValidColumn("email", CustomerAllowedSortColumns) {
		t.Error("CustomerAllowedSortColumns should contain email")
	}

	// Test JobAllowedSortColumns
	if !IsValidColumn("created_at", JobAllowedSortColumns) {
		t.Error("JobAllowedSortColumns should contain created_at")
	}
	if !IsValidColumn("status", JobAllowedSortColumns) {
		t.Error("JobAllowedSortColumns should contain status")
	}

	// Test InvoiceAllowedSortColumns
	if !IsValidColumn("created_at", InvoiceAllowedSortColumns) {
		t.Error("InvoiceAllowedSortColumns should contain created_at")
	}
	if !IsValidColumn("invoice_number", InvoiceAllowedSortColumns) {
		t.Error("InvoiceAllowedSortColumns should contain invoice_number")
	}

	// Test PaymentHistoryAllowedSortColumns
	if !IsValidColumn("created_at", PaymentHistoryAllowedSortColumns) {
		t.Error("PaymentHistoryAllowedSortColumns should contain created_at")
	}

	// Test AnalyticsEventAllowedSortColumns
	if !IsValidColumn("timestamp", AnalyticsEventAllowedSortColumns) {
		t.Error("AnalyticsEventAllowedSortColumns should contain timestamp")
	}
}

// TestSQLInjectionPatterns tests common SQL injection patterns
func TestSQLInjectionPatterns(t *testing.T) {
	allowedColumns := []string{"id", "name", "created_at"}
	defaultColumn := "created_at"

	// Common SQL injection patterns that should all be rejected
	injectionPatterns := []string{
		// Basic injection
		"' OR '1'='1",
		"'; DROP TABLE users;--",
		"1; DROP TABLE users",
		"1 OR 1=1",
		"1' OR '1'='1",

		// Comment-based
		"name--",
		"name/*",
		"name#",

		// Union-based
		"name UNION SELECT * FROM users",
		"name UNION ALL SELECT",
		"1 UNION SELECT password FROM users",

		// Subquery-based
		"(SELECT password FROM users)",
		"name AND (SELECT 1 FROM users)",

		// Stacked queries
		"name; DELETE FROM users",
		"name; UPDATE users SET",
		"name; INSERT INTO users",

		// Special characters - note: trailing whitespace is trimmed, so "name\n" becomes "name"
		// We test patterns where special chars are in the middle
		"name\x00injection",
		"name\ninjection",
		"name\rinjection",
		"name\tinjection",

		// Boolean-based blind
		"name' AND '1'='1",
		"name' AND SLEEP(5)",

		// Time-based blind
		"name AND SLEEP(5)",
		"name; WAITFOR DELAY '00:00:05'",

		// Out-of-band
		"name AND LOAD_FILE('/etc/passwd')",

		// Encoding tricks
		"name%27",
		"name%00",
	}

	for _, pattern := range injectionPatterns {
		t.Run("column_injection_"+pattern[:min(20, len(pattern))], func(t *testing.T) {
			result := SafeOrderColumn(pattern, allowedColumns, defaultColumn)
			if result != defaultColumn {
				t.Errorf("SafeOrderColumn should reject injection pattern %q, but returned %q",
					pattern, result)
			}
		})

		t.Run("direction_injection_"+pattern[:min(20, len(pattern))], func(t *testing.T) {
			result := SafeOrderDirection(pattern)
			if result != "DESC" && result != "ASC" {
				t.Errorf("SafeOrderDirection should only return ASC or DESC for %q, but returned %q",
					pattern, result)
			}
		})
	}
}
