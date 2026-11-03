package stripe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// D10: cards 0.4%; bank 1% capped at $10 — "$10 on a $5,000 bank payment".
func TestApplicationFee(t *testing.T) {
	cases := []struct {
		name   string
		units  int64
		method string
		want   int64
	}{
		{"bank $5,000 hits the $10 cap", 500000, PayMethodBank, 1000},
		{"bank $9,882 capped", 988200, PayMethodBank, 1000},
		{"bank $540 is 1%", 54000, PayMethodBank, 540},
		{"bank exactly $1,000 is $10", 100000, PayMethodBank, 1000},
		{"bank rounds half up", 12350, PayMethodBank, 124}, // 123.5 -> 124
		{"card $960 is 0.4%", 96000, PayMethodCard, 384},
		{"card $5,000 is $20 (no cap)", 500000, PayMethodCard, 2000},
		{"card rounds half up", 12375, PayMethodCard, 50}, // 49.5 -> 50
		{"tiny card charge has no fee", 100, PayMethodCard, 0},
		{"zero", 0, PayMethodBank, 0},
		{"unknown method charges the card fee", 96000, "", 384},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ApplicationFee(tc.units, "usd", tc.method))
		})
	}
}

// The $10 cap is in the charge currency's minor units.
func TestApplicationFee_CapUsesCurrencyExponent(t *testing.T) {
	assert.Equal(t, int64(10), ApplicationFee(100000, "jpy", PayMethodBank), "JPY has no minor unit")
	assert.Equal(t, int64(10000), ApplicationFee(10000000, "kwd", PayMethodBank), "KWD has three decimals")
}
