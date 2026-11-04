package quickbooks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
)

// QuickBooks field limits.
const (
	maxDisplayName = 500
	maxNamePart    = 100
	maxDocNumber   = 21
	maxRefNum      = 21
	maxItemName    = 100
	maxPrivateNote = 4000
)

// truncate cuts s to n runes.
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:n]))
}

// cleanName removes characters QuickBooks rejects in names (':' separates
// sub-customers; tabs and newlines).
func cleanName(s string) string {
	r := strings.NewReplacer(":", "-", "\t", " ", "\n", " ", "\r", " ")
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}

// money formats a decimal as a JSON number with two places (never float64).
func money(d decimal.Decimal) json.Number {
	return json.Number(d.StringFixed(2))
}

// customerDisplayName is the QuickBooks DisplayName for a customer: the
// company name for businesses, else the person's name.
func customerDisplayName(c *models.Customer) string {
	name := ""
	if c.CompanyName != nil {
		name = cleanName(*c.CompanyName)
	}
	if name == "" {
		name = cleanName(c.FirstName + " " + c.LastName)
	}
	return truncate(name, maxDisplayName)
}

// customerPayload maps a customer to a QuickBooks Customer.
func customerPayload(c *models.Customer) qbo.Customer {
	out := qbo.Customer{
		DisplayName: customerDisplayName(c),
		GivenName:   truncate(cleanName(c.FirstName), maxNamePart),
		FamilyName:  truncate(cleanName(c.LastName), maxNamePart),
	}
	if c.CompanyName != nil {
		out.CompanyName = truncate(cleanName(*c.CompanyName), maxNamePart)
	}
	if strings.TrimSpace(c.Email) != "" {
		out.PrimaryEmailAddr = &qbo.EmailAddr{Address: strings.TrimSpace(c.Email)}
	}
	if strings.TrimSpace(c.PhonePrimary) != "" {
		out.PrimaryPhone = &qbo.Phone{FreeFormNumber: strings.TrimSpace(c.PhonePrimary)}
	}
	out.BillAddr = &qbo.Address{
		Line1: c.BillingAddressStreet, City: c.BillingAddressCity,
		CountrySubDivisionCode: strings.TrimSpace(c.BillingAddressState), PostalCode: c.BillingAddressZip,
	}
	return out
}

// invoiceRefs are the QuickBooks IDs an invoice payload points at.
type invoiceRefs struct {
	CustomerID string
	ItemID     string
	TaxCodeID  string // optional TxnTaxCodeRef (companies without automated sales tax)
}

// invoicePayload maps an invoice and its lines to a QuickBooks Invoice.
//
// Lines carry the line's net amount (after line discounts). Quantity and
// unit price are sent only when they multiply to exactly that amount
// (QuickBooks rejects a line whose Amount isn't Qty × UnitPrice). The
// invoice-level discount becomes a DiscountLineDetail line, and the tax we
// computed is sent as TxnTaxDetail.TotalTax, so the QuickBooks total matches
// ours. US lines are marked TAX/NON by whether they were taxed.
func invoicePayload(inv *models.Invoice, lines []models.InvoiceLine, refs invoiceRefs) qbo.Invoice {
	out := qbo.Invoice{
		DocNumber:   truncate(inv.InvoiceNumber, maxDocNumber),
		TxnDate:     inv.IssueDate.Format("2006-01-02"),
		DueDate:     inv.DueDate.Format("2006-01-02"),
		CustomerRef: qbo.Ref{Value: refs.CustomerID},
	}
	for _, l := range lines {
		gross := l.Quantity.Mul(l.UnitPrice)
		net := gross.Sub(l.DiscountAmount).Round(2)
		detail := &qbo.SalesItemLineDetail{ItemRef: qbo.Ref{Value: refs.ItemID}}
		if l.DiscountAmount.IsZero() && gross.Equal(gross.Round(2)) {
			qty, price := json.Number(l.Quantity.String()), json.Number(l.UnitPrice.StringFixed(2))
			detail.Qty, detail.UnitPrice = &qty, &price
		}
		code := "NON"
		if l.Taxable && l.TaxAmount.IsPositive() {
			code = "TAX"
		}
		detail.TaxCodeRef = &qbo.Ref{Value: code}
		desc := l.Description
		if l.SiteName != nil && *l.SiteName != "" {
			desc = *l.SiteName + " · " + desc
		}
		out.Line = append(out.Line, qbo.InvoiceLine{
			DetailType: "SalesItemLineDetail", Amount: money(net), Description: truncate(desc, maxPrivateNote),
			SalesItemLineDetail: detail,
		})
	}
	if inv.DiscountAmount.IsPositive() {
		out.Line = append(out.Line, qbo.InvoiceLine{
			DetailType: "DiscountLineDetail", Amount: money(inv.DiscountAmount),
			DiscountLineDetail: &qbo.DiscountLineDetail{PercentBased: false},
		})
	}
	tax := &qbo.TxnTaxDetail{TotalTax: money(inv.TaxAmount)}
	if refs.TaxCodeID != "" && inv.TaxAmount.IsPositive() {
		tax.TxnTaxCodeRef = &qbo.Ref{Value: refs.TaxCodeID}
	}
	out.TxnTaxDetail = tax
	note := "ServicePro invoice " + inv.InvoiceNumber
	if strings.TrimSpace(inv.PONumber) != "" {
		note += " · PO " + strings.TrimSpace(inv.PONumber)
	}
	out.PrivateNote = truncate(note, maxPrivateNote)
	return out
}

// paymentPayload maps a payment on an invoice to a QuickBooks Payment
// applied to that invoice.
func paymentPayload(p *models.InvoicePayment, invoiceNumber, customerID, invoiceID, depositAccountID string) qbo.Payment {
	refNum := p.ReferenceNumber
	if strings.HasPrefix(refNum, "pi_") || strings.HasPrefix(refNum, "ch_") {
		refNum = "" // Stripe IDs are longer than QuickBooks allows; kept in the note
	}
	out := qbo.Payment{
		CustomerRef:   qbo.Ref{Value: customerID},
		TotalAmt:      money(p.Amount),
		TxnDate:       p.PaymentDate.Format("2006-01-02"),
		PaymentRefNum: truncate(refNum, maxRefNum),
		Line: []qbo.PaymentLine{{
			Amount:    money(p.Amount),
			LinkedTxn: []qbo.LinkedTxn{{TxnID: invoiceID, TxnType: "Invoice"}},
		}},
	}
	if depositAccountID != "" {
		out.DepositToAccountRef = &qbo.Ref{Value: depositAccountID}
	}
	note := "ServicePro payment on invoice " + invoiceNumber
	if p.PaymentMethod != "" {
		note += " · " + p.PaymentMethod
	}
	if p.ReferenceNumber != "" {
		note += " · ref " + p.ReferenceNumber
	}
	out.PrivateNote = truncate(note, maxPrivateNote)
	return out
}

// itemName is the QuickBooks item ServicePro creates for an income account.
func itemName(accountName string) string {
	return truncate(cleanName("ServicePro - "+accountName), maxItemName)
}

// payloadHash fingerprints a payload so unchanged records aren't re-sent.
func payloadHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
