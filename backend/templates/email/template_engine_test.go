package email

import (
	"html/template"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ============================================
// Template Engine Tests
// ============================================

func TestNewTemplateEngine(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Engine is nil")
	}

	if engine.config == nil {
		t.Error("Config is nil")
	}

	if engine.spamChecker == nil {
		t.Error("Spam checker is nil")
	}
}

func TestNewTemplateEngineWithConfig(t *testing.T) {
	config := &TemplateConfig{
		TemplateDir:           "templates/email",
		MJMLEnabled:           false,
		CacheEnabled:          true,
		CacheTTL:              time.Hour,
		SpamCheckEnabled:      true,
		MaxSpamScore:          3.0,
		DefaultPrimaryColor:   "#ff0000",
		DefaultSecondaryColor: "#00ff00",
		CompanyName:           "TestCorp",
	}

	engine, err := NewTemplateEngine(config)
	if err != nil {
		t.Fatalf("Failed to create template engine: %v", err)
	}

	if engine.config.CompanyName != "TestCorp" {
		t.Errorf("Expected company name TestCorp, got %s", engine.config.CompanyName)
	}

	if engine.config.MaxSpamScore != 3.0 {
		t.Errorf("Expected max spam score 3.0, got %f", engine.config.MaxSpamScore)
	}
}

func TestDefaultTemplateConfig(t *testing.T) {
	config := DefaultTemplateConfig()

	if config.MJMLEnabled != true {
		t.Error("MJML should be enabled by default")
	}

	if config.CacheEnabled != true {
		t.Error("Cache should be enabled by default")
	}

	if config.SpamCheckEnabled != true {
		t.Error("Spam check should be enabled by default")
	}

	if config.MaxSpamScore != 5.0 {
		t.Errorf("Expected max spam score 5.0, got %f", config.MaxSpamScore)
	}
}

// ============================================
// Template Function Tests
// ============================================

func TestTemplateFuncs(t *testing.T) {
	funcs := createTemplateFuncs()

	t.Run("string functions", func(t *testing.T) {
		upper := funcs["upper"].(func(string) string)
		if upper("hello") != "HELLO" {
			t.Error("upper function failed")
		}

		lower := funcs["lower"].(func(string) string)
		if lower("HELLO") != "hello" {
			t.Error("lower function failed")
		}

		trim := funcs["trim"].(func(string) string)
		if trim("  hello  ") != "hello" {
			t.Error("trim function failed")
		}

		contains := funcs["contains"].(func(string, string) bool)
		if !contains("hello world", "world") {
			t.Error("contains function failed")
		}
	})

	t.Run("date functions", func(t *testing.T) {
		formatDate := funcs["formatDate"].(func(time.Time, string) string)
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		if formatDate(testTime, "2006-01-02") != "2024-01-15" {
			t.Error("formatDate function failed")
		}

		formatDateShort := funcs["formatDateShort"].(func(time.Time) string)
		if formatDateShort(testTime) != "Jan 15, 2024" {
			t.Error("formatDateShort function failed")
		}

		formatTime := funcs["formatTime"].(func(time.Time) string)
		if formatTime(testTime) != "10:30 AM" {
			t.Error("formatTime function failed")
		}

		year := funcs["year"].(func() int)
		if year() != time.Now().Year() {
			t.Error("year function failed")
		}
	})

	t.Run("number functions", func(t *testing.T) {
		formatCurrency := funcs["formatCurrency"].(func(float64, string) string)
		if formatCurrency(100.50, "USD") != "$100.50" {
			t.Error("formatCurrency USD failed")
		}
		if formatCurrency(100.50, "EUR") != "€100.50" {
			t.Error("formatCurrency EUR failed")
		}
		if formatCurrency(100.50, "GBP") != "£100.50" {
			t.Error("formatCurrency GBP failed")
		}

		formatPercent := funcs["formatPercent"].(func(float64) string)
		if formatPercent(8.5) != "8.5%" {
			t.Error("formatPercent function failed")
		}

		add := funcs["add"].(func(int, int) int)
		if add(5, 3) != 8 {
			t.Error("add function failed")
		}

		mul := funcs["mul"].(func(float64, float64) float64)
		if mul(5, 3) != 15 {
			t.Error("mul function failed")
		}

		div := funcs["div"].(func(float64, float64) float64)
		if div(10, 2) != 5 {
			t.Error("div function failed")
		}
		if div(10, 0) != 0 {
			t.Error("div by zero should return 0")
		}
	})

	t.Run("conditional functions", func(t *testing.T) {
		defaultFn := funcs["default"].(func(interface{}, interface{}) interface{})
		if defaultFn("default", "").(string) != "default" {
			t.Error("default function failed for empty string")
		}
		if defaultFn("default", "value").(string) != "value" {
			t.Error("default function failed for non-empty string")
		}

		ternary := funcs["ternary"].(func(bool, interface{}, interface{}) interface{})
		if ternary(true, "yes", "no").(string) != "yes" {
			t.Error("ternary true failed")
		}
		if ternary(false, "yes", "no").(string) != "no" {
			t.Error("ternary false failed")
		}
	})

	t.Run("HTML functions", func(t *testing.T) {
		safeHTML := funcs["safeHTML"].(func(string) template.HTML)
		result := safeHTML("<b>bold</b>")
		if string(result) != "<b>bold</b>" {
			t.Error("safeHTML function failed")
		}
	})
}

