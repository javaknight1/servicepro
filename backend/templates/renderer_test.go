package templates

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Renderer Tests
// =============================================================================

func TestNewRenderer(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	if renderer == nil {
		t.Fatal("renderer should not be nil")
	}

	// Check that templates are loaded
	if len(renderer.templates) == 0 {
		t.Error("templates should be loaded")
	}
}

func TestRenderQuote(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "QT-TEST-001",
		"Date":           time.Now(),
		"ValidUntil":     time.Now().Add(14 * 24 * time.Hour),
		"Status":         "pending",
		"CustomerName":   "Test Customer",
		"CompanyName":    "Test Company",
		"LineItems": []map[string]interface{}{
			{
				"Description": "Test Item",
				"Quantity":    1,
				"UnitPrice":   100.00,
				"Amount":      100.00,
			},
		},
		"Subtotal": 100.00,
		"Total":    100.00,
	}

	html, err := renderer.Render(DocumentTypeQuote, data, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify content
	if !strings.Contains(html, "QT-TEST-001") {
		t.Error("HTML should contain document number")
	}
	if !strings.Contains(html, "Test Customer") {
		t.Error("HTML should contain customer name")
	}
	if !strings.Contains(html, "QUOTE") {
		t.Error("HTML should contain document type")
	}
}

func TestRenderInvoice(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "INV-TEST-001",
		"Date":           time.Now(),
		"DueDate":        time.Now().Add(30 * 24 * time.Hour),
		"CustomerName":   "Invoice Customer",
		"CompanyName":    "Test Company",
		"PaymentTerms":   "Net 30",
		"LineItems": []map[string]interface{}{
			{
				"Description": "Service",
				"Quantity":    1,
				"UnitPrice":   200.00,
				"Amount":      200.00,
			},
		},
		"Subtotal":  200.00,
		"Total":     200.00,
		"AmountDue": 200.00,
	}

	html, err := renderer.Render(DocumentTypeInvoice, data, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "INV-TEST-001") {
		t.Error("HTML should contain invoice number")
	}
	if !strings.Contains(html, "INVOICE") {
		t.Error("HTML should contain document type")
	}
}

func TestRenderReport(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber":  "SR-TEST-001",
		"WorkOrderNumber": "WO-TEST-001",
		"ServiceDate":     time.Now(),
		"Status":          "completed",
		"CustomerName":    "Report Customer",
		"CompanyName":     "Test Company",
		"Technician": map[string]interface{}{
			"Name":  "Test Tech",
			"Phone": "(555) 123-4567",
		},
		"ArrivalTime":   time.Now().Add(-2 * time.Hour),
		"DepartureTime": time.Now(),
		"TotalHours":    2.0,
		"WorkPerformed": "Test work performed",
	}

	html, err := renderer.Render(DocumentTypeReport, data, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "SR-TEST-001") {
		t.Error("HTML should contain report number")
	}
	if !strings.Contains(html, "SERVICE REPORT") {
		t.Error("HTML should contain document type")
	}
}

func TestRenderWithBranding(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "QT-BRAND-001",
		"CustomerName":   "Test Customer",
	}

	opts := &RenderOptions{
		Branding: &BrandingOptions{
			CompanyName:    "Custom Company",
			CompanyTagline: "Custom Tagline",
			CompanyPhone:   "(800) 555-0000",
			CompanyEmail:   "custom@company.com",
			FooterText:     "Custom Footer Message",
		},
	}

	html, err := renderer.Render(DocumentTypeQuote, data, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "Custom Company") {
		t.Error("HTML should contain custom company name")
	}
	if !strings.Contains(html, "Custom Footer Message") {
		t.Error("HTML should contain custom footer")
	}
}

func TestRenderWithInlineCSS(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "QT-CSS-001",
		"CustomerName":   "Test Customer",
	}

	opts := &RenderOptions{
		InlineCSS: true,
	}

	html, err := renderer.Render(DocumentTypeQuote, data, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should contain style tag instead of link
	if !strings.Contains(html, "<style>") {
		t.Error("HTML should contain inline styles")
	}
}

