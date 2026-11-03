package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
)

// T188: a consolidated invoice's PDF prints the PO, the service period, a
// heading per site and the approver under extra work.
func TestDocumentPDFService_ConsolidatedInvoice(t *testing.T) {
	cfg := pdf.DefaultConfig()
	cfg.Storage.Type = "local"
	cfg.Storage.LocalPath = t.TempDir()
	cfg.Performance.EnableCompression = false // readable text in the output
	gen, err := pdf.NewGenerator(context.Background(), cfg, nil)
	require.NoError(t, err)
	svc := NewDocumentPDFService(gen, nil)

	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	lofts, river := "Harborview Lofts", "Riverside Commons"
	note := "Approved by Karen Doyle for 450.00"
	line := func(desc string, amount int64, site *string, note *string) models.InvoiceLine {
		return models.InvoiceLine{Description: desc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(amount), SiteName: site, ApprovalNote: note}
	}
	inv := &models.Invoice{
		ID: uuid.New(), InvoiceNumber: "INV-1042", IssueDate: end.AddDate(0, 0, 1), DueDate: end.AddDate(0, 1, 0),
		PONumber: "7731-HV", BillingPeriodStart: &start, BillingPeriodEnd: &end,
		Subtotal: decimal.NewFromInt(4650), TotalAmount: decimal.NewFromInt(4650),
		Lines: []models.InvoiceLine{
			line("Nightly cleaning · 22 visits", 2400, &lofts, nil),
			line("Carpet extraction, lobby · J-2291", 450, &lofts, &note),
			line("Cleaning 3x weekly · 13 visits", 1800, &river, nil),
		},
	}

	data := svc.buildInvoicePDFData(inv)
	assert.Equal(t, "7731-HV", data.PurchaseOrder)
	assert.Equal(t, "Oct 1, 2026 – Oct 31, 2026", data.ServicePeriod)
	require.Len(t, data.LineItems, 3)
	assert.Equal(t, lofts, data.LineItems[1].Site)
	assert.Equal(t, note, data.LineItems[1].Note)

	res, err := svc.GenerateInvoicePDF(context.Background(), inv)
	require.NoError(t, err)
	out := string(res.Content)
	for _, want := range []string{"PO Number:", "7731-HV", "Service Period:", lofts, river, "Approved by Karen Doyle for 450.00"} {
		assert.True(t, strings.Contains(out, want), "PDF should contain %q", want)
	}
	assert.Equal(t, 1, strings.Count(out, "("+lofts+")"), "one heading per site")
}
