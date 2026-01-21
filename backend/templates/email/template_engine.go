package email

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed *.html *.mjml
var templateFS embed.FS

// Standard errors
var (
	ErrTemplateNotFound  = errors.New("template not found")
	ErrTemplateInvalid   = errors.New("template is invalid")
	ErrVersionNotFound   = errors.New("template version not found")
	ErrRenderFailed      = errors.New("template render failed")
	ErrMJMLCompileFailed = errors.New("MJML compilation failed")
	ErrSpamScoreHigh     = errors.New("email spam score too high")
)

// TemplateType represents the type of email template
type TemplateType string

const (
	TemplateTypeBase         TemplateType = "base"
	TemplateTypeAppointment  TemplateType = "appointment"
	TemplateTypeInvoice      TemplateType = "invoice"
	TemplateTypeAlert        TemplateType = "alert"
	TemplateTypeWelcome      TemplateType = "welcome"
	TemplateTypeReset        TemplateType = "reset"
	TemplateTypeNotification TemplateType = "notification"
)

// TemplateVersion represents a versioned template
type TemplateVersion struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Type        TemplateType           `json:"type"`
	Name        string                 `json:"name"`
	Subject     string                 `json:"subject"`
	HTMLContent string                 `json:"html_content"`
	MJMLContent string                 `json:"mjml_content,omitempty"`
	TextContent string                 `json:"text_content,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	IsActive    bool                   `json:"is_active"`
	Hash        string                 `json:"hash"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateData represents common data for all templates
type TemplateData struct {
	// Common fields
	CompanyName    string `json:"company_name"`
	CompanyLogo    string `json:"company_logo"`
	CompanyAddress string `json:"company_address"`
	CompanyPhone   string `json:"company_phone"`
	CompanyEmail   string `json:"company_email"`
	CompanyWebsite string `json:"company_website"`

	// Recipient fields
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`

	// Content fields
	Subject     string        `json:"subject"`
	PreviewText string        `json:"preview_text"`
	Heading     string        `json:"heading"`
	Body        template.HTML `json:"body"`
	Footer      string        `json:"footer"`

	// Action button
	ActionURL  string `json:"action_url"`
	ActionText string `json:"action_text"`

	// Custom data
	Custom map[string]interface{} `json:"custom,omitempty"`

	// Tracking
	TrackingID     string `json:"tracking_id,omitempty"`
	UnsubscribeURL string `json:"unsubscribe_url,omitempty"`

	// Styling
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`

	// Date/Time
	CurrentYear int    `json:"current_year"`
	Timestamp   string `json:"timestamp"`
}

// AppointmentData represents appointment-specific template data
type AppointmentData struct {
	TemplateData

	// Appointment details
	AppointmentID      string    `json:"appointment_id"`
	ServiceName        string    `json:"service_name"`
	ServiceDescription string    `json:"service_description"`
	ScheduledDate      time.Time `json:"scheduled_date"`
	ScheduledTime      string    `json:"scheduled_time"`
	Duration           string    `json:"duration"`
	Location           string    `json:"location"`
	TechnicianName     string    `json:"technician_name"`
	TechnicianPhone    string    `json:"technician_phone"`
	Notes              string    `json:"notes"`

	// Status
	Status          string `json:"status"`
	IsConfirmed     bool   `json:"is_confirmed"`
	ConfirmationURL string `json:"confirmation_url"`
	RescheduleURL   string `json:"reschedule_url"`
	CancelURL       string `json:"cancel_url"`
}

