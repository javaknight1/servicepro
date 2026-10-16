package services

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// ErrInvalidLineItem is returned for a line with a bad quantity, price or discount
	ErrInvalidLineItem = errors.New("invalid line item")
	// ErrInvalidDiscount is returned when the invoice discount is negative or exceeds the subtotal
	ErrInvalidDiscount = errors.New("discount must be between zero and the invoice subtotal")
	// ErrInvalidTaxRate is returned for a tax rate outside 0..1
	ErrInvalidTaxRate = errors.New("tax rate must be between 0 and 1 (e.g. 0.0825 for 8.25%)")
)

var hundred = decimal.NewFromInt(100)

// invoiceTotals are an invoice's money columns.
type invoiceTotals struct {
	Subtotal decimal.Decimal // sum of line nets (after line discounts)
	Discount decimal.Decimal // invoice-level discount
	Tax      decimal.Decimal // sum of line taxes
	Total    decimal.Decimal // Subtotal - Discount + Tax
}

// calculateInvoiceTotals is the single server-side money calculation for an
// invoice (T225 / audit 7.H3, 7.M7). It normalises and fills each line's
// DiscountAmount, TaxRate and TaxAmount in place.
//
// Order of operations, everything rounded half away from zero to cents (the
// same rounding Postgres applies to the DECIMAL(12,2) columns):
//
//  1. line gross = quantity x unit price
//  2. line discount = DiscountPercentage% of gross when a percentage is
//     given, else DiscountAmount; line net = gross - line discount
//  3. subtotal = sum of line nets (so invoice lines always add up to it)
//  4. the invoice-level discount is applied BEFORE tax: it is allocated to
//     the lines pro rata to their net (remainder cents to the last line)
//  5. line tax = (line net - its share of the discount) x rate, taxable
//     lines only; tax = sum of line taxes
//  6. total = subtotal - discount + tax
func calculateInvoiceTotals(lines []models.InvoiceLine, discount, taxRate decimal.Decimal) (invoiceTotals, error) {
	if taxRate.IsNegative() || taxRate.GreaterThan(decimal.NewFromInt(1)) {
		return invoiceTotals{}, ErrInvalidTaxRate
	}
	discount = discount.Round(2)
	if discount.IsNegative() {
		return invoiceTotals{}, ErrInvalidDiscount
	}

	nets := make([]decimal.Decimal, len(lines))
	subtotal := decimal.Zero
	for i := range lines {
		l := &lines[i]
		l.Quantity = l.Quantity.Round(2)
		l.UnitPrice = l.UnitPrice.Round(2)
		if l.Quantity.LessThanOrEqual(decimal.Zero) {
			return invoiceTotals{}, fmt.Errorf("%w on line %d - quantity must be greater than 0", ErrInvalidLineItem, i+1)
		}
		if l.UnitPrice.IsNegative() {
			return invoiceTotals{}, fmt.Errorf("%w on line %d - unit price must not be negative", ErrInvalidLineItem, i+1)
		}
		gross := l.Quantity.Mul(l.UnitPrice)
		if l.DiscountPercentage.IsNegative() || l.DiscountPercentage.GreaterThan(hundred) {
			return invoiceTotals{}, fmt.Errorf("%w on line %d - discount percentage must be between 0 and 100", ErrInvalidLineItem, i+1)
		}
		if l.DiscountPercentage.IsPositive() {
			l.DiscountAmount = gross.Mul(l.DiscountPercentage).Div(hundred).Round(2)
		}
		l.DiscountAmount = l.DiscountAmount.Round(2)
		if l.DiscountAmount.IsNegative() || l.DiscountAmount.GreaterThan(gross) {
			return invoiceTotals{}, fmt.Errorf("%w on line %d - discount must be between 0 and the line amount", ErrInvalidLineItem, i+1)
		}
		nets[i] = gross.Sub(l.DiscountAmount).Round(2)
		subtotal = subtotal.Add(nets[i])
	}

	if discount.GreaterThan(subtotal) {
		return invoiceTotals{}, ErrInvalidDiscount
	}

	tax := decimal.Zero
	allocated := decimal.Zero
	for i := range lines {
		l := &lines[i]
		share := decimal.Zero
		if discount.IsPositive() {
			if i == len(lines)-1 {
				share = discount.Sub(allocated)
			} else {
				share = discount.Mul(nets[i]).Div(subtotal).Round(2)
			}
			allocated = allocated.Add(share)
		}
		l.TaxRate = taxRate
		l.TaxAmount = decimal.Zero
		if l.Taxable {
			l.TaxAmount = nets[i].Sub(share).Mul(taxRate).Round(2)
		}
		tax = tax.Add(l.TaxAmount)
	}

	return invoiceTotals{
		Subtotal: subtotal,
		Discount: discount,
		Tax:      tax,
		Total:    subtotal.Sub(discount).Add(tax),
	}, nil
}

// lineNet is a line's amount after its own discount, as stored in the
// generated line_total column.
func lineNet(l models.InvoiceLine) decimal.Decimal {
	return l.Quantity.Mul(l.UnitPrice).Sub(l.DiscountAmount).Round(2)
}

// InvoiceRuleError is a business-rule violation in invoice input (bad dates,
// missing customer, total below the amount paid). Handlers map it to 400.
type InvoiceRuleError struct{ msg string }

func (e *InvoiceRuleError) Error() string { return e.msg }

func invalidInvoice(msg string) error { return &InvoiceRuleError{msg: msg} }
