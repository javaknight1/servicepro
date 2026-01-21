package templates

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Standard errors
var (
	ErrTemplateNotFound    = errors.New("template not found")
	ErrTemplateExists      = errors.New("template already exists")
	ErrInvalidTemplate     = errors.New("invalid template")
	ErrInvalidVariables    = errors.New("invalid variables")
	ErrMissingVariable     = errors.New("missing required variable")
	ErrVersionConflict     = errors.New("version conflict")
	ErrTemplateArchived    = errors.New("template is archived")
	ErrInvalidTemplateType = errors.New("invalid template type")
	ErrContentTooLong      = errors.New("content exceeds maximum length")
	ErrEmptyContent        = errors.New("template content cannot be empty")
)

// TemplateType represents the type of SMS template
type TemplateType string

const (
	TemplateTypeTransactional TemplateType = "transactional"
	TemplateTypeMarketing     TemplateType = "marketing"
	TemplateTypeOTP           TemplateType = "otp"
	TemplateTypeNotification  TemplateType = "notification"
	TemplateTypeReminder      TemplateType = "reminder"
	TemplateTypeAlert         TemplateType = "alert"
)

// ValidTemplateTypes contains all valid template types
var ValidTemplateTypes = map[TemplateType]bool{
	TemplateTypeTransactional: true,
	TemplateTypeMarketing:     true,
	TemplateTypeOTP:           true,
	TemplateTypeNotification:  true,
	TemplateTypeReminder:      true,
	TemplateTypeAlert:         true,
}

// SMS character limits
const (
	MaxSMSLength            = 160  // Single SMS segment
	MaxConcatenatedLength   = 1600 // Maximum for concatenated SMS (10 segments)
	MaxVariableNameLength   = 50
	MaxVariablesPerTemplate = 20
)

// Variable pattern for substitution: {{variable_name}}
var variablePattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

// SMSTemplate represents an SMS message template
type SMSTemplate struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       TemplateType      `json:"type"`
	Content    string            `json:"content"`
	Variables  []string          `json:"variables"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Version    int               `json:"version"`
	IsActive   bool              `json:"is_active"`
	IsArchived bool              `json:"is_archived"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	CreatedBy  string            `json:"created_by,omitempty"`
	UpdatedBy  string            `json:"updated_by,omitempty"`
}

// TemplateVersion represents a historical version of a template
type TemplateVersion struct {
	TemplateID string    `json:"template_id"`
	Version    int       `json:"version"`
	Content    string    `json:"content"`
	Variables  []string  `json:"variables"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by,omitempty"`
	ChangeNote string    `json:"change_note,omitempty"`
}

// TemplatePreview represents a preview of a rendered template
type TemplatePreview struct {
	TemplateID      string            `json:"template_id"`
	TemplateName    string            `json:"template_name"`
	OriginalContent string            `json:"original_content"`
	RenderedContent string            `json:"rendered_content"`
	Variables       map[string]string `json:"variables"`
	CharacterCount  int               `json:"character_count"`
	SegmentCount    int               `json:"segment_count"`
	Warnings        []string          `json:"warnings,omitempty"`
}

// CreateTemplateInput holds the input for creating a template
type CreateTemplateInput struct {
	Name      string            `json:"name"`
	Type      TemplateType      `json:"type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedBy string            `json:"created_by,omitempty"`
}

// UpdateTemplateInput holds the input for updating a template
type UpdateTemplateInput struct {
	Name       string            `json:"name,omitempty"`
	Content    string            `json:"content,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	IsActive   *bool             `json:"is_active,omitempty"`
	UpdatedBy  string            `json:"updated_by,omitempty"`
	ChangeNote string            `json:"change_note,omitempty"`
}

// TemplateFilter holds filters for listing templates
type TemplateFilter struct {
	Type          TemplateType `json:"type,omitempty"`
	IsActive      *bool        `json:"is_active,omitempty"`
	IsArchived    *bool        `json:"is_archived,omitempty"`
	CreatedAfter  *time.Time   `json:"created_after,omitempty"`
	CreatedBefore *time.Time   `json:"created_before,omitempty"`
	Search        string       `json:"search,omitempty"`
}

// TemplateStore defines the interface for template storage
type TemplateStore interface {
	Create(ctx context.Context, template *SMSTemplate) error
	Get(ctx context.Context, id string) (*SMSTemplate, error)
	GetByName(ctx context.Context, name string) (*SMSTemplate, error)
	Update(ctx context.Context, template *SMSTemplate) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *TemplateFilter) ([]*SMSTemplate, error)
	SaveVersion(ctx context.Context, version *TemplateVersion) error
	GetVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error)
	GetVersion(ctx context.Context, templateID string, version int) (*TemplateVersion, error)
}