// InvoiceData represents invoice-specific template data
type InvoiceData struct {
	TemplateData

	// Invoice details
	InvoiceID     string    `json:"invoice_id"`
	InvoiceNumber string    `json:"invoice_number"`
	InvoiceDate   time.Time `json:"invoice_date"`
	DueDate       time.Time `json:"due_date"`

	// Customer
	CustomerName    string `json:"customer_name"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email"`

	// Line items
	LineItems []InvoiceLineItem `json:"line_items"`

	// Totals
	Subtotal   float64 `json:"subtotal"`
	TaxRate    float64 `json:"tax_rate"`
	TaxAmount  float64 `json:"tax_amount"`
	Discount   float64 `json:"discount"`
	Total      float64 `json:"total"`
	AmountPaid float64 `json:"amount_paid"`
	AmountDue  float64 `json:"amount_due"`
	Currency   string  `json:"currency"`

	// Payment
	PaymentURL     string   `json:"payment_url"`
	PaymentMethods []string `json:"payment_methods"`
	PaymentTerms   string   `json:"payment_terms"`

	// Status
	Status    string `json:"status"`
	IsPaid    bool   `json:"is_paid"`
	IsOverdue bool   `json:"is_overdue"`
}

// InvoiceLineItem represents a line item in an invoice
type InvoiceLineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// AlertData represents alert-specific template data
type AlertData struct {
	TemplateData

	// Alert details
	AlertID   string    `json:"alert_id"`
	AlertType string    `json:"alert_type"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`

	// Context
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`

	// Actions
	AcknowledgeURL string `json:"acknowledge_url"`
	ResolveURL     string `json:"resolve_url"`
	DetailsURL     string `json:"details_url"`

	// Additional info
	Metadata map[string]string `json:"metadata"`
}

// ============================================
// Template Engine
// ============================================

// TemplateEngine handles email template rendering
type TemplateEngine struct {
	templates     map[string]*template.Template
	versions      map[string]map[string]*TemplateVersion // type -> version -> template
	activeVersion map[string]string                      // type -> active version
	mjmlEnabled   bool
	mjmlPath      string
	spamChecker   *SpamChecker
	cache         *templateCache
	mu            sync.RWMutex

	// Template functions
	funcMap template.FuncMap

	// Configuration
	config *TemplateConfig
}

// TemplateConfig holds template engine configuration
type TemplateConfig struct {
	// Paths
	TemplateDir string `json:"template_dir"`

	// MJML
	MJMLEnabled bool   `json:"mjml_enabled"`
	MJMLPath    string `json:"mjml_path"`
	MJMLMinify  bool   `json:"mjml_minify"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`

	// Spam checking
	SpamCheckEnabled bool    `json:"spam_check_enabled"`
	MaxSpamScore     float64 `json:"max_spam_score"`

	// Defaults
	DefaultPrimaryColor   string `json:"default_primary_color"`
	DefaultSecondaryColor string `json:"default_secondary_color"`
	CompanyName           string `json:"company_name"`
	CompanyLogo           string `json:"company_logo"`
	CompanyWebsite        string `json:"company_website"`
}

// DefaultTemplateConfig returns default configuration
func DefaultTemplateConfig() *TemplateConfig {
	return &TemplateConfig{
		TemplateDir:           "templates/email",
		MJMLEnabled:           true,
		MJMLPath:              "mjml",
		MJMLMinify:            true,
		CacheEnabled:          true,
		CacheTTL:              time.Hour,
		SpamCheckEnabled:      true,
		MaxSpamScore:          5.0,
		DefaultPrimaryColor:   "#2563eb",
		DefaultSecondaryColor: "#64748b",
		CompanyName:           "ServicePro",
		CompanyLogo:           "",
		CompanyWebsite:        "",
	}
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(config *TemplateConfig) (*TemplateEngine, error) {
	if config == nil {
		config = DefaultTemplateConfig()
	}

	engine := &TemplateEngine{
		templates:     make(map[string]*template.Template),
		versions:      make(map[string]map[string]*TemplateVersion),
		activeVersion: make(map[string]string),
		mjmlEnabled:   config.MJMLEnabled,
		mjmlPath:      config.MJMLPath,
		config:        config,
		funcMap:       createTemplateFuncs(),
	}

	// Initialize cache
	if config.CacheEnabled {
		engine.cache = newTemplateCache(config.CacheTTL)
	}

	// Initialize spam checker
	if config.SpamCheckEnabled {
		engine.spamChecker = NewSpamChecker(config.MaxSpamScore)
	}

	// Load embedded templates
	if err := engine.loadEmbeddedTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return engine, nil
}

// createTemplateFuncs creates the template function map
func createTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// String functions
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join":      strings.Join,

		// Date/Time functions
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"formatDateShort": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"formatDateLong": func(t time.Time) string {
			return t.Format("Monday, January 2, 2006")
		},
		"formatTime": func(t time.Time) string {
			return t.Format("3:04 PM")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 at 3:04 PM")
		},
		"now": time.Now,
		"year": func() int {
			return time.Now().Year()
		},

		// Number functions
		"formatCurrency": func(amount float64, currency string) string {
			symbol := "$"
			switch currency {
			case "EUR":
				symbol = "€"
			case "GBP":
				symbol = "£"
			case "JPY":
				symbol = "¥"
			}
			return fmt.Sprintf("%s%.2f", symbol, amount)
		},
		"formatPercent": func(value float64) string {
			return fmt.Sprintf("%.1f%%", value)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
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

		// Conditional functions
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"ternary": func(cond bool, trueVal, falseVal interface{}) interface{} {
			if cond {
				return trueVal
			}
			return falseVal
		},

		// HTML functions
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},

		// Collection functions
		"first": func(arr []interface{}) interface{} {
			if len(arr) > 0 {
				return arr[0]
			}
			return nil
		},
		"last": func(arr []interface{}) interface{} {
			if len(arr) > 0 {
				return arr[len(arr)-1]
			}
			return nil
		},
		"len": func(arr interface{}) int {
			switch v := arr.(type) {
			case []interface{}:
				return len(v)
			case []string:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},

		// Color functions
		"lighten": func(color string, percent int) string {
			// Simplified - in production use a proper color library
			return color
		},
		"darken": func(color string, percent int) string {
			return color
		},
	}
}

// loadEmbeddedTemplates loads templates from the embedded filesystem
func (e *TemplateEngine) loadEmbeddedTemplates() error {
	entries, err := fs.ReadDir(templateFS, ".")
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".html") && !strings.HasSuffix(name, ".mjml") {
			continue
		}

		content, err := fs.ReadFile(templateFS, name)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", name, err)
		}

		// Compile MJML if needed
		htmlContent := string(content)
		if strings.HasSuffix(name, ".mjml") && e.mjmlEnabled {
			compiled, err := e.compileMJML(htmlContent)
			if err != nil {
				return fmt.Errorf("failed to compile MJML %s: %w", name, err)
			}
			htmlContent = compiled
			name = strings.TrimSuffix(name, ".mjml") + ".html"
		}

		// Parse template
		tmpl, err := template.New(name).Funcs(e.funcMap).Parse(htmlContent)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		e.templates[name] = tmpl

		// Create initial version
		templateType := e.getTemplateType(name)
		version := &TemplateVersion{
			ID:          generateVersionID(),
			Version:     "1.0.0",
			Type:        templateType,
			Name:        name,
			HTMLContent: htmlContent,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			IsActive:    true,
			Hash:        hashContent(htmlContent),
		}

		e.addVersion(templateType, version)
	}

	return nil
}

// getTemplateType determines template type from filename
func (e *TemplateEngine) getTemplateType(name string) TemplateType {
	name = strings.TrimSuffix(name, ".html")
	name = strings.TrimSuffix(name, ".mjml")

	switch {
	case strings.Contains(name, "base"):
		return TemplateTypeBase
	case strings.Contains(name, "appointment"):
		return TemplateTypeAppointment
	case strings.Contains(name, "invoice"):
		return TemplateTypeInvoice
	case strings.Contains(name, "alert"):
		return TemplateTypeAlert
	case strings.Contains(name, "welcome"):
		return TemplateTypeWelcome
	case strings.Contains(name, "reset"):
		return TemplateTypeReset
	default:
		return TemplateTypeNotification
	}
}

// compileMJML compiles MJML to HTML
func (e *TemplateEngine) compileMJML(mjml string) (string, error) {
	if !e.mjmlEnabled {
		return mjml, nil
	}

	// Try to use mjml CLI
	cmd := exec.Command(e.mjmlPath, "-s")
	if e.config.MJMLMinify {
		cmd = exec.Command(e.mjmlPath, "-s", "--config.minify", "true")
	}

	cmd.Stdin = strings.NewReader(mjml)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If mjml CLI not available, use fallback
		return e.mjmlFallback(mjml), nil
	}

	return stdout.String(), nil
}

// mjmlFallback provides basic MJML-like responsive structure without CLI
func (e *TemplateEngine) mjmlFallback(mjml string) string {
	// This is a simplified fallback - in production, use a proper MJML library
	// or ensure mjml CLI is available

	// Basic conversion of MJML tags to HTML
	html := mjml

	// Convert mj-body
	html = regexp.MustCompile(`<mj-body[^>]*>`).ReplaceAllString(html, `<body style="background-color:#f4f4f4;">`)
	html = strings.ReplaceAll(html, "</mj-body>", "</body>")

	// Convert mj-section
	html = regexp.MustCompile(`<mj-section[^>]*>`).ReplaceAllString(html, `<table width="100%" cellpadding="0" cellspacing="0"><tr><td>`)
	html = strings.ReplaceAll(html, "</mj-section>", "</td></tr></table>")

	// Convert mj-column
	html = regexp.MustCompile(`<mj-column[^>]*>`).ReplaceAllString(html, `<td style="padding:20px;">`)
	html = strings.ReplaceAll(html, "</mj-column>", "</td>")

	// Convert mj-text
	html = regexp.MustCompile(`<mj-text[^>]*>`).ReplaceAllString(html, `<div style="font-family:Arial,sans-serif;font-size:14px;line-height:1.5;">`)
	html = strings.ReplaceAll(html, "</mj-text>", "</div>")

	// Convert mj-button
	buttonRegex := regexp.MustCompile(`<mj-button[^>]*href="([^"]*)"[^>]*>([^<]*)</mj-button>`)
	html = buttonRegex.ReplaceAllString(html, `<a href="$1" style="background-color:#2563eb;color:#ffffff;padding:12px 24px;text-decoration:none;border-radius:4px;display:inline-block;">$2</a>`)

	// Convert mj-image
	imageRegex := regexp.MustCompile(`<mj-image[^>]*src="([^"]*)"[^>]*/?>`)
	html = imageRegex.ReplaceAllString(html, `<img src="$1" style="max-width:100%;height:auto;" />`)

	// Convert mj-divider
	html = regexp.MustCompile(`<mj-divider[^>]*/>`).ReplaceAllString(html, `<hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0;" />`)

	// Remove remaining mj- tags
	html = regexp.MustCompile(`<mj-[^>]*>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`</mj-[^>]*>`).ReplaceAllString(html, "")

	// Wrap in basic email HTML structure
	if !strings.Contains(html, "<!DOCTYPE") {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Email</title>
</head>
%s
</html>`, html)
	}

	return html
}

