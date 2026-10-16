package database

import (
	"strings"
	"testing"
	"time"
)

func TestLatestMigrationVersion(t *testing.T) {
	v, err := LatestMigrationVersion()
	if err != nil {
		t.Fatalf("LatestMigrationVersion: %v", err)
	}
	if v < BaselineVersion {
		t.Fatalf("latest embedded version = %d, want >= %d", v, BaselineVersion)
	}
}

func TestUsesTransactionPooler(t *testing.T) {
	cases := map[string]bool{
		"postgresql://postgres:pw@localhost:5432/servicepro?sslmode=disable":             false,
		"postgresql://postgres.ref:pw@aws-0-us-east-1.pooler.supabase.com:6543/postgres": true,
		"postgresql://postgres.ref:pw@aws-0-us-east-1.pooler.supabase.com:5432/postgres": false,
		"host=db.example.com port=6543 user=postgres dbname=servicepro":                  true,
		"postgres://u:p@[::1]:6543/db":                                                   true,
	}
	for url, want := range cases {
		if got := UsesTransactionPooler(url); got != want {
			t.Errorf("UsesTransactionPooler(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestMigrateURL(t *testing.T) {
	got := migrateURL("postgresql://u:p@localhost:5432/db?sslmode=disable")
	if !strings.HasPrefix(got, "pgx5://") {
		t.Errorf("migrateURL scheme not rewritten: %s", got)
	}
	if !strings.Contains(got, "&default_query_exec_mode=simple_protocol") {
		t.Errorf("migrateURL missing simple protocol: %s", got)
	}
	got = migrateURL("postgres://u:p@localhost:5432/db")
	if !strings.HasPrefix(got, "pgx5://") || !strings.Contains(got, "?default_query_exec_mode=") {
		t.Errorf("migrateURL without query: %s", got)
	}
}

func TestSchemaMonitor_RechecksOnlyWhenUnhealthy(t *testing.T) {
	calls := 0
	states := []SchemaState{
		{Status: SchemaBehind, Applied: 1, Expected: 2},
		{Status: SchemaOK, Applied: 2, Expected: 2},
	}
	sm := &SchemaMonitor{
		recheckInterval: 0, // recheck immediately while unhealthy
		check: func(string) SchemaState {
			s := states[min(calls, len(states)-1)]
			calls++
			return s
		},
	}
	sm.refresh()
	if got := sm.State().Status; got != SchemaOK {
		t.Fatalf("after recheck status = %q, want ok", got)
	}
	// Healthy state is cached: no further checks.
	sm.recheckInterval = time.Nanosecond
	_ = sm.State()
	_ = sm.State()
	if calls != 2 {
		t.Fatalf("check called %d times, want 2", calls)
	}
}

func TestSchemaState_Healthy(t *testing.T) {
	for status, want := range map[string]bool{
		SchemaOK: true, SchemaAhead: true, SchemaBehind: false, SchemaDirty: false, SchemaUnknown: false,
	} {
		if got := (SchemaState{Status: status}).Healthy(); got != want {
			t.Errorf("Healthy(%q) = %v, want %v", status, got, want)
		}
	}
}
