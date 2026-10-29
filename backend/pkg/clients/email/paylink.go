package email

import (
	"fmt"
	"html"
)

// PayNowButtonHTML is the "Pay Now" button for the plain invoice email
// templates, or "" when paymentURL is empty — online invoice payment is off
// until Stripe Connect routes money to the tenant (T261).
func PayNowButtonHTML(paymentURL string) string {
	if paymentURL == "" {
		return ""
	}
	return fmt.Sprintf(`<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none;">Pay Now</a></p>`,
		html.EscapeString(paymentURL))
}