// InMemoryTemplateStore implements TemplateStore using in-memory storage
type InMemoryTemplateStore struct {
	mu        sync.RWMutex
	templates map[string]*SMSTemplate
	versions  map[string][]*TemplateVersion // templateID -> versions
	nameIndex map[string]string             // name -> id
}

// NewInMemoryTemplateStore creates a new in-memory template store
func NewInMemoryTemplateStore() *InMemoryTemplateStore {
	return &InMemoryTemplateStore{
		templates: make(map[string]*SMSTemplate),
		versions:  make(map[string][]*TemplateVersion),
		nameIndex: make(map[string]string),
	}
}

// Create stores a new template
func (s *InMemoryTemplateStore) Create(ctx context.Context, template *SMSTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.templates[template.ID]; exists {
		return ErrTemplateExists
	}

	if _, exists := s.nameIndex[template.Name]; exists {
		return fmt.Errorf("%w: name '%s' already in use", ErrTemplateExists, template.Name)
	}

	s.templates[template.ID] = template
	s.nameIndex[template.Name] = template.ID
	return nil
}

// Get retrieves a template by ID
func (s *InMemoryTemplateStore) Get(ctx context.Context, id string) (*SMSTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	template, exists := s.templates[id]
	if !exists {
		return nil, ErrTemplateNotFound
	}

	// Return a copy to prevent external modification
	copy := *template
	return &copy, nil
}

// GetByName retrieves a template by name
func (s *InMemoryTemplateStore) GetByName(ctx context.Context, name string) (*SMSTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.nameIndex[name]
	if !exists {
		return nil, ErrTemplateNotFound
	}

	template := s.templates[id]
	copy := *template
	return &copy, nil
}

// Update updates an existing template
func (s *InMemoryTemplateStore) Update(ctx context.Context, template *SMSTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.templates[template.ID]
	if !exists {
		return ErrTemplateNotFound
	}

	// Update name index if name changed
	if existing.Name != template.Name {
		// Check if new name is already in use
		if existingID, nameExists := s.nameIndex[template.Name]; nameExists && existingID != template.ID {
			return fmt.Errorf("%w: name '%s' already in use", ErrTemplateExists, template.Name)
		}
		delete(s.nameIndex, existing.Name)
		s.nameIndex[template.Name] = template.ID
	}

	s.templates[template.ID] = template
	return nil
}

// Delete removes a template
func (s *InMemoryTemplateStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	template, exists := s.templates[id]
	if !exists {
		return ErrTemplateNotFound
	}

	delete(s.nameIndex, template.Name)
	delete(s.templates, id)
	delete(s.versions, id)
	return nil
}

// List returns templates matching the filter
func (s *InMemoryTemplateStore) List(ctx context.Context, filter *TemplateFilter) ([]*SMSTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*SMSTemplate

	for _, template := range s.templates {
		if matchesFilter(template, filter) {
			copy := *template
			results = append(results, &copy)
		}
	}

	// Sort by created date descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results, nil
}

// SaveVersion saves a template version
func (s *InMemoryTemplateStore) SaveVersion(ctx context.Context, version *TemplateVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.versions[version.TemplateID] = append(s.versions[version.TemplateID], version)
	return nil
}

// GetVersions returns all versions of a template
func (s *InMemoryTemplateStore) GetVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions, exists := s.versions[templateID]
	if !exists {
		return []*TemplateVersion{}, nil
	}

	// Return copies
	result := make([]*TemplateVersion, len(versions))
	for i, v := range versions {
		copy := *v
		result[i] = &copy
	}

	return result, nil
}

// GetVersion returns a specific version of a template
func (s *InMemoryTemplateStore) GetVersion(ctx context.Context, templateID string, version int) (*TemplateVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions, exists := s.versions[templateID]
	if !exists {
		return nil, ErrTemplateNotFound
	}

	for _, v := range versions {
		if v.Version == version {
			copy := *v
			return &copy, nil
		}
	}

	return nil, ErrTemplateNotFound
}

// matchesFilter checks if a template matches the given filter
func matchesFilter(template *SMSTemplate, filter *TemplateFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Type != "" && template.Type != filter.Type {
		return false
	}

	if filter.IsActive != nil && template.IsActive != *filter.IsActive {
		return false
	}

	if filter.IsArchived != nil && template.IsArchived != *filter.IsArchived {
		return false
	}

	if filter.CreatedAfter != nil && template.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && template.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	if filter.Search != "" {
		search := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(template.Name), search) &&
			!strings.Contains(strings.ToLower(template.Content), search) {
			return false
		}
	}

	return true
}