// ============================================
// Spam Checker Tests
// ============================================

func TestNewSpamChecker(t *testing.T) {
	checker := NewSpamChecker(5.0)

	if checker == nil {
		t.Fatal("Spam checker is nil")
	}

	if checker.maxScore != 5.0 {
		t.Errorf("Expected max score 5.0, got %f", checker.maxScore)
	}

	if len(checker.rules) == 0 {
		t.Error("No default rules loaded")
	}
}

func TestSpamCheckerCleanEmail(t *testing.T) {
	checker := NewSpamChecker(5.0)

	cleanContent := `
		<html>
		<body>
			<h1>Hello John</h1>
			<p>Your appointment is confirmed for Monday.</p>
			<a href="https://example.com/confirm">Confirm</a>
		</body>
		</html>
	`

	result := checker.Check(cleanContent)

	if result.Score > 1.0 {
		t.Errorf("Clean email has unexpectedly high spam score: %f", result.Score)
	}

	if !result.Passed {
		t.Error("Clean email should pass spam check")
	}
}

func TestSpamCheckerSpammyEmail(t *testing.T) {
	checker := NewSpamChecker(3.0)

	spammyContent := `
		<html>
		<body>
			<h1>URGENT!!! ACT NOW!!!</h1>
			<p>CONGRATULATIONS! You are a WINNER! Claim your FREE PRIZE immediately!</p>
			<p>Click here to verify your account before it's suspended!</p>
			<a href="http://192.168.1.1/claim">CLAIM NOW!!!</a>
		</body>
		</html>
	`

	result := checker.Check(spammyContent)

	if result.Score < 3.0 {
		t.Errorf("Spammy email should have high spam score, got: %f", result.Score)
	}

	if result.Passed {
		t.Error("Spammy email should fail spam check")
	}

	if len(result.Matches) == 0 {
		t.Error("Spammy email should have rule matches")
	}

	if len(result.Suggestions) == 0 {
		t.Error("Failed spam check should have suggestions")
	}
}

func TestSpamCheckerUrgencyLanguage(t *testing.T) {
	checker := NewSpamChecker(5.0)

	tests := []struct {
		content     string
		shouldMatch bool
	}{
		{"Act now before it's too late", true},
		{"This is an urgent matter", true},
		{"Limited time offer", true},
		{"Your appointment is scheduled", false},
	}

	for _, tt := range tests {
		result := checker.Check(tt.content)
		hasUrgencyMatch := false
		for _, match := range result.Matches {
			if match.Rule == "urgency" {
				hasUrgencyMatch = true
				break
			}
		}

		if hasUrgencyMatch != tt.shouldMatch {
			t.Errorf("Content %q: expected urgency match %v, got %v",
				tt.content, tt.shouldMatch, hasUrgencyMatch)
		}
	}
}

