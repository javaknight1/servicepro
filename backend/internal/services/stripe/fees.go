package stripe

import "github.com/shopspring/decimal"

// Payment methods a customer chooses on the pay link (mockup 17).
const (
	PayMethodBank = "bank" // US bank account (ACH), Checkout type us_bank_account
	PayMethodCard = "card" // cards and the other dynamic methods, ACH excluded
)

// ServicePro's application fees on direct charges (D10). Stripe bills its own
// processing fees to the tenant (fees_collector = stripe).
var (
	CardFeeRate     = decimal.RequireFromString("0.004") // 0.4%
	ACHFeeRate      = decimal.RequireFromString("0.01")  // 1%
	ACHFeeCapAmount = decimal.NewFromInt(10)             // $10, in major units
)

// ApplicationFee is ServicePro's fee for a charge of units (minor units of
// currency) paid by method: 0.4% for cards, 1% capped at $10 for bank. It is
// rounded half up to a whole minor unit, and is 0 (no fee) when it would
// round to nothing or not be below the charge (Stripe requires 0 < fee < amount).
func ApplicationFee(units int64, currency, method string) int64 {
	if units <= 0 {
		return 0
	}
	amount := decimal.NewFromInt(units)
	var fee decimal.Decimal
	switch method {
	case PayMethodBank:
		fee = amount.Mul(ACHFeeRate)
		capUnits := ACHFeeCapAmount.Shift(MinorUnitExponent(currency))
		if fee.GreaterThan(capUnits) {
			fee = capUnits
		}
	default:
		fee = amount.Mul(CardFeeRate)
	}
	f := fee.Round(0).IntPart()
	if f <= 0 || f >= units {
		return 0
	}
	return f
}
