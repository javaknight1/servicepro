package services

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
)

func newTestPDFGenerator(t *testing.T) *pdf.Generator {
	t.Helper()
	cfg := pdf.DefaultConfig()
	cfg.Storage.Type = "local"
	cfg.Storage.LocalPath = t.TempDir()
	gen, err := pdf.NewGenerator(context.Background(), cfg, nil)
	require.NoError(t, err)
	return gen
}

func assertPDF(t *testing.T, content []byte) {
	t.Helper()
	require.NotEmpty(t, content)
	assert.True(t, bytes.HasPrefix(content, []byte("%PDF-")), "not a PDF")
}

// F-8 regression: user content in export rows is table text, never markup.
func TestBuildPDFReport_UserContentIsText(t *testing.T) {
	payload := `<iframe src='file:///proc/self/environ'></iframe>`
	report, ok := buildPDFReport(
		[]models.Customer{{FirstName: `<img src=x onerror=alert(1)>`, LastName: "Doe", Email: "a@b.test"}},
		&models.ExportConfig{ExportType: models.ExportTypeCustomers},
		&models.PDFExportOptions{Title: "<script>x</script>", HeaderText: payload, IncludeSummary: true},
	)
	require.True(t, ok)
	require.NoError(t, report.Validate())
	assert.Equal(t, payload, report.Subtitle)
	assert.Equal(t, "Total Customers: 1", report.Summary)
	assert.Equal(t, "a@b.test", report.Rows[0][2])
}

func TestBuildPDFReport_NonTabularDataHasNoLayout(t *testing.T) {
	_, ok := buildPDFReport(map[string]interface{}{"type": "revenue_report"},
		&models.ExportConfig{ExportType: models.ExportTypeRevenueReport}, models.DefaultPDFOptions())
	assert.False(t, ok)
}

// T260: export PDFs come from gofpdf — no external binary.
func TestExportToPDF_GofpdfTable(t *testing.T) {
	svc := &unifiedExportService{pdfGen: newTestPDFGenerator(t)}
	customers := make([]models.Customer, 120) // forces page breaks
	for i := range customers {
		customers[i] = models.Customer{FirstName: fmt.Sprintf("Zoë %d", i), LastName: "Müller-Longname-That-Overflows-The-Column",
			Email: fmt.Sprintf("c%d@example.test", i), Status: models.CustomerStatusActive}
	}
	for _, orientation := range []string{"portrait", "landscape"} {
		out, err := svc.ExportToPDF(context.Background(), customers, &models.ExportConfig{
			ExportType: models.ExportTypeCustomers, Format: models.ExportFormatPDF,
			PDFOptions: &models.PDFExportOptions{PageSize: "A4", Orientation: orientation, IncludeSummary: true, FooterText: "Acme"},
		})
		require.NoError(t, err, orientation)
		assertPDF(t, out)
	}

	out, err := svc.ExportToPDF(context.Background(), []models.Invoice{{InvoiceNumber: "INV-1", DueDate: time.Now()}},
		&models.ExportConfig{ExportType: models.ExportTypeInvoices, Format: models.ExportFormatPDF})
	require.NoError(t, err)
	assertPDF(t, out)
}

func TestExportValidation_PDFOnlyForListExports(t *testing.T) {
	svc := &unifiedExportService{}
	err := svc.validateConfig(&models.ExportConfig{ExportType: models.ExportTypeRevenueReport, Format: models.ExportFormatPDF})
	appErr, ok := apperrors.As(err)
	require.True(t, ok, "typed validation error")
	assert.Equal(t, 400, appErr.Status)

	assert.NoError(t, svc.validateConfig(&models.ExportConfig{ExportType: models.ExportTypeRevenueReport, Format: models.ExportFormatCSV}))
	assert.NoError(t, svc.validateConfig(&models.ExportConfig{ExportType: models.ExportTypeQuotes, Format: models.ExportFormatPDF}))
}

// Quote, invoice and receipt PDFs all render through the one engine (T260).
func TestDocumentPDFs_QuoteInvoiceReceipt(t *testing.T) {
	svc := NewDocumentPDFService(newTestPDFGenerator(t), nil)
	ctx := context.Background()
	now := time.Now()
	company := "Acme Cleaning"
	customer := &models.Customer{FirstName: "Ana", LastName: "Lopez", CompanyName: &company, Email: "ana@example.test"}

	quote := &models.Quote{
		ID: uuid.New(), QuoteNumber: "Q-1", Customer: customer, ValidUntil: now.Add(72 * time.Hour), CreatedAt: now,
		Subtotal: decimal.NewFromInt(200), Total: decimal.NewFromInt(200),
		Items: []models.QuoteItem{{Description: "Deep clean", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(100), Total: decimal.NewFromInt(200)}},
	}
	q, err := svc.GenerateQuotePDF(ctx, quote)
	require.NoError(t, err)
	assertPDF(t, q.Content)

	inv := &models.Invoice{
		ID: uuid.New(), InvoiceNumber: "INV-1", Customer: customer, IssueDate: now, DueDate: now.Add(24 * time.Hour), PaidDate: &now,
		Subtotal: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), AmountPaid: decimal.NewFromInt(100),
		Lines: []models.InvoiceLine{{Description: "Visit", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)}},
	}
	i, err := svc.GenerateInvoicePDF(ctx, inv)
	require.NoError(t, err)
	assertPDF(t, i.Content)

	r, err := svc.GenerateReceiptPDF(ctx, inv)
	require.NoError(t, err)
	assertPDF(t, r.Content)
}