// Render renders a template with the given data
func (e *TemplateEngine) Render(templateType TemplateType, data interface{}) (*RenderedEmail, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get active version
	version, err := e.getActiveVersion(templateType)
	if err != nil {
		return nil, err
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s:%s:%s", templateType, version.Version, hashData(data))
	if e.cache != nil {
		if cached, ok := e.cache.get(cacheKey); ok {
			return cached, nil
		}
	}

	// Get template
	tmpl, ok := e.templates[version.Name]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	// Render HTML
	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	htmlContent := htmlBuf.String()

	// Generate text version
	textContent := e.htmlToText(htmlContent)

	// Check spam score
	if e.spamChecker != nil {
		result := e.spamChecker.Check(htmlContent)
		if result.Score > e.config.MaxSpamScore {
			return nil, fmt.Errorf("%w: score %.1f exceeds max %.1f",
				ErrSpamScoreHigh, result.Score, e.config.MaxSpamScore)
		}
	}

	rendered := &RenderedEmail{
		HTML:       htmlContent,
		Text:       textContent,
		TemplateID: version.ID,
		Version:    version.Version,
		RenderedAt: time.Now(),
	}

	// Cache result
	if e.cache != nil {
		e.cache.set(cacheKey, rendered)
	}

	return rendered, nil
}

// RenderWithVersion renders a specific version of a template
func (e *TemplateEngine) RenderWithVersion(templateType TemplateType, version string, data interface{}) (*RenderedEmail, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	versions, ok := e.versions[string(templateType)]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	tmplVersion, ok := versions[version]
	if !ok {
		return nil, ErrVersionNotFound
	}

	// Parse and render
	tmpl, err := template.New("temp").Funcs(e.funcMap).Parse(tmplVersion.HTMLContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateInvalid, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	htmlContent := buf.String()

	return &RenderedEmail{
		HTML:       htmlContent,
		Text:       e.htmlToText(htmlContent),
		TemplateID: tmplVersion.ID,
		Version:    tmplVersion.Version,
		RenderedAt: time.Now(),
	}, nil
}

// RenderedEmail represents a rendered email
type RenderedEmail struct {
	HTML       string    `json:"html"`
	Text       string    `json:"text"`
	TemplateID string    `json:"template_id"`
	Version    string    `json:"version"`
	RenderedAt time.Time `json:"rendered_at"`
}

// htmlToText converts HTML to plain text
func (e *TemplateEngine) htmlToText(html string) string {
	// Remove script and style tags
	html = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`).ReplaceAllString(html, "")

	// Replace common elements
	html = regexp.MustCompile(`<br\s*/?>|</p>|</div>|</tr>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`</td>`).ReplaceAllString(html, "\t")
	html = regexp.MustCompile(`<h[1-6][^>]*>`).ReplaceAllString(html, "\n\n")
	html = regexp.MustCompile(`</h[1-6]>`).ReplaceAllString(html, "\n")

	// Extract link URLs
	linkRegex := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	html = linkRegex.ReplaceAllString(html, "$2 ($1)")

	// Remove all remaining HTML tags
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")

	// Decode HTML entities
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")

	// Clean up whitespace
	html = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(html, "\n\n")
	html = strings.TrimSpace(html)

	return html
}

// ============================================
// Version Management
// ============================================

// addVersion adds a new template version
func (e *TemplateEngine) addVersion(templateType TemplateType, version *TemplateVersion) {
	typeStr := string(templateType)

	if e.versions[typeStr] == nil {
		e.versions[typeStr] = make(map[string]*TemplateVersion)
	}

	e.versions[typeStr][version.Version] = version

	if version.IsActive {
		e.activeVersion[typeStr] = version.Version
	}
}

// AddTemplateVersion adds a new version of a template
func (e *TemplateEngine) AddTemplateVersion(templateType TemplateType, content string, version string) (*TemplateVersion, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Compile MJML if needed
	htmlContent := content
	mjmlContent := ""
	if strings.Contains(content, "<mj-") {
		mjmlContent = content
		var err error
		htmlContent, err = e.compileMJML(content)
		if err != nil {
			return nil, fmt.Errorf("failed to compile MJML: %w", err)
		}
	}

	// Validate template
	_, err := template.New("validate").Funcs(e.funcMap).Parse(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateInvalid, err)
	}

	// Create version
	v := &TemplateVersion{
		ID:          generateVersionID(),
		Version:     version,
		Type:        templateType,
		Name:        fmt.Sprintf("%s_v%s.html", templateType, version),
		HTMLContent: htmlContent,
		MJMLContent: mjmlContent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    false,
		Hash:        hashContent(htmlContent),
	}

	e.addVersion(templateType, v)

	return v, nil
}