func TestRenderWithCustomCSS(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "QT-CUSTOM-001",
		"CustomerName":   "Test Customer",
	}

	opts := &RenderOptions{
		CustomCSS: ".custom-class { color: red; }",
	}

	html, err := renderer.Render(DocumentTypeQuote, data, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// CustomCSS should be in the output (if rendered with template)
	// The template checks for CustomCSS
	if !strings.Contains(html, "custom-class") {
		t.Error("HTML should contain custom CSS")
	}
}

func TestRenderToWriter(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := map[string]interface{}{
		"DocumentNumber": "QT-WRITER-001",
		"CustomerName":   "Test Customer",
	}

	var buf bytes.Buffer
	err = renderer.RenderToWriter(&buf, DocumentTypeQuote, data, nil)
	if err != nil {
		t.Fatalf("RenderToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("buffer should contain content")
	}

	if !strings.Contains(buf.String(), "QT-WRITER-001") {
		t.Error("buffer should contain document number")
	}
}

func TestRenderInvalidDocumentType(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	_, err = renderer.Render("invalid_type", nil, nil)
	if err == nil {
		t.Error("should return error for invalid document type")
	}
}

func TestGetCSS(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	tests := []struct {
		docType DocumentType
	}{
		{DocumentTypeQuote},
		{DocumentTypeInvoice},
		{DocumentTypeReport},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			css, err := renderer.GetCSS(tt.docType)
			if err != nil {
				t.Fatalf("GetCSS failed: %v", err)
			}

			if css == "" {
				t.Error("CSS should not be empty")
			}

			// Should contain shared CSS variables
			if !strings.Contains(css, "--color-primary") {
				t.Error("CSS should contain shared variables")
			}
		})
	}
}

func TestGetSharedCSS(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	css, err := renderer.GetSharedCSS()
	if err != nil {
		t.Fatalf("GetSharedCSS failed: %v", err)
	}

	if css == "" {
		t.Error("shared CSS should not be empty")
	}

	// Verify shared CSS contains expected content
	expectations := []string{
		"--color-primary",
		"--font-family",
		"--spacing",
		".badge",
		".btn",
	}

	for _, expected := range expectations {
		if !strings.Contains(css, expected) {
			t.Errorf("shared CSS should contain %s", expected)
		}
	}
}

// =============================================================================
// Preview Data Tests
// =============================================================================

func TestGetPreviewData(t *testing.T) {
	preview := GetPreviewData()

	if preview == nil {
		t.Fatal("preview data should not be nil")
	}

	// Check quote data
	if preview.Quote == nil {
		t.Error("quote preview data should not be nil")
	}
	if preview.Quote["DocumentNumber"] == nil {
		t.Error("quote should have document number")
	}

	// Check invoice data
	if preview.Invoice == nil {
		t.Error("invoice preview data should not be nil")
	}
	if preview.Invoice["DocumentNumber"] == nil {
		t.Error("invoice should have document number")
	}

	// Check report data
	if preview.Report == nil {
		t.Error("report preview data should not be nil")
	}
	if preview.Report["DocumentNumber"] == nil {
		t.Error("report should have document number")
	}
}

func TestRenderPreview(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	tests := []struct {
		docType  DocumentType
		expected string
	}{
		{DocumentTypeQuote, "QUOTE"},
		{DocumentTypeInvoice, "INVOICE"},
		{DocumentTypeReport, "SERVICE REPORT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			html, err := renderer.RenderPreview(tt.docType, nil)
			if err != nil {
				t.Fatalf("RenderPreview failed: %v", err)
			}

			if !strings.Contains(html, tt.expected) {
				t.Errorf("preview should contain %s", tt.expected)
			}

			// Should be valid HTML
			if !strings.HasPrefix(html, "<!DOCTYPE html>") {
				t.Error("should be valid HTML document")
			}
		})
	}
}

// =============================================================================
// Template Function Tests
// =============================================================================

