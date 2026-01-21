package templates

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Template CRUD Tests
// =============================================================================

func TestCreateTemplate(t *testing.T) {
	tests := []struct {
		name    string
		input   *CreateTemplateInput
		wantErr bool
		errType error
	}{
		{
			name: "valid transactional template",
			input: &CreateTemplateInput{
				Name:    "welcome_message",
				Type:    TemplateTypeTransactional,
				Content: "Welcome {{name}}! Your account is ready.",
			},
			wantErr: false,
		},
		{
			name: "valid OTP template",
			input: &CreateTemplateInput{
				Name:    "otp_verification",
				Type:    TemplateTypeOTP,
				Content: "Your verification code is {{code}}. Valid for {{minutes}} minutes.",
			},
			wantErr: false,
		},
		{
			name: "valid template with metadata",
			input: &CreateTemplateInput{
				Name:    "marketing_promo",
				Type:    TemplateTypeMarketing,
				Content: "Hi {{name}}, check out our sale!",
				Metadata: map[string]string{
					"campaign": "summer_sale",
					"version":  "A",
				},
			},
			wantErr: false,
		},
		{
			name: "template without variables",
			input: &CreateTemplateInput{
				Name:    "static_message",
				Type:    TemplateTypeNotification,
				Content: "System maintenance scheduled for tonight.",
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
			errType: ErrInvalidTemplate,
		},
		{
			name: "empty name",
			input: &CreateTemplateInput{
				Name:    "",
				Type:    TemplateTypeTransactional,
				Content: "Test content",
			},
			wantErr: true,
			errType: ErrInvalidTemplate,
		},
		{
			name: "empty type",
			input: &CreateTemplateInput{
				Name:    "test",
				Type:    "",
				Content: "Test content",
			},
			wantErr: true,
			errType: ErrInvalidTemplate,
		},
		{
			name: "invalid type",
			input: &CreateTemplateInput{
				Name:    "test",
				Type:    "invalid_type",
				Content: "Test content",
			},
			wantErr: true,
			errType: ErrInvalidTemplateType,
		},
		{
			name: "empty content",
			input: &CreateTemplateInput{
				Name:    "test",
				Type:    TemplateTypeTransactional,
				Content: "",
			},
			wantErr: true,
			errType: ErrEmptyContent,
		},
		{
			name: "content too long",
			input: &CreateTemplateInput{
				Name:    "test",
				Type:    TemplateTypeTransactional,
				Content: strings.Repeat("a", MaxConcatenatedLength+1),
			},
			wantErr: true,
			errType: ErrContentTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryTemplateStore()
			manager := NewTemplateManager(store)

			template, err := manager.CreateTemplate(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTemplate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateTemplate() unexpected error: %v", err)
				return
			}

			if template.ID == "" {
				t.Errorf("Template ID should not be empty")
			}

			if template.Name != tt.input.Name {
				t.Errorf("Name = %s, want %s", template.Name, tt.input.Name)
			}

			if template.Type != tt.input.Type {
				t.Errorf("Type = %s, want %s", template.Type, tt.input.Type)
			}

			if template.Version != 1 {
				t.Errorf("Initial version should be 1, got %d", template.Version)
			}

			if !template.IsActive {
				t.Errorf("New template should be active")
			}

			if template.IsArchived {
				t.Errorf("New template should not be archived")
			}
		})
	}
}

func TestCreateTemplateDuplicateName(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create first template
	input := &CreateTemplateInput{
		Name:    "unique_name",
		Type:    TemplateTypeTransactional,
		Content: "First template",
	}

	_, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("First CreateTemplate() failed: %v", err)
	}

	// Try to create with same name
	_, err = manager.CreateTemplate(ctx, input)
	if err == nil {
		t.Errorf("Second CreateTemplate() should fail with duplicate name")
	}
}

func TestGetTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "test_template",
		Type:    TemplateTypeNotification,
		Content: "Hello {{name}}!",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Get by ID
	template, err := manager.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Errorf("GetTemplate() failed: %v", err)
	}

	if template.Name != input.Name {
		t.Errorf("Name = %s, want %s", template.Name, input.Name)
	}

	// Get by name
	template, err = manager.GetTemplateByName(ctx, input.Name)
	if err != nil {
		t.Errorf("GetTemplateByName() failed: %v", err)
	}

	if template.ID != created.ID {
		t.Errorf("ID = %s, want %s", template.ID, created.ID)
	}

	// Get non-existent
	_, err = manager.GetTemplate(ctx, "non-existent-id")
	if err != ErrTemplateNotFound {
		t.Errorf("GetTemplate() for non-existent should return ErrTemplateNotFound, got %v", err)
	}
}

func TestUpdateTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "update_test",
		Type:    TemplateTypeTransactional,
		Content: "Original content {{var1}}",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Update name only (no version bump)
	updateInput := &UpdateTemplateInput{
		Name: "updated_name",
	}

	updated, err := manager.UpdateTemplate(ctx, created.ID, updateInput)
	if err != nil {
		t.Errorf("UpdateTemplate() failed: %v", err)
	}

	if updated.Name != "updated_name" {
		t.Errorf("Name not updated, got %s", updated.Name)
	}

	if updated.Version != 1 {
		t.Errorf("Version should not change for name-only update, got %d", updated.Version)
	}

	// Update content (should bump version)
	updateInput = &UpdateTemplateInput{
		Content:    "New content {{var2}}",
		ChangeNote: "Updated content",
	}

	updated, err = manager.UpdateTemplate(ctx, created.ID, updateInput)
	if err != nil {
		t.Errorf("UpdateTemplate() failed: %v", err)
	}

	if updated.Content != "New content {{var2}}" {
		t.Errorf("Content not updated")
	}

	if updated.Version != 2 {
		t.Errorf("Version should be 2 after content update, got %d", updated.Version)
	}

	// Check variables were updated
	if len(updated.Variables) != 1 || updated.Variables[0] != "var2" {
		t.Errorf("Variables not updated correctly, got %v", updated.Variables)
	}
}

func TestUpdateArchivedTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create and archive a template
	input := &CreateTemplateInput{
		Name:    "archived_test",
		Type:    TemplateTypeNotification,
		Content: "Test content",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	err = manager.DeleteTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteTemplate() failed: %v", err)
	}

	// Try to update archived template
	_, err = manager.UpdateTemplate(ctx, created.ID, &UpdateTemplateInput{
		Content: "New content",
	})

	if err != ErrTemplateArchived {
		t.Errorf("UpdateTemplate() should fail for archived template, got %v", err)
	}
}

func TestDeleteTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "delete_test",
		Type:    TemplateTypeAlert,
		Content: "Alert message",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Soft delete
	err = manager.DeleteTemplate(ctx, created.ID)
	if err != nil {
		t.Errorf("DeleteTemplate() failed: %v", err)
	}

	// Template should still be retrievable but archived
	template, err := manager.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Errorf("GetTemplate() failed after soft delete: %v", err)
	}

	if !template.IsArchived {
		t.Errorf("Template should be archived after DeleteTemplate()")
	}

	if template.IsActive {
		t.Errorf("Template should not be active after DeleteTemplate()")
	}

	// Hard delete
	err = manager.HardDeleteTemplate(ctx, created.ID)
	if err != nil {
		t.Errorf("HardDeleteTemplate() failed: %v", err)
	}

	// Template should not be retrievable
	_, err = manager.GetTemplate(ctx, created.ID)
	if err != ErrTemplateNotFound {
		t.Errorf("GetTemplate() should return ErrTemplateNotFound after hard delete")
	}
}