// TemplateManager manages SMS templates
type TemplateManager struct {
	store TemplateStore
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(store TemplateStore) *TemplateManager {
	return &TemplateManager{store: store}
}

// CreateTemplate creates a new SMS template
func (m *TemplateManager) CreateTemplate(ctx context.Context, input *CreateTemplateInput) (*SMSTemplate, error) {
	// Validate input
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	// Extract variables from content
	variables := extractVariables(input.Content)

	// Create template
	now := time.Now()
	template := &SMSTemplate{
		ID:         uuid.New().String(),
		Name:       input.Name,
		Type:       input.Type,
		Content:    input.Content,
		Variables:  variables,
		Metadata:   input.Metadata,
		Version:    1,
		IsActive:   true,
		IsArchived: false,
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  input.CreatedBy,
	}

	// Store template
	if err := m.store.Create(ctx, template); err != nil {
		return nil, err
	}

	// Save initial version
	version := &TemplateVersion{
		TemplateID: template.ID,
		Version:    1,
		Content:    template.Content,
		Variables:  template.Variables,
		CreatedAt:  now,
		CreatedBy:  input.CreatedBy,
		ChangeNote: "Initial version",
	}
	if err := m.store.SaveVersion(ctx, version); err != nil {
		// Log but don't fail - template was created successfully
	}

	return template, nil
}

// UpdateTemplate updates an existing template
func (m *TemplateManager) UpdateTemplate(ctx context.Context, id string, input *UpdateTemplateInput) (*SMSTemplate, error) {
	// Get existing template
	template, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if template.IsArchived {
		return nil, ErrTemplateArchived
	}

	// Track if content changed for versioning
	contentChanged := false
	oldContent := template.Content

	// Apply updates
	if input.Name != "" {
		template.Name = input.Name
	}

	if input.Content != "" {
		if err := validateContent(input.Content); err != nil {
			return nil, err
		}
		if input.Content != template.Content {
			contentChanged = true
			template.Content = input.Content
			template.Variables = extractVariables(input.Content)
		}
	}

	if input.Metadata != nil {
		template.Metadata = input.Metadata
	}

	if input.IsActive != nil {
		template.IsActive = *input.IsActive
	}

	template.UpdatedAt = time.Now()
	template.UpdatedBy = input.UpdatedBy

	// Increment version if content changed
	if contentChanged {
		template.Version++

		// Save version history
		version := &TemplateVersion{
			TemplateID: template.ID,
			Version:    template.Version,
			Content:    template.Content,
			Variables:  template.Variables,
			CreatedAt:  template.UpdatedAt,
			CreatedBy:  input.UpdatedBy,
			ChangeNote: input.ChangeNote,
		}
		if err := m.store.SaveVersion(ctx, version); err != nil {
			// Log but don't fail
			_ = oldContent // suppress unused warning
		}
	}

	// Update in store
	if err := m.store.Update(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (m *TemplateManager) GetTemplate(ctx context.Context, id string) (*SMSTemplate, error) {
	return m.store.Get(ctx, id)
}

// GetTemplateByName retrieves a template by name
func (m *TemplateManager) GetTemplateByName(ctx context.Context, name string) (*SMSTemplate, error) {
	return m.store.GetByName(ctx, name)
}

// DeleteTemplate soft-deletes a template by archiving it
func (m *TemplateManager) DeleteTemplate(ctx context.Context, id string) error {
	template, err := m.store.Get(ctx, id)
	if err != nil {
		return err
	}

	template.IsArchived = true
	template.IsActive = false
	template.UpdatedAt = time.Now()

	return m.store.Update(ctx, template)
}

// HardDeleteTemplate permanently deletes a template
func (m *TemplateManager) HardDeleteTemplate(ctx context.Context, id string) error {
	return m.store.Delete(ctx, id)
}

// ListTemplates lists templates with optional filtering
func (m *TemplateManager) ListTemplates(ctx context.Context, filter *TemplateFilter) ([]*SMSTemplate, error) {
	return m.store.List(ctx, filter)
}

// GetTemplateVersions returns all versions of a template
func (m *TemplateManager) GetTemplateVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error) {
	return m.store.GetVersions(ctx, templateID)
}

// GetTemplateVersion returns a specific version of a template
func (m *TemplateManager) GetTemplateVersion(ctx context.Context, templateID string, version int) (*TemplateVersion, error) {
	return m.store.GetVersion(ctx, templateID, version)
}

// RestoreVersion restores a template to a previous version
func (m *TemplateManager) RestoreVersion(ctx context.Context, templateID string, version int, updatedBy string) (*SMSTemplate, error) {
	// Get the version to restore
	targetVersion, err := m.store.GetVersion(ctx, templateID, version)
	if err != nil {
		return nil, err
	}

	// Update the template with the old version's content
	input := &UpdateTemplateInput{
		Content:    targetVersion.Content,
		UpdatedBy:  updatedBy,
		ChangeNote: fmt.Sprintf("Restored to version %d", version),
	}

	return m.UpdateTemplate(ctx, templateID, input)
}

// SubstituteVariables replaces variables in template content with provided values
func (m *TemplateManager) SubstituteVariables(ctx context.Context, templateID string, variables map[string]string) (string, error) {
	template, err := m.store.Get(ctx, templateID)
	if err != nil {
		return "", err
	}

	return substituteVariables(template.Content, template.Variables, variables)
}

// SubstituteVariablesInContent substitutes variables in arbitrary content
func SubstituteVariablesInContent(content string, variables map[string]string) (string, error) {
	templateVars := extractVariables(content)
	return substituteVariables(content, templateVars, variables)
}

// substituteVariables performs the actual variable substitution
func substituteVariables(content string, templateVars []string, values map[string]string) (string, error) {
	// Check for missing required variables
	for _, varName := range templateVars {
		if _, exists := values[varName]; !exists {
			return "", fmt.Errorf("%w: %s", ErrMissingVariable, varName)
		}
	}

	// Perform substitution
	result := variablePattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from {{name}}
		varName := match[2 : len(match)-2]
		if value, exists := values[varName]; exists {
			return value
		}
		return match // Keep original if not found
	})

	return result, nil
}

