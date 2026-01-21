package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

// =============================================================================
// Generator Service
// =============================================================================

// Generator is the main PDF generation service
type Generator struct {
	config   *Config
	registry *TemplateRegistry
	s3Client *S3Client
	logger   Logger

	// Worker pool for concurrent generation
	workerPool chan struct{}
	mu         sync.RWMutex

	// Metrics
	generatedCount int64
	errorCount     int64
}

// NewGenerator creates a new PDF generator service
func NewGenerator(ctx context.Context, config *Config, logger Logger) (*Generator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = &DefaultLogger{}
	}

	g := &Generator{
		config:     config,
		registry:   NewTemplateRegistry(),
		logger:     logger,
		workerPool: make(chan struct{}, config.Performance.MaxConcurrent),
	}

	// Initialize S3 client if needed
	if config.Storage.Type == "s3" || config.Storage.Type == "both" {
		s3Client, err := NewS3Client(ctx, &config.Storage)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 client: %w", err)
		}
		g.s3Client = s3Client
	}

	// Create local storage directory if needed
	if config.Storage.Type == "local" || config.Storage.Type == "both" {
		if err := os.MkdirAll(config.Storage.LocalPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Create temp directory
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	logger.Info("PDF generator initialized",
		"storage_type", config.Storage.Type,
		"max_concurrent", config.Performance.MaxConcurrent,
	)

	return g, nil
}

// =============================================================================
// Generation Result
// =============================================================================

// GenerationResult contains the result of PDF generation
type GenerationResult struct {
	// Unique ID for this generation
	ID string `json:"id"`

	// File information
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`

	// Storage locations
	LocalPath string `json:"local_path,omitempty"`
	S3Key     string `json:"s3_key,omitempty"`
	S3URL     string `json:"s3_url,omitempty"`
	PublicURL string `json:"public_url,omitempty"`

	// PDF content (if requested)
	Content []byte `json:"-"`

	// Metadata
	TemplateType TemplateType  `json:"template_type"`
	GeneratedAt  time.Time     `json:"generated_at"`
	Duration     time.Duration `json:"duration"`
}

// =============================================================================
// Generation Options
// =============================================================================

// GenerationOptions contains options for PDF generation
type GenerationOptions struct {
	// Output filename (auto-generated if empty)
	Filename string `json:"filename"`

	// Return content in result
	ReturnContent bool `json:"return_content"`

	// Storage options
	SkipLocalStorage bool `json:"skip_local_storage"`
	SkipS3Upload     bool `json:"skip_s3_upload"`

	// Custom branding (overrides config)
	Branding *BrandingConfig `json:"branding,omitempty"`

	// Custom page settings
	Page *PageConfig `json:"page,omitempty"`

	// Metadata to include in PDF
	Metadata map[string]string `json:"metadata,omitempty"`

	// Watermark text (optional)
	Watermark string `json:"watermark,omitempty"`

	// Password protection (optional)
	Password string `json:"password,omitempty"`
}

// DefaultOptions returns default generation options
func DefaultOptions() *GenerationOptions {
	return &GenerationOptions{
		ReturnContent:    false,
		SkipLocalStorage: false,
		SkipS3Upload:     false,
	}
}

// =============================================================================
// PDF Generation Methods
// =============================================================================

// Generate generates a PDF from template data
func (g *Generator) Generate(ctx context.Context, data TemplateData, opts *GenerationOptions) (*GenerationResult, error) {
	// Acquire worker slot
	select {
	case g.workerPool <- struct{}{}:
		defer func() { <-g.workerPool }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	startTime := time.Now()

	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate data
	if err := data.Validate(); err != nil {
		g.incrementErrorCount()
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get template
	templateFunc, err := g.registry.GetTemplate(data.GetType())
	if err != nil {
		g.incrementErrorCount()
		return nil, err
	}

	// Generate filename
	filename := opts.Filename
	if filename == "" {
		filename = g.generateFilename(data.GetType())
	}

	// Create PDF
	pdf, err := g.createPDF(opts)
	if err != nil {
		g.incrementErrorCount()
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	// Apply watermark if specified
	if opts.Watermark != "" {
		g.addWatermark(pdf, opts.Watermark)
	}

	// Render template
	if err := templateFunc(pdf, data, g.getEffectiveConfig(opts)); err != nil {
		g.incrementErrorCount()
		return nil, fmt.Errorf("%w: %v", ErrTemplateRender, err)
	}

	// Render footer
	g.renderFooter(pdf, opts)

	// Generate PDF content
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		g.incrementErrorCount()
		return nil, fmt.Errorf("%w: %v", ErrPDFGenerationFailed, err)
	}

	content := buf.Bytes()

	result := &GenerationResult{
		ID:           uuid.New().String(),
		Filename:     filename,
		Size:         int64(len(content)),
		ContentType:  "application/pdf",
		TemplateType: data.GetType(),
		GeneratedAt:  time.Now(),
		Duration:     time.Since(startTime),
	}

	if opts.ReturnContent {
		result.Content = content
	}

	// Save to storage
	if err := g.saveToStorage(ctx, content, filename, result, opts); err != nil {
		g.incrementErrorCount()
		return nil, err
	}

	g.incrementGeneratedCount()

	g.logger.Info("PDF generated",
		"id", result.ID,
		"filename", filename,
		"size", result.Size,
		"duration", result.Duration,
	)

	return result, nil
}

// GenerateToWriter generates a PDF and writes to the provided writer
func (g *Generator) GenerateToWriter(ctx context.Context, data TemplateData, w io.Writer, opts *GenerationOptions) error {
	// Acquire worker slot
	select {
	case g.workerPool <- struct{}{}:
		defer func() { <-g.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	if opts == nil {
		opts = DefaultOptions()
	}

	// Validate data
	if err := data.Validate(); err != nil {
		g.incrementErrorCount()
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get template
	templateFunc, err := g.registry.GetTemplate(data.GetType())
	if err != nil {
		g.incrementErrorCount()
		return err
	}

	// Create PDF
	pdf, err := g.createPDF(opts)
	if err != nil {
		g.incrementErrorCount()
		return fmt.Errorf("failed to create PDF: %w", err)
	}

	// Apply watermark
	if opts.Watermark != "" {
		g.addWatermark(pdf, opts.Watermark)
	}

	// Render template
	if err := templateFunc(pdf, data, g.getEffectiveConfig(opts)); err != nil {
		g.incrementErrorCount()
		return fmt.Errorf("%w: %v", ErrTemplateRender, err)
	}

	// Render footer
	g.renderFooter(pdf, opts)

	// Output to writer
	if err := pdf.Output(w); err != nil {
		g.incrementErrorCount()
		return fmt.Errorf("%w: %v", ErrPDFGenerationFailed, err)
	}

	g.incrementGeneratedCount()
	return nil
}

// GenerateInvoice is a convenience method for generating invoices
func (g *Generator) GenerateInvoice(ctx context.Context, data *InvoiceData, opts *GenerationOptions) (*GenerationResult, error) {
	return g.Generate(ctx, data, opts)
}

// GenerateQuote is a convenience method for generating quotes
func (g *Generator) GenerateQuote(ctx context.Context, data *QuoteData, opts *GenerationOptions) (*GenerationResult, error) {
	return g.Generate(ctx, data, opts)
}

// GenerateWorkOrder is a convenience method for generating work orders
func (g *Generator) GenerateWorkOrder(ctx context.Context, data *WorkOrderData, opts *GenerationOptions) (*GenerationResult, error) {
	return g.Generate(ctx, data, opts)
}

// GenerateServiceReport is a convenience method for generating service reports
func (g *Generator) GenerateServiceReport(ctx context.Context, data *ServiceReportData, opts *GenerationOptions) (*GenerationResult, error) {
	return g.Generate(ctx, data, opts)
}

// =============================================================================
// Storage Methods
// =============================================================================

// saveToStorage saves the PDF to configured storage locations
func (g *Generator) saveToStorage(ctx context.Context, content []byte, filename string, result *GenerationResult, opts *GenerationOptions) error {
	// Save to local storage
	if (g.config.Storage.Type == "local" || g.config.Storage.Type == "both") && !opts.SkipLocalStorage {
		localPath := filepath.Join(g.config.Storage.LocalPath, filename)
		if err := os.WriteFile(localPath, content, 0644); err != nil {
			return fmt.Errorf("failed to save locally: %w", err)
		}
		result.LocalPath = localPath
	}

	// Upload to S3
	if (g.config.Storage.Type == "s3" || g.config.Storage.Type == "both") && !opts.SkipS3Upload && g.s3Client != nil {
		key := g.s3Client.GetKey(filename)

		_, err := g.s3Client.GetClient().PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(g.s3Client.GetBucket()),
			Key:         aws.String(key),
			Body:        bytes.NewReader(content),
			ContentType: aws.String("application/pdf"),
		})
		if err != nil {
			return fmt.Errorf("%w: %v", ErrS3UploadFailed, err)
		}

		result.S3Key = key
		result.S3URL = fmt.Sprintf("s3://%s/%s", g.s3Client.GetBucket(), key)
		result.PublicURL = g.s3Client.GetPublicURL(filename)
	}

	return nil
}

// GetPresignedURL generates a presigned URL for downloading a PDF from S3
func (g *Generator) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if g.s3Client == nil {
		return "", fmt.Errorf("S3 client not configured")
	}

	presigner := s3.NewPresignClient(g.s3Client.GetClient())

	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(g.s3Client.GetBucket()),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// DeletePDF deletes a PDF from storage
func (g *Generator) DeletePDF(ctx context.Context, filename string) error {
	// Delete from local storage
	if g.config.Storage.Type == "local" || g.config.Storage.Type == "both" {
		localPath := filepath.Join(g.config.Storage.LocalPath, filename)
		if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete local file: %w", err)
		}
	}

	// Delete from S3
	if (g.config.Storage.Type == "s3" || g.config.Storage.Type == "both") && g.s3Client != nil {
		key := g.s3Client.GetKey(filename)
		_, err := g.s3Client.GetClient().DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(g.s3Client.GetBucket()),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("failed to delete from S3: %w", err)
		}
	}

	return nil
}

// =============================================================================
// PDF Creation Helpers
// =============================================================================

// createPDF creates a new PDF document with configured settings
func (g *Generator) createPDF(opts *GenerationOptions) (*gofpdf.Fpdf, error) {
	pageConfig := g.config.PageDefaults
	if opts.Page != nil {
		pageConfig = *opts.Page
	}

	pdf := gofpdf.New(pageConfig.Orientation, pageConfig.Unit, pageConfig.Size, g.config.FontDir)

	// Set margins
	pdf.SetMargins(pageConfig.MarginLeft, pageConfig.MarginTop, pageConfig.MarginRight)
	pdf.SetAutoPageBreak(true, pageConfig.MarginBottom+pageConfig.FooterHeight)

	// Load custom fonts
	for _, font := range g.config.Fonts.CustomFonts {
		if font.Unicode {
			pdf.AddUTF8Font(font.Family, font.Style, font.Path)
		} else {
			pdf.AddFont(font.Family, font.Style, font.Path)
		}
	}

	// Set default font
	pdf.SetFont(g.config.Fonts.DefaultFamily, "", g.config.Fonts.SizeBody)

	// Set compression
	pdf.SetCompression(g.config.Performance.EnableCompression)

	// Add PDF metadata
	pdf.SetTitle("Generated Document", true)
	pdf.SetAuthor(g.config.Branding.CompanyName, true)
	pdf.SetCreator("ServicePro PDF Generator", true)
	pdf.SetCreationDate(time.Now())

	if opts.Metadata != nil {
		for key, value := range opts.Metadata {
			pdf.SetKeywords(key+":"+value, true)
		}
	}

	// Add first page
	pdf.AddPage()

	return pdf, nil
}

// addWatermark adds a watermark to the PDF
func (g *Generator) addWatermark(pdf *gofpdf.Fpdf, text string) {
	// Save current settings
	currentFont, _ := pdf.GetFontSize()

	// Configure watermark
	pdf.SetFont(g.config.Fonts.DefaultFamily, "B", 60)
	pdf.SetTextColor(200, 200, 200)

	// Get page dimensions
	w, h := pdf.GetPageSize()

	// Add watermark to each page
	pdf.SetAcceptPageBreakFunc(func() bool {
		// Draw watermark on current page
		pdf.TransformBegin()
		pdf.TransformRotate(45, w/2, h/2)
		pdf.SetXY(w/4, h/2)
		pdf.CellFormat(w/2, 20, text, "", 0, "C", false, 0, "")
		pdf.TransformEnd()
		return true
	})

	// Restore settings
	pdf.SetFont(g.config.Fonts.DefaultFamily, "", currentFont)
	pdf.SetTextColor(0, 0, 0)
}

// renderFooter renders the page footer
func (g *Generator) renderFooter(pdf *gofpdf.Fpdf, opts *GenerationOptions) {
	branding := g.config.Branding
	if opts.Branding != nil {
		branding = *opts.Branding
	}

	// Footer callback
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont(g.config.Fonts.DefaultFamily, "", g.config.Fonts.SizeFooter)
		r, g, b := hexToRGB(branding.MutedColor)
		pdf.SetTextColor(r, g, b)

		// Footer text
		footerText := branding.FooterText
		if footerText == "" {
			footerText = fmt.Sprintf("%s | Generated on %s", branding.CompanyName, time.Now().Format("Jan 2, 2006"))
		}
		pdf.CellFormat(0, 10, footerText, "", 0, "C", false, 0, "")

		// Page number
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AliasNbPages("{nb}")
}

// getEffectiveConfig returns config with any option overrides applied
func (g *Generator) getEffectiveConfig(opts *GenerationOptions) *Config {
	if opts == nil || (opts.Branding == nil && opts.Page == nil) {
		return g.config
	}

	// Create a copy of config with overrides
	cfg := *g.config
	if opts.Branding != nil {
		cfg.Branding = *opts.Branding
	}
	if opts.Page != nil {
		cfg.PageDefaults = *opts.Page
	}
	return &cfg
}

// generateFilename generates a unique filename for a PDF
func (g *Generator) generateFilename(templateType TemplateType) string {
	timestamp := time.Now().Format("20060102-150405")
	id := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s-%s.pdf", templateType, timestamp, id)
}

// =============================================================================
// Template Registry Methods
// =============================================================================

// RegisterTemplate registers a custom template
func (g *Generator) RegisterTemplate(templateType TemplateType, fn TemplateFunc) {
	g.registry.RegisterTemplate(templateType, fn)
}

// HasTemplate checks if a template exists
func (g *Generator) HasTemplate(templateType TemplateType) bool {
	return g.registry.HasTemplate(templateType)
}

// =============================================================================
// Metrics
// =============================================================================

func (g *Generator) incrementGeneratedCount() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.generatedCount++
}

func (g *Generator) incrementErrorCount() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errorCount++
}

// GetMetrics returns generation metrics
func (g *Generator) GetMetrics() map[string]int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return map[string]int64{
		"generated_count": g.generatedCount,
		"error_count":     g.errorCount,
	}
}

// =============================================================================
// Batch Generation
// =============================================================================

// BatchGenerationResult contains results from batch generation
type BatchGenerationResult struct {
	Results []*GenerationResult `json:"results"`
	Errors  []BatchError        `json:"errors,omitempty"`
	Total   int                 `json:"total"`
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
}

// BatchError represents an error during batch generation
type BatchError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// GenerateBatch generates multiple PDFs concurrently
func (g *Generator) GenerateBatch(ctx context.Context, items []TemplateData, opts *GenerationOptions) *BatchGenerationResult {
	result := &BatchGenerationResult{
		Results: make([]*GenerationResult, 0, len(items)),
		Errors:  make([]BatchError, 0),
		Total:   len(items),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	resultsChan := make(chan *GenerationResult, len(items))
	errorsChan := make(chan BatchError, len(items))

	for i, data := range items {
		wg.Add(1)
		go func(idx int, d TemplateData) {
			defer wg.Done()

			res, err := g.Generate(ctx, d, opts)
			if err != nil {
				errorsChan <- BatchError{Index: idx, Error: err.Error()}
				return
			}
			resultsChan <- res
		}(i, data)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	for res := range resultsChan {
		mu.Lock()
		result.Results = append(result.Results, res)
		result.Success++
		mu.Unlock()
	}

	// Collect errors
	for err := range errorsChan {
		mu.Lock()
		result.Errors = append(result.Errors, err)
		result.Failed++
		mu.Unlock()
	}

	return result
}

// =============================================================================
// Configuration Access
// =============================================================================

// GetConfig returns the current configuration
func (g *Generator) GetConfig() *Config {
	return g.config
}

// UpdateBranding updates the branding configuration
func (g *Generator) UpdateBranding(branding *BrandingConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config.Branding = *branding
}