// SetActiveVersion sets the active version for a template type
func (e *TemplateEngine) SetActiveVersion(templateType TemplateType, version string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	typeStr := string(templateType)
	versions, ok := e.versions[typeStr]
	if !ok {
		return ErrTemplateNotFound
	}

	v, ok := versions[version]
	if !ok {
		return ErrVersionNotFound
	}

	// Deactivate current active version
	if activeVersion, exists := e.activeVersion[typeStr]; exists {
		if av, ok := versions[activeVersion]; ok {
			av.IsActive = false
		}
	}

	// Activate new version
	v.IsActive = true
	v.UpdatedAt = time.Now()
	e.activeVersion[typeStr] = version

	// Update templates map
	tmpl, err := template.New(v.Name).Funcs(e.funcMap).Parse(v.HTMLContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	e.templates[v.Name] = tmpl

	return nil
}

// getActiveVersion gets the active version for a template type
func (e *TemplateEngine) getActiveVersion(templateType TemplateType) (*TemplateVersion, error) {
	typeStr := string(templateType)

	versionStr, ok := e.activeVersion[typeStr]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	versions, ok := e.versions[typeStr]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	version, ok := versions[versionStr]
	if !ok {
		return nil, ErrVersionNotFound
	}

	return version, nil
}

// GetVersions gets all versions for a template type
func (e *TemplateEngine) GetVersions(templateType TemplateType) ([]*TemplateVersion, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	versions, ok := e.versions[string(templateType)]
	if !ok {
		return nil, ErrTemplateNotFound
	}

	result := make([]*TemplateVersion, 0, len(versions))
	for _, v := range versions {
		result = append(result, v)
	}

	return result, nil
}

// GetActiveVersionInfo returns info about the active version
func (e *TemplateEngine) GetActiveVersionInfo(templateType TemplateType) (*TemplateVersion, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.getActiveVersion(templateType)
}

// ============================================
// Template Cache
// ============================================

type templateCache struct {
	items map[string]*cacheItem
	ttl   time.Duration
	mu    sync.RWMutex
}

type cacheItem struct {
	email     *RenderedEmail
	expiresAt time.Time
}

func newTemplateCache(ttl time.Duration) *templateCache {
	cache := &templateCache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *templateCache) get(key string) (*RenderedEmail, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.email, true
}

func (c *templateCache) set(key string, email *RenderedEmail) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		email:     email,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *templateCache) cleanup() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// ============================================
// Spam Checker
// ============================================

