package email

import (
	"strings"
	"testing"
)

func TestPayNowButtonHTML(t *testing.T) {
	if got := PayNowButtonHTML(""); got != "" {
		t.Fatalf("empty URL: want no button, got %q", got)
	}
	got := PayNowButtonHTML("https://api.example.test/api/v1/public/invoices/pay/abc")
	if !strings.Contains(got, `href="https://api.example.test/api/v1/public/invoices/pay/abc"`) || !strings.Contains(got, "Pay Now") {
		t.Fatalf("want a Pay Now link, got %q", got)
	}
}
