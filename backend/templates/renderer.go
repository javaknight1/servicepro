package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
	"time"
)

//go:embed quotes/*.html quotes/*.css invoices/*.html invoices/*.css reports/*.html reports/*.css shared/*.css
var templateFS embed.FS

// =============================================================================
// Template Types
// =============================================================================

// DocumentType represents the type of document template
type DocumentType string

const (
	DocumentTypeQuote   DocumentType = "quote"
	DocumentTypeInvoice DocumentType = "invoice"
	DocumentTypeReport  DocumentType = "report"
)

// =============================================================================
// Template Functions
// =============================================================================

// templateFuncs provides custom template functions
var templateFuncs = template.FuncMap{
	// Date/Time formatting
	"formatDate": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("January 2, 2006")
	},
	"formatDateShort": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("01/02/2006")
	},
	"formatDateTime": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("January 2, 2006 at 3:04 PM")
	},
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("3:04 PM")
	},

	// Currency formatting
	"formatCurrency": func(amount float64) string {
		negative := amount < 0
		if negative {
			amount = -amount
		}

		// Format with commas
		intPart := int64(amount)
		decPart := int64((amount - float64(intPart)) * 100)

		var result string
		if intPart == 0 {
			result = "0"
		} else {
			parts := []string{}
			for intPart > 0 {
				parts = append([]string{fmt.Sprintf("%03d", intPart%1000)}, parts...)
				intPart /= 1000
			}
			result = strings.TrimLeft(strings.Join(parts, ","), "0,")
			if result == "" {
				result = "0"
			}
		}

		if negative {
			return fmt.Sprintf("-$%s.%02d", result, decPart)
		}
		return fmt.Sprintf("$%s.%02d", result, decPart)
	},

	// Percent formatting
	"formatPercent": func(value float64) string {
		if value == float64(int(value)) {
			return fmt.Sprintf("%d%%", int(value))
		}
		return fmt.Sprintf("%.1f%%", value)
	},

	// Text transformations
	"uppercase": func(s string) string {
		return strings.ToUpper(s)
	},
	"lowercase": func(s string) string {
		return strings.ToLower(s)
	},
	"title": func(s string) string {
		return strings.Title(strings.ToLower(s))
	},

	// HTML helpers
	"nl2br": func(s string) template.HTML {
		escaped := template.HTMLEscapeString(s)
		return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>"))
	},
	"safe": func(s string) template.HTML {
		return template.HTML(s)
	},

	// Conditionals
	"default": func(defaultVal, val interface{}) interface{} {
		if val == nil || val == "" || val == 0 {
			return defaultVal
		}
		return val
	},

	// Math
	"add": func(a, b float64) float64 {
		return a + b
	},
	"sub": func(a, b float64) float64 {
		return a - b
	},
	"mul": func(a, b float64) float64 {
		return a * b
	},
	"div": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},

	// Comparison
	"eq": func(a, b interface{}) bool {
		return a == b
	},
	"ne": func(a, b interface{}) bool {
		return a != b
	},
	"gt": func(a, b float64) bool {
		return a > b
	},
	"gte": func(a, b float64) bool {
		return a >= b
	},
	"lt": func(a, b float64) bool {
		return a < b
	},
	"lte": func(a, b float64) bool {
		return a <= b
	},

	// Iteration helpers
	"seq": func(n int) []int {
		result := make([]int, n)
		for i := range result {
			result[i] = i + 1
		}
		return result
	},
}

// =============================================================================
// Renderer
// =============================================================================

// Renderer handles HTML template rendering
type Renderer struct {
	templates map[DocumentType]*template.Template
	mu        sync.RWMutex
	basePath  string
}

// NewRenderer creates a new template renderer
func NewRenderer() (*Renderer, error) {
	r := &Renderer{
		templates: make(map[DocumentType]*template.Template),
	}

	if err := r.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return r, nil
}