// SpamChecker checks email content for spam indicators
type SpamChecker struct {
	maxScore float64
	rules    []SpamRule
}

// SpamRule represents a spam detection rule
type SpamRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Score       float64
	Description string
}

// SpamCheckResult represents the result of a spam check
type SpamCheckResult struct {
	Score       float64         `json:"score"`
	MaxScore    float64         `json:"max_score"`
	Passed      bool            `json:"passed"`
	Matches     []SpamRuleMatch `json:"matches"`
	Suggestions []string        `json:"suggestions"`
}

// SpamRuleMatch represents a matched spam rule
type SpamRuleMatch struct {
	Rule        string  `json:"rule"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

// NewSpamChecker creates a new spam checker
func NewSpamChecker(maxScore float64) *SpamChecker {
	return &SpamChecker{
		maxScore: maxScore,
		rules:    defaultSpamRules(),
	}
}

// defaultSpamRules returns default spam detection rules
func defaultSpamRules() []SpamRule {
	return []SpamRule{
		// Urgency and pressure
		{Name: "urgency", Pattern: regexp.MustCompile(`(?i)\b(urgent|act now|limited time|expires|hurry|immediately|instant)\b`), Score: 0.5, Description: "Urgency language"},
		{Name: "pressure", Pattern: regexp.MustCompile(`(?i)\b(don't miss|last chance|final notice|deadline)\b`), Score: 0.4, Description: "Pressure tactics"},

		// Money-related
		{Name: "money", Pattern: regexp.MustCompile(`(?i)\b(free|winner|prize|congratulations|claim|award|lottery|cash)\b`), Score: 0.8, Description: "Money/prize language"},
		{Name: "discount", Pattern: regexp.MustCompile(`(?i)\b(discount|save|sale|offer|deal|bargain|cheap)\b`), Score: 0.3, Description: "Excessive discount language"},

		// Suspicious phrases
		{Name: "suspicious", Pattern: regexp.MustCompile(`(?i)\b(click here|click below|verify your account|confirm your identity)\b`), Score: 0.6, Description: "Suspicious phrases"},
		{Name: "threat", Pattern: regexp.MustCompile(`(?i)\b(suspended|terminated|blocked|unauthorized|violation)\b`), Score: 0.5, Description: "Threat language"},

		// ALL CAPS abuse
		{Name: "caps", Pattern: regexp.MustCompile(`[A-Z]{5,}`), Score: 0.3, Description: "Excessive capitals"},

		// Multiple exclamation marks
		{Name: "exclamation", Pattern: regexp.MustCompile(`!{2,}`), Score: 0.3, Description: "Multiple exclamation marks"},

		// Suspicious formatting
		{Name: "hidden_text", Pattern: regexp.MustCompile(`(?i)display:\s*none|visibility:\s*hidden|font-size:\s*0`), Score: 1.0, Description: "Hidden text"},
		{Name: "tiny_text", Pattern: regexp.MustCompile(`(?i)font-size:\s*[0-5]px`), Score: 0.8, Description: "Tiny text"},

		// Image-heavy (ratio check would need more context)
		{Name: "image_only", Pattern: regexp.MustCompile(`(?i)^<img[^>]+>$`), Score: 1.0, Description: "Image-only content"},

		// Suspicious URLs
		{Name: "ip_url", Pattern: regexp.MustCompile(`https?://\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`), Score: 1.5, Description: "IP address URL"},
		{Name: "suspicious_tld", Pattern: regexp.MustCompile(`(?i)https?://[^\s/]+\.(xyz|top|club|work|tk|ml|ga|cf|gq)\b`), Score: 0.8, Description: "Suspicious TLD"},
	}
}