// PreviewTemplate generates a preview of a rendered template
func (m *TemplateManager) PreviewTemplate(ctx context.Context, templateID string, variables map[string]string) (*TemplatePreview, error) {
	template, err := m.store.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}

	return generatePreview(template, variables)
}

// PreviewTemplateContent generates a preview for arbitrary content
func PreviewTemplateContent(content string, templateType TemplateType, variables map[string]string) (*TemplatePreview, error) {
	template := &SMSTemplate{
		ID:        "preview",
		Name:      "Preview",
		Type:      templateType,
		Content:   content,
		Variables: extractVariables(content),
	}

	return generatePreview(template, variables)
}

// generatePreview creates a preview from a template and variables
func generatePreview(template *SMSTemplate, variables map[string]string) (*TemplatePreview, error) {
	// Fill in missing variables with placeholders for preview
	previewVars := make(map[string]string)
	for k, v := range variables {
		previewVars[k] = v
	}

	// Add placeholders for missing variables
	warnings := []string{}
	for _, varName := range template.Variables {
		if _, exists := previewVars[varName]; !exists {
			previewVars[varName] = fmt.Sprintf("[%s]", varName)
			warnings = append(warnings, fmt.Sprintf("Variable '%s' not provided, using placeholder", varName))
		}
	}

	// Render content
	rendered, err := substituteVariables(template.Content, template.Variables, previewVars)
	if err != nil {
		return nil, err
	}

	// Calculate character count and segments
	charCount := len(rendered)
	segmentCount := calculateSegments(rendered)

	// Add warnings for length
	if charCount > MaxSMSLength {
		warnings = append(warnings, fmt.Sprintf("Message exceeds single SMS length (%d chars), will be sent as %d segments", charCount, segmentCount))
	}
	if charCount > MaxConcatenatedLength {
		warnings = append(warnings, fmt.Sprintf("Message exceeds maximum length of %d characters", MaxConcatenatedLength))
	}

	// Check for non-GSM characters
	if hasNonGSMCharacters(rendered) {
		warnings = append(warnings, "Message contains non-GSM characters, character limits may be reduced")
	}

	return &TemplatePreview{
		TemplateID:      template.ID,
		TemplateName:    template.Name,
		OriginalContent: template.Content,
		RenderedContent: rendered,
		Variables:       previewVars,
		CharacterCount:  charCount,
		SegmentCount:    segmentCount,
		Warnings:        warnings,
	}, nil
}

// extractVariables extracts variable names from template content
func extractVariables(content string) []string {
	matches := variablePattern.FindAllStringSubmatch(content, -1)

	// Use map to deduplicate
	varMap := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			varMap[match[1]] = true
		}
	}

	// Convert to sorted slice
	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}
	sort.Strings(vars)

	return vars
}

