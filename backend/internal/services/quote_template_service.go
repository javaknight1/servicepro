package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// ErrTemplateNotFound is returned when a template is not found
	ErrTemplateNotFound = errors.New("template not found")
	// ErrCategoryNotFound is returned when a category is not found
	ErrCategoryNotFound = errors.New("category not found")
)

// QuoteTemplateService handles quote template operations
type QuoteTemplateService struct {
	db     *gorm.DB
	engine *TemplateEngine
}

// NewQuoteTemplateService creates a new quote template service
func NewQuoteTemplateService(db *gorm.DB) *QuoteTemplateService {
	return &QuoteTemplateService{
		db:     db,
		engine: NewTemplateEngine(),
	}
}

// CreateTemplate creates a new quote template
func (s *QuoteTemplateService) CreateTemplate(ctx context.Context, template *models.QuoteTemplate, userID uuid.UUID) (*models.QuoteTemplate, error) {
	// Validate template content
	if err := s.engine.ValidateTemplateContent(template.Content, template.Variables); err != nil {
		return nil, fmt.Errorf("invalid template content: %w", err)
	}

	// Set metadata
	template.CreatedBy = userID
	template.Version = 1
	template.UsageCount = 0

	// Create template
	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (s *QuoteTemplateService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.QuoteTemplate, error) {
	var template models.QuoteTemplate
	if err := s.db.WithContext(ctx).First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// UpdateTemplate updates an existing template
func (s *QuoteTemplateService) UpdateTemplate(ctx context.Context, id uuid.UUID, updates *models.QuoteTemplate, userID uuid.UUID) (*models.QuoteTemplate, error) {
	// Get existing template
	existing, err := s.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate updated content
	if updates.Content != "" {
		if err := s.engine.ValidateTemplateContent(updates.Content, updates.Variables); err != nil {
			return nil, fmt.Errorf("invalid template content: %w", err)
		}
	}

	// Increment version
	updates.Version = existing.Version + 1
	updates.UpdatedBy = userID

	// Update template
	if err := s.db.WithContext(ctx).Model(&models.QuoteTemplate{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return s.GetTemplate(ctx, id)
}

// DeleteTemplate soft deletes a template
func (s *QuoteTemplateService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Model(&models.QuoteTemplate{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to delete template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}

// ListTemplates retrieves templates with filtering
func (s *QuoteTemplateService) ListTemplates(ctx context.Context, filter *models.TemplateListFilter) ([]models.QuoteTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&models.QuoteTemplate{})

	// Apply filters
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if len(filter.Tags) > 0 {
		query = query.Where("tags && ?", filter.Tags)
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Apply sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = filter.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	page := 1
	if filter.Page > 0 {
		page = filter.Page
	}
	pageSize := 20
	if filter.PageSize > 0 {
		pageSize = filter.PageSize
	}
	offset := (page - 1) * pageSize

	query = query.Offset(offset).Limit(pageSize)

	// Execute query
	var templates []models.QuoteTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// RenderTemplate renders a template with variables
func (s *QuoteTemplateService) RenderTemplate(ctx context.Context, req *models.TemplateRenderRequest) (*models.TemplateRenderResult, error) {
	// Get template
	template, err := s.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	// Render template
	result, err := s.engine.RenderTemplate(template, req.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Increment usage count
	s.db.WithContext(ctx).Model(&models.QuoteTemplate{}).
		Where("id = ?", req.TemplateID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))

	return result, nil
}

// DuplicateTemplate creates a copy of an existing template
func (s *QuoteTemplateService) DuplicateTemplate(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.QuoteTemplate, error) {
	// Get original template
	original, err := s.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create copy
	duplicate := &models.QuoteTemplate{
		Name:           original.Name + " (Copy)",
		Description:    original.Description,
		Category:       original.Category,
		Content:        original.Content,
		Variables:      original.Variables,
		LineItems:      original.LineItems,
		ValidDays:      original.ValidDays,
		PaymentTerms:   original.PaymentTerms,
		DeliveryInfo:   original.DeliveryInfo,
		WarrantyInfo:   original.WarrantyInfo,
		DefaultTaxRate: original.DefaultTaxRate,
		Tags:           original.Tags,
		IsPublic:       false, // Always private by default
		CreatedBy:      userID,
	}

	return s.CreateTemplate(ctx, duplicate, userID)
}

// ExportTemplates exports templates to JSON
func (s *QuoteTemplateService) ExportTemplates(ctx context.Context, templateIDs []uuid.UUID) (*models.TemplateImportExport, error) {
	var templates []models.QuoteTemplate

	// Get templates
	if len(templateIDs) == 0 {
		// Export all active templates
		if err := s.db.WithContext(ctx).
			Where("is_active = ?", true).
			Find(&templates).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch templates: %w", err)
		}
	} else {
		if err := s.db.WithContext(ctx).
			Where("id IN ?", templateIDs).
			Find(&templates).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch templates: %w", err)
		}
	}

	// Convert to export format
	exports := make([]models.QuoteTemplateExport, len(templates))
	categories := make(map[string]bool)

	for i, t := range templates {
		exports[i] = models.QuoteTemplateExport{
			Name:           t.Name,
			Description:    t.Description,
			Category:       t.Category,
			Content:        t.Content,
			Variables:      t.Variables,
			LineItems:      t.LineItems,
			ValidDays:      t.ValidDays,
			PaymentTerms:   t.PaymentTerms,
			DeliveryInfo:   t.DeliveryInfo,
			WarrantyInfo:   t.WarrantyInfo,
			DefaultTaxRate: t.DefaultTaxRate,
			Tags:           t.Tags,
			IsPublic:       t.IsPublic,
		}
		categories[t.Category] = true
	}

	// Get unique categories
	var categoryRecords []models.TemplateCategory
	categoryNames := make([]string, 0, len(categories))
	for cat := range categories {
		if cat != "" {
			categoryNames = append(categoryNames, cat)
		}
	}

	if len(categoryNames) > 0 {
		s.db.WithContext(ctx).
			Where("name IN ?", categoryNames).
			Find(&categoryRecords)
	}

	categoryExports := make([]models.TemplateCategoryExport, len(categoryRecords))
	for i, cat := range categoryRecords {
		categoryExports[i] = models.TemplateCategoryExport{
			Name:        cat.Name,
			Description: cat.Description,
			Icon:        cat.Icon,
			Color:       cat.Color,
			SortOrder:   cat.SortOrder,
		}
	}

	return &models.TemplateImportExport{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Templates:  exports,
		Categories: categoryExports,
	}, nil
}

// ImportTemplates imports templates from JSON
func (s *QuoteTemplateService) ImportTemplates(ctx context.Context, data *models.TemplateImportExport, userID uuid.UUID, overwrite bool) (int, error) {
	imported := 0

	// Import categories first
	for _, catExport := range data.Categories {
		var existing models.TemplateCategory
		err := s.db.WithContext(ctx).
			Where("name = ?", catExport.Name).
			First(&existing).Error

		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new category
			category := &models.TemplateCategory{
				Name:        catExport.Name,
				Description: catExport.Description,
				Icon:        catExport.Icon,
				Color:       catExport.Color,
				SortOrder:   catExport.SortOrder,
				IsActive:    true,
			}
			s.db.WithContext(ctx).Create(category)
		} else if overwrite {
			// Update existing
			s.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
				"description": catExport.Description,
				"icon":        catExport.Icon,
				"color":       catExport.Color,
				"sort_order":  catExport.SortOrder,
			})
		}
	}

	// Import templates
	for _, tplExport := range data.Templates {
		var existing models.QuoteTemplate
		err := s.db.WithContext(ctx).
			Where("name = ? AND created_by = ?", tplExport.Name, userID).
			First(&existing).Error

		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new template
			template := &models.QuoteTemplate{
				Name:           tplExport.Name,
				Description:    tplExport.Description,
				Category:       tplExport.Category,
				Content:        tplExport.Content,
				Variables:      tplExport.Variables,
				LineItems:      tplExport.LineItems,
				ValidDays:      tplExport.ValidDays,
				PaymentTerms:   tplExport.PaymentTerms,
				DeliveryInfo:   tplExport.DeliveryInfo,
				WarrantyInfo:   tplExport.WarrantyInfo,
				DefaultTaxRate: tplExport.DefaultTaxRate,
				Tags:           tplExport.Tags,
				IsPublic:       tplExport.IsPublic,
				CreatedBy:      userID,
				IsActive:       true,
			}

			if _, err := s.CreateTemplate(ctx, template, userID); err == nil {
				imported++
			}
		} else if overwrite {
			// Update existing
			updates := &models.QuoteTemplate{
				Description:    tplExport.Description,
				Category:       tplExport.Category,
				Content:        tplExport.Content,
				Variables:      tplExport.Variables,
				LineItems:      tplExport.LineItems,
				ValidDays:      tplExport.ValidDays,
				PaymentTerms:   tplExport.PaymentTerms,
				DeliveryInfo:   tplExport.DeliveryInfo,
				WarrantyInfo:   tplExport.WarrantyInfo,
				DefaultTaxRate: tplExport.DefaultTaxRate,
				Tags:           tplExport.Tags,
			}

			if _, err := s.UpdateTemplate(ctx, existing.ID, updates, userID); err == nil {
				imported++
			}
		}
	}

	return imported, nil
}

// GetTemplateStats retrieves template statistics
func (s *QuoteTemplateService) GetTemplateStats(ctx context.Context) (*models.TemplateStats, error) {
	stats := &models.TemplateStats{
		ByCategory: make(map[string]int),
	}

	var totalTemplates, activeTemplates, totalCategories int64

	// Total templates
	s.db.WithContext(ctx).Model(&models.QuoteTemplate{}).Count(&totalTemplates)
	stats.TotalTemplates = int(totalTemplates)

	// Active templates
	s.db.WithContext(ctx).Model(&models.QuoteTemplate{}).
		Where("is_active = ?", true).
		Count(&activeTemplates)
	stats.ActiveTemplates = int(activeTemplates)

	// Total categories
	s.db.WithContext(ctx).Model(&models.TemplateCategory{}).
		Where("is_active = ?", true).
		Count(&totalCategories)
	stats.TotalCategories = int(totalCategories)

	// Total usage
	s.db.WithContext(ctx).Model(&models.QuoteTemplate{}).
		Select("COALESCE(SUM(usage_count), 0)").
		Scan(&stats.TotalUsage)

	// By category
	rows, err := s.db.WithContext(ctx).
		Model(&models.QuoteTemplate{}).
		Select("category, COUNT(*) as count").
		Where("is_active = ?", true).
		Group("category").
		Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int
			rows.Scan(&category, &count)
			stats.ByCategory[category] = count
		}
	}

	// Most used templates
	s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("usage_count DESC").
		Limit(5).
		Find(&stats.MostUsed)

	// Recently updated
	s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("updated_at DESC").
		Limit(5).
		Find(&stats.RecentlyUpdated)

	return stats, nil
}

// Category Management

// CreateCategory creates a new template category
func (s *QuoteTemplateService) CreateCategory(ctx context.Context, category *models.TemplateCategory) (*models.TemplateCategory, error) {
	if err := s.db.WithContext(ctx).Create(category).Error; err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	return category, nil
}

// ListCategories retrieves all active categories
func (s *QuoteTemplateService) ListCategories(ctx context.Context) ([]models.TemplateCategory, error) {
	var categories []models.TemplateCategory
	if err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}

// ExportTemplatesToJSON exports templates to JSON string
func (s *QuoteTemplateService) ExportTemplatesToJSON(ctx context.Context, templateIDs []uuid.UUID) (string, error) {
	export, err := s.ExportTemplates(ctx, templateIDs)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// ImportTemplatesFromJSON imports templates from JSON string
func (s *QuoteTemplateService) ImportTemplatesFromJSON(ctx context.Context, jsonData string, userID uuid.UUID, overwrite bool) (int, error) {
	var data models.TemplateImportExport
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return s.ImportTemplates(ctx, &data, userID, overwrite)
}