// loadTemplates loads all embedded templates
func (r *Renderer) loadTemplates() error {
	templateDirs := map[DocumentType]string{
		DocumentTypeQuote:   "quotes",
		DocumentTypeInvoice: "invoices",
		DocumentTypeReport:  "reports",
	}

	for docType, dir := range templateDirs {
		htmlPath := fmt.Sprintf("%s/base.html", dir)
		cssPath := fmt.Sprintf("%s/styles.css", dir)
		sharedCSSPath := "shared/base.css"

		// Read HTML template
		htmlContent, err := templateFS.ReadFile(htmlPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", htmlPath, err)
		}

		// Read CSS
		cssContent, err := templateFS.ReadFile(cssPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", cssPath, err)
		}

		// Read shared CSS
		sharedCSS, err := templateFS.ReadFile(sharedCSSPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", sharedCSSPath, err)
		}

		// Combine CSS (shared first, then template-specific)
		combinedCSS := string(sharedCSS) + "\n" + string(cssContent)

		// Create template with embedded CSS
		tmpl, err := template.New(string(docType)).Funcs(templateFuncs).Parse(string(htmlContent))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", docType, err)
		}

		// Store CSS for inline embedding
		tmpl, err = tmpl.New("styles").Parse(combinedCSS)
		if err != nil {
			return fmt.Errorf("failed to parse styles template: %w", err)
		}

		r.templates[docType] = tmpl
	}

	return nil
}

// RenderOptions configures template rendering
type RenderOptions struct {
	// InlineCSS embeds CSS directly in the HTML
	InlineCSS bool

	// CustomCSS adds additional CSS rules
	CustomCSS string

	// Branding overrides default branding
	Branding *BrandingOptions
}

// BrandingOptions allows customization of document branding
type BrandingOptions struct {
	CompanyName    string
	CompanyTagline string
	CompanyAddress string
	CompanyPhone   string
	CompanyEmail   string
	CompanyWebsite string
	CompanyLogo    string
	PrimaryColor   string
	FooterText     string
}

// Render renders a template with the given data
func (r *Renderer) Render(docType DocumentType, data interface{}, opts *RenderOptions) (string, error) {
	r.mu.RLock()
	tmpl, exists := r.templates[docType]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template not found: %s", docType)
	}

	// Prepare template data
	templateData := r.prepareData(data, opts)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()

	// Inline CSS if requested
	if opts != nil && opts.InlineCSS {
		result = r.inlineCSS(result, docType, opts.CustomCSS)
	}

	return result, nil
}

// RenderToWriter renders a template directly to a writer
func (r *Renderer) RenderToWriter(w io.Writer, docType DocumentType, data interface{}, opts *RenderOptions) error {
	html, err := r.Render(docType, data, opts)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(html))
	return err
}

// prepareData prepares the template data with branding and metadata
func (r *Renderer) prepareData(data interface{}, opts *RenderOptions) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy original data
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			result[k] = val
		}
	default:
		result["Data"] = data
	}

	// Add metadata
	result["GeneratedAt"] = time.Now()

	// Apply branding
	if opts != nil && opts.Branding != nil {
		b := opts.Branding
		if b.CompanyName != "" {
			result["CompanyName"] = b.CompanyName
		}
		if b.CompanyTagline != "" {
			result["CompanyTagline"] = b.CompanyTagline
		}
		if b.CompanyAddress != "" {
			result["CompanyAddress"] = b.CompanyAddress
		}
		if b.CompanyPhone != "" {
			result["CompanyPhone"] = b.CompanyPhone
		}
		if b.CompanyEmail != "" {
			result["CompanyEmail"] = b.CompanyEmail
		}
		if b.CompanyWebsite != "" {
			result["CompanyWebsite"] = b.CompanyWebsite
		}
		if b.CompanyLogo != "" {
			result["CompanyLogo"] = b.CompanyLogo
		}
		if b.FooterText != "" {
			result["FooterText"] = b.FooterText
		}
	}

	// Add custom CSS
	if opts != nil && opts.CustomCSS != "" {
		result["CustomCSS"] = opts.CustomCSS
	}

	return result
}

// inlineCSS embeds CSS directly in the HTML
func (r *Renderer) inlineCSS(html string, docType DocumentType, customCSS string) string {
	// Get template-specific CSS
	var cssPath string
	switch docType {
	case DocumentTypeQuote:
		cssPath = "quotes/styles.css"
	case DocumentTypeInvoice:
		cssPath = "invoices/styles.css"
	case DocumentTypeReport:
		cssPath = "reports/styles.css"
	}

	css, err := templateFS.ReadFile(cssPath)
	if err != nil {
		return html
	}

	sharedCSS, err := templateFS.ReadFile("shared/base.css")
	if err != nil {
		sharedCSS = []byte{}
	}

	// Build inline style tag
	styleTag := fmt.Sprintf("<style>\n%s\n%s\n%s\n</style>",
		string(sharedCSS),
		string(css),
		customCSS,
	)

	// Replace external stylesheet link with inline styles
	// This is a simple replacement - a more robust solution would use HTML parsing
	html = strings.Replace(html,
		`<link rel="stylesheet" href="styles.css">`,
		styleTag,
		1,
	)

	return html
}

