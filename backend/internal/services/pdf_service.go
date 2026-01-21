package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aymerick/raymond"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// PDFService handles PDF generation from templates
type PDFService struct {
	wkhtmltopdfPath string
	tempDir         string
	outputDir       string
}

// PDFGenerationOptions contains options for PDF generation
type PDFGenerationOptions struct {
	PageSize         models.PageSize
	PageOrientation  models.PageOrientation
	MarginTop        string
	MarginRight      string
	MarginBottom     string
	MarginLeft       string
	HeaderHTML       string
	FooterHTML       string
	EnableJavascript bool
	DPI              int
	ImageQuality     int
}

// NewPDFService creates a new PDF service
func NewPDFService(wkhtmltopdfPath, tempDir, outputDir string) *PDFService {
	if wkhtmltopdfPath == "" {
		wkhtmltopdfPath = "wkhtmltopdf" // Use PATH
	}
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	if outputDir == "" {
		outputDir = "./generated_pdfs"
	}

	// Ensure directories exist
	os.MkdirAll(tempDir, 0755)
	os.MkdirAll(outputDir, 0755)

	return &PDFService{
		wkhtmltopdfPath: wkhtmltopdfPath,
		tempDir:         tempDir,
		outputDir:       outputDir,
	}
}

// GeneratePDFFromTemplate generates a PDF from an invoice template
func (s *PDFService) GeneratePDFFromTemplate(
	ctx context.Context,
	template *models.InvoiceTemplate,
	data map[string]interface{},
	outputPath string,
) (*models.GeneratePDFResponse, error) {
	startTime := time.Now()

	// Render the template with data
	htmlContent, err := s.RenderTemplate(template.HTMLContent, data)
	if err != nil {
		return &models.GeneratePDFResponse{
			Success:          false,
			Error:            fmt.Sprintf("Failed to render template: %v", err),
			GenerationTimeMs: int(time.Since(startTime).Milliseconds()),
			GeneratedAt:      time.Now(),
		}, err
	}

	// Apply CSS
	fullHTML := s.ApplyCSS(htmlContent, template.CSSContent)

	// Add watermark if enabled
	if template.WatermarkEnabled && template.WatermarkText != "" {
		fullHTML = s.AddWatermark(fullHTML, template.WatermarkText,
			template.WatermarkOpacity.InexactFloat64(), template.WatermarkRotation)
	}

	// Generate output path if not provided
	if outputPath == "" {
		outputPath = filepath.Join(s.outputDir, fmt.Sprintf("invoice_%s.pdf", uuid.New().String()))
	}

	// Build PDF generation options
	options := &PDFGenerationOptions{
		PageSize:         template.PageSize,
		PageOrientation:  template.PageOrientation,
		MarginTop:        fmt.Sprintf("%fmm", template.MarginTop.InexactFloat64()),
		MarginRight:      fmt.Sprintf("%fmm", template.MarginRight.InexactFloat64()),
		MarginBottom:     fmt.Sprintf("%fmm", template.MarginBottom.InexactFloat64()),
		MarginLeft:       fmt.Sprintf("%fmm", template.MarginLeft.InexactFloat64()),
		HeaderHTML:       template.HeaderHTML,
		FooterHTML:       s.BuildFooter(template),
		EnableJavascript: false,
		DPI:              300,
		ImageQuality:     94,
	}

	// Generate PDF
	fileSize, err := s.GeneratePDF(ctx, fullHTML, outputPath, options)
	if err != nil {
		return &models.GeneratePDFResponse{
			Success:          false,
			Error:            fmt.Sprintf("Failed to generate PDF: %v", err),
			GenerationTimeMs: int(time.Since(startTime).Milliseconds()),
			GeneratedAt:      time.Now(),
		}, err
	}

	return &models.GeneratePDFResponse{
		Success:          true,
		FilePath:         outputPath,
		FileSize:         fileSize,
		GenerationTimeMs: int(time.Since(startTime).Milliseconds()),
		GeneratedAt:      time.Now(),
	}, nil
}

