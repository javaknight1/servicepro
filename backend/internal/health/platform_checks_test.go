package health

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/pkg/database"
)

type fakeSchema struct{ state database.SchemaState }

func (f fakeSchema) State() database.SchemaState { return f.state }

func TestSchemaCheck_Statuses(t *testing.T) {
	cases := map[string]Status{
		database.SchemaOK:      StatusHealthy,
		database.SchemaAhead:   StatusHealthy,
		database.SchemaBehind:  StatusUnhealthy,
		database.SchemaDirty:   StatusUnhealthy,
		database.SchemaUnknown: StatusDegraded,
	}
	for schemaStatus, want := range cases {
		res := SchemaCheck(fakeSchema{database.SchemaState{Status: schemaStatus, Applied: 1, Expected: 1}})(context.Background())
		if res.Status != want {
			t.Errorf("schema %q: status = %q, want %q", schemaStatus, res.Status, want)
		}
		if res.Details["state"] != schemaStatus {
			t.Errorf("schema %q: details.state = %v", schemaStatus, res.Details["state"])
		}
	}
}

func TestReadiness_SchemaBehindIsUnhealthy(t *testing.T) {
	checker := NewChecker("test", "test")
	checker.Register(CheckConfig{
		Name:     "schema",
		Critical: true,
		Check:    SchemaCheck(fakeSchema{database.SchemaState{Status: database.SchemaBehind, Applied: 1, Expected: 2}}),
	})
	checker.Register(CheckConfig{
		Name:  "redis",
		Check: RedisFallbackCheck(RedisModeNotConfigured),
	})

	report := checker.GetReport(context.Background())
	if report.Status != StatusUnhealthy {
		t.Fatalf("report status = %q, want unhealthy", report.Status)
	}
	if got := report.Checks["redis"].Message; got != RedisModeNotConfigured {
		t.Fatalf("redis message = %q, want %q", got, RedisModeNotConfigured)
	}
}

func TestRedisFallbackCheck_NeverUnhealthy(t *testing.T) {
	if s := RedisFallbackCheck(RedisModeNotConfigured)(context.Background()).Status; s != StatusHealthy {
		t.Errorf("not_configured status = %q, want healthy", s)
	}
	if s := RedisFallbackCheck(RedisModeInMemoryFallback)(context.Background()).Status; s != StatusDegraded {
		t.Errorf("in_memory_fallback status = %q, want degraded", s)
	}
}