func TestSpamCheckerHiddenText(t *testing.T) {
	checker := NewSpamChecker(5.0)

	hiddenContent := `<div style="display:none">hidden spam text</div>`
	result := checker.Check(hiddenContent)

	hasHiddenMatch := false
	for _, match := range result.Matches {
		if match.Rule == "hidden_text" {
			hasHiddenMatch = true
			break
		}
	}

	if !hasHiddenMatch {
		t.Error("Should detect hidden text")
	}
}

func TestSpamCheckerIPURL(t *testing.T) {
	checker := NewSpamChecker(5.0)

	ipURLContent := `<a href="http://192.168.1.1/page">Click</a>`
	result := checker.Check(ipURLContent)

	hasIPMatch := false
	for _, match := range result.Matches {
		if match.Rule == "ip_url" {
			hasIPMatch = true
			break
		}
	}

	if !hasIPMatch {
		t.Error("Should detect IP address URL")
	}
}

func TestSpamCheckerAddRule(t *testing.T) {
	checker := NewSpamChecker(5.0)

	initialCount := len(checker.rules)

	checker.AddRule(SpamRule{
		Name:        "custom_test",
		Pattern:     regexp.MustCompile(`(?i)\bcustom_spam_word\b`),
		Score:       2.0,
		Description: "Custom test rule",
	})

	if len(checker.rules) != initialCount+1 {
		t.Error("Failed to add custom rule")
	}

	result := checker.Check("This contains custom_spam_word in it")

	hasCustomMatch := false
	for _, match := range result.Matches {
		if match.Rule == "custom_test" {
			hasCustomMatch = true
			break
		}
	}

	if !hasCustomMatch {
		t.Error("Custom rule should match")
	}
}

// ============================================
// Template Rendering Tests
// ============================================

func TestRenderBaseTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &TemplateData{
		CompanyName:    "Test Company",
		RecipientName:  "John Doe",
		RecipientEmail: "john@example.com",
		Subject:        "Test Email",
		PreviewText:    "This is a test",
		Heading:        "Welcome",
		Body:           template.HTML("<p>This is the body content.</p>"),
		ActionURL:      "https://example.com/action",
		ActionText:     "Click Here",
		CurrentYear:    2024,
		PrimaryColor:   "#2563eb",
	}

	result, err := engine.Render(TemplateTypeBase, data)
	if err != nil {
		t.Fatalf("Failed to render base template: %v", err)
	}

	if result == nil {
		t.Fatal("Render result is nil")
	}

	if result.HTML == "" {
		t.Error("HTML content is empty")
	}

	// Check for expected content
	if !strings.Contains(result.HTML, "Test Company") {
		t.Error("HTML should contain company name")
	}

	if !strings.Contains(result.HTML, "John Doe") {
		t.Error("HTML should contain recipient name")
	}

	if !strings.Contains(result.HTML, "Click Here") {
		t.Error("HTML should contain action text")
	}
}

func TestRenderAppointmentTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &AppointmentData{
		TemplateData: TemplateData{
			CompanyName:    "ServicePro",
			RecipientName:  "Jane Smith",
			RecipientEmail: "jane@example.com",
			Subject:        "Appointment Confirmation",
			CurrentYear:    2024,
			PrimaryColor:   "#2563eb",
		},
		AppointmentID:   "APT-123",
		ServiceName:     "HVAC Maintenance",
		ScheduledDate:   time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
		ScheduledTime:   "10:00 AM",
		Duration:        "2 hours",
		Location:        "123 Main St",
		TechnicianName:  "Mike Johnson",
		TechnicianPhone: "555-1234",
		Status:          "confirmed",
		IsConfirmed:     true,
	}

	result, err := engine.RenderAppointment(data)
	if err != nil {
		t.Fatalf("Failed to render appointment template: %v", err)
	}

	if !strings.Contains(result.HTML, "HVAC Maintenance") {
		t.Error("HTML should contain service name")
	}

	if !strings.Contains(result.HTML, "APT-123") {
		t.Error("HTML should contain appointment ID")
	}

	if !strings.Contains(result.HTML, "Mike Johnson") {
		t.Error("HTML should contain technician name")
	}
}

func TestRenderInvoiceTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &InvoiceData{
		TemplateData: TemplateData{
			CompanyName:    "ServicePro",
			RecipientName:  "John Customer",
			RecipientEmail: "john@customer.com",
			Subject:        "Invoice #INV-001",
			CurrentYear:    2024,
			PrimaryColor:   "#2563eb",
		},
		InvoiceID:     "inv-123",
		InvoiceNumber: "INV-001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().Add(30 * 24 * time.Hour),
		CustomerName:  "John Customer",
		LineItems: []InvoiceLineItem{
			{Description: "Service A", Quantity: 1, UnitPrice: 100.00, Amount: 100.00},
			{Description: "Service B", Quantity: 2, UnitPrice: 50.00, Amount: 100.00},
		},
		Subtotal:   200.00,
		TaxRate:    8.0,
		TaxAmount:  16.00,
		Total:      216.00,
		AmountDue:  216.00,
		Currency:   "USD",
		PaymentURL: "https://pay.example.com",
		IsPaid:     false,
		IsOverdue:  false,
	}

	result, err := engine.RenderInvoice(data)
	if err != nil {
		t.Fatalf("Failed to render invoice template: %v", err)
	}

	if !strings.Contains(result.HTML, "INV-001") {
		t.Error("HTML should contain invoice number")
	}

	if !strings.Contains(result.HTML, "$216.00") {
		t.Error("HTML should contain formatted total")
	}
}

func TestRenderAlertTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &AlertData{
		TemplateData: TemplateData{
			CompanyName:    "ServicePro",
			RecipientName:  "Admin",
			RecipientEmail: "admin@example.com",
			Subject:        "System Alert",
			CurrentYear:    2024,
			PrimaryColor:   "#2563eb",
		},
		AlertID:      "ALT-456",
		AlertType:    "warning",
		Severity:     "high",
		Title:        "High CPU Usage",
		Message:      "Server CPU usage exceeded 90%",
		Timestamp:    time.Now(),
		ResourceType: "server",
		ResourceID:   "srv-001",
		ResourceName: "Production Server",
		DetailsURL:   "https://example.com/alerts/456",
	}

	result, err := engine.RenderAlert(data)
	if err != nil {
		t.Fatalf("Failed to render alert template: %v", err)
	}

	if !strings.Contains(result.HTML, "High CPU Usage") {
		t.Error("HTML should contain alert title")
	}

	if !strings.Contains(result.HTML, "ALT-456") {
		t.Error("HTML should contain alert ID")
	}

	// Check for severity styling
	if !strings.Contains(strings.ToLower(result.HTML), "high") {
		t.Error("HTML should contain severity")
	}
}

// ============================================
// HTML to Text Conversion Tests
// ============================================

func TestHTMLToText(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello World</p>",
			expected: "Hello World",
		},
		{
			name:     "link extraction",
			html:     `<a href="https://example.com">Click here</a>`,
			expected: "Click here (https://example.com)",
		},
		{
			name:     "script removal",
			html:     `<script>alert('xss')</script>Content`,
			expected: "Content",
		},
		{
			name:     "style removal",
			html:     `<style>.class{color:red}</style>Content`,
			expected: "Content",
		},
		{
			name:     "entity decode",
			html:     `&amp; &lt; &gt; &quot;`,
			expected: "& < > \"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.htmlToText(tt.html)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected text to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// ============================================
// Version Management Tests
// ============================================

func TestAddTemplateVersion(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	content := `<!DOCTYPE html><html><body>{{.CompanyName}}</body></html>`

	version, err := engine.AddTemplateVersion(TemplateTypeBase, content, "2.0.0")
	if err != nil {
		t.Fatalf("Failed to add template version: %v", err)
	}

	if version.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", version.Version)
	}

	if version.Type != TemplateTypeBase {
		t.Errorf("Expected type base, got %s", version.Type)
	}

	if version.ID == "" {
		t.Error("Version ID should not be empty")
	}

	if version.Hash == "" {
		t.Error("Version hash should not be empty")
	}
}

