package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
)

// T256: without object storage (production until R2) PDFs are still
// generated for email attachments and downloads — just not cached or linked.
func TestDocumentPDFService_WithoutStorage(t *testing.T) {
	cfg := pdf.DefaultConfig()
	cfg.Storage.Type = "local"
	cfg.Storage.LocalPath = t.TempDir()
	gen, err := pdf.NewGenerator(context.Background(), cfg, nil)
	require.NoError(t, err)
	svc := NewDocumentPDFService(gen, nil)
	ctx := context.Background()

	now := time.Now()
	inv := &models.Invoice{
		ID: uuid.New(), InvoiceNumber: "INV-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Subtotal: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100),
		Lines: []models.InvoiceLine{{Description: "Visit", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)}},
	}
	res, err := svc.GenerateInvoicePDF(ctx, inv)
	require.NoError(t, err, "generation must not need storage")
	assert.NotEmpty(t, res.Content)

	_, _, err = svc.DownloadInvoicePDF(ctx, inv.ID)
	assert.ErrorIs(t, err, ErrPDFStorageUnavailable, "no cached copy: callers regenerate")
	_, _, err = svc.GetInvoicePDFURL(ctx, inv.ID)
	assert.ErrorIs(t, err, ErrPDFStorageUnavailable, "no link without storage")
}