// calculateSegments calculates the number of SMS segments needed
func calculateSegments(content string) int {
	length := len(content)

	if hasNonGSMCharacters(content) {
		// Unicode SMS: 70 chars per segment, 67 for concatenated
		if length <= 70 {
			return 1
		}
		return (length + 66) / 67 // Ceiling division
	}

	// GSM-7 SMS: 160 chars per segment, 153 for concatenated
	if length <= 160 {
		return 1
	}
	return (length + 152) / 153 // Ceiling division
}

// hasNonGSMCharacters checks if content contains non-GSM-7 characters
func hasNonGSMCharacters(content string) bool {
	// GSM-7 basic character set (simplified check)
	for _, r := range content {
		if r > 127 {
			return true
		}
	}
	return false
}

// validateCreateInput validates the input for creating a template
func validateCreateInput(input *CreateTemplateInput) error {
	if input == nil {
		return fmt.Errorf("%w: input is nil", ErrInvalidTemplate)
	}

	if input.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTemplate)
	}

	if input.Type == "" {
		return fmt.Errorf("%w: type is required", ErrInvalidTemplate)
	}

	if !ValidTemplateTypes[input.Type] {
		return fmt.Errorf("%w: %s", ErrInvalidTemplateType, input.Type)
	}

	return validateContent(input.Content)
}

// validateContent validates template content
func validateContent(content string) error {
	if content == "" {
		return ErrEmptyContent
	}

	if len(content) > MaxConcatenatedLength {
		return fmt.Errorf("%w: %d characters (max %d)", ErrContentTooLong, len(content), MaxConcatenatedLength)
	}

	// Validate variables in content
	variables := extractVariables(content)
	if len(variables) > MaxVariablesPerTemplate {
		return fmt.Errorf("%w: %d variables (max %d)", ErrInvalidVariables, len(variables), MaxVariablesPerTemplate)
	}

	for _, varName := range variables {
		if len(varName) > MaxVariableNameLength {
			return fmt.Errorf("%w: variable name '%s' too long (max %d)", ErrInvalidVariables, varName, MaxVariableNameLength)
		}
	}

	return nil
}

// ValidateVariables validates that all required variables are provided
func ValidateVariables(template *SMSTemplate, variables map[string]string) error {
	for _, varName := range template.Variables {
		if _, exists := variables[varName]; !exists {
			return fmt.Errorf("%w: %s", ErrMissingVariable, varName)
		}
	}
	return nil
}

// CloneTemplate creates a copy of a template with a new name
func (m *TemplateManager) CloneTemplate(ctx context.Context, templateID, newName, createdBy string) (*SMSTemplate, error) {
	// Get the source template
	source, err := m.store.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Create a new template based on the source
	input := &CreateTemplateInput{
		Name:      newName,
		Type:      source.Type,
		Content:   source.Content,
		Metadata:  source.Metadata,
		CreatedBy: createdBy,
	}

	return m.CreateTemplate(ctx, input)
}

// BulkSubstitute performs variable substitution for multiple recipients
func (m *TemplateManager) BulkSubstitute(ctx context.Context, templateID string, recipientVariables []map[string]string) ([]string, []error) {
	template, err := m.store.Get(ctx, templateID)
	if err != nil {
		errors := make([]error, len(recipientVariables))
		for i := range errors {
			errors[i] = err
		}
		return nil, errors
	}

	results := make([]string, len(recipientVariables))
	errs := make([]error, len(recipientVariables))

	for i, vars := range recipientVariables {
		rendered, err := substituteVariables(template.Content, template.Variables, vars)
		if err != nil {
			errs[i] = err
		} else {
			results[i] = rendered
		}
	}

	return results, errs
}

// GetTemplateStats returns statistics about templates
func (m *TemplateManager) GetTemplateStats(ctx context.Context) (*TemplateStats, error) {
	templates, err := m.store.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	stats := &TemplateStats{
		TotalTemplates:    len(templates),
		ByType:            make(map[TemplateType]int),
		ActiveTemplates:   0,
		ArchivedTemplates: 0,
	}

	for _, t := range templates {
		stats.ByType[t.Type]++
		if t.IsActive {
			stats.ActiveTemplates++
		}
		if t.IsArchived {
			stats.ArchivedTemplates++
		}
	}

	return stats, nil
}

// TemplateStats holds statistics about templates
type TemplateStats struct {
	TotalTemplates    int                  `json:"total_templates"`
	ActiveTemplates   int                  `json:"active_templates"`
	ArchivedTemplates int                  `json:"archived_templates"`
	ByType            map[TemplateType]int `json:"by_type"`
}