// GetCSS returns the CSS for a document type
func (r *Renderer) GetCSS(docType DocumentType) (string, error) {
	var cssPath string
	switch docType {
	case DocumentTypeQuote:
		cssPath = "quotes/styles.css"
	case DocumentTypeInvoice:
		cssPath = "invoices/styles.css"
	case DocumentTypeReport:
		cssPath = "reports/styles.css"
	default:
		return "", fmt.Errorf("unknown document type: %s", docType)
	}

	css, err := templateFS.ReadFile(cssPath)
	if err != nil {
		return "", fmt.Errorf("failed to read CSS: %w", err)
	}

	sharedCSS, err := templateFS.ReadFile("shared/base.css")
	if err != nil {
		return string(css), nil
	}

	return string(sharedCSS) + "\n" + string(css), nil
}

// GetSharedCSS returns the shared CSS
func (r *Renderer) GetSharedCSS() (string, error) {
	css, err := templateFS.ReadFile("shared/base.css")
	if err != nil {
		return "", fmt.Errorf("failed to read shared CSS: %w", err)
	}
	return string(css), nil
}

// =============================================================================
// Preview Support
// =============================================================================

// PreviewData provides sample data for template previews
type PreviewData struct {
	Quote   map[string]interface{}
	Invoice map[string]interface{}
	Report  map[string]interface{}
}