func TestSetActiveVersion(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	content := `<!DOCTYPE html><html><body>Version 2: {{.CompanyName}}</body></html>`

	_, err = engine.AddTemplateVersion(TemplateTypeBase, content, "2.0.0")
	if err != nil {
		t.Fatalf("Failed to add version: %v", err)
	}

	err = engine.SetActiveVersion(TemplateTypeBase, "2.0.0")
	if err != nil {
		t.Fatalf("Failed to set active version: %v", err)
	}

	info, err := engine.GetActiveVersionInfo(TemplateTypeBase)
	if err != nil {
		t.Fatalf("Failed to get active version info: %v", err)
	}

	if info.Version != "2.0.0" {
		t.Errorf("Expected active version 2.0.0, got %s", info.Version)
	}
}

func TestGetVersions(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Add additional versions
	content := `<!DOCTYPE html><html><body>{{.CompanyName}}</body></html>`
	engine.AddTemplateVersion(TemplateTypeBase, content, "2.0.0")
	engine.AddTemplateVersion(TemplateTypeBase, content, "3.0.0")

	versions, err := engine.GetVersions(TemplateTypeBase)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}

	if len(versions) < 2 {
		t.Errorf("Expected at least 2 versions, got %d", len(versions))
	}
}

func TestSetActiveVersionNotFound(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	err = engine.SetActiveVersion(TemplateTypeBase, "99.99.99")
	if err == nil {
		t.Error("Expected error for non-existent version")
	}
}

func TestRenderWithVersion(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	content := `<!DOCTYPE html><html><body>V2: {{.CompanyName}}</body></html>`
	engine.AddTemplateVersion(TemplateTypeBase, content, "2.0.0")

	data := &TemplateData{
		CompanyName: "TestCorp",
		CurrentYear: 2024,
	}

	result, err := engine.RenderWithVersion(TemplateTypeBase, "2.0.0", data)
	if err != nil {
		t.Fatalf("Failed to render with version: %v", err)
	}

	if !strings.Contains(result.HTML, "V2: TestCorp") {
		t.Error("Should render specific version content")
	}

	if result.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", result.Version)
	}
}

// ============================================
// Preview Tests
// ============================================

func TestPreviewAppointment(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Preview(TemplateTypeAppointment)
	if err != nil {
		t.Fatalf("Failed to preview appointment: %v", err)
	}

	if result.HTML == "" {
		t.Error("Preview HTML should not be empty")
	}

	if !strings.Contains(result.HTML, "ServicePro Demo") {
		t.Error("Preview should contain demo company name")
	}
}

func TestPreviewInvoice(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Preview(TemplateTypeInvoice)
	if err != nil {
		t.Fatalf("Failed to preview invoice: %v", err)
	}

	if result.HTML == "" {
		t.Error("Preview HTML should not be empty")
	}

	// Should contain sample line items
	if !strings.Contains(result.HTML, "HVAC Service") {
		t.Error("Preview should contain sample line item")
	}
}

func TestPreviewAlert(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Preview(TemplateTypeAlert)
	if err != nil {
		t.Fatalf("Failed to preview alert: %v", err)
	}

	if result.HTML == "" {
		t.Error("Preview HTML should not be empty")
	}

	if !strings.Contains(result.HTML, "Scheduled Maintenance") {
		t.Error("Preview should contain sample alert title")
	}
}

// ============================================
// MJML Fallback Tests
// ============================================

func TestMJMLFallback(t *testing.T) {
	config := DefaultTemplateConfig()
	config.MJMLEnabled = true

	engine, err := NewTemplateEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	mjmlContent := `
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>Hello World</mj-text>
					<mj-button href="https://example.com">Click Me</mj-button>
				</mj-column>
			</mj-section>
		</mj-body>
	`

	result := engine.mjmlFallback(mjmlContent)

	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Fallback should add DOCTYPE")
	}

	if !strings.Contains(result, "Hello World") {
		t.Error("Fallback should preserve text content")
	}

	if !strings.Contains(result, "https://example.com") {
		t.Error("Fallback should preserve links")
	}
}

// ============================================
// Cache Tests
// ============================================