// Check checks email content for spam indicators
func (s *SpamChecker) Check(content string) *SpamCheckResult {
	result := &SpamCheckResult{
		MaxScore: s.maxScore,
		Matches:  make([]SpamRuleMatch, 0),
	}

	for _, rule := range s.rules {
		if rule.Pattern.MatchString(content) {
			matches := rule.Pattern.FindAllString(content, -1)
			// Increase score based on number of matches, with diminishing returns
			matchScore := rule.Score * (1 + float64(len(matches)-1)*0.2)

			result.Score += matchScore
			result.Matches = append(result.Matches, SpamRuleMatch{
				Rule:        rule.Name,
				Score:       matchScore,
				Description: rule.Description,
			})
		}
	}

	result.Passed = result.Score <= s.maxScore

	// Add suggestions
	if !result.Passed {
		result.Suggestions = s.getSuggestions(result.Matches)
	}

	return result
}

// getSuggestions returns suggestions to improve spam score
func (s *SpamChecker) getSuggestions(matches []SpamRuleMatch) []string {
	suggestions := make([]string, 0)

	for _, match := range matches {
		switch match.Rule {
		case "urgency", "pressure":
			suggestions = append(suggestions, "Reduce urgency language - use softer calls to action")
		case "money":
			suggestions = append(suggestions, "Avoid prize/lottery language")
		case "caps":
			suggestions = append(suggestions, "Reduce use of ALL CAPS")
		case "exclamation":
			suggestions = append(suggestions, "Use single exclamation marks")
		case "hidden_text", "tiny_text":
			suggestions = append(suggestions, "Remove hidden or tiny text")
		case "ip_url":
			suggestions = append(suggestions, "Use domain names instead of IP addresses")
		case "suspicious":
			suggestions = append(suggestions, "Rephrase 'click here' to more specific text")
		}
	}

	return suggestions
}