// GetPreviewData returns sample data for previewing templates
func GetPreviewData() *PreviewData {
	now := time.Now()

	return &PreviewData{
		Quote: map[string]interface{}{
			"DocumentNumber":  "QT-2024-001",
			"Date":            now,
			"ValidUntil":      now.Add(14 * 24 * time.Hour),
			"Status":          "pending",
			"CustomerName":    "John Smith",
			"CustomerCompany": "Smith Industries",
			"CustomerAddress": "123 Business Ave\nSuite 100\nNew York, NY 10001",
			"CustomerEmail":   "john@smithind.com",
			"CustomerPhone":   "(555) 123-4567",
			"CompanyName":     "ServicePro",
			"CompanyAddress":  "456 Service Blvd\nLos Angeles, CA 90001",
			"CompanyPhone":    "(800) 555-0199",
			"CompanyEmail":    "info@servicepro.com",
			"LineItems": []map[string]interface{}{
				{
					"Description": "Annual HVAC Maintenance Contract",
					"Details":     "Includes quarterly inspections and priority service",
					"Quantity":    1,
					"UnitPrice":   1200.00,
					"Amount":      1200.00,
					"Optional":    false,
				},
				{
					"Description": "Extended Parts Warranty",
					"Details":     "2-year extended coverage on all parts",
					"Quantity":    1,
					"UnitPrice":   299.00,
					"Amount":      299.00,
					"Optional":    true,
				},
			},
			"Subtotal":      1200.00,
			"TaxRate":       8.5,
			"TaxAmount":     102.00,
			"Total":         1302.00,
			"OptionalTotal": 299.00,
			"ShowDiscount":  false,
			"Notes":         "Quote valid for 14 days from date of issue.",
			"Terms":         "50% deposit required to begin work. Balance due upon completion.",
		},
		Invoice: map[string]interface{}{
			"DocumentNumber":  "INV-2024-001",
			"Date":            now,
			"DueDate":         now.Add(30 * 24 * time.Hour),
			"PONumber":        "PO-12345",
			"PaymentTerms":    "Net 30",
			"CustomerName":    "Jane Doe",
			"CustomerCompany": "Doe Enterprises",
			"BillingAddress":  "789 Commerce St\nChicago, IL 60601",
			"ShippingAddress": "100 Warehouse Way\nChicago, IL 60602",
			"CustomerEmail":   "jane@doeent.com",
			"CustomerPhone":   "(555) 987-6543",
			"CompanyName":     "ServicePro",
			"CompanyAddress":  "456 Service Blvd\nLos Angeles, CA 90001",
			"CompanyPhone":    "(800) 555-0199",
			"CompanyEmail":    "billing@servicepro.com",
			"ServiceDate":     now.Add(-7 * 24 * time.Hour),
			"Technician":      "Mike Johnson",
			"WorkOrderNumber": "WO-2024-042",
			"LineItems": []map[string]interface{}{
				{
					"Description": "Emergency HVAC Repair",
					"Details":     "Replaced compressor and recharged system",
					"Quantity":    1,
					"UnitPrice":   450.00,
					"Amount":      450.00,
					"Taxable":     true,
				},
				{
					"Description": "Compressor Unit",
					"Details":     "Part #COMP-2000-A",
					"Quantity":    1,
					"UnitPrice":   650.00,
					"Amount":      650.00,
					"Taxable":     true,
				},
				{
					"Description": "Labor (2 hours)",
					"Quantity":    2,
					"Unit":        "hrs",
					"UnitPrice":   85.00,
					"Amount":      170.00,
					"Taxable":     false,
				},
			},
			"Subtotal":      1270.00,
			"TaxRate":       8.5,
			"TaxAmount":     93.50,
			"Total":         1363.50,
			"AmountPaid":    0.00,
			"AmountDue":     1363.50,
			"ShowTaxColumn": true,
			"Notes":         "Thank you for choosing ServicePro!",
		},
		Report: map[string]interface{}{
			"DocumentNumber":  "SR-2024-001",
			"WorkOrderNumber": "WO-2024-042",
			"ServiceDate":     now,
			"Status":          "completed",
			"CustomerName":    "ABC Corporation",
			"CustomerPhone":   "(555) 111-2222",
			"CustomerEmail":   "facilities@abccorp.com",
			"ServiceAddress":  "500 Corporate Plaza\nSuite 300\nHouston, TX 77001",
			"CompanyName":     "ServicePro",
			"CompanyPhone":    "(800) 555-0199",
			"CompanyEmail":    "service@servicepro.com",
			"Technician": map[string]interface{}{
				"Name":          "Robert Williams",
				"Phone":         "(555) 333-4444",
				"LicenseNumber": "HVAC-TX-12345",
			},
			"ArrivalTime":   now.Add(-4 * time.Hour),
			"DepartureTime": now.Add(-1 * time.Hour),
			"TotalHours":    3.0,
			"Equipment": []map[string]interface{}{
				{
					"Name":         "Rooftop HVAC Unit",
					"Model":        "Carrier 48TM016",
					"SerialNumber": "SN-2019-45678",
					"Location":     "Rooftop - Unit 1",
					"Condition":    "Good",
				},
			},
			"WorkPerformed":    "Performed scheduled maintenance inspection.\nReplaced air filter and cleaned condenser coils.\nChecked refrigerant levels and electrical connections.\nTested thermostat operation.",
			"Findings":         "Unit operating within normal parameters.\nMinor wear observed on belt - recommend replacement within 6 months.\nDrain pan clear, no signs of water damage.",
			"Recommendations":  "Schedule belt replacement during next maintenance visit.\nConsider upgrading thermostat to smart model for energy savings.",
			"NextServiceDate":  now.Add(90 * 24 * time.Hour),
			"NextServiceNotes": "Quarterly maintenance - include belt replacement",
			"Tasks": []map[string]interface{}{
				{"Description": "Visual inspection", "Status": "completed"},
				{"Description": "Replace air filter", "Status": "completed"},
				{"Description": "Clean condenser coils", "Status": "completed"},
				{"Description": "Check refrigerant levels", "Status": "completed", "Notes": "Levels normal"},
				{"Description": "Test thermostat", "Status": "completed"},
			},
			"PartsUsed": []map[string]interface{}{
				{
					"PartNumber":  "FLT-20X20X1",
					"Description": "Air Filter 20x20x1",
					"Quantity":    1,
					"UnitCost":    15.00,
					"Total":       15.00,
				},
			},
			"PartsTotal":        15.00,
			"LaborHours":        3.0,
			"LaborRate":         85.00,
			"LaborTotal":        255.00,
			"TotalLaborCharges": 255.00,
			"TotalCharges":      270.00,
			"CustomerSignature": map[string]interface{}{
				"Name": "Tom Manager",
				"Date": now,
			},
			"TechnicianSignature": map[string]interface{}{
				"Name": "Robert Williams",
				"Date": now,
			},
		},
	}
}

// RenderPreview renders a template with sample data
func (r *Renderer) RenderPreview(docType DocumentType, opts *RenderOptions) (string, error) {
	preview := GetPreviewData()

	var data map[string]interface{}
	switch docType {
	case DocumentTypeQuote:
		data = preview.Quote
	case DocumentTypeInvoice:
		data = preview.Invoice
	case DocumentTypeReport:
		data = preview.Report
	default:
		return "", fmt.Errorf("unknown document type: %s", docType)
	}

	return r.Render(docType, data, opts)
}