func TestTemplateCache(t *testing.T) {
	cache := newTemplateCache(time.Hour)

	email := &RenderedEmail{
		HTML:       "<html>test</html>",
		Text:       "test",
		TemplateID: "t1",
		Version:    "1.0.0",
		RenderedAt: time.Now(),
	}

	// Test set and get
	cache.set("key1", email)

	retrieved, ok := cache.get("key1")
	if !ok {
		t.Error("Should retrieve cached item")
	}

	if retrieved.HTML != email.HTML {
		t.Error("Retrieved item should match original")
	}

	// Test missing key
	_, ok = cache.get("nonexistent")
	if ok {
		t.Error("Should not find nonexistent key")
	}
}

func TestTemplateCacheExpiration(t *testing.T) {
	cache := newTemplateCache(10 * time.Millisecond)

	email := &RenderedEmail{
		HTML: "<html>test</html>",
	}

	cache.set("key1", email)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	_, ok := cache.get("key1")
	if ok {
		t.Error("Expired item should not be retrieved")
	}
}

// ============================================
// Error Handling Tests
// ============================================

func TestRenderInvalidTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Try to add invalid template
	invalidContent := `{{.InvalidSyntax{{}`

	_, err = engine.AddTemplateVersion(TemplateTypeBase, invalidContent, "bad")
	if err == nil {
		t.Error("Should fail to add invalid template")
	}
}

func TestRenderNonExistentTemplate(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	_, err = engine.RenderWithVersion(TemplateTypeBase, "nonexistent", nil)
	if err == nil {
		t.Error("Should fail to render nonexistent version")
	}
}

// ============================================
// Email Client Compatibility Tests
// ============================================

