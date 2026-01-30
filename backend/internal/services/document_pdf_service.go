package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

const (
	// PDF URL expiration time
	PDFURLExpiration = 1 * time.Hour
)

// PDFResult contains the result of PDF generation
type PDFResult struct {
	S3Key    string
	FileName string
	Size     int64
	Content  []byte
}

// DocumentPDFService handles PDF generation and storage for documents
type DocumentPDFService struct {
	pdfGenerator *pdf.Generator
	storage      storage.Client
}

// NewDocumentPDFService creates a new document PDF service
func NewDocumentPDFService(pdfGenerator *pdf.Generator, storageClient storage.Client) *DocumentPDFService {
	return &DocumentPDFService{
		pdfGenerator: pdfGenerator,
		storage:      storageClient,
	}
}

// GenerateQuotePDF generates a PDF for a quote and stores it in S3
func (s *DocumentPDFService) GenerateQuotePDF(ctx context.Context, quote *models.Quote) (*PDFResult, error) {
	if quote == nil {
		return nil, fmt.Errorf("quote is required")
	}

	// Build PDF data from quote
	data := s.buildQuotePDFData(quote)

	// Generate filename
	filename := fmt.Sprintf("Quote-%s.pdf", quote.QuoteNumber)

	// Generate PDF
	result, err := s.pdfGenerator.GenerateQuote(ctx, data, &pdf.GenerationOptions{
		Filename:         filename,
		ReturnContent:    true,
		SkipLocalStorage: true,
		SkipS3Upload:     true, // We'll handle S3 upload ourselves with custom path
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate quote PDF: %w", err)
	}

	// Upload to S3 with document-specific path
	s3Key := s.getQuoteS3Key(quote.ID)
	err = s.uploadToStorage(ctx, result.Content, s3Key, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload quote PDF: %w", err)
	}

	log.Printf("[DOCUMENT-PDF] Generated quote PDF: %s (size: %d bytes)", s3Key, result.Size)

	return &PDFResult{
		S3Key:    s3Key,
		FileName: filename,
		Size:     result.Size,
		Content:  result.Content,
	}, nil
}

// GenerateInvoicePDF generates a PDF for an invoice and stores it in S3
func (s *DocumentPDFService) GenerateInvoicePDF(ctx context.Context, invoice *models.Invoice) (*PDFResult, error) {
	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}

	// Build PDF data from invoice
	data := s.buildInvoicePDFData(invoice)

	// Generate filename
	filename := fmt.Sprintf("Invoice-%s.pdf", invoice.InvoiceNumber)

	// Generate PDF
	result, err := s.pdfGenerator.GenerateInvoice(ctx, data, &pdf.GenerationOptions{
		Filename:         filename,
		ReturnContent:    true,
		SkipLocalStorage: true,
		SkipS3Upload:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice PDF: %w", err)
	}

	// Upload to S3 with document-specific path
	s3Key := s.getInvoiceS3Key(invoice.ID)
	err = s.uploadToStorage(ctx, result.Content, s3Key, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload invoice PDF: %w", err)
	}

	log.Printf("[DOCUMENT-PDF] Generated invoice PDF: %s (size: %d bytes)", s3Key, result.Size)

	return &PDFResult{
		S3Key:    s3Key,
		FileName: filename,
		Size:     result.Size,
		Content:  result.Content,
	}, nil
}

// GenerateReceiptPDF generates a receipt PDF for a paid invoice and stores it in S3
func (s *DocumentPDFService) GenerateReceiptPDF(ctx context.Context, invoice *models.Invoice) (*PDFResult, error) {
	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}

	// Build PDF data from invoice for receipt
	data := s.buildReceiptPDFData(invoice)

	// Generate filename
	filename := fmt.Sprintf("Receipt-%s.pdf", invoice.InvoiceNumber)

	// Generate PDF using the receipt template
	result, err := s.pdfGenerator.Generate(ctx, data, &pdf.GenerationOptions{
		Filename:         filename,
		ReturnContent:    true,
		SkipLocalStorage: true,
		SkipS3Upload:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate receipt PDF: %w", err)
	}

	// Upload to S3 with document-specific path
	s3Key := s.getReceiptS3Key(invoice.ID)
	err = s.uploadToStorage(ctx, result.Content, s3Key, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload receipt PDF: %w", err)
	}

	log.Printf("[DOCUMENT-PDF] Generated receipt PDF: %s (size: %d bytes)", s3Key, result.Size)

	return &PDFResult{
		S3Key:    s3Key,
		FileName: filename,
		Size:     result.Size,
		Content:  result.Content,
	}, nil
}

// GetQuotePDFURL returns a presigned URL for downloading a quote PDF
func (s *DocumentPDFService) GetQuotePDFURL(ctx context.Context, quoteID uuid.UUID) (string, time.Time, error) {
	s3Key := s.getQuoteS3Key(quoteID)
	return s.getPresignedURL(ctx, s3Key)
}

// GetInvoicePDFURL returns a presigned URL for downloading an invoice PDF
func (s *DocumentPDFService) GetInvoicePDFURL(ctx context.Context, invoiceID uuid.UUID) (string, time.Time, error) {
	s3Key := s.getInvoiceS3Key(invoiceID)
	return s.getPresignedURL(ctx, s3Key)
}

// GetReceiptPDFURL returns a presigned URL for downloading a receipt PDF
func (s *DocumentPDFService) GetReceiptPDFURL(ctx context.Context, invoiceID uuid.UUID) (string, time.Time, error) {
	s3Key := s.getReceiptS3Key(invoiceID)
	return s.getPresignedURL(ctx, s3Key)
}

// DownloadQuotePDF downloads the quote PDF content directly
func (s *DocumentPDFService) DownloadQuotePDF(ctx context.Context, quoteID uuid.UUID) ([]byte, string, error) {
	s3Key := s.getQuoteS3Key(quoteID)
	filename := fmt.Sprintf("Quote-%s.pdf", quoteID.String())
	content, err := s.storage.DownloadBytes(ctx, s3Key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download quote PDF: %w", err)
	}
	return content, filename, nil
}

// DownloadInvoicePDF downloads the invoice PDF content directly
func (s *DocumentPDFService) DownloadInvoicePDF(ctx context.Context, invoiceID uuid.UUID) ([]byte, string, error) {
	s3Key := s.getInvoiceS3Key(invoiceID)
	filename := fmt.Sprintf("Invoice-%s.pdf", invoiceID.String())
	content, err := s.storage.DownloadBytes(ctx, s3Key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download invoice PDF: %w", err)
	}
	return content, filename, nil
}

// DownloadReceiptPDF downloads the receipt PDF content directly
func (s *DocumentPDFService) DownloadReceiptPDF(ctx context.Context, invoiceID uuid.UUID) ([]byte, string, error) {
	s3Key := s.getReceiptS3Key(invoiceID)
	filename := fmt.Sprintf("Receipt-%s.pdf", invoiceID.String())
	content, err := s.storage.DownloadBytes(ctx, s3Key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download receipt PDF: %w", err)
	}
	return content, filename, nil
}

// S3 key helpers
func (s *DocumentPDFService) getQuoteS3Key(quoteID uuid.UUID) string {
	return fmt.Sprintf("quotes/%s.pdf", quoteID.String())
}

func (s *DocumentPDFService) getInvoiceS3Key(invoiceID uuid.UUID) string {
	return fmt.Sprintf("invoices/%s.pdf", invoiceID.String())
}

func (s *DocumentPDFService) getReceiptS3Key(invoiceID uuid.UUID) string {
	return fmt.Sprintf("receipts/%s_receipt.pdf", invoiceID.String())
}

// uploadToStorage uploads PDF content to storage
func (s *DocumentPDFService) uploadToStorage(ctx context.Context, content []byte, key, filename string) error {
	input := storage.NewUploadInput(key).
		WithContentType("application/pdf").
		WithSize(int64(len(content))).
		WithMetadata("filename", filename)

	_, err := s.storage.UploadBytes(ctx, content, input)
	return err
}

// getPresignedURL generates a presigned URL for downloading
func (s *DocumentPDFService) getPresignedURL(ctx context.Context, key string) (string, time.Time, error) {
	// Check if file exists
	exists, err := s.storage.Exists(ctx, key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to check if PDF exists: %w", err)
	}
	if !exists {
		return "", time.Time{}, fmt.Errorf("PDF not found: %s", key)
	}

	url, err := s.storage.GetPresignedURL(ctx, key, PDFURLExpiration)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := time.Now().Add(PDFURLExpiration)
	return url, expiresAt, nil
}

// buildQuotePDFData converts a Quote model to PDF QuoteData
func (s *DocumentPDFService) buildQuotePDFData(quote *models.Quote) *pdf.QuoteData {
	var notes string
	if quote.Notes != nil {
		notes = *quote.Notes
	}

	data := &pdf.QuoteData{
		BaseDocumentData: pdf.BaseDocumentData{
			DocumentNumber: quote.QuoteNumber,
			DocumentDate:   quote.CreatedAt,
			DueDate:        &quote.ValidUntil,
			Notes:          notes,
		},
		QuoteNumber:    quote.QuoteNumber,
		ExpirationDate: &quote.ValidUntil,
		Status:         string(quote.Status),
		TaxRate:        quote.TaxRate.InexactFloat64() * 100, // Convert to percentage
	}

	// Set customer info (Quote uses Customer model)
	if quote.Customer != nil {
		customerName := strings.TrimSpace(quote.Customer.FirstName + " " + quote.Customer.LastName)
		if quote.Customer.CompanyName != nil && *quote.Customer.CompanyName != "" {
			customerName = *quote.Customer.CompanyName
		}
		if customerName == "" && quote.Customer.Email != "" {
			customerName = quote.Customer.Email
		}
		if customerName == "" {
			customerName = "Customer"
		}
		data.CustomerName = customerName
		data.CustomerEmail = quote.Customer.Email
		data.CustomerPhone = quote.Customer.PhonePrimary
		data.CustomerAddress = formatCustomerAddress(quote.Customer)
	} else {
		// No customer associated - use placeholder
		data.CustomerName = "Customer"
	}

	// Build line items and calculate totals
	var subtotal float64
	for _, item := range quote.Items {
		lineItem := pdf.QuoteLineItem{
			Description: item.Description,
			Quantity:    item.Quantity.InexactFloat64(),
			UnitPrice:   item.UnitPrice.InexactFloat64(),
			Amount:      item.Total.InexactFloat64(),
		}
		data.LineItems = append(data.LineItems, lineItem)
		subtotal += item.Total.InexactFloat64()
	}

	data.Subtotal = subtotal
	data.TaxAmount = quote.TaxAmount.InexactFloat64()
	data.Total = quote.Total.InexactFloat64()

	return data
}

// buildInvoicePDFData converts an Invoice model to PDF InvoiceData
func (s *DocumentPDFService) buildInvoicePDFData(invoice *models.Invoice) *pdf.InvoiceData {
	data := &pdf.InvoiceData{
		BaseDocumentData: pdf.BaseDocumentData{
			DocumentNumber: invoice.InvoiceNumber,
			DocumentDate:   invoice.IssueDate,
			DueDate:        &invoice.DueDate,
			Notes:          invoice.Notes,
		},
		InvoiceNumber: invoice.InvoiceNumber,
		Status:        string(invoice.Status),
	}

	// Set customer info (Invoice uses Customer model)
	if invoice.Customer != nil {
		customerName := strings.TrimSpace(invoice.Customer.FirstName + " " + invoice.Customer.LastName)
		if invoice.Customer.CompanyName != nil && *invoice.Customer.CompanyName != "" {
			customerName = *invoice.Customer.CompanyName
		}
		if customerName == "" && invoice.Customer.Email != "" {
			customerName = invoice.Customer.Email
		}
		if customerName == "" {
			customerName = "Customer"
		}
		data.CustomerName = customerName
		data.CustomerEmail = invoice.Customer.Email
		data.CustomerPhone = invoice.Customer.PhonePrimary

		// Build address string
		data.CustomerAddress = formatCustomerAddress(invoice.Customer)
	} else {
		data.CustomerName = "Customer"
	}

	// Build line items
	for _, line := range invoice.Lines {
		lineItem := pdf.InvoiceLineItem{
			Description: line.Description,
			Quantity:    line.Quantity.InexactFloat64(),
			UnitPrice:   line.UnitPrice.InexactFloat64(),
			Amount:      line.Quantity.Mul(line.UnitPrice).InexactFloat64(),
		}
		data.LineItems = append(data.LineItems, lineItem)
	}

	data.Subtotal = invoice.Subtotal.InexactFloat64()
	data.TaxAmount = invoice.TaxAmount.InexactFloat64()
	data.Total = invoice.TotalAmount.InexactFloat64()
	data.AmountPaid = invoice.AmountPaid.InexactFloat64()
	data.AmountDue = invoice.AmountDue.InexactFloat64()

	return data
}

// buildReceiptPDFData converts an Invoice model to PDF ReceiptData
func (s *DocumentPDFService) buildReceiptPDFData(invoice *models.Invoice) *pdf.ReceiptData {
	paymentDate := time.Now()
	if invoice.PaidDate != nil {
		paymentDate = *invoice.PaidDate
	}

	data := &pdf.ReceiptData{
		BaseDocumentData: pdf.BaseDocumentData{
			DocumentNumber: fmt.Sprintf("RCP-%s", invoice.InvoiceNumber),
			DocumentDate:   paymentDate,
		},
		ReceiptNumber: fmt.Sprintf("RCP-%s", invoice.InvoiceNumber),
		InvoiceNumber: invoice.InvoiceNumber,
		PaymentDate:   paymentDate,
	}

	// Set payment method from last payment if available
	if len(invoice.Payments) > 0 {
		lastPayment := invoice.Payments[len(invoice.Payments)-1]
		data.PaymentMethod = lastPayment.PaymentMethod
		data.PaymentReference = lastPayment.ReferenceNumber
	}

	// Set customer info (Invoice uses Customer model)
	if invoice.Customer != nil {
		customerName := strings.TrimSpace(invoice.Customer.FirstName + " " + invoice.Customer.LastName)
		if invoice.Customer.CompanyName != nil && *invoice.Customer.CompanyName != "" {
			customerName = *invoice.Customer.CompanyName
		}
		if customerName == "" && invoice.Customer.Email != "" {
			customerName = invoice.Customer.Email
		}
		if customerName == "" {
			customerName = "Customer"
		}
		data.CustomerName = customerName
		data.CustomerEmail = invoice.Customer.Email
		data.CustomerPhone = invoice.Customer.PhonePrimary

		// Build address string
		data.CustomerAddress = formatCustomerAddress(invoice.Customer)
	} else {
		data.CustomerName = "Customer"
	}

	// Build line items
	for _, line := range invoice.Lines {
		lineItem := pdf.InvoiceLineItem{
			Description: line.Description,
			Quantity:    line.Quantity.InexactFloat64(),
			UnitPrice:   line.UnitPrice.InexactFloat64(),
			Amount:      line.Quantity.Mul(line.UnitPrice).InexactFloat64(),
		}
		data.LineItems = append(data.LineItems, lineItem)
	}

	data.Subtotal = invoice.Subtotal.InexactFloat64()
	data.TaxAmount = invoice.TaxAmount.InexactFloat64()
	data.Total = invoice.TotalAmount.InexactFloat64()
	data.AmountPaid = invoice.AmountPaid.InexactFloat64()

	return data
}

// formatCustomerAddress formats a customer's billing address as a string
func formatCustomerAddress(customer *models.Customer) string {
	if customer == nil {
		return ""
	}

	var parts []string

	if customer.BillingAddressStreet != "" {
		parts = append(parts, customer.BillingAddressStreet)
	}

	cityStateZip := ""
	if customer.BillingAddressCity != "" {
		cityStateZip = customer.BillingAddressCity
	}
	if customer.BillingAddressState != "" {
		if cityStateZip != "" {
			cityStateZip += ", "
		}
		cityStateZip += customer.BillingAddressState
	}
	if customer.BillingAddressZip != "" {
		if cityStateZip != "" {
			cityStateZip += " "
		}
		cityStateZip += customer.BillingAddressZip
	}

	if cityStateZip != "" {
		parts = append(parts, cityStateZip)
	}

	return strings.Join(parts, "\n")
}
