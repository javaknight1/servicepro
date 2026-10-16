package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

// Audit 7.L1: item totals were rounded but the subtotal summed the unrounded
// products, so the subtotal could differ from the items by a cent.
func TestQuoteCalculateTotals_SubtotalIsSumOfRoundedItems(t *testing.T) {
	q := &Quote{TaxRate: decimal.RequireFromString("0.1")}
	for _, qty := range []string{"0.33", "0.33", "0.33"} {
		item := QuoteItem{Quantity: decimal.RequireFromString(qty), UnitPrice: decimal.RequireFromString("0.05")}
		item.CalculateItemTotal() // 0.0165 -> 0.02
		q.Items = append(q.Items, item)
	}
	q.CalculateTotals()

	sum := decimal.Zero
	for _, it := range q.Items {
		sum = sum.Add(it.Total)
	}
	if !q.Subtotal.Equal(sum) {
		t.Fatalf("subtotal %s != sum of item totals %s", q.Subtotal, sum)
	}
	if !q.Total.Equal(q.Subtotal.Add(q.TaxAmount)) {
		t.Fatalf("total %s != subtotal + tax", q.Total)
	}
}
