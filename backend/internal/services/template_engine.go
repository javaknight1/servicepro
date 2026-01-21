package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// ErrMissingVariable is returned when a required variable is missing
	ErrMissingVariable = errors.New("missing required variable")
	// ErrInvalidVariable is returned when a variable value is invalid
	ErrInvalidVariable = errors.New("invalid variable value")
)

// TemplateEngine handles variable substitution and template rendering
type TemplateEngine struct {
	// Variable syntax: {{variable_name}}
	variablePattern *regexp.Regexp
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		variablePattern: regexp.MustCompile(`\{\{([^}]+)\}\}`),
	}
}

// RenderTemplate renders a template with the provided variables
func (e *TemplateEngine) RenderTemplate(template *models.QuoteTemplate, variables models.TemplateVariableMap) (*models.TemplateRenderResult, error) {
	// Validate variables
	if err := e.ValidateVariables(template.Variables, variables); err != nil {
		return nil, err
	}

	// Render content
	content, err := e.SubstituteVariables(template.Content, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}

	// Render payment terms
	paymentTerms, err := e.SubstituteVariables(template.PaymentTerms, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render payment terms: %w", err)
	}

	// Render delivery info
	deliveryInfo, err := e.SubstituteVariables(template.DeliveryInfo, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render delivery info: %w", err)
	}

	// Render warranty info
	warrantyInfo, err := e.SubstituteVariables(template.WarrantyInfo, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render warranty info: %w", err)
	}

	// Process line items
	lineItems := make([]models.QuoteItem, len(template.LineItems))
	for i, item := range template.LineItems {
		description, err := e.SubstituteVariables(item.Description, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render line item %d: %w", i, err)
		}

		lineItems[i] = models.QuoteItem{
			Description: description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Total:       item.Quantity.Mul(item.UnitPrice),
			SortOrder:   i,
		}
	}

	// Calculate valid until date
	validUntil := time.Now().AddDate(0, 0, template.ValidDays)

	return &models.TemplateRenderResult{
		Content:      content,
		LineItems:    lineItems,
		PaymentTerms: paymentTerms,
		DeliveryInfo: deliveryInfo,
		WarrantyInfo: warrantyInfo,
		TaxRate:      template.DefaultTaxRate,
		ValidUntil:   validUntil,
		Variables:    variables,
	}, nil
}

// SubstituteVariables replaces variables in text with their values
func (e *TemplateEngine) SubstituteVariables(text string, variables models.TemplateVariableMap) (string, error) {
	result := text
	missingVars := []string{}

	// Find all variables in the text
	matches := e.variablePattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		fullMatch := match[0]                  // {{variable_name}}
		varName := strings.TrimSpace(match[1]) // variable_name

		// Get variable value
		value, exists := variables[varName]
		if !exists {
			missingVars = append(missingVars, varName)
			continue
		}

		// Format value based on type
		formattedValue := e.FormatValue(value)

		// Replace in result
		result = strings.ReplaceAll(result, fullMatch, formattedValue)
	}

	if len(missingVars) > 0 {
		return result, fmt.Errorf("%w: %s", ErrMissingVariable, strings.Join(missingVars, ", "))
	}

	return result, nil
}

// FormatValue formats a value for display
func (e *TemplateEngine) FormatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case decimal.Decimal:
		return v.String()
	case time.Time:
		return v.Format("January 2, 2006")
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ValidateVariables validates that all required variables are provided and valid
func (e *TemplateEngine) ValidateVariables(definitions []models.TemplateVariable, variables models.TemplateVariableMap) error {
	for _, def := range definitions {
		value, exists := variables[def.Name]

		// Check required
		if def.Required && !exists {
			return fmt.Errorf("%w: %s is required", ErrMissingVariable, def.Name)
		}

		if !exists {
			// Use default value if not required
			if def.DefaultValue != nil {
				variables[def.Name] = def.DefaultValue
			}
			continue
		}

		// Validate based on type
		if err := e.ValidateVariableValue(def, value); err != nil {
			return fmt.Errorf("%w: %s - %s", ErrInvalidVariable, def.Name, err)
		}
	}

	return nil
}

// ValidateVariableValue validates a single variable value
func (e *TemplateEngine) ValidateVariableValue(def models.TemplateVariable, value interface{}) error {
	if def.Validation == nil {
		return nil
	}

	switch def.Type {
	case "text":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

		if def.Validation.MinLength != nil && len(str) < *def.Validation.MinLength {
			return fmt.Errorf("minimum length is %d", *def.Validation.MinLength)
		}

		if def.Validation.MaxLength != nil && len(str) > *def.Validation.MaxLength {
			return fmt.Errorf("maximum length is %d", *def.Validation.MaxLength)
		}

		if def.Validation.Pattern != nil {
			matched, err := regexp.MatchString(*def.Validation.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern: %w", err)
			}
			if !matched {
				return fmt.Errorf("does not match required pattern")
			}
		}

		if len(def.Validation.Options) > 0 {
			valid := false
			for _, option := range def.Validation.Options {
				if str == option {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("must be one of: %s", strings.Join(def.Validation.Options, ", "))
			}
		}

	case "number", "currency":
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

		if def.Validation.MinValue != nil && num < *def.Validation.MinValue {
			return fmt.Errorf("minimum value is %.2f", *def.Validation.MinValue)
		}

		if def.Validation.MaxValue != nil && num > *def.Validation.MaxValue {
			return fmt.Errorf("maximum value is %.2f", *def.Validation.MaxValue)
		}

	case "date":
		_, ok := value.(time.Time)
		if !ok {
			// Try parsing string
			if str, ok := value.(string); ok {
				_, err := time.Parse("2006-01-02", str)
				if err != nil {
					return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
				}
			} else {
				return fmt.Errorf("expected date, got %T", value)
			}
		}
	}

	return nil
}

// ExtractVariables extracts all variable names from a text
func (e *TemplateEngine) ExtractVariables(text string) []string {
	matches := e.variablePattern.FindAllStringSubmatch(text, -1)
	variables := make([]string, 0, len(matches))

	seen := make(map[string]bool)
	for _, match := range matches {
		varName := strings.TrimSpace(match[1])
		if !seen[varName] {
			variables = append(variables, varName)
			seen[varName] = true
		}
	}

	return variables
}

// ValidateTemplateContent validates that all variables in content are defined
func (e *TemplateEngine) ValidateTemplateContent(content string, variables []models.TemplateVariable) error {
	usedVars := e.ExtractVariables(content)
	definedVars := make(map[string]bool)

	for _, v := range variables {
		definedVars[v.Name] = true
	}

	undefinedVars := []string{}
	for _, varName := range usedVars {
		if !definedVars[varName] {
			undefinedVars = append(undefinedVars, varName)
		}
	}

	if len(undefinedVars) > 0 {
		return fmt.Errorf("undefined variables in template: %s", strings.Join(undefinedVars, ", "))
	}

	return nil
}
