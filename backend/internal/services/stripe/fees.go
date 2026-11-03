package stripe

import "github.com/shopspring/decimal"

// Payment methods a customer chooses on the pay link (mockup 17).
const (
	PayMethodBank = "bank" // US bank account (ACH), Checkout type us_bank_account
	PayMethodCard = "card" // cards and the other dynamic methods, ACH excluded
)

// PlatformFees is ServicePro's application fee schedule on direct charges.
// Stripe bills its own processing fees to the tenant (fees_collector = stripe).
type PlatformFees struct {
	CardBps int64           // basis points on card charges
	BankBps int64           // basis points on bank (ACH) charges
	BankCap decimal.Decimal // cap per bank charge, in major units of the charge currency
}

// CardRate is the card fee as a fraction (40 bps = 0.004).
func (f PlatformFees) CardRate() decimal.Decimal { return decimal.New(f.CardBps, -4) }

// BankRate is the bank fee as a fraction (100 bps = 0.01).
func (f PlatformFees) BankRate() decimal.Decimal { return decimal.New(f.BankBps, -4) }

// DefaultPlatformFees is D10: cards 0.4%, ACH 1% capped at $10. Legacy and
// unknown tiers pay these.
var DefaultPlatformFees = PlatformFees{CardBps: 40, BankBps: 100, BankCap: decimal.NewFromInt(10)}

// feesByTier maps each plan to its fees (D15: the D10 fees on every plan for
// now; mapped per plan so they can differ later).
var feesByTier = map[string]PlatformFees{
	"solo": DefaultPlatformFees,
	"team": DefaultPlatformFees,
	"pro":  DefaultPlatformFees,
}

// PlatformFeesForTier returns the fee schedule of a tier (membership_tiers.name).
func PlatformFeesForTier(tier string) PlatformFees {
	if f, ok := feesByTier[tier]; ok {
		return f
	}
	return DefaultPlatformFees
}

// D10 fee values, kept for callers that show the default schedule.
var (
	CardFeeRate     = DefaultPlatformFees.CardRate()
	ACHFeeRate      = DefaultPlatformFees.BankRate()
	ACHFeeCapAmount = DefaultPlatformFees.BankCap
)

// ApplicationFee is ServicePro's default (D10) fee for a charge of units
// (minor units of currency) paid by method.
func ApplicationFee(units int64, currency, method string) int64 {
	return DefaultPlatformFees.ApplicationFee(units, currency, method)
}

// ApplicationFee is the fee for a charge of units (minor units of currency)
// paid by method: the card rate for cards, the bank rate capped at BankCap
// for bank. It is rounded half up to a whole minor unit, and is 0 (no fee)
// when it would round to nothing or not be below the charge (Stripe requires
// 0 < fee < amount).
func (f PlatformFees) ApplicationFee(units int64, currency, method string) int64 {
	if units <= 0 {
		return 0
	}
	amount := decimal.NewFromInt(units)
	var fee decimal.Decimal
	switch method {
	case PayMethodBank:
		fee = amount.Mul(f.BankRate())
		capUnits := f.BankCap.Shift(MinorUnitExponent(currency))
		if fee.GreaterThan(capUnits) {
			fee = capUnits
		}
	default:
		fee = amount.Mul(f.CardRate())
	}
	n := fee.Round(0).IntPart()
	if n <= 0 || n >= units {
		return 0
	}
	return n
}