// RenderTemplate renders an HTML template with the provided data using Handlebars
func (s *PDFService) RenderTemplate(templateContent string, data map[string]interface{}) (string, error) {
	// Register custom helpers
	raymond.RegisterHelper("format_currency", func(value interface{}) string {
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("%.2f", v)
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	})

	raymond.RegisterHelper("format_date", func(value interface{}, format string) string {
		switch v := value.(type) {
		case time.Time:
			if format == "" {
				format = "01/02/2006"
			}
			return v.Format(format)
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	})

	// Render template
	result, err := raymond.Render(templateContent, data)
	if err != nil {
		return "", fmt.Errorf("template rendering failed: %w", err)
	}

	return result, nil
}

// ApplyCSS combines HTML content with CSS
func (s *PDFService) ApplyCSS(htmlContent, cssContent string) string {
	var builder strings.Builder

	builder.WriteString("<!DOCTYPE html>\n")
	builder.WriteString("<html>\n<head>\n")
	builder.WriteString("<meta charset=\"UTF-8\">\n")

	if cssContent != "" {
		builder.WriteString("<style>\n")
		builder.WriteString(cssContent)
		builder.WriteString("\n</style>\n")
	}

	builder.WriteString("</head>\n<body>\n")
	builder.WriteString(htmlContent)
	builder.WriteString("\n</body>\n</html>")

	return builder.String()
}

// AddWatermark adds a watermark to the HTML content
func (s *PDFService) AddWatermark(htmlContent, text string, opacity float64, rotation int) string {
	watermarkCSS := fmt.Sprintf(`
<style>
.watermark {
    position: fixed;
    top: 50%%;
    left: 50%%;
    transform: translate(-50%%, -50%%) rotate(%ddeg);
    font-size: 120px;
    font-weight: bold;
    color: rgba(200, 200, 200, %.2f);
    opacity: %.2f;
    z-index: -1;
    pointer-events: none;
    white-space: nowrap;
    user-select: none;
}
</style>
<div class="watermark">%s</div>
`, rotation, opacity, opacity, template.HTMLEscapeString(text))

	// Insert watermark after <body> tag
	return strings.Replace(htmlContent, "<body>", "<body>\n"+watermarkCSS, 1)
}

// BuildFooter builds the footer HTML with page numbers if enabled
func (s *PDFService) BuildFooter(tmpl *models.InvoiceTemplate) string {
	if !tmpl.ShowPageNumbers {
		if tmpl.FooterHTML != "" {
			return tmpl.FooterHTML
		}
		return ""
	}

	var builder strings.Builder

	// Add custom footer if exists
	if tmpl.FooterHTML != "" {
		builder.WriteString(tmpl.FooterHTML)
		builder.WriteString("\n")
	}

	// Add page numbers
	pageNumberFormat := tmpl.PageNumberFormat
	if pageNumberFormat == "" {
		pageNumberFormat = "Page {page} of {total}"
	}

	// wkhtmltopdf uses [page] and [topage] placeholders
	pageNumberHTML := strings.ReplaceAll(pageNumberFormat, "{page}", "[page]")
	pageNumberHTML = strings.ReplaceAll(pageNumberHTML, "{total}", "[topage]")

	position := tmpl.PageNumberPosition
	if position == "" {
		position = "bottom-center"
	}

	textAlign := "center"
	if strings.Contains(position, "left") {
		textAlign = "left"
	} else if strings.Contains(position, "right") {
		textAlign = "right"
	}

	builder.WriteString(fmt.Sprintf(`
<div style="text-align: %s; font-size: 10px; color: #666; padding: 5px;">
    %s
</div>
`, textAlign, pageNumberHTML))

	return builder.String()
}

// GeneratePDF generates a PDF file from HTML content using wkhtmltopdf
func (s *PDFService) GeneratePDF(ctx context.Context, htmlContent, outputPath string, options *PDFGenerationOptions) (int64, error) {
	// Create temporary HTML file
	tempHTML := filepath.Join(s.tempDir, fmt.Sprintf("temp_%s.html", uuid.New().String()))
	if err := os.WriteFile(tempHTML, []byte(htmlContent), 0644); err != nil {
		return 0, fmt.Errorf("failed to write temporary HTML file: %w", err)
	}
	defer os.Remove(tempHTML)

	// Build wkhtmltopdf command arguments
	args := s.buildWkhtmltopdfArgs(tempHTML, outputPath, options)

	// Execute wkhtmltopdf
	cmd := exec.CommandContext(ctx, s.wkhtmltopdfPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("wkhtmltopdf execution failed: %w, stderr: %s", err, stderr.String())
	}

	// Get file size
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get PDF file info: %w", err)
	}

	return fileInfo.Size(), nil
}

