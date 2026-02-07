package database

import (
	"strings"
	"testing"
)

func TestParseSQLInfo(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantOp    string
		wantTable string
	}{
		{
			name:      "SELECT with quoted table",
			sql:       `SELECT * FROM "users" WHERE id = $1`,
			wantOp:    "SELECT",
			wantTable: "users",
		},
		{
			name:      "SELECT with unquoted table",
			sql:       `SELECT id, name FROM customers WHERE active = true`,
			wantOp:    "SELECT",
			wantTable: "customers",
		},
		{
			name:      "INSERT with quoted table",
			sql:       `INSERT INTO "jobs" ("id","title") VALUES ($1,$2)`,
			wantOp:    "INSERT",
			wantTable: "jobs",
		},
		{
			name:      "INSERT with unquoted table",
			sql:       `INSERT INTO invoices (id, amount) VALUES ($1, $2)`,
			wantOp:    "INSERT",
			wantTable: "invoices",
		},
		{
			name:      "UPDATE with quoted table",
			sql:       `UPDATE "quotes" SET status = $1 WHERE id = $2`,
			wantOp:    "UPDATE",
			wantTable: "quotes",
		},
		{
			name:      "UPDATE with unquoted table",
			sql:       `UPDATE users SET name = $1 WHERE id = $2`,
			wantOp:    "UPDATE",
			wantTable: "users",
		},
		{
			name:      "DELETE with quoted table",
			sql:       `DELETE FROM "line_items" WHERE invoice_id = $1`,
			wantOp:    "DELETE",
			wantTable: "line_items",
		},
		{
			name:      "DELETE with unquoted table",
			sql:       `DELETE FROM sessions WHERE expired_at < $1`,
			wantOp:    "DELETE",
			wantTable: "sessions",
		},
		{
			name:      "SELECT with subquery",
			sql:       `SELECT * FROM "orders" WHERE customer_id IN (SELECT id FROM customers)`,
			wantOp:    "SELECT",
			wantTable: "orders",
		},
		{
			name:      "empty SQL",
			sql:       "",
			wantOp:    "OTHER",
			wantTable: "unknown",
		},
		{
			name:      "BEGIN transaction",
			sql:       "BEGIN",
			wantOp:    "OTHER",
			wantTable: "unknown",
		},
		{
			name:      "COMMIT transaction",
			sql:       "COMMIT",
			wantOp:    "OTHER",
			wantTable: "unknown",
		},
		{
			name:      "case insensitive select",
			sql:       `select * from "tenants" where id = $1`,
			wantOp:    "SELECT",
			wantTable: "tenants",
		},
		{
			name:      "SELECT with JOIN",
			sql:       `SELECT j.* FROM "jobs" j JOIN "customers" c ON j.customer_id = c.id`,
			wantOp:    "SELECT",
			wantTable: "jobs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSQLInfo(tt.sql)
			if got.Operation != tt.wantOp {
				t.Errorf("parseSQLInfo(%q).Operation = %q, want %q", tt.sql, got.Operation, tt.wantOp)
			}
			if got.Table != tt.wantTable {
				t.Errorf("parseSQLInfo(%q).Table = %q, want %q", tt.sql, got.Table, tt.wantTable)
			}
		})
	}
}

func TestTruncateSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		wantFull bool // if true, output should equal input
		wantSuf  string
	}{
		{
			name:     "short SQL unchanged",
			sql:      `SELECT * FROM users WHERE id = $1`,
			wantFull: true,
		},
		{
			name:     "exactly at limit unchanged",
			sql:      strings.Repeat("x", maxSQLLength),
			wantFull: true,
		},
		{
			name:    "over limit truncated",
			sql:     strings.Repeat("x", maxSQLLength+100),
			wantSuf: "...[truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSQL(tt.sql)
			if tt.wantFull {
				if got != tt.sql {
					t.Errorf("truncateSQL() changed short SQL: got len=%d, want len=%d", len(got), len(tt.sql))
				}
			} else {
				if !strings.HasSuffix(got, tt.wantSuf) {
					t.Errorf("truncateSQL() should end with %q, got %q", tt.wantSuf, got[len(got)-20:])
				}
				expectedLen := maxSQLLength + len("...[truncated]")
				if len(got) != expectedLen {
					t.Errorf("truncateSQL() len = %d, want %d", len(got), expectedLen)
				}
			}
		})
	}
}