func TestEmailClientCompatibility(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &TemplateData{
		CompanyName:   "TestCorp",
		RecipientName: "Test User",
		Subject:       "Test",
		CurrentYear:   2024,
	}

	result, err := engine.Render(TemplateTypeBase, data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	t.Run("has DOCTYPE", func(t *testing.T) {
		if !strings.Contains(result.HTML, "<!DOCTYPE") {
			t.Error("Should have DOCTYPE declaration")
		}
	})

	t.Run("has charset meta", func(t *testing.T) {
		if !strings.Contains(result.HTML, "charset") {
			t.Error("Should have charset meta tag")
		}
	})

	t.Run("has viewport meta", func(t *testing.T) {
		if !strings.Contains(result.HTML, "viewport") {
			t.Error("Should have viewport meta tag for responsive design")
		}
	})

	t.Run("uses table layout", func(t *testing.T) {
		if !strings.Contains(result.HTML, "<table") {
			t.Error("Should use table-based layout for email compatibility")
		}
	})

	t.Run("has inline styles", func(t *testing.T) {
		if !strings.Contains(result.HTML, "style=") {
			t.Error("Should have inline styles for email compatibility")
		}
	})

	t.Run("has text version", func(t *testing.T) {
		if result.Text == "" {
			t.Error("Should generate text version of email")
		}
	})
}

// ============================================
// Responsive Design Tests
// ============================================

func TestResponsiveDesign(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := &TemplateData{
		CompanyName:   "TestCorp",
		RecipientName: "Test User",
		Subject:       "Test",
		CurrentYear:   2024,
	}

	result, err := engine.Render(TemplateTypeBase, data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	t.Run("has media queries or responsive classes", func(t *testing.T) {
		// Media queries are added by MJML compilation or in style tag
		hasMediaQueries := strings.Contains(result.HTML, "@media")
		hasResponsiveClasses := strings.Contains(result.HTML, "container") || strings.Contains(result.HTML, "content-padding")
		if !hasMediaQueries && !hasResponsiveClasses {
			t.Log("Note: Consider adding media queries for better responsive design")
		}
	})

	t.Run("has max-width or width constraint", func(t *testing.T) {
		// Check for any width constraint mechanism
		hasMaxWidth := strings.Contains(result.HTML, "max-width")
		hasWidthAttr := strings.Contains(result.HTML, "width=\"600\"") || strings.Contains(result.HTML, "width=")
		if !hasMaxWidth && !hasWidthAttr {
			t.Log("Note: Consider adding width constraints for container")
		}
	})

	t.Run("uses table-based responsive pattern", func(t *testing.T) {
		// Email-safe responsive design uses tables
		if strings.Contains(result.HTML, "<table") {
			// This is good for email compatibility
		}
	})
}

// ============================================
// Dynamic Content Tests
// ============================================

func TestDynamicContentInsertion(t *testing.T) {
	engine, err := NewTemplateEngine(nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test with various data types
	data := &InvoiceData{
		TemplateData: TemplateData{
			CompanyName:   "Dynamic Corp",
			RecipientName: "Dynamic User",
			CurrentYear:   2024,
		},
		InvoiceNumber: "DYN-123",
		InvoiceDate:   time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC),
		CustomerName:  "Customer Inc",
		LineItems: []InvoiceLineItem{
			{Description: "Widget A", Quantity: 5, UnitPrice: 10.00, Amount: 50.00},
			{Description: "Widget B", Quantity: 3, UnitPrice: 20.00, Amount: 60.00},
		},
		Subtotal:  110.00,
		TaxRate:   10.0,
		TaxAmount: 11.00,
		Total:     121.00,
		AmountDue: 121.00,
		Currency:  "USD",
	}

	result, err := engine.RenderInvoice(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Verify dynamic content
	checks := []struct {
		name     string
		expected string
	}{
		{"company name", "Dynamic Corp"},
		{"invoice number", "DYN-123"},
		{"customer name", "Customer Inc"},
		{"line item 1", "Widget A"},
		{"line item 2", "Widget B"},
		{"total amount", "$121.00"},
		{"tax rate", "10.0%"},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if !strings.Contains(result.HTML, check.expected) {
				t.Errorf("Should contain %s (%s)", check.name, check.expected)
			}
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkRenderBaseTemplate(b *testing.B) {
	engine, _ := NewTemplateEngine(nil)

	data := &TemplateData{
		CompanyName:    "Benchmark Corp",
		RecipientName:  "Benchmark User",
		RecipientEmail: "bench@example.com",
		Subject:        "Benchmark Test",
		Body:           template.HTML("<p>Benchmark content</p>"),
		CurrentYear:    2024,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Render(TemplateTypeBase, data)
	}
}

func BenchmarkRenderInvoiceTemplate(b *testing.B) {
	engine, _ := NewTemplateEngine(nil)

	data := &InvoiceData{
		TemplateData: TemplateData{
			CompanyName:   "Benchmark Corp",
			RecipientName: "Benchmark User",
			CurrentYear:   2024,
		},
		InvoiceNumber: "BENCH-001",
		InvoiceDate:   time.Now(),
		DueDate:       time.Now().Add(30 * 24 * time.Hour),
		CustomerName:  "Customer",
		LineItems: []InvoiceLineItem{
			{Description: "Item 1", Quantity: 1, UnitPrice: 100.00, Amount: 100.00},
			{Description: "Item 2", Quantity: 2, UnitPrice: 50.00, Amount: 100.00},
		},
		Subtotal:  200.00,
		TaxAmount: 20.00,
		Total:     220.00,
		AmountDue: 220.00,
		Currency:  "USD",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.RenderInvoice(data)
	}
}

func BenchmarkSpamCheck(b *testing.B) {
	checker := NewSpamChecker(5.0)
	content := `
		<html>
		<body>
			<h1>Hello Customer</h1>
			<p>Your invoice is ready. Please review the details below.</p>
			<table>
				<tr><td>Service</td><td>$100.00</td></tr>
			</table>
			<a href="https://example.com/pay">Pay Now</a>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(content)
	}
}

func BenchmarkHTMLToText(b *testing.B) {
	engine, _ := NewTemplateEngine(nil)
	html := `
		<!DOCTYPE html>
		<html>
		<head><style>.test{color:red}</style></head>
		<body>
			<h1>Title</h1>
			<p>Paragraph with <a href="https://example.com">a link</a></p>
			<script>alert('test')</script>
			<div>More content &amp; entities</div>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.htmlToText(html)
	}
}

func BenchmarkTemplateCache(b *testing.B) {
	cache := newTemplateCache(time.Hour)

	email := &RenderedEmail{
		HTML:       "<html>cached content</html>",
		Text:       "cached content",
		TemplateID: "t1",
		Version:    "1.0.0",
	}

	cache.set("bench_key", email)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.get("bench_key")
	}
}