// buildWkhtmltopdfArgs builds command line arguments for wkhtmltopdf
func (s *PDFService) buildWkhtmltopdfArgs(inputPath, outputPath string, options *PDFGenerationOptions) []string {
	args := []string{
		"--encoding", "UTF-8",
		"--quiet",
	}

	// Page size
	if options.PageSize != "" {
		args = append(args, "--page-size", string(options.PageSize))
	}

	// Page orientation
	if options.PageOrientation != "" {
		args = append(args, "--orientation", string(options.PageOrientation))
	}

	// Margins
	if options.MarginTop != "" {
		args = append(args, "--margin-top", options.MarginTop)
	}
	if options.MarginRight != "" {
		args = append(args, "--margin-right", options.MarginRight)
	}
	if options.MarginBottom != "" {
		args = append(args, "--margin-bottom", options.MarginBottom)
	}
	if options.MarginLeft != "" {
		args = append(args, "--margin-left", options.MarginLeft)
	}

	// Header/Footer HTML
	if options.HeaderHTML != "" {
		headerFile := filepath.Join(s.tempDir, fmt.Sprintf("header_%s.html", uuid.New().String()))
		os.WriteFile(headerFile, []byte(options.HeaderHTML), 0644)
		args = append(args, "--header-html", headerFile)
		defer os.Remove(headerFile)
	}

	if options.FooterHTML != "" {
		footerFile := filepath.Join(s.tempDir, fmt.Sprintf("footer_%s.html", uuid.New().String()))
		os.WriteFile(footerFile, []byte(options.FooterHTML), 0644)
		args = append(args, "--footer-html", footerFile)
		defer os.Remove(footerFile)
	}

	// JavaScript
	if options.EnableJavascript {
		args = append(args, "--enable-javascript")
	} else {
		args = append(args, "--no-stop-slow-scripts")
	}

	// DPI
	if options.DPI > 0 {
		args = append(args, "--dpi", fmt.Sprintf("%d", options.DPI))
	}

	// Image quality
	if options.ImageQuality > 0 {
		args = append(args, "--image-quality", fmt.Sprintf("%d", options.ImageQuality))
	}

	// Enable local file access
	args = append(args, "--enable-local-file-access")

	// Input and output files
	args = append(args, inputPath, outputPath)

	return args
}

// GeneratePreview generates a preview PDF with sample data
func (s *PDFService) GeneratePreview(ctx context.Context, htmlContent, cssContent string, sampleData map[string]interface{}) (string, error) {
	// Render template with sample data
	rendered, err := s.RenderTemplate(htmlContent, sampleData)
	if err != nil {
		return "", err
	}

	// Apply CSS
	fullHTML := s.ApplyCSS(rendered, cssContent)

	// Generate preview PDF
	previewPath := filepath.Join(s.tempDir, fmt.Sprintf("preview_%s.pdf", uuid.New().String()))

	options := &PDFGenerationOptions{
		PageSize:        models.PageSizeA4,
		PageOrientation: models.PageOrientationPortrait,
		MarginTop:       "10mm",
		MarginRight:     "10mm",
		MarginBottom:    "10mm",
		MarginLeft:      "10mm",
		DPI:             150, // Lower DPI for preview
		ImageQuality:    75,  // Lower quality for preview
	}

	_, err = s.GeneratePDF(ctx, fullHTML, previewPath, options)
	if err != nil {
		return "", err
	}

	return previewPath, nil
}

