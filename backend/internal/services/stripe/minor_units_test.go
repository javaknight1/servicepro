package stripe

import (
	"testing"

	"github.com/shopspring/decimal"
)

// Audit 7.L3: amounts were truncated (IntPart) and assumed 2-decimal
// currencies.
func TestToMinorUnits(t *testing.T) {
	cases := []struct {
		amount   string
		currency string
		want     int64
	}{
		{"10.005", "usd", 1001}, // rounds, doesn't truncate to 1000
		{"10.004", "usd", 1000},
		{"19.999", "USD", 2000},
		{"0.1", "eur", 10},
		{"1234.5", "jpy", 1235}, // zero-decimal currency
		{"1.2345", "kwd", 1235}, // three-decimal currency
		{"-2.555", "usd", -256},
	}
	for _, c := range cases {
		got := ToMinorUnits(decimal.RequireFromString(c.amount), c.currency)
		if got != c.want {
			t.Errorf("ToMinorUnits(%s %s) = %d, want %d", c.amount, c.currency, got, c.want)
		}
	}
}

func TestFromMinorUnits(t *testing.T) {
	if got := FromMinorUnits(1001, "usd"); !got.Equal(decimal.RequireFromString("10.01")) {
		t.Errorf("usd: %s", got)
	}
	if got := FromMinorUnits(1235, "jpy"); !got.Equal(decimal.NewFromInt(1235)) {
		t.Errorf("jpy: %s", got)
	}
	if got := FromMinorUnits(1235, "kwd"); !got.Equal(decimal.RequireFromString("1.235")) {
		t.Errorf("kwd: %s", got)
	}
}