// AddRule adds a custom spam rule
func (s *SpamChecker) AddRule(rule SpamRule) {
	s.rules = append(s.rules, rule)
}

// ============================================
// Utility Functions
// ============================================

func generateVersionID() string {
	return fmt.Sprintf("tv_%d", time.Now().UnixNano())
}

func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8])
}

func hashData(data interface{}) string {
	bytes, _ := json.Marshal(data)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:8])
}

// ============================================
// Convenience Methods
// ============================================

// RenderAppointment renders an appointment email
func (e *TemplateEngine) RenderAppointment(data *AppointmentData) (*RenderedEmail, error) {
	// Set defaults
	data.CurrentYear = time.Now().Year()
	if data.PrimaryColor == "" {
		data.PrimaryColor = e.config.DefaultPrimaryColor
	}

	return e.Render(TemplateTypeAppointment, data)
}

// RenderInvoice renders an invoice email
func (e *TemplateEngine) RenderInvoice(data *InvoiceData) (*RenderedEmail, error) {
	// Set defaults
	data.CurrentYear = time.Now().Year()
	if data.PrimaryColor == "" {
		data.PrimaryColor = e.config.DefaultPrimaryColor
	}
	if data.Currency == "" {
		data.Currency = "USD"
	}

	return e.Render(TemplateTypeInvoice, data)
}