// ProcessImage downloads and processes an image, returning a data URL
func (s *PDFService) ProcessImage(imageURL string, maxWidth, maxHeight int) (string, error) {
	// Download image
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	// Read image
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Decode image to get dimensions
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate resize if needed
	if (maxWidth > 0 && width > maxWidth) || (maxHeight > 0 && height > maxHeight) {
		// Image needs resizing - for now, just use original
		// In production, you'd use an image processing library like imaging
	}

	// Convert to base64 data URL
	mimeType := "image/" + format
	base64Data := base64.StdEncoding.EncodeToString(imgData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return dataURL, nil
}

// ValidateTemplate validates that a template can be rendered
func (s *PDFService) ValidateTemplate(htmlContent, cssContent string, sampleData map[string]interface{}) error {
	// Try to render template
	_, err := s.RenderTemplate(htmlContent, sampleData)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Validate CSS (basic check)
	if cssContent != "" && strings.Contains(cssContent, "</script>") {
		return fmt.Errorf("CSS content contains potentially dangerous script tags")
	}

	return nil
}

// CleanupOldFiles removes generated PDF files older than the specified duration
func (s *PDFService) CleanupOldFiles(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	return filepath.Walk(s.outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pdf") {
			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove old file %s: %w", path, err)
				}
			}
		}

		return nil
	})
}

// GetDefaultSampleData returns sample data for template preview
func (s *PDFService) GetDefaultSampleData() map[string]interface{} {
	return map[string]interface{}{
		"invoice_number":   "INV-2024-00001",
		"issue_date":       time.Now().Format("01/02/2006"),
		"due_date":         time.Now().AddDate(0, 0, 30).Format("01/02/2006"),
		"company_name":     "Sample Company Inc.",
		"company_address":  "123 Business Street",
		"company_city":     "New York",
		"company_state":    "NY",
		"company_zip":      "10001",
		"company_phone":    "(555) 123-4567",
		"company_email":    "info@samplecompany.com",
		"customer_name":    "John Doe",
		"customer_address": "456 Customer Ave",
		"customer_city":    "Los Angeles",
		"customer_state":   "CA",
		"customer_zip":     "90001",
		"customer_email":   "john.doe@example.com",
		"line_items": []map[string]interface{}{
			{
				"description": "Website Development - 40 hours",
				"quantity":    "40.00",
				"unit_price":  "125.00",
				"line_total":  "5000.00",
			},
			{
				"description": "Hosting Services - Annual",
				"quantity":    "1.00",
				"unit_price":  "500.00",
				"line_total":  "500.00",
			},
			{
				"description": "Domain Registration",
				"quantity":    "1.00",
				"unit_price":  "15.00",
				"line_total":  "15.00",
			},
		},
		"subtotal":        "5515.00",
		"tax_rate":        "8.25",
		"tax_amount":      "454.99",
		"discount_amount": "100.00",
		"total_amount":    "5869.99",
		"amount_paid":     "0.00",
		"amount_due":      "5869.99",
		"notes":           "Thank you for your business. Payment is due within 30 days.",
		"payment_terms":   "Net 30. Late payments may be subject to a 5% late fee.",
	}
}

// CheckWkhtmltopdfInstalled checks if wkhtmltopdf is installed and accessible
func (s *PDFService) CheckWkhtmltopdfInstalled() error {
	cmd := exec.Command(s.wkhtmltopdfPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wkhtmltopdf not found or not executable: %w", err)
	}
	return nil
}
