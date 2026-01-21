package services

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// InvoiceTemplateService handles invoice template operations
type InvoiceTemplateService struct {
	db         *gorm.DB
	pdfService *PDFService
	assetDir   string
}

// NewInvoiceTemplateService creates a new invoice template service
func NewInvoiceTemplateService(db *gorm.DB, pdfService *PDFService, assetDir string) *InvoiceTemplateService {
	if assetDir == "" {
		assetDir = "./template_assets"
	}
	os.MkdirAll(assetDir, 0755)

	return &InvoiceTemplateService{
		db:         db,
		pdfService: pdfService,
		assetDir:   assetDir,
	}
}

// CreateTemplate creates a new invoice template
func (s *InvoiceTemplateService) CreateTemplate(ctx context.Context, req *models.CreateTemplateRequest, userID uuid.UUID) (*models.InvoiceTemplate, error) {
	// Validate template content
	if err := s.pdfService.ValidateTemplate(req.HTMLContent, req.CSSContent, s.pdfService.GetDefaultSampleData()); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	template := &models.InvoiceTemplate{
		Name:        req.Name,
		Description: req.Description,
		HTMLContent: req.HTMLContent,
		CSSContent:  req.CSSContent,
		HeaderHTML:  req.HeaderHTML,
		FooterHTML:  req.FooterHTML,
		Status:      req.Status,
		IsDefault:   req.IsDefault,

		PageSize:        req.PageSize,
		PageOrientation: req.PageOrientation,
		MarginTop:       req.MarginTop,
		MarginRight:     req.MarginRight,
		MarginBottom:    req.MarginBottom,
		MarginLeft:      req.MarginLeft,

		LogoURL:      req.LogoURL,
		LogoPosition: req.LogoPosition,
		LogoWidth:    req.LogoWidth,
		LogoHeight:   req.LogoHeight,

		WatermarkText:     req.WatermarkText,
		WatermarkOpacity:  req.WatermarkOpacity,
		WatermarkRotation: req.WatermarkRotation,
		WatermarkEnabled:  req.WatermarkEnabled,

		ShowPageNumbers:    req.ShowPageNumbers,
		PageNumberFormat:   req.PageNumberFormat,
		PageNumberPosition: req.PageNumberPosition,

		CustomFonts:   req.CustomFonts,
		FieldMappings: req.FieldMappings,
		PreviewData:   req.PreviewData,

		Tags:      req.Tags,
		CreatedBy: userID,
	}

	// Set defaults if not provided
	if template.Status == "" {
		template.Status = models.TemplateStatusDraft
	}
	if template.PageSize == "" {
		template.PageSize = models.PageSizeA4
	}
	if template.PageOrientation == "" {
		template.PageOrientation = models.PageOrientationPortrait
	}
	if template.MarginTop.IsZero() {
		template.MarginTop = decimal.NewFromFloat(10.0)
	}
	if template.MarginRight.IsZero() {
		template.MarginRight = decimal.NewFromFloat(10.0)
	}
	if template.MarginBottom.IsZero() {
		template.MarginBottom = decimal.NewFromFloat(10.0)
	}
	if template.MarginLeft.IsZero() {
		template.MarginLeft = decimal.NewFromFloat(10.0)
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (s *InvoiceTemplateService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.InvoiceTemplate, error) {
	var template models.InvoiceTemplate
	if err := s.db.WithContext(ctx).
		Preload("Assets").
		Where("id = ?", id).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return &template, nil
}

// ListTemplates retrieves all templates with optional filtering
func (s *InvoiceTemplateService) ListTemplates(ctx context.Context, filters map[string]interface{}, page, pageSize int) ([]models.InvoiceTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&models.InvoiceTemplate{})

	// Apply filters
	if filters != nil {
		if status, ok := filters["status"]; ok {
			query = query.Where("status = ?", status)
		}

		if isDefault, ok := filters["is_default"]; ok {
			query = query.Where("is_default = ?", isDefault)
		}

		if tags, ok := filters["tags"]; ok {
			if tagArray, ok := tags.([]string); ok && len(tagArray) > 0 {
				query = query.Where("tags && ?", tagArray)
			}
		}

		if search, ok := filters["search"]; ok {
			if searchStr, ok := search.(string); ok && searchStr != "" {
				searchPattern := "%" + searchStr + "%"
				query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
			}
		}
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Get templates
	var templates []models.InvoiceTemplate
	query = query.Order("created_at DESC")

	// Calculate offset from page
	offset := (page - 1) * pageSize
	if pageSize > 0 {
		query = query.Limit(pageSize)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// UpdateTemplate updates an existing template
func (s *InvoiceTemplateService) UpdateTemplate(ctx context.Context, id uuid.UUID, req *models.UpdateTemplateRequest, userID uuid.UUID) (*models.InvoiceTemplate, error) {
	template, err := s.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate new content if provided
	htmlContent := template.HTMLContent
	cssContent := template.CSSContent

	if req.HTMLContent != nil {
		htmlContent = *req.HTMLContent
	}
	if req.CSSContent != nil {
		cssContent = *req.CSSContent
	}

	if req.HTMLContent != nil || req.CSSContent != nil {
		if err := s.pdfService.ValidateTemplate(htmlContent, cssContent, s.pdfService.GetDefaultSampleData()); err != nil {
			return nil, fmt.Errorf("template validation failed: %w", err)
		}
	}

	// Build updates
	updates := make(map[string]interface{})
	updates["updated_by"] = userID

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.HTMLContent != nil {
		updates["html_content"] = *req.HTMLContent
	}
	if req.CSSContent != nil {
		updates["css_content"] = *req.CSSContent
	}
	if req.HeaderHTML != nil {
		updates["header_html"] = *req.HeaderHTML
	}
	if req.FooterHTML != nil {
		updates["footer_html"] = *req.FooterHTML
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.PageSize != nil {
		updates["page_size"] = *req.PageSize
	}
	if req.PageOrientation != nil {
		updates["page_orientation"] = *req.PageOrientation
	}
	if req.MarginTop != nil {
		updates["margin_top"] = *req.MarginTop
	}
	if req.MarginRight != nil {
		updates["margin_right"] = *req.MarginRight
	}
	if req.MarginBottom != nil {
		updates["margin_bottom"] = *req.MarginBottom
	}
	if req.MarginLeft != nil {
		updates["margin_left"] = *req.MarginLeft
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.LogoPosition != nil {
		updates["logo_position"] = *req.LogoPosition
	}
	if req.LogoWidth != nil {
		updates["logo_width"] = *req.LogoWidth
	}
	if req.LogoHeight != nil {
		updates["logo_height"] = *req.LogoHeight
	}
	if req.WatermarkText != nil {
		updates["watermark_text"] = *req.WatermarkText
	}
	if req.WatermarkOpacity != nil {
		updates["watermark_opacity"] = *req.WatermarkOpacity
	}
	if req.WatermarkRotation != nil {
		updates["watermark_rotation"] = *req.WatermarkRotation
	}
	if req.WatermarkEnabled != nil {
		updates["watermark_enabled"] = *req.WatermarkEnabled
	}
	if req.ShowPageNumbers != nil {
		updates["show_page_numbers"] = *req.ShowPageNumbers
	}
	if req.PageNumberFormat != nil {
		updates["page_number_format"] = *req.PageNumberFormat
	}
	if req.PageNumberPosition != nil {
		updates["page_number_position"] = *req.PageNumberPosition
	}
	if req.CustomFonts != nil {
		updates["custom_fonts"] = *req.CustomFonts
	}
	if req.FieldMappings != nil {
		updates["field_mappings"] = *req.FieldMappings
	}
	if req.PreviewData != nil {
		updates["preview_data"] = *req.PreviewData
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.VersionNotes != nil {
		updates["version_notes"] = *req.VersionNotes
	}

	if err := s.db.WithContext(ctx).Model(&models.InvoiceTemplate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return s.GetTemplate(ctx, id)
}

// DeleteTemplate deletes or archives a template
func (s *InvoiceTemplateService) DeleteTemplate(ctx context.Context, id uuid.UUID, hardDelete bool) error {
	// Check if template exists and is default
	var template models.InvoiceTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("template not found")
		}
		return fmt.Errorf("failed to find template: %w", err)
	}

	// Don't allow deletion of default template
	if template.IsDefault {
		return errors.New("cannot delete default template")
	}

	// If soft delete (archive), just update status
	if !hardDelete {
		return s.db.WithContext(ctx).Model(&models.InvoiceTemplate{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": models.TemplateStatusArchived,
			}).Error
	}

	// Hard delete: Delete template and associated assets
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete assets from filesystem
		var assets []models.InvoiceTemplateAsset
		if err := tx.Where("template_id = ?", id).Find(&assets).Error; err != nil {
			return err
		}

		for _, asset := range assets {
			os.Remove(asset.FilePath) // Ignore errors
		}

		// Delete from database
		if err := tx.Where("template_id = ?", id).Delete(&models.InvoiceTemplateAsset{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&models.InvoiceTemplate{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

// CloneTemplate creates a copy of an existing template
func (s *InvoiceTemplateService) CloneTemplate(ctx context.Context, id uuid.UUID, newName string, userID uuid.UUID) (*models.InvoiceTemplate, error) {
	var newID uuid.UUID
	err := s.db.WithContext(ctx).Raw("SELECT clone_template(?, ?, ?)", id, newName, userID).Scan(&newID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to clone template: %w", err)
	}

	return s.GetTemplate(ctx, newID)
}

// GetDefaultTemplate retrieves the default template
func (s *InvoiceTemplateService) GetDefaultTemplate(ctx context.Context) (*models.InvoiceTemplate, error) {
	var template models.InvoiceTemplate
	if err := s.db.WithContext(ctx).
		Where("is_default = ? AND status = ?", true, models.TemplateStatusActive).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no default template found")
		}
		return nil, fmt.Errorf("failed to get default template: %w", err)
	}
	return &template, nil
}

// SetDefaultTemplate sets a template as the default
func (s *InvoiceTemplateService) SetDefaultTemplate(ctx context.Context, id uuid.UUID) error {
	template, err := s.GetTemplate(ctx, id)
	if err != nil {
		return err
	}

	if template.Status != models.TemplateStatusActive {
		return fmt.Errorf("only active templates can be set as default")
	}

	return s.db.WithContext(ctx).Model(&models.InvoiceTemplate{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}

// GetTemplateStatistics retrieves usage statistics for a template
func (s *InvoiceTemplateService) GetTemplateStatistics(ctx context.Context, id uuid.UUID) (*models.TemplateStatistics, error) {
	var stats models.TemplateStatistics
	err := s.db.WithContext(ctx).Raw("SELECT * FROM get_template_statistics(?)", id).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get template statistics: %w", err)
	}
	return &stats, nil
}

// GetVersionHistory retrieves the version history for a template
func (s *InvoiceTemplateService) GetVersionHistory(ctx context.Context, id uuid.UUID) ([]models.InvoiceTemplateVersion, error) {
	var versions []models.InvoiceTemplateVersion
	if err := s.db.WithContext(ctx).
		Where("template_id = ?", id).
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}
	return versions, nil
}

// RestoreVersion restores a template to a specific version
func (s *InvoiceTemplateService) RestoreVersion(ctx context.Context, templateID uuid.UUID, versionNumber int, userID uuid.UUID) (*models.InvoiceTemplate, error) {
	var version models.InvoiceTemplateVersion
	if err := s.db.WithContext(ctx).
		Where("template_id = ? AND version_number = ?", templateID, versionNumber).
		First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("version not found")
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// Update template with version content
	updates := &models.UpdateTemplateRequest{
		HTMLContent:  &version.HTMLContent,
		CSSContent:   &version.CSSContent,
		HeaderHTML:   &version.HeaderHTML,
		FooterHTML:   &version.FooterHTML,
		VersionNotes: stringPtr(fmt.Sprintf("Restored from version %d", versionNumber)),
	}

	return s.UpdateTemplate(ctx, templateID, updates, userID)
}

// UploadAsset uploads an asset (logo, image, font) for a template
func (s *InvoiceTemplateService) UploadAsset(ctx context.Context, templateID uuid.UUID, assetType, assetName string, file multipart.File, header *multipart.FileHeader) (*models.InvoiceTemplateAsset, error) {
	// Validate asset type
	validTypes := map[string]bool{
		"logo":  true,
		"image": true,
		"font":  true,
		"css":   true,
		"other": true,
	}
	if !validTypes[assetType] {
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}

	// Create asset directory for template
	templateAssetDir := filepath.Join(s.assetDir, templateID.String())
	if err := os.MkdirAll(templateAssetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create asset directory: %w", err)
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%s%s", assetType, uuid.New().String(), ext)
	filePath := filepath.Join(templateAssetDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save asset file: %w", err)
	}

	// Get file info
	fileInfo, _ := os.Stat(filePath)

	// Create asset record
	asset := &models.InvoiceTemplateAsset{
		TemplateID: templateID,
		AssetType:  assetType,
		AssetName:  assetName,
		FilePath:   filePath,
		FileSize:   fileInfo.Size(),
		MimeType:   header.Header.Get("Content-Type"),
	}

	// Get image dimensions if it's an image
	if assetType == "logo" || assetType == "image" {
		if width, height, err := s.getImageDimensions(filePath); err == nil {
			asset.Width = width
			asset.Height = height
		}
	}

	if err := s.db.WithContext(ctx).Create(asset).Error; err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create asset record: %w", err)
	}

	return asset, nil
}

// DeleteAsset deletes an asset
func (s *InvoiceTemplateService) DeleteAsset(ctx context.Context, assetID uuid.UUID) error {
	var asset models.InvoiceTemplateAsset
	if err := s.db.WithContext(ctx).First(&asset, assetID).Error; err != nil {
		return fmt.Errorf("asset not found: %w", err)
	}

	// Delete file
	os.Remove(asset.FilePath) // Ignore error

	// Delete record
	return s.db.WithContext(ctx).Delete(&asset).Error
}

// GeneratePDF generates a PDF for an invoice using a template
func (s *InvoiceTemplateService) GeneratePDF(ctx context.Context, templateID, invoiceID uuid.UUID, data map[string]interface{}, userID *uuid.UUID) (*models.GeneratePDFResponse, error) {
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Generate PDF
	result, err := s.pdfService.GeneratePDFFromTemplate(ctx, template, data, "")

	// Record usage
	usage := &models.InvoiceTemplateUsage{
		TemplateID:       templateID,
		InvoiceID:        invoiceID,
		GeneratedBy:      userID,
		GenerationTimeMs: result.GenerationTimeMs,
		Success:          result.Success,
	}

	if result.Success {
		usage.PDFFilePath = result.FilePath
		usage.PDFFileSize = result.FileSize
	} else {
		usage.ErrorMessage = result.Error
	}

	s.db.WithContext(ctx).Create(usage) // Ignore error

	return result, err
}

// GeneratePreview generates a preview PDF of a template
func (s *InvoiceTemplateService) GeneratePreview(ctx context.Context, templateID uuid.UUID) (string, error) {
	template, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", err
	}

	// Use template's preview data or default sample data
	data := s.pdfService.GetDefaultSampleData()
	if template.PreviewData != nil && len(template.PreviewData) > 0 {
		data = template.PreviewData
	}

	return s.pdfService.GeneratePreview(ctx, template.HTMLContent, template.CSSContent, data)
}

// GeneratePreviewFromContent generates a preview from HTML/CSS content without saving
func (s *InvoiceTemplateService) GeneratePreviewFromContent(ctx context.Context, htmlContent, cssContent string, data map[string]interface{}) (string, error) {
	if data == nil {
		data = s.pdfService.GetDefaultSampleData()
	}

	return s.pdfService.GeneratePreview(ctx, htmlContent, cssContent, data)
}

// getImageDimensions returns the width and height of an image file
func (s *InvoiceTemplateService) getImageDimensions(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