func TestTemplateFuncFormatDate(t *testing.T) {
	fn := templateFuncs["formatDate"].(func(time.Time) string)

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "valid date",
			time: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			want: "March 15, 2024",
		},
		{
			name: "zero time",
			time: time.Time{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fn(tt.time)
			if got != tt.want {
				t.Errorf("formatDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateFuncFormatCurrency(t *testing.T) {
	fn := templateFuncs["formatCurrency"].(func(float64) string)

	tests := []struct {
		amount float64
		want   string
	}{
		{0, "$0.00"},
		{100, "$100.00"},
		{1234.56, "$1,234.56"},
		{1000000, "$1,000,000.00"},
		{-50.00, "-$50.00"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := fn(tt.amount)
			if got != tt.want {
				t.Errorf("formatCurrency(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestTemplateFuncFormatPercent(t *testing.T) {
	fn := templateFuncs["formatPercent"].(func(float64) string)

	tests := []struct {
		value float64
		want  string
	}{
		{10, "10%"},
		{8.5, "8.5%"},
		{100, "100%"},
		{0, "0%"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := fn(tt.value)
			if got != tt.want {
				t.Errorf("formatPercent(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestTemplateFuncNl2br(t *testing.T) {
	fn := templateFuncs["nl2br"].(func(string) template.HTML)

	input := "Line 1\nLine 2\nLine 3"
	result := fn(input)

	if !strings.Contains(string(result), "<br>") {
		t.Error("nl2br should convert newlines to <br>")
	}

	// Count <br> tags
	count := strings.Count(string(result), "<br>")
	if count != 2 {
		t.Errorf("expected 2 <br> tags, got %d", count)
	}
}

func TestTemplateFuncUpperLower(t *testing.T) {
	upper := templateFuncs["uppercase"].(func(string) string)
	lower := templateFuncs["lowercase"].(func(string) string)

	if upper("hello") != "HELLO" {
		t.Error("uppercase should convert to upper case")
	}

	if lower("HELLO") != "hello" {
		t.Error("lowercase should convert to lower case")
	}
}

func TestTemplateFuncMath(t *testing.T) {
	add := templateFuncs["add"].(func(float64, float64) float64)
	sub := templateFuncs["sub"].(func(float64, float64) float64)
	mul := templateFuncs["mul"].(func(float64, float64) float64)
	div := templateFuncs["div"].(func(float64, float64) float64)

	if add(1, 2) != 3 {
		t.Error("add should add numbers")
	}
	if sub(5, 3) != 2 {
		t.Error("sub should subtract numbers")
	}
	if mul(3, 4) != 12 {
		t.Error("mul should multiply numbers")
	}
	if div(10, 2) != 5 {
		t.Error("div should divide numbers")
	}
	if div(10, 0) != 0 {
		t.Error("div by zero should return 0")
	}
}

// =============================================================================
// HTTP Handler Tests
// =============================================================================

func TestPreviewHandlerGet(t *testing.T) {
	handler, err := NewPreviewHandler()
	if err != nil {
		t.Fatalf("NewPreviewHandler failed: %v", err)
	}

	tests := []struct {
		name       string
		query      string
		statusCode int
	}{
		{"quote preview", "?type=quote", http.StatusOK},
		{"invoice preview", "?type=invoice", http.StatusOK},
		{"report preview", "?type=report", http.StatusOK},
		{"default preview", "", http.StatusOK},
		{"invalid type", "?type=invalid", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/preview"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.statusCode == http.StatusOK {
				if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
					t.Error("content type should be text/html")
				}
			}
		})
	}
}

func TestPreviewHandlerPost(t *testing.T) {
	handler, err := NewPreviewHandler()
	if err != nil {
		t.Fatalf("NewPreviewHandler failed: %v", err)
	}

	body := `{
		"data": {
			"DocumentNumber": "POST-TEST-001",
			"CustomerName": "Post Test Customer"
		},
		"inline_css": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/preview?type=quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "POST-TEST-001") {
		t.Error("response should contain posted document number")
	}
}

func TestCSSHandler(t *testing.T) {
	handler, err := NewCSSHandler()
	if err != nil {
		t.Fatalf("NewCSSHandler failed: %v", err)
	}

	tests := []struct {
		name       string
		query      string
		statusCode int
	}{
		{"shared CSS", "", http.StatusOK},
		{"shared CSS explicit", "?type=shared", http.StatusOK},
		{"quote CSS", "?type=quote", http.StatusOK},
		{"invoice CSS", "?type=invoice", http.StatusOK},
		{"report CSS", "?type=report", http.StatusOK},
		{"invalid type", "?type=invalid", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/css"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}

			if tt.statusCode == http.StatusOK {
				if w.Header().Get("Content-Type") != "text/css; charset=utf-8" {
					t.Error("content type should be text/css")
				}
			}
		})
	}
}

func TestListHandler(t *testing.T) {
	handler := NewListHandler()

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("content type should be application/json")
	}

	body := w.Body.String()
	if !strings.Contains(body, "quote") {
		t.Error("response should list quote template")
	}
	if !strings.Contains(body, "invoice") {
		t.Error("response should list invoice template")
	}
	if !strings.Contains(body, "report") {
		t.Error("response should list report template")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	previewHandler, _ := NewPreviewHandler()
	cssHandler, _ := NewCSSHandler()
	listHandler := NewListHandler()

	handlers := []http.Handler{previewHandler, cssHandler, listHandler}

	for _, handler := range handlers {
		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405 for DELETE, got %d", w.Code)
		}
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestHTMLValidity(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	docTypes := []DocumentType{
		DocumentTypeQuote,
		DocumentTypeInvoice,
		DocumentTypeReport,
	}

	for _, docType := range docTypes {
		t.Run(string(docType), func(t *testing.T) {
			html, err := renderer.RenderPreview(docType, nil)
			if err != nil {
				t.Fatalf("RenderPreview failed: %v", err)
			}

			// Basic HTML structure checks
			checks := []struct {
				name     string
				contains string
			}{
				{"DOCTYPE", "<!DOCTYPE html>"},
				{"html tag", "<html"},
				{"head tag", "<head>"},
				{"body tag", "<body>"},
				{"closing html", "</html>"},
				{"charset", "charset="},
				{"viewport", "viewport"},
			}

			for _, check := range checks {
				if !strings.Contains(html, check.contains) {
					t.Errorf("HTML should contain %s (%s)", check.name, check.contains)
				}
			}

			// Check for balanced tags (simple check)
			openDiv := strings.Count(html, "<div")
			closeDiv := strings.Count(html, "</div>")
			if openDiv != closeDiv {
				t.Errorf("unbalanced div tags: %d open, %d close", openDiv, closeDiv)
			}
		})
	}
}

func TestCSSValidity(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	css, err := renderer.GetSharedCSS()
	if err != nil {
		t.Fatalf("GetSharedCSS failed: %v", err)
	}

	// Check for CSS structure
	checks := []struct {
		name     string
		contains string
	}{
		{"root variables", ":root"},
		{"color variable", "--color-"},
		{"font variable", "--font-"},
		{"spacing variable", "--spacing-"},
		{"media query", "@media"},
		{"print styles", "@media print"},
	}

	for _, check := range checks {
		if !strings.Contains(css, check.contains) {
			t.Errorf("CSS should contain %s (%s)", check.name, check.contains)
		}
	}

	// Check balanced braces (simple check)
	openBrace := strings.Count(css, "{")
	closeBrace := strings.Count(css, "}")
	if openBrace != closeBrace {
		t.Errorf("unbalanced braces: %d open, %d close", openBrace, closeBrace)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkRenderQuote(b *testing.B) {
	renderer, _ := NewRenderer()
	data := GetPreviewData().Quote

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(DocumentTypeQuote, data, nil)
	}
}

func BenchmarkRenderInvoice(b *testing.B) {
	renderer, _ := NewRenderer()
	data := GetPreviewData().Invoice

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(DocumentTypeInvoice, data, nil)
	}
}

func BenchmarkRenderReport(b *testing.B) {
	renderer, _ := NewRenderer()
	data := GetPreviewData().Report

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(DocumentTypeReport, data, nil)
	}
}

func BenchmarkRenderWithInlineCSS(b *testing.B) {
	renderer, _ := NewRenderer()
	data := GetPreviewData().Quote
	opts := &RenderOptions{InlineCSS: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(DocumentTypeQuote, data, opts)
	}
}
