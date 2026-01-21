package templates

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// HTTP Handler for Template Previews
// =============================================================================

// PreviewHandler handles template preview requests
type PreviewHandler struct {
	renderer *Renderer
}

// NewPreviewHandler creates a new preview handler
func NewPreviewHandler() (*PreviewHandler, error) {
	renderer, err := NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	return &PreviewHandler{
		renderer: renderer,
	}, nil
}

// ServeHTTP implements http.Handler
func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet handles GET requests for template previews
func (h *PreviewHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	docTypeStr := r.URL.Query().Get("type")
	if docTypeStr == "" {
		docTypeStr = "quote"
	}

	docType := DocumentType(docTypeStr)
	if !isValidDocumentType(docType) {
		http.Error(w, "Invalid document type", http.StatusBadRequest)
		return
	}

	// Parse options from query params
	opts := &RenderOptions{
		InlineCSS: r.URL.Query().Get("inline") == "true",
		Branding:  h.parseBrandingFromQuery(r),
	}

	html, err := h.renderer.RenderPreview(docType, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handlePost handles POST requests for custom template rendering
func (h *PreviewHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	docTypeStr := r.URL.Query().Get("type")
	if docTypeStr == "" {
		http.Error(w, "Document type required", http.StatusBadRequest)
		return
	}

	docType := DocumentType(docTypeStr)
	if !isValidDocumentType(docType) {
		http.Error(w, "Invalid document type", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	opts := &RenderOptions{
		InlineCSS: req.InlineCSS,
		CustomCSS: req.CustomCSS,
	}

	if req.Branding != nil {
		opts.Branding = req.Branding
	}

	// Use custom data or preview data
	var html string
	var err error

	if req.Data != nil {
		html, err = h.renderer.Render(docType, req.Data, opts)
	} else {
		html, err = h.renderer.RenderPreview(docType, opts)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// PreviewRequest represents a preview request body
type PreviewRequest struct {
	Data      map[string]interface{} `json:"data,omitempty"`
	InlineCSS bool                   `json:"inline_css"`
	CustomCSS string                 `json:"custom_css,omitempty"`
	Branding  *BrandingOptions       `json:"branding,omitempty"`
}

// parseBrandingFromQuery extracts branding options from query parameters
func (h *PreviewHandler) parseBrandingFromQuery(r *http.Request) *BrandingOptions {
	branding := &BrandingOptions{}
	hasValue := false

	if v := r.URL.Query().Get("company_name"); v != "" {
		branding.CompanyName = v
		hasValue = true
	}
	if v := r.URL.Query().Get("company_tagline"); v != "" {
		branding.CompanyTagline = v
		hasValue = true
	}
	if v := r.URL.Query().Get("company_phone"); v != "" {
		branding.CompanyPhone = v
		hasValue = true
	}
	if v := r.URL.Query().Get("company_email"); v != "" {
		branding.CompanyEmail = v
		hasValue = true
	}
	if v := r.URL.Query().Get("primary_color"); v != "" {
		branding.PrimaryColor = v
		hasValue = true
	}
	if v := r.URL.Query().Get("footer_text"); v != "" {
		branding.FooterText = v
		hasValue = true
	}

	if !hasValue {
		return nil
	}
	return branding
}

// isValidDocumentType checks if the document type is valid
func isValidDocumentType(docType DocumentType) bool {
	switch docType {
	case DocumentTypeQuote, DocumentTypeInvoice, DocumentTypeReport:
		return true
	default:
		return false
	}
}

// =============================================================================
// CSS Handler
// =============================================================================

// CSSHandler serves template CSS
type CSSHandler struct {
	renderer *Renderer
}

// NewCSSHandler creates a new CSS handler
func NewCSSHandler() (*CSSHandler, error) {
	renderer, err := NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	return &CSSHandler{
		renderer: renderer,
	}, nil
}

// ServeHTTP implements http.Handler
func (h *CSSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docTypeStr := r.URL.Query().Get("type")

	var css string
	var err error

	if docTypeStr == "" || docTypeStr == "shared" {
		css, err = h.renderer.GetSharedCSS()
	} else {
		docType := DocumentType(docTypeStr)
		if !isValidDocumentType(docType) {
			http.Error(w, "Invalid document type", http.StatusBadRequest)
			return
		}
		css, err = h.renderer.GetCSS(docType)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(css))
}

// =============================================================================
// Template List Handler
// =============================================================================

// ListHandler returns available templates and their metadata
type ListHandler struct{}

// NewListHandler creates a new list handler
func NewListHandler() *ListHandler {
	return &ListHandler{}
}

// TemplateInfo describes a template
type TemplateInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PreviewURL  string `json:"preview_url"`
	CSSURL      string `json:"css_url"`
}

// ServeHTTP implements http.Handler
func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templates := []TemplateInfo{
		{
			Type:        "quote",
			Name:        "Quote",
			Description: "Professional quote template with line items, totals, and acceptance section",
			PreviewURL:  "/api/templates/preview?type=quote",
			CSSURL:      "/api/templates/css?type=quote",
		},
		{
			Type:        "invoice",
			Name:        "Invoice",
			Description: "Invoice template with payment tracking, line items, and payment instructions",
			PreviewURL:  "/api/templates/preview?type=invoice",
			CSSURL:      "/api/templates/css?type=invoice",
		},
		{
			Type:        "report",
			Name:        "Service Report",
			Description: "Service report template with equipment, tasks, findings, and signatures",
			PreviewURL:  "/api/templates/preview?type=report",
			CSSURL:      "/api/templates/css?type=report",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(templates)
}

// =============================================================================
// Route Registration Helper
// =============================================================================

// RegisterRoutes registers all template-related HTTP routes
func RegisterRoutes(mux *http.ServeMux, basePath string) error {
	previewHandler, err := NewPreviewHandler()
	if err != nil {
		return fmt.Errorf("failed to create preview handler: %w", err)
	}

	cssHandler, err := NewCSSHandler()
	if err != nil {
		return fmt.Errorf("failed to create CSS handler: %w", err)
	}

	listHandler := NewListHandler()

	mux.Handle(basePath+"/preview", previewHandler)
	mux.Handle(basePath+"/css", cssHandler)
	mux.Handle(basePath+"/list", listHandler)

	return nil
}