// RenderAlert renders an alert email
func (e *TemplateEngine) RenderAlert(data *AlertData) (*RenderedEmail, error) {
	// Set defaults
	data.CurrentYear = time.Now().Year()
	if data.PrimaryColor == "" {
		data.PrimaryColor = e.config.DefaultPrimaryColor
	}

	return e.Render(TemplateTypeAlert, data)
}

// Preview generates a preview of a template with sample data
func (e *TemplateEngine) Preview(templateType TemplateType) (*RenderedEmail, error) {
	var data interface{}

	switch templateType {
	case TemplateTypeAppointment:
		data = &AppointmentData{
			TemplateData: TemplateData{
				CompanyName:    "ServicePro Demo",
				RecipientName:  "John Doe",
				RecipientEmail: "john@example.com",
				Subject:        "Appointment Confirmation",
				PreviewText:    "Your appointment has been confirmed",
				CurrentYear:    time.Now().Year(),
				PrimaryColor:   e.config.DefaultPrimaryColor,
			},
			AppointmentID:  "APT-123456",
			ServiceName:    "HVAC Maintenance",
			ScheduledDate:  time.Now().Add(24 * time.Hour),
			ScheduledTime:  "10:00 AM",
			Duration:       "2 hours",
			Location:       "123 Main St, Anytown, USA",
			TechnicianName: "Mike Smith",
			Status:         "confirmed",
		}
	case TemplateTypeInvoice:
		data = &InvoiceData{
			TemplateData: TemplateData{
				CompanyName:    "ServicePro Demo",
				RecipientName:  "Jane Smith",
				RecipientEmail: "jane@example.com",
				Subject:        "Invoice #INV-2024-001",
				PreviewText:    "Your invoice is ready",
				CurrentYear:    time.Now().Year(),
				PrimaryColor:   e.config.DefaultPrimaryColor,
			},
			InvoiceNumber: "INV-2024-001",
			InvoiceDate:   time.Now(),
			DueDate:       time.Now().Add(30 * 24 * time.Hour),
			LineItems: []InvoiceLineItem{
				{Description: "HVAC Service", Quantity: 1, UnitPrice: 150.00, Amount: 150.00},
				{Description: "Filter Replacement", Quantity: 2, UnitPrice: 25.00, Amount: 50.00},
			},
			Subtotal:  200.00,
			TaxRate:   8.0,
			TaxAmount: 16.00,
			Total:     216.00,
			AmountDue: 216.00,
			Currency:  "USD",
		}
	case TemplateTypeAlert:
		data = &AlertData{
			TemplateData: TemplateData{
				CompanyName:    "ServicePro Demo",
				RecipientName:  "Admin User",
				RecipientEmail: "admin@example.com",
				Subject:        "System Alert",
				PreviewText:    "Action required",
				CurrentYear:    time.Now().Year(),
				PrimaryColor:   e.config.DefaultPrimaryColor,
			},
			AlertID:      "ALT-789",
			AlertType:    "warning",
			Severity:     "high",
			Title:        "Scheduled Maintenance",
			Message:      "The system will undergo maintenance tonight from 2-4 AM EST.",
			Timestamp:    time.Now(),
			ResourceType: "system",
		}
	default:
		data = &TemplateData{
			CompanyName:    "ServicePro Demo",
			RecipientName:  "User",
			RecipientEmail: "user@example.com",
			Subject:        "Notification",
			CurrentYear:    time.Now().Year(),
			PrimaryColor:   e.config.DefaultPrimaryColor,
		}
	}

	return e.Render(templateType, data)
}