func TestListTemplates(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create multiple templates
	templates := []struct {
		name    string
		ttype   TemplateType
		content string
	}{
		{"template1", TemplateTypeTransactional, "Content 1"},
		{"template2", TemplateTypeMarketing, "Content 2"},
		{"template3", TemplateTypeTransactional, "Content 3"},
		{"template4", TemplateTypeOTP, "OTP: {{code}}"},
	}

	for _, tt := range templates {
		_, err := manager.CreateTemplate(ctx, &CreateTemplateInput{
			Name:    tt.name,
			Type:    tt.ttype,
			Content: tt.content,
		})
		if err != nil {
			t.Fatalf("CreateTemplate() failed: %v", err)
		}
	}

	// List all
	all, err := manager.ListTemplates(ctx, nil)
	if err != nil {
		t.Errorf("ListTemplates() failed: %v", err)
	}

	if len(all) != 4 {
		t.Errorf("ListTemplates() returned %d templates, want 4", len(all))
	}

	// Filter by type
	filtered, err := manager.ListTemplates(ctx, &TemplateFilter{
		Type: TemplateTypeTransactional,
	})
	if err != nil {
		t.Errorf("ListTemplates() with filter failed: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("ListTemplates(transactional) returned %d templates, want 2", len(filtered))
	}

	// Filter by active status
	active := true
	filtered, err = manager.ListTemplates(ctx, &TemplateFilter{
		IsActive: &active,
	})
	if err != nil {
		t.Errorf("ListTemplates() with active filter failed: %v", err)
	}

	if len(filtered) != 4 {
		t.Errorf("ListTemplates(active=true) returned %d templates, want 4", len(filtered))
	}

	// Filter by search
	filtered, err = manager.ListTemplates(ctx, &TemplateFilter{
		Search: "OTP",
	})
	if err != nil {
		t.Errorf("ListTemplates() with search failed: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("ListTemplates(search=OTP) returned %d templates, want 1", len(filtered))
	}
}

// =============================================================================
// Variable Substitution Tests
// =============================================================================

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single variable",
			content:  "Hello {{name}}!",
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			content:  "Hello {{first_name}} {{last_name}}!",
			expected: []string{"first_name", "last_name"},
		},
		{
			name:     "duplicate variables",
			content:  "{{name}} is {{name}}'s name",
			expected: []string{"name"},
		},
		{
			name:     "no variables",
			content:  "Hello World!",
			expected: []string{},
		},
		{
			name:     "variables with numbers",
			content:  "{{var1}} and {{var2}}",
			expected: []string{"var1", "var2"},
		},
		{
			name:     "underscore variables",
			content:  "{{first_name}} {{_private}}",
			expected: []string{"_private", "first_name"},
		},
		{
			name:     "mixed content",
			content:  "Dear {{customer_name}}, your order #{{order_id}} is ready. Total: {{amount}}",
			expected: []string{"amount", "customer_name", "order_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables := extractVariables(tt.content)

			if len(variables) != len(tt.expected) {
				t.Errorf("extractVariables() returned %d variables, want %d", len(variables), len(tt.expected))
				return
			}

			for i, v := range variables {
				if v != tt.expected[i] {
					t.Errorf("Variable %d = %s, want %s", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "substitution_test",
		Type:    TemplateTypeTransactional,
		Content: "Hello {{name}}, your code is {{code}}. Valid for {{minutes}} minutes.",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	tests := []struct {
		name      string
		variables map[string]string
		expected  string
		wantErr   bool
	}{
		{
			name: "all variables provided",
			variables: map[string]string{
				"name":    "John",
				"code":    "123456",
				"minutes": "5",
			},
			expected: "Hello John, your code is 123456. Valid for 5 minutes.",
			wantErr:  false,
		},
		{
			name: "missing variable",
			variables: map[string]string{
				"name": "John",
				"code": "123456",
			},
			wantErr: true,
		},
		{
			name:      "all variables missing",
			variables: map[string]string{},
			wantErr:   true,
		},
		{
			name: "extra variables ignored",
			variables: map[string]string{
				"name":    "John",
				"code":    "123456",
				"minutes": "5",
				"extra":   "ignored",
			},
			expected: "Hello John, your code is 123456. Valid for 5 minutes.",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.SubstituteVariables(ctx, created.ID, tt.variables)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SubstituteVariables() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SubstituteVariables() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Result = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestSubstituteVariablesInContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		expected  string
		wantErr   bool
	}{
		{
			name:    "simple substitution",
			content: "Hello {{name}}!",
			variables: map[string]string{
				"name": "World",
			},
			expected: "Hello World!",
			wantErr:  false,
		},
		{
			name:    "no variables in content",
			content: "Static message",
			variables: map[string]string{
				"unused": "value",
			},
			expected: "Static message",
			wantErr:  false,
		},
		{
			name:    "special characters in value",
			content: "Message: {{message}}",
			variables: map[string]string{
				"message": "Hello & goodbye! <test>",
			},
			expected: "Message: Hello & goodbye! <test>",
			wantErr:  false,
		},
		{
			name:    "unicode in value",
			content: "Greeting: {{greeting}}",
			variables: map[string]string{
				"greeting": "Hello",
			},
			expected: "Greeting: Hello",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SubstituteVariablesInContent(tt.content, tt.variables)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SubstituteVariablesInContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SubstituteVariablesInContent() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Result = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestBulkSubstitute(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "bulk_test",
		Type:    TemplateTypeMarketing,
		Content: "Hi {{name}}, your code is {{code}}.",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	recipientVars := []map[string]string{
		{"name": "Alice", "code": "111"},
		{"name": "Bob", "code": "222"},
		{"name": "Charlie"}, // Missing code
		{"name": "Diana", "code": "444"},
	}

	results, errs := manager.BulkSubstitute(ctx, created.ID, recipientVars)

	// Check successful substitutions
	if results[0] != "Hi Alice, your code is 111." {
		t.Errorf("Result[0] = %s, want 'Hi Alice, your code is 111.'", results[0])
	}

	if results[1] != "Hi Bob, your code is 222." {
		t.Errorf("Result[1] = %s, want 'Hi Bob, your code is 222.'", results[1])
	}

	// Check error for missing variable
	if errs[2] == nil {
		t.Errorf("Expected error for recipient 2 (missing code)")
	}

	if results[3] != "Hi Diana, your code is 444." {
		t.Errorf("Result[3] = %s, want 'Hi Diana, your code is 444.'", results[3])
	}
}

// =============================================================================
// Version Management Tests
// =============================================================================

func TestVersionControl(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:      "version_test",
		Type:      TemplateTypeTransactional,
		Content:   "Version 1 content",
		CreatedBy: "user1",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Check initial version
	if created.Version != 1 {
		t.Errorf("Initial version should be 1, got %d", created.Version)
	}

	// Update content multiple times
	for i := 2; i <= 5; i++ {
		_, err = manager.UpdateTemplate(ctx, created.ID, &UpdateTemplateInput{
			Content:    "Version " + string(rune('0'+i)) + " content",
			UpdatedBy:  "user1",
			ChangeNote: "Update to version " + string(rune('0'+i)),
		})
		if err != nil {
			t.Fatalf("UpdateTemplate() failed: %v", err)
		}
	}

	// Get current version
	current, err := manager.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTemplate() failed: %v", err)
	}

	if current.Version != 5 {
		t.Errorf("Current version should be 5, got %d", current.Version)
	}

	// Get all versions
	versions, err := manager.GetTemplateVersions(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTemplateVersions() failed: %v", err)
	}

	if len(versions) != 5 {
		t.Errorf("Expected 5 versions, got %d", len(versions))
	}

	// Get specific version
	v2, err := manager.GetTemplateVersion(ctx, created.ID, 2)
	if err != nil {
		t.Fatalf("GetTemplateVersion() failed: %v", err)
	}

	if v2.Version != 2 {
		t.Errorf("Version should be 2, got %d", v2.Version)
	}

	if v2.Content != "Version 2 content" {
		t.Errorf("Content = %s, want 'Version 2 content'", v2.Content)
	}
}

func TestRestoreVersion(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "restore_test",
		Type:    TemplateTypeNotification,
		Content: "Original content {{var1}}",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Update content
	_, err = manager.UpdateTemplate(ctx, created.ID, &UpdateTemplateInput{
		Content: "Updated content {{var2}}",
	})
	if err != nil {
		t.Fatalf("UpdateTemplate() failed: %v", err)
	}

	// Restore to version 1
	restored, err := manager.RestoreVersion(ctx, created.ID, 1, "admin")
	if err != nil {
		t.Fatalf("RestoreVersion() failed: %v", err)
	}

	if restored.Content != "Original content {{var1}}" {
		t.Errorf("Content not restored correctly")
	}

	// Version should increment
	if restored.Version != 3 {
		t.Errorf("Version should be 3 after restore, got %d", restored.Version)
	}

	// Variables should be restored
	if len(restored.Variables) != 1 || restored.Variables[0] != "var1" {
		t.Errorf("Variables not restored correctly, got %v", restored.Variables)
	}
}

// =============================================================================
// Preview Tests
// =============================================================================

func TestPreviewTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	input := &CreateTemplateInput{
		Name:    "preview_test",
		Type:    TemplateTypeOTP,
		Content: "Your code is {{code}}. Valid for {{minutes}} minutes.",
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Preview with all variables
	preview, err := manager.PreviewTemplate(ctx, created.ID, map[string]string{
		"code":    "123456",
		"minutes": "5",
	})
	if err != nil {
		t.Fatalf("PreviewTemplate() failed: %v", err)
	}

	if preview.RenderedContent != "Your code is 123456. Valid for 5 minutes." {
		t.Errorf("RenderedContent = %s, want 'Your code is 123456. Valid for 5 minutes.'", preview.RenderedContent)
	}

	expectedLen := len("Your code is 123456. Valid for 5 minutes.")
	if preview.CharacterCount != expectedLen {
		t.Errorf("CharacterCount = %d, want %d", preview.CharacterCount, expectedLen)
	}

	if preview.SegmentCount != 1 {
		t.Errorf("SegmentCount = %d, want 1", preview.SegmentCount)
	}

	// Preview with missing variables (should use placeholders)
	preview, err = manager.PreviewTemplate(ctx, created.ID, map[string]string{
		"code": "123456",
	})
	if err != nil {
		t.Fatalf("PreviewTemplate() failed: %v", err)
	}

	if preview.RenderedContent != "Your code is 123456. Valid for [minutes] minutes." {
		t.Errorf("RenderedContent with placeholder = %s", preview.RenderedContent)
	}

	if len(preview.Warnings) == 0 {
		t.Errorf("Expected warning for missing variable")
	}
}

func TestPreviewLongMessage(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template with long content
	longContent := "This is a very long message. " + strings.Repeat("Adding more text to make it longer. ", 10)
	input := &CreateTemplateInput{
		Name:    "long_preview_test",
		Type:    TemplateTypeMarketing,
		Content: longContent,
	}

	created, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	preview, err := manager.PreviewTemplate(ctx, created.ID, map[string]string{})
	if err != nil {
		t.Fatalf("PreviewTemplate() failed: %v", err)
	}

	if preview.SegmentCount <= 1 {
		t.Errorf("Long message should require multiple segments")
	}

	// Should have warning about length
	hasLengthWarning := false
	for _, w := range preview.Warnings {
		if strings.Contains(w, "exceeds single SMS length") {
			hasLengthWarning = true
			break
		}
	}

	if !hasLengthWarning {
		t.Errorf("Expected warning about message length")
	}
}

func TestPreviewTemplateContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		variables    map[string]string
		expectedLen  int
		expectedSegs int
	}{
		{
			name:         "short message",
			content:      "Hello {{name}}!",
			variables:    map[string]string{"name": "World"},
			expectedLen:  12,
			expectedSegs: 1,
		},
		{
			name:         "exactly 160 chars",
			content:      strings.Repeat("a", 160),
			variables:    map[string]string{},
			expectedLen:  160,
			expectedSegs: 1,
		},
		{
			name:         "161 chars (2 segments)",
			content:      strings.Repeat("a", 161),
			variables:    map[string]string{},
			expectedLen:  161,
			expectedSegs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview, err := PreviewTemplateContent(tt.content, TemplateTypeTransactional, tt.variables)
			if err != nil {
				t.Errorf("PreviewTemplateContent() failed: %v", err)
				return
			}

			if preview.CharacterCount != tt.expectedLen {
				t.Errorf("CharacterCount = %d, want %d", preview.CharacterCount, tt.expectedLen)
			}

			if preview.SegmentCount != tt.expectedSegs {
				t.Errorf("SegmentCount = %d, want %d", preview.SegmentCount, tt.expectedSegs)
			}
		})
	}
}

// =============================================================================
// Additional Tests
// =============================================================================

func TestCloneTemplate(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create original template
	input := &CreateTemplateInput{
		Name:    "original",
		Type:    TemplateTypeTransactional,
		Content: "Original content {{var}}",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	original, err := manager.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Clone the template
	clone, err := manager.CloneTemplate(ctx, original.ID, "cloned", "admin")
	if err != nil {
		t.Fatalf("CloneTemplate() failed: %v", err)
	}

	if clone.Name != "cloned" {
		t.Errorf("Clone name = %s, want 'cloned'", clone.Name)
	}

	if clone.Content != original.Content {
		t.Errorf("Clone content doesn't match original")
	}

	if clone.ID == original.ID {
		t.Errorf("Clone should have different ID")
	}

	if clone.Version != 1 {
		t.Errorf("Clone should start at version 1")
	}
}

func TestGetTemplateStats(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create templates of different types
	templates := []struct {
		name  string
		ttype TemplateType
	}{
		{"t1", TemplateTypeTransactional},
		{"t2", TemplateTypeTransactional},
		{"t3", TemplateTypeMarketing},
		{"t4", TemplateTypeOTP},
	}

	for _, tt := range templates {
		_, err := manager.CreateTemplate(ctx, &CreateTemplateInput{
			Name:    tt.name,
			Type:    tt.ttype,
			Content: "Content",
		})
		if err != nil {
			t.Fatalf("CreateTemplate() failed: %v", err)
		}
	}

	// Archive one
	all, _ := manager.ListTemplates(ctx, nil)
	_ = manager.DeleteTemplate(ctx, all[0].ID)

	stats, err := manager.GetTemplateStats(ctx)
	if err != nil {
		t.Fatalf("GetTemplateStats() failed: %v", err)
	}

	if stats.TotalTemplates != 4 {
		t.Errorf("TotalTemplates = %d, want 4", stats.TotalTemplates)
	}

	if stats.ActiveTemplates != 3 {
		t.Errorf("ActiveTemplates = %d, want 3", stats.ActiveTemplates)
	}

	if stats.ArchivedTemplates != 1 {
		t.Errorf("ArchivedTemplates = %d, want 1", stats.ArchivedTemplates)
	}

	if stats.ByType[TemplateTypeTransactional] != 2 {
		t.Errorf("ByType[transactional] = %d, want 2", stats.ByType[TemplateTypeTransactional])
	}
}

func TestValidateVariables(t *testing.T) {
	template := &SMSTemplate{
		Variables: []string{"name", "code"},
	}

	tests := []struct {
		name      string
		variables map[string]string
		wantErr   bool
	}{
		{
			name: "all variables provided",
			variables: map[string]string{
				"name": "John",
				"code": "123",
			},
			wantErr: false,
		},
		{
			name: "missing one variable",
			variables: map[string]string{
				"name": "John",
			},
			wantErr: true,
		},
		{
			name:      "all variables missing",
			variables: map[string]string{},
			wantErr:   true,
		},
		{
			name: "extra variables ok",
			variables: map[string]string{
				"name":  "John",
				"code":  "123",
				"extra": "ignored",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariables(template, tt.variables)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateVariables() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateVariables() unexpected error: %v", err)
			}
		})
	}
}

func TestCalculateSegments(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty", "", 1},
		{"short", "Hello", 1},
		{"exactly 160", strings.Repeat("a", 160), 1},
		{"161 chars", strings.Repeat("a", 161), 2},
		{"320 chars", strings.Repeat("a", 320), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSegments(tt.content)
			if result != tt.expected {
				t.Errorf("calculateSegments() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestHasNonGSMCharacters(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"ascii only", "Hello World!", false},
		{"with high unicode", "Hello \u00e9", true}, // é character
		{"with accented char", "Caf\u00e9", true},   // Café
		{"numbers", "123456", false},
		{"special ascii", "Hello! @#$%", false},
		{"with non-ascii", string([]byte{200}), true}, // Byte > 127
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNonGSMCharacters(tt.content)
			if result != tt.expected {
				t.Errorf("hasNonGSMCharacters(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestConcurrentTemplateOperations(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create a template
	template, err := manager.CreateTemplate(ctx, &CreateTemplateInput{
		Name:    "concurrent_test",
		Type:    TemplateTypeTransactional,
		Content: "Initial content {{var}}",
	})
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Concurrent reads and substitutions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			// Read
			_, err := manager.GetTemplate(ctx, template.ID)
			if err != nil {
				t.Errorf("Concurrent GetTemplate() failed: %v", err)
			}

			// Substitute
			_, err = manager.SubstituteVariables(ctx, template.ID, map[string]string{
				"var": "value",
			})
			if err != nil {
				t.Errorf("Concurrent SubstituteVariables() failed: %v", err)
			}

			// Preview
			_, err = manager.PreviewTemplate(ctx, template.ID, map[string]string{
				"var": "value",
			})
			if err != nil {
				t.Errorf("Concurrent PreviewTemplate() failed: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkExtractVariables(b *testing.B) {
	content := "Hello {{first_name}} {{last_name}}, your order #{{order_id}} totals {{amount}}."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractVariables(content)
	}
}

func BenchmarkSubstituteVariables(b *testing.B) {
	content := "Hello {{first_name}} {{last_name}}, your order #{{order_id}} totals {{amount}}."
	templateVars := []string{"first_name", "last_name", "order_id", "amount"}
	values := map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"order_id":   "12345",
		"amount":     "$99.99",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		substituteVariables(content, templateVars, values)
	}
}

func BenchmarkCalculateSegments(b *testing.B) {
	content := strings.Repeat("This is a test message. ", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateSegments(content)
	}
}

func BenchmarkPreviewTemplate(b *testing.B) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	template, _ := manager.CreateTemplate(ctx, &CreateTemplateInput{
		Name:    "bench_preview",
		Type:    TemplateTypeTransactional,
		Content: "Hello {{name}}, your order #{{order}} is ready for {{amount}}.",
	})

	variables := map[string]string{
		"name":   "John",
		"order":  "12345",
		"amount": "$99.99",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.PreviewTemplate(ctx, template.ID, variables)
	}
}

func BenchmarkCreateTemplate(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := NewInMemoryTemplateStore()
		manager := NewTemplateManager(store)
		b.StartTimer()

		manager.CreateTemplate(ctx, &CreateTemplateInput{
			Name:    "bench_template",
			Type:    TemplateTypeTransactional,
			Content: "Hello {{name}}!",
		})
	}
}

// =============================================================================
// Time-based Tests
// =============================================================================

func TestTemplateTimestamps(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	beforeCreate := time.Now()

	template, err := manager.CreateTemplate(ctx, &CreateTemplateInput{
		Name:    "timestamp_test",
		Type:    TemplateTypeTransactional,
		Content: "Test content",
	})
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	afterCreate := time.Now()

	if template.CreatedAt.Before(beforeCreate) || template.CreatedAt.After(afterCreate) {
		t.Errorf("CreatedAt timestamp out of range")
	}

	if template.UpdatedAt.Before(beforeCreate) || template.UpdatedAt.After(afterCreate) {
		t.Errorf("UpdatedAt timestamp out of range")
	}

	// Wait a bit and update
	time.Sleep(10 * time.Millisecond)
	beforeUpdate := time.Now()

	updated, err := manager.UpdateTemplate(ctx, template.ID, &UpdateTemplateInput{
		Content: "Updated content",
	})
	if err != nil {
		t.Fatalf("UpdateTemplate() failed: %v", err)
	}

	afterUpdate := time.Now()

	if updated.UpdatedAt.Before(beforeUpdate) || updated.UpdatedAt.After(afterUpdate) {
		t.Errorf("UpdatedAt timestamp not updated correctly")
	}

	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Errorf("UpdatedAt should be after CreatedAt")
	}
}

func TestListTemplatesDateFilter(t *testing.T) {
	store := NewInMemoryTemplateStore()
	manager := NewTemplateManager(store)
	ctx := context.Background()

	// Create templates at different times
	_, err := manager.CreateTemplate(ctx, &CreateTemplateInput{
		Name:    "old_template",
		Type:    TemplateTypeTransactional,
		Content: "Old",
	})
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	midTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	_, err = manager.CreateTemplate(ctx, &CreateTemplateInput{
		Name:    "new_template",
		Type:    TemplateTypeTransactional,
		Content: "New",
	})
	if err != nil {
		t.Fatalf("CreateTemplate() failed: %v", err)
	}

	// Filter by created after
	filtered, err := manager.ListTemplates(ctx, &TemplateFilter{
		CreatedAfter: &midTime,
	})
	if err != nil {
		t.Fatalf("ListTemplates() failed: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("Expected 1 template created after midTime, got %d", len(filtered))
	}

	// Filter by created before
	filtered, err = manager.ListTemplates(ctx, &TemplateFilter{
		CreatedBefore: &midTime,
	})
	if err != nil {
		t.Fatalf("ListTemplates() failed: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("Expected 1 template created before midTime, got %d", len(filtered))
	}
}
