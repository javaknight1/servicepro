package pdf

import (
	"fmt"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// =============================================================================
// Template Types
// =============================================================================

// TemplateType represents the type of document template
type TemplateType string

const (
	TemplateTypeInvoice       TemplateType = "invoice"
	TemplateTypeQuote         TemplateType = "quote"
	TemplateTypeReceipt       TemplateType = "receipt"
	TemplateTypeWorkOrder     TemplateType = "work_order"
	TemplateTypeServiceReport TemplateType = "service_report"
	TemplateTypeContract      TemplateType = "contract"
	TemplateTypeStatement     TemplateType = "statement"
	TemplateTypeCustom        TemplateType = "custom"
)

// =============================================================================
// Template Data Structures
// =============================================================================

// TemplateData is the base interface for all template data
type TemplateData interface {
	GetType() TemplateType
	Validate() error
}

// BaseDocumentData contains common fields for all documents
type BaseDocumentData struct {
	// Document metadata
	DocumentNumber string     `json:"document_number"`
	DocumentDate   time.Time  `json:"document_date"`
	DueDate        *time.Time `json:"due_date,omitempty"`

	// Company info (overrides branding config)
	CompanyName    string `json:"company_name,omitempty"`
	CompanyAddress string `json:"company_address,omitempty"`
	CompanyPhone   string `json:"company_phone,omitempty"`
	CompanyEmail   string `json:"company_email,omitempty"`
	CompanyWebsite string `json:"company_website,omitempty"`
	LogoPath       string `json:"logo_path,omitempty"`

	// Customer info
	CustomerName    string `json:"customer_name"`
	CustomerAddress string `json:"customer_address,omitempty"`
	CustomerEmail   string `json:"customer_email,omitempty"`
	CustomerPhone   string `json:"customer_phone,omitempty"`

	// Notes
	Notes      string `json:"notes,omitempty"`
	Terms      string `json:"terms,omitempty"`
	FooterText string `json:"footer_text,omitempty"`
}

// InvoiceData contains data for invoice generation
type InvoiceData struct {
	BaseDocumentData

	// Invoice specific fields
	InvoiceNumber string `json:"invoice_number"`
	PurchaseOrder string `json:"purchase_order,omitempty"`
	Status        string `json:"status,omitempty"`

	// Line items
	LineItems []InvoiceLineItem `json:"line_items"`

	// Totals
	Subtotal   float64 `json:"subtotal"`
	TaxRate    float64 `json:"tax_rate"`
	TaxAmount  float64 `json:"tax_amount"`
	Discount   float64 `json:"discount,omitempty"`
	Total      float64 `json:"total"`
	AmountPaid float64 `json:"amount_paid,omitempty"`
	AmountDue  float64 `json:"amount_due"`

	// Payment info
	PaymentTerms        string `json:"payment_terms,omitempty"`
	PaymentMethod       string `json:"payment_method,omitempty"`
	PaymentInstructions string `json:"payment_instructions,omitempty"`
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
	TaxRate     float64 `json:"tax_rate,omitempty"`
	Discount    float64 `json:"discount,omitempty"`
}

// GetType returns the template type for invoice
func (d *InvoiceData) GetType() TemplateType {
	return TemplateTypeInvoice
}

// Validate validates invoice data
func (d *InvoiceData) Validate() error {
	if d.InvoiceNumber == "" && d.DocumentNumber == "" {
		return fmt.Errorf("%w: invoice number required", ErrInvalidContent)
	}
	if d.CustomerName == "" {
		return fmt.Errorf("%w: customer name required", ErrInvalidContent)
	}
	return nil
}

// QuoteData contains data for quote generation
type QuoteData struct {
	BaseDocumentData

	// Quote specific fields
	QuoteNumber    string     `json:"quote_number"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
	Status         string     `json:"status,omitempty"`

	// Line items
	LineItems []QuoteLineItem `json:"line_items"`

	// Totals
	Subtotal  float64 `json:"subtotal"`
	TaxRate   float64 `json:"tax_rate"`
	TaxAmount float64 `json:"tax_amount"`
	Discount  float64 `json:"discount,omitempty"`
	Total     float64 `json:"total"`

	// Quote terms
	ValidityPeriod  string `json:"validity_period,omitempty"`
	AcceptanceTerms string `json:"acceptance_terms,omitempty"`
}

// ReceiptData contains data for receipt generation
type ReceiptData struct {
	BaseDocumentData

	// Receipt specific fields
	ReceiptNumber    string    `json:"receipt_number"`
	InvoiceNumber    string    `json:"invoice_number"`
	PaymentDate      time.Time `json:"payment_date"`
	PaymentMethod    string    `json:"payment_method,omitempty"`
	PaymentReference string    `json:"payment_reference,omitempty"`

	// Line items (same as invoice)
	LineItems []InvoiceLineItem `json:"line_items"`

	// Totals
	Subtotal   float64 `json:"subtotal"`
	TaxRate    float64 `json:"tax_rate"`
	TaxAmount  float64 `json:"tax_amount"`
	Discount   float64 `json:"discount,omitempty"`
	Total      float64 `json:"total"`
	AmountPaid float64 `json:"amount_paid"`
}

// GetType returns the template type for receipt
func (d *ReceiptData) GetType() TemplateType {
	return TemplateTypeReceipt
}

// Validate validates receipt data
func (d *ReceiptData) Validate() error {
	if d.ReceiptNumber == "" && d.DocumentNumber == "" {
		return fmt.Errorf("%w: receipt number required", ErrInvalidContent)
	}
	if d.CustomerName == "" {
		return fmt.Errorf("%w: customer name required", ErrInvalidContent)
	}
	return nil
}

// QuoteLineItem represents a line item on a quote
type QuoteLineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
	Optional    bool    `json:"optional,omitempty"`
}

// GetType returns the template type for quote
func (d *QuoteData) GetType() TemplateType {
	return TemplateTypeQuote
}

// Validate validates quote data
func (d *QuoteData) Validate() error {
	if d.QuoteNumber == "" && d.DocumentNumber == "" {
		return fmt.Errorf("%w: quote number required", ErrInvalidContent)
	}
	if d.CustomerName == "" {
		return fmt.Errorf("%w: customer name required", ErrInvalidContent)
	}
	return nil
}

// WorkOrderData contains data for work order generation
type WorkOrderData struct {
	BaseDocumentData

	// Work order specific fields
	WorkOrderNumber string     `json:"work_order_number"`
	ScheduledDate   *time.Time `json:"scheduled_date,omitempty"`
	ScheduledTime   string     `json:"scheduled_time,omitempty"`
	Status          string     `json:"status,omitempty"`
	Priority        string     `json:"priority,omitempty"`

	// Service details
	ServiceType     string `json:"service_type,omitempty"`
	ServiceLocation string `json:"service_location,omitempty"`
	Description     string `json:"description,omitempty"`

	// Assigned technicians
	Technicians []TechnicianInfo `json:"technicians,omitempty"`

	// Equipment/materials
	Equipment []EquipmentItem `json:"equipment,omitempty"`

	// Tasks
	Tasks []TaskItem `json:"tasks,omitempty"`

	// Signatures
	CustomerSignature   *SignatureInfo `json:"customer_signature,omitempty"`
	TechnicianSignature *SignatureInfo `json:"technician_signature,omitempty"`
}

// TechnicianInfo contains technician details
type TechnicianInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// EquipmentItem represents equipment or materials
type EquipmentItem struct {
	Name        string  `json:"name"`
	Quantity    float64 `json:"quantity"`
	Description string  `json:"description,omitempty"`
}

// TaskItem represents a task on a work order
type TaskItem struct {
	Description string `json:"description"`
	Status      string `json:"status,omitempty"`
	Completed   bool   `json:"completed"`
}

// SignatureInfo contains signature details
type SignatureInfo struct {
	Name      string    `json:"name"`
	SignedAt  time.Time `json:"signed_at"`
	ImagePath string    `json:"image_path,omitempty"`
	ImageData []byte    `json:"image_data,omitempty"`
}

// GetType returns the template type for work order
func (d *WorkOrderData) GetType() TemplateType {
	return TemplateTypeWorkOrder
}

// Validate validates work order data
func (d *WorkOrderData) Validate() error {
	if d.WorkOrderNumber == "" && d.DocumentNumber == "" {
		return fmt.Errorf("%w: work order number required", ErrInvalidContent)
	}
	return nil
}

// ServiceReportData contains data for service report generation
type ServiceReportData struct {
	BaseDocumentData

	// Report specific fields
	ReportNumber  string     `json:"report_number"`
	ServiceDate   time.Time  `json:"service_date"`
	CompletedDate *time.Time `json:"completed_date,omitempty"`

	// Related work order
	WorkOrderNumber string `json:"work_order_number,omitempty"`

	// Service details
	ServiceType     string `json:"service_type,omitempty"`
	ServiceLocation string `json:"service_location,omitempty"`

	// Work performed
	WorkPerformed string     `json:"work_performed"`
	PartsUsed     []PartItem `json:"parts_used,omitempty"`
	LaborHours    float64    `json:"labor_hours,omitempty"`

	// Findings
	Findings        string `json:"findings,omitempty"`
	Recommendations string `json:"recommendations,omitempty"`

	// Technician
	TechnicianName string `json:"technician_name,omitempty"`

	// Photos
	Photos []PhotoItem `json:"photos,omitempty"`

	// Signatures
	CustomerSignature   *SignatureInfo `json:"customer_signature,omitempty"`
	TechnicianSignature *SignatureInfo `json:"technician_signature,omitempty"`
}

// PartItem represents a part used in service
type PartItem struct {
	PartNumber  string  `json:"part_number,omitempty"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost,omitempty"`
}

// PhotoItem represents a photo attachment
type PhotoItem struct {
	Path        string `json:"path,omitempty"`
	Data        []byte `json:"data,omitempty"`
	Caption     string `json:"caption,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// GetType returns the template type for service report
func (d *ServiceReportData) GetType() TemplateType {
	return TemplateTypeServiceReport
}

// Validate validates service report data
func (d *ServiceReportData) Validate() error {
	if d.ReportNumber == "" && d.DocumentNumber == "" {
		return fmt.Errorf("%w: report number required", ErrInvalidContent)
	}
	if d.WorkPerformed == "" {
		return fmt.Errorf("%w: work performed description required", ErrInvalidContent)
	}
	return nil
}

// =============================================================================
// Template Registry
// =============================================================================

// TemplateFunc is a function that renders a template
type TemplateFunc func(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error

// TemplateRegistry manages document templates
type TemplateRegistry struct {
	templates map[TemplateType]TemplateFunc
	mu        sync.RWMutex
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	r := &TemplateRegistry{
		templates: make(map[TemplateType]TemplateFunc),
	}

	// Register default templates
	r.RegisterTemplate(TemplateTypeInvoice, renderInvoiceTemplate)
	r.RegisterTemplate(TemplateTypeQuote, renderQuoteTemplate)
	r.RegisterTemplate(TemplateTypeReceipt, renderReceiptTemplate)
	r.RegisterTemplate(TemplateTypeWorkOrder, renderWorkOrderTemplate)
	r.RegisterTemplate(TemplateTypeServiceReport, renderServiceReportTemplate)

	return r
}

// RegisterTemplate registers a template function
func (r *TemplateRegistry) RegisterTemplate(templateType TemplateType, fn TemplateFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[templateType] = fn
}

// GetTemplate retrieves a template function
func (r *TemplateRegistry) GetTemplate(templateType TemplateType) (TemplateFunc, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.templates[templateType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingTemplate, templateType)
	}
	return fn, nil
}

// HasTemplate checks if a template exists
func (r *TemplateRegistry) HasTemplate(templateType TemplateType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.templates[templateType]
	return ok
}

// =============================================================================
// Default Template Implementations
// =============================================================================

// renderInvoiceTemplate renders an invoice document
func renderInvoiceTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	invoiceData, ok := data.(*InvoiceData)
	if !ok {
		return fmt.Errorf("%w: expected InvoiceData", ErrInvalidContent)
	}

	// Get invoice number
	invoiceNumber := invoiceData.InvoiceNumber
	if invoiceNumber == "" {
		invoiceNumber = invoiceData.DocumentNumber
	}

	// Render header
	renderDocumentHeader(pdf, config, "INVOICE", invoiceNumber)

	// Customer info section
	pdf.SetY(pdf.GetY() + 5)
	renderCustomerSection(pdf, config, &invoiceData.BaseDocumentData)

	// Invoice details
	pdf.SetY(pdf.GetY() + 5)
	renderKeyValuePair(pdf, config, "Invoice Date:", formatDate(invoiceData.DocumentDate))
	if invoiceData.DueDate != nil {
		renderKeyValuePair(pdf, config, "Due Date:", formatDate(*invoiceData.DueDate))
	}
	if invoiceData.PurchaseOrder != "" {
		renderKeyValuePair(pdf, config, "PO Number:", invoiceData.PurchaseOrder)
	}

	// Line items table
	pdf.SetY(pdf.GetY() + 10)
	headers := []string{"Description", "Qty", "Unit Price", "Amount"}
	widths := []float64{90, 20, 35, 35}

	var rows [][]string
	for _, item := range invoiceData.LineItems {
		rows = append(rows, []string{
			item.Description,
			fmt.Sprintf("%.2f", item.Quantity),
			formatCurrency(item.UnitPrice),
			formatCurrency(item.Amount),
		})
	}
	renderTable(pdf, config, headers, widths, rows)

	// Totals
	pdf.SetY(pdf.GetY() + 5)
	totalsX := 145.0
	renderTotalLine(pdf, config, totalsX, "Subtotal:", invoiceData.Subtotal)
	if invoiceData.Discount > 0 {
		renderTotalLine(pdf, config, totalsX, "Discount:", -invoiceData.Discount)
	}
	if invoiceData.TaxAmount > 0 {
		taxLabel := fmt.Sprintf("Tax (%.1f%%):", invoiceData.TaxRate)
		renderTotalLine(pdf, config, totalsX, taxLabel, invoiceData.TaxAmount)
	}
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	renderTotalLine(pdf, config, totalsX, "Total:", invoiceData.Total)

	if invoiceData.AmountPaid > 0 {
		renderTotalLine(pdf, config, totalsX, "Amount Paid:", invoiceData.AmountPaid)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody+2)
		renderTotalLine(pdf, config, totalsX, "Amount Due:", invoiceData.AmountDue)
	}

	// Payment instructions
	if invoiceData.PaymentInstructions != "" {
		pdf.SetY(pdf.GetY() + 15)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Payment Instructions:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
		pdf.MultiCell(0, 4, invoiceData.PaymentInstructions, "", "", false)
	}

	// Notes
	if invoiceData.Notes != "" {
		renderNotesSection(pdf, config, invoiceData.Notes)
	}

	return nil
}

// renderQuoteTemplate renders a quote document
func renderQuoteTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	quoteData, ok := data.(*QuoteData)
	if !ok {
		return fmt.Errorf("%w: expected QuoteData", ErrInvalidContent)
	}

	quoteNumber := quoteData.QuoteNumber
	if quoteNumber == "" {
		quoteNumber = quoteData.DocumentNumber
	}

	// Render header
	renderDocumentHeader(pdf, config, "QUOTE", quoteNumber)

	// Customer info
	pdf.SetY(pdf.GetY() + 5)
	renderCustomerSection(pdf, config, &quoteData.BaseDocumentData)

	// Quote details
	pdf.SetY(pdf.GetY() + 5)
	renderKeyValuePair(pdf, config, "Quote Date:", formatDate(quoteData.DocumentDate))
	if quoteData.ExpirationDate != nil {
		renderKeyValuePair(pdf, config, "Valid Until:", formatDate(*quoteData.ExpirationDate))
	}

	// Line items
	pdf.SetY(pdf.GetY() + 10)
	headers := []string{"Description", "Qty", "Unit Price", "Amount"}
	widths := []float64{90, 20, 35, 35}

	var rows [][]string
	for _, item := range quoteData.LineItems {
		desc := item.Description
		if item.Optional {
			desc = "(Optional) " + desc
		}
		rows = append(rows, []string{
			desc,
			fmt.Sprintf("%.2f", item.Quantity),
			formatCurrency(item.UnitPrice),
			formatCurrency(item.Amount),
		})
	}
	renderTable(pdf, config, headers, widths, rows)

	// Totals
	pdf.SetY(pdf.GetY() + 5)
	totalsX := 145.0
	renderTotalLine(pdf, config, totalsX, "Subtotal:", quoteData.Subtotal)
	if quoteData.Discount > 0 {
		renderTotalLine(pdf, config, totalsX, "Discount:", -quoteData.Discount)
	}
	if quoteData.TaxAmount > 0 {
		renderTotalLine(pdf, config, totalsX, "Tax:", quoteData.TaxAmount)
	}
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	renderTotalLine(pdf, config, totalsX, "Total:", quoteData.Total)

	// Terms
	if quoteData.Terms != "" || quoteData.AcceptanceTerms != "" {
		pdf.SetY(pdf.GetY() + 15)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Terms & Conditions:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
		if quoteData.Terms != "" {
			pdf.MultiCell(0, 4, quoteData.Terms, "", "", false)
		}
		if quoteData.AcceptanceTerms != "" {
			pdf.Ln(4)
			pdf.MultiCell(0, 4, quoteData.AcceptanceTerms, "", "", false)
		}
	}

	// Notes
	if quoteData.Notes != "" {
		renderNotesSection(pdf, config, quoteData.Notes)
	}

	return nil
}

// renderWorkOrderTemplate renders a work order document
func renderWorkOrderTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	woData, ok := data.(*WorkOrderData)
	if !ok {
		return fmt.Errorf("%w: expected WorkOrderData", ErrInvalidContent)
	}

	woNumber := woData.WorkOrderNumber
	if woNumber == "" {
		woNumber = woData.DocumentNumber
	}

	// Render header
	renderDocumentHeader(pdf, config, "WORK ORDER", woNumber)

	// Customer info
	pdf.SetY(pdf.GetY() + 5)
	renderCustomerSection(pdf, config, &woData.BaseDocumentData)

	// Work order details
	pdf.SetY(pdf.GetY() + 5)
	if woData.ScheduledDate != nil {
		renderKeyValuePair(pdf, config, "Scheduled Date:", formatDate(*woData.ScheduledDate))
	}
	if woData.ScheduledTime != "" {
		renderKeyValuePair(pdf, config, "Scheduled Time:", woData.ScheduledTime)
	}
	if woData.ServiceType != "" {
		renderKeyValuePair(pdf, config, "Service Type:", woData.ServiceType)
	}
	if woData.Priority != "" {
		renderKeyValuePair(pdf, config, "Priority:", woData.Priority)
	}
	if woData.ServiceLocation != "" {
		renderKeyValuePair(pdf, config, "Location:", woData.ServiceLocation)
	}

	// Assigned technicians
	if len(woData.Technicians) > 0 {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Assigned Technicians:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
		for _, tech := range woData.Technicians {
			techInfo := tech.Name
			if tech.Phone != "" {
				techInfo += " - " + tech.Phone
			}
			pdf.Cell(0, 5, "  - "+techInfo)
			pdf.Ln(5)
		}
	}

	// Description
	if woData.Description != "" {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Description:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
		pdf.MultiCell(0, 5, woData.Description, "", "", false)
	}

	// Tasks
	if len(woData.Tasks) > 0 {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Tasks:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
		for _, task := range woData.Tasks {
			checkbox := "[ ]"
			if task.Completed {
				checkbox = "[X]"
			}
			pdf.Cell(0, 5, "  "+checkbox+" "+task.Description)
			pdf.Ln(5)
		}
	}

	// Equipment/Materials
	if len(woData.Equipment) > 0 {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Equipment/Materials:")
		pdf.Ln(6)

		headers := []string{"Item", "Quantity", "Notes"}
		widths := []float64{80, 30, 70}
		var rows [][]string
		for _, eq := range woData.Equipment {
			rows = append(rows, []string{
				eq.Name,
				fmt.Sprintf("%.2f", eq.Quantity),
				eq.Description,
			})
		}
		renderTable(pdf, config, headers, widths, rows)
	}

	// Signature area
	pdf.SetY(pdf.GetY() + 20)
	renderSignatureBlock(pdf, config, "Customer Signature:", woData.CustomerSignature)
	pdf.SetY(pdf.GetY() + 15)
	renderSignatureBlock(pdf, config, "Technician Signature:", woData.TechnicianSignature)

	return nil
}

// renderServiceReportTemplate renders a service report document
func renderServiceReportTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	reportData, ok := data.(*ServiceReportData)
	if !ok {
		return fmt.Errorf("%w: expected ServiceReportData", ErrInvalidContent)
	}

	reportNumber := reportData.ReportNumber
	if reportNumber == "" {
		reportNumber = reportData.DocumentNumber
	}

	// Render header
	renderDocumentHeader(pdf, config, "SERVICE REPORT", reportNumber)

	// Customer info
	pdf.SetY(pdf.GetY() + 5)
	renderCustomerSection(pdf, config, &reportData.BaseDocumentData)

	// Report details
	pdf.SetY(pdf.GetY() + 5)
	renderKeyValuePair(pdf, config, "Service Date:", formatDate(reportData.ServiceDate))
	if reportData.CompletedDate != nil {
		renderKeyValuePair(pdf, config, "Completed:", formatDate(*reportData.CompletedDate))
	}
	if reportData.WorkOrderNumber != "" {
		renderKeyValuePair(pdf, config, "Work Order:", reportData.WorkOrderNumber)
	}
	if reportData.ServiceType != "" {
		renderKeyValuePair(pdf, config, "Service Type:", reportData.ServiceType)
	}
	if reportData.TechnicianName != "" {
		renderKeyValuePair(pdf, config, "Technician:", reportData.TechnicianName)
	}
	if reportData.LaborHours > 0 {
		renderKeyValuePair(pdf, config, "Labor Hours:", fmt.Sprintf("%.2f", reportData.LaborHours))
	}

	// Work performed
	pdf.SetY(pdf.GetY() + 8)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	pdf.Cell(40, 6, "Work Performed:")
	pdf.Ln(6)
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
	pdf.MultiCell(0, 5, reportData.WorkPerformed, "", "", false)

	// Parts used
	if len(reportData.PartsUsed) > 0 {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Parts Used:")
		pdf.Ln(6)

		headers := []string{"Part #", "Description", "Qty"}
		widths := []float64{40, 100, 30}
		var rows [][]string
		for _, part := range reportData.PartsUsed {
			rows = append(rows, []string{
				part.PartNumber,
				part.Description,
				fmt.Sprintf("%.2f", part.Quantity),
			})
		}
		renderTable(pdf, config, headers, widths, rows)
	}

	// Findings
	if reportData.Findings != "" {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Findings:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
		pdf.MultiCell(0, 5, reportData.Findings, "", "", false)
	}

	// Recommendations
	if reportData.Recommendations != "" {
		pdf.SetY(pdf.GetY() + 8)
		pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
		pdf.Cell(40, 6, "Recommendations:")
		pdf.Ln(6)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
		pdf.MultiCell(0, 5, reportData.Recommendations, "", "", false)
	}

	// Signatures
	pdf.SetY(pdf.GetY() + 20)
	renderSignatureBlock(pdf, config, "Customer Signature:", reportData.CustomerSignature)
	pdf.SetY(pdf.GetY() + 15)
	renderSignatureBlock(pdf, config, "Technician Signature:", reportData.TechnicianSignature)

	return nil
}

// renderReceiptTemplate renders a payment receipt document
func renderReceiptTemplate(pdf *gofpdf.Fpdf, data TemplateData, config *Config) error {
	receiptData, ok := data.(*ReceiptData)
	if !ok {
		return fmt.Errorf("%w: expected ReceiptData", ErrInvalidContent)
	}

	receiptNumber := receiptData.ReceiptNumber
	if receiptNumber == "" {
		receiptNumber = receiptData.DocumentNumber
	}

	// Render header
	renderDocumentHeader(pdf, config, "PAYMENT RECEIPT", receiptNumber)

	// Add PAID stamp
	renderPaidStamp(pdf, config)

	// Customer info section
	pdf.SetY(pdf.GetY() + 5)
	renderCustomerSection(pdf, config, &receiptData.BaseDocumentData)

	// Receipt details
	pdf.SetY(pdf.GetY() + 5)
	renderKeyValuePair(pdf, config, "Receipt Date:", formatDate(receiptData.PaymentDate))
	renderKeyValuePair(pdf, config, "Invoice Number:", receiptData.InvoiceNumber)
	if receiptData.PaymentMethod != "" {
		renderKeyValuePair(pdf, config, "Payment Method:", receiptData.PaymentMethod)
	}
	if receiptData.PaymentReference != "" {
		renderKeyValuePair(pdf, config, "Reference:", receiptData.PaymentReference)
	}

	// Line items table
	pdf.SetY(pdf.GetY() + 10)
	headers := []string{"Description", "Qty", "Unit Price", "Amount"}
	widths := []float64{90, 20, 35, 35}

	var rows [][]string
	for _, item := range receiptData.LineItems {
		rows = append(rows, []string{
			item.Description,
			fmt.Sprintf("%.2f", item.Quantity),
			formatCurrency(item.UnitPrice),
			formatCurrency(item.Amount),
		})
	}
	renderTable(pdf, config, headers, widths, rows)

	// Totals
	pdf.SetY(pdf.GetY() + 5)
	totalsX := 145.0
	renderTotalLine(pdf, config, totalsX, "Subtotal:", receiptData.Subtotal)
	if receiptData.Discount > 0 {
		renderTotalLine(pdf, config, totalsX, "Discount:", -receiptData.Discount)
	}
	if receiptData.TaxAmount > 0 {
		taxLabel := fmt.Sprintf("Tax (%.1f%%):", receiptData.TaxRate)
		renderTotalLine(pdf, config, totalsX, taxLabel, receiptData.TaxAmount)
	}
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	renderTotalLine(pdf, config, totalsX, "Total:", receiptData.Total)

	// Amount paid with green styling
	pdf.SetY(pdf.GetY() + 5)
	pdf.SetTextColor(34, 139, 34) // Forest green
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody+2)
	pdf.SetX(totalsX)
	pdf.Cell(25, 6, "Amount Paid:")
	pdf.Cell(30, 6, formatCurrency(receiptData.AmountPaid))
	pdf.SetTextColor(0, 0, 0) // Reset to black

	// Thank you message
	pdf.SetY(pdf.GetY() + 25)
	pdf.SetFont(config.Fonts.DefaultFamily, "I", config.Fonts.SizeBody)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 6, "Thank you for your payment!")
	pdf.SetTextColor(0, 0, 0)

	// Notes
	if receiptData.Notes != "" {
		renderNotesSection(pdf, config, receiptData.Notes)
	}

	return nil
}

// renderPaidStamp renders a "PAID" stamp on the receipt
func renderPaidStamp(pdf *gofpdf.Fpdf, config *Config) {
	// Save current state
	currentFont, currentSize := pdf.GetFontSize()

	// Position stamp in upper right area
	pdf.SetXY(150, 45)

	// Green "PAID" text with border
	pdf.SetTextColor(34, 139, 34) // Forest green
	pdf.SetDrawColor(34, 139, 34)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", 24)

	// Draw border
	pdf.SetLineWidth(1.5)
	pdf.Rect(145, 42, 45, 18, "D")

	// Draw text centered in border
	pdf.SetXY(145, 46)
	pdf.CellFormat(45, 10, "PAID", "", 0, "C", false, 0, "")

	// Restore state
	pdf.SetTextColor(0, 0, 0)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
	pdf.SetFont(config.Fonts.DefaultFamily, "", currentSize)
	_ = currentFont // Font family doesn't need explicit restoration
}

// =============================================================================
// Helper Rendering Functions
// =============================================================================

// renderDocumentHeader renders the document header with logo and title
func renderDocumentHeader(pdf *gofpdf.Fpdf, config *Config, docType, docNumber string) {
	// Logo
	if config.Branding.LogoPath != "" {
		pdf.Image(config.Branding.LogoPath, 20, 15, config.Branding.LogoWidth, config.Branding.LogoHeight, false, "", 0, "")
	}

	// Company info on right
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeHeading)
	pdf.SetXY(120, 15)
	pdf.Cell(0, 6, config.Branding.CompanyName)

	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
	pdf.SetXY(120, 22)
	if config.Branding.CompanyAddress != "" {
		pdf.MultiCell(70, 4, config.Branding.CompanyAddress, "", "L", false)
	}

	y := pdf.GetY()
	if config.Branding.CompanyPhone != "" {
		pdf.SetXY(120, y)
		pdf.Cell(0, 4, config.Branding.CompanyPhone)
		pdf.Ln(4)
	}
	if config.Branding.CompanyEmail != "" {
		pdf.Cell(0, 4, config.Branding.CompanyEmail)
		pdf.Ln(4)
	}

	// Document title
	pdf.SetY(50)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeTitle)
	r, g, b := hexToRGB(config.Branding.PrimaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 10, docType)
	pdf.Ln(12)

	// Document number
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(40, 6, fmt.Sprintf("%s #: %s", docType, docNumber))
	pdf.Ln(10)
}

// renderCustomerSection renders the customer information section
func renderCustomerSection(pdf *gofpdf.Fpdf, config *Config, data *BaseDocumentData) {
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	pdf.Cell(40, 6, "Bill To:")
	pdf.Ln(6)

	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
	pdf.Cell(0, 5, data.CustomerName)
	pdf.Ln(5)

	if data.CustomerAddress != "" {
		pdf.MultiCell(80, 4, data.CustomerAddress, "", "", false)
	}
	if data.CustomerEmail != "" {
		pdf.Cell(0, 4, data.CustomerEmail)
		pdf.Ln(4)
	}
	if data.CustomerPhone != "" {
		pdf.Cell(0, 4, data.CustomerPhone)
		pdf.Ln(4)
	}
}

// renderKeyValuePair renders a label-value pair
func renderKeyValuePair(pdf *gofpdf.Fpdf, config *Config, label, value string) {
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeBody)
	pdf.Cell(40, 5, label)
	pdf.Cell(0, 5, value)
	pdf.Ln(5)
}

// renderTable renders a table with headers and rows
func renderTable(pdf *gofpdf.Fpdf, config *Config, headers []string, widths []float64, rows [][]string) {
	// Header
	r, g, b := hexToRGB(config.Branding.PrimaryColor)
	pdf.SetFillColor(r, g, b)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeSmall)

	for i, header := range headers {
		pdf.CellFormat(widths[i], 7, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Rows
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
	pdf.SetFillColor(249, 250, 251)

	for rowIdx, row := range rows {
		fill := rowIdx%2 == 1
		for i, cell := range row {
			align := "L"
			if i > 0 {
				align = "R"
			}
			pdf.CellFormat(widths[i], 6, cell, "1", 0, align, fill, 0, "")
		}
		pdf.Ln(-1)
	}
}

// renderTotalLine renders a totals line
func renderTotalLine(pdf *gofpdf.Fpdf, config *Config, x float64, label string, amount float64) {
	pdf.SetX(x)
	pdf.Cell(25, 6, label)
	pdf.Cell(30, 6, formatCurrency(amount))
	pdf.Ln(6)
}

// renderNotesSection renders the notes section
func renderNotesSection(pdf *gofpdf.Fpdf, config *Config, notes string) {
	pdf.SetY(pdf.GetY() + 15)
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	pdf.Cell(40, 6, "Notes:")
	pdf.Ln(6)
	pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
	pdf.MultiCell(0, 4, notes, "", "", false)
}

// renderSignatureBlock renders a signature block
func renderSignatureBlock(pdf *gofpdf.Fpdf, config *Config, label string, sig *SignatureInfo) {
	pdf.SetFont(config.Fonts.DefaultFamily, "B", config.Fonts.SizeBody)
	pdf.Cell(40, 6, label)
	pdf.Ln(6)

	if sig != nil && sig.Name != "" {
		if sig.ImagePath != "" {
			// Render signature image
			pdf.Image(sig.ImagePath, pdf.GetX(), pdf.GetY(), 50, 15, false, "", 0, "")
			pdf.SetY(pdf.GetY() + 18)
		}
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
		pdf.Cell(50, 4, sig.Name)
		pdf.Cell(50, 4, formatDate(sig.SignedAt))
	} else {
		// Empty signature line
		pdf.Line(pdf.GetX(), pdf.GetY()+12, pdf.GetX()+60, pdf.GetY()+12)
		pdf.SetY(pdf.GetY() + 15)
		pdf.SetFont(config.Fonts.DefaultFamily, "", config.Fonts.SizeSmall)
		pdf.Cell(30, 4, "Signature")
		pdf.Cell(30, 4, "Date")
	}
}
