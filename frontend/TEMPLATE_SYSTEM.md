# Quote Template System

A comprehensive template system for creating reusable quote templates with variable substitution and import/export functionality.

## Features

- **Template CRUD**: Create, read, update, delete quote templates
- **Variable System**: Dynamic variable substitution with {{variable}} syntax
- **Category Management**: Organize templates by categories
- **Import/Export**: JSON-based template sharing
- **Template Preview**: Real-time preview with variable substitution
- **Line Items**: Template line items with variable pricing
- **Validation**: Variable validation with custom rules

## Backend Architecture

### Models (`internal/models/quote_template.go`)

**QuoteTemplate:**

```go
type QuoteTemplate struct {
    ID             uuid.UUID
    Name           string
    Description    string
    Category       string
    Content        string              // Template content with {{variables}}
    Variables      []TemplateVariable  // Variable definitions
    LineItems      []TemplateLineItem  // Line item templates
    ValidDays      int                 // Default validity period
    PaymentTerms   string              // With variables
    DeliveryInfo   string              // With variables
    WarrantyInfo   string              // With variables
    DefaultTaxRate decimal.Decimal
    Tags           []string
    Version        int
    IsActive       bool
    IsPublic       bool
    UsageCount     int
    CreatedBy      uuid.UUID
    UpdatedBy      uuid.UUID
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**TemplateVariable:**

```go
type TemplateVariable struct {
    Name         string      // e.g., "customer_name"
    Type         string      // "text", "number", "date", "currency"
    Label        string      // Display label
    Description  string      // Help text
    DefaultValue interface{} // Optional default
    Required     bool
    Placeholder  string
    Validation   *VariableValidation
}
```

**VariableValidation:**

```go
type VariableValidation struct {
    MinLength *int
    MaxLength *int
    MinValue  *float64
    MaxValue  *float64
    Pattern   *string  // Regex
    Options   []string // For select/dropdown
}
```

### Template Engine (`internal/services/template_engine.go`)

**Variable Syntax:** `{{variable_name}}`

**Core Methods:**

- `RenderTemplate()`: Renders template with variables
- `SubstituteVariables()`: Replaces {{var}} with values
- `ValidateVariables()`: Validates all variables
- `ExtractVariables()`: Gets variable names from content
- `FormatValue()`: Formats values by type

**Example Usage:**

```go
engine := NewTemplateEngine()

// Render template
result, err := engine.RenderTemplate(template, variables)

// result.Content has all {{variables}} replaced
// result.LineItems has processed line items
// result.ValidUntil is calculated from ValidDays
```

**Variable Substitution:**

```go
// Template content
content := "Dear {{customer_name}}, your quote for {{project_name}} is ready."

// Variables
variables := map[string]interface{}{
    "customer_name": "John Doe",
    "project_name": "Website Redesign",
}

// Result
"Dear John Doe, your quote for Website Redesign is ready."
```

### Template Service (`internal/services/quote_template_service.go`)

**CRUD Operations:**

```go
service := NewQuoteTemplateService(db)

// Create
template, err := service.CreateTemplate(ctx, &template, userID)

// Read
template, err := service.GetTemplate(ctx, id)

// Update
template, err := service.UpdateTemplate(ctx, id, &updates, userID)

// Delete (soft delete)
err := service.DeleteTemplate(ctx, id)

// List with filters
templates, total, err := service.ListTemplates(ctx, &filter)
```

**Advanced Operations:**

```go
// Duplicate
duplicate, err := service.DuplicateTemplate(ctx, id, userID)

// Render
result, err := service.RenderTemplate(ctx, &renderRequest)

// Export
exportData, err := service.ExportTemplates(ctx, templateIDs)

// Import
imported, err := service.ImportTemplates(ctx, data, userID, overwrite)

// Stats
stats, err := service.GetTemplateStats(ctx)
```

## Frontend Architecture

### Types (`types/template.ts`)

All TypeScript interfaces match backend models:

- `QuoteTemplate`
- `TemplateVariable`
- `TemplateLineItem`
- `TemplateCategory`
- `TemplateImportExport`
- etc.

### Service (`services/templateService.ts`)

```typescript
// Create
const template = await templateService.createTemplate(data);

// List
const response = await templateService.listTemplates({
  category: 'Service',
  is_active: true,
  page: 1,
  page_size: 20,
});

// Render
const result = await templateService.renderTemplate({
  template_id: 'uuid',
  variables: {
    customer_name: 'John Doe',
    project_name: 'Website',
  },
});

// Export/Import
await templateService.downloadTemplatesJSON(['id1', 'id2']);
const count = await templateService.uploadTemplatesJSON(file, false);
```

## Component Structure

### 1. Template Editor

```typescript
// components/templates/TemplateEditor.tsx
interface TemplateEditorProps {
  templateId?: string;
  onSave?: (template: QuoteTemplate) => void;
  onCancel?: () => void;
}

// Features:
- Name, description, category fields
- Content editor with variable syntax highlighting
- Variable management (add/edit/delete)
- Line item templates
- Payment terms, delivery, warranty editors
- Real-time validation
- Save/cancel actions
```

### 2. Variable Editor

```typescript
// components/templates/VariableEditor.tsx
interface VariableEditorProps {
  variables: TemplateVariable[];
  onChange: (variables: TemplateVariable[]) => void;
}

// Features:
- Add/remove variables
- Configure name, type, label
- Set validation rules
- Default values
- Required flag
```

### 3. Template Preview

```typescript
// components/templates/TemplatePreview.tsx
interface TemplatePreviewProps {
  template: QuoteTemplate;
  variables: TemplateVariableMap;
  onVariableChange: (vars: TemplateVariableMap) => void;
}

// Features:
- Live preview of rendered template
- Variable input form
- Validation feedback
- Preview content, line items, terms
```

### 4. Template List

```typescript
// components/templates/TemplateList.tsx
interface TemplateListProps {
  onSelect?: (template: QuoteTemplate) => void;
  filter?: TemplateListFilter;
}

// Features:
- Filterable list
- Search by name/description
- Category filter
- Tag filter
- Actions: Edit, Duplicate, Delete, Use
- Usage statistics
```

### 5. Import/Export

```typescript
// components/templates/TemplateImportExport.tsx
interface TemplateImportExportProps {
  onImportComplete?: (count: number) => void;
}

// Features:
- Export selected templates to JSON
- Import templates from JSON file
- Overwrite option
- Progress feedback
- Error handling
```

### 6. Category Manager

```typescript
// components/templates/CategoryManager.tsx

// Features:
- Create/edit/delete categories
- Icon and color selection
- Sort order management
- Category list with template counts
```

## Variable System

### Supported Variable Types

**Text:**

```typescript
{
  name: 'customer_name',
  type: 'text',
  label: 'Customer Name',
  required: true,
  validation: {
    min_length: 2,
    max_length: 200,
  }
}
```

**Number:**

```typescript
{
  name: 'quantity',
  type: 'number',
  label: 'Quantity',
  validation: {
    min_value: 1,
    max_value: 9999,
  }
}
```

**Currency:**

```typescript
{
  name: 'price',
  type: 'currency',
  label: 'Price',
  validation: {
    min_value: 0,
    max_value: 999999.99,
  }
}
```

**Date:**

```typescript
{
  name: 'project_start',
  type: 'date',
  label: 'Project Start Date',
}
```

### Variable Usage in Templates

**In Content:**

```
Dear {{customer_name}},

Thank you for your interest in {{project_name}}. This quote is valid for {{valid_days}} days.

Project Details:
- Start Date: {{project_start}}
- Duration: {{project_duration}} weeks
- Total Cost: ${{total_cost}}

Best regards,
{{company_name}}
```

**In Line Items:**

```typescript
{
  description: "{{service_name}} - {{hours}} hours",
  quantity: 1,
  unit_price: "{{hourly_rate}}",
  is_variable: true
}
```

**In Terms:**

```
Payment Terms: {{payment_terms}}
Delivery: {{delivery_timeline}}
Warranty: {{warranty_period}} year(s)
```

## Import/Export Format

### Export Structure

```json
{
  "version": "1.0",
  "exported_at": "2024-01-15T10:00:00Z",
  "templates": [
    {
      "name": "Standard Service Quote",
      "description": "Template for service quotes",
      "category": "Service",
      "content": "Dear {{customer_name}}...",
      "variables": [
        {
          "name": "customer_name",
          "type": "text",
          "label": "Customer Name",
          "required": true
        }
      ],
      "line_items": [
        {
          "description": "{{service_name}}",
          "quantity": 1,
          "unit_price": 100,
          "is_variable": true
        }
      ],
      "valid_days": 30,
      "payment_terms": "Net 30",
      "tags": ["service", "standard"]
    }
  ],
  "categories": [
    {
      "name": "Service",
      "description": "Service-related templates",
      "icon": "wrench",
      "color": "#3B82F6"
    }
  ]
}
```

### Import/Export Operations

**Export:**

```typescript
// Export all templates
await templateService.downloadTemplatesJSON();

// Export specific templates
await templateService.downloadTemplatesJSON(['id1', 'id2']);
```

**Import:**

```typescript
// Import from file
const file = event.target.files[0];
const count = await templateService.uploadTemplatesJSON(file, false);
console.log(`Imported ${count} templates`);

// Import with overwrite
const count = await templateService.uploadTemplatesJSON(file, true);
```

## Usage Example

### Creating a Template

```typescript
const template: Partial<QuoteTemplate> = {
  name: 'Website Development Quote',
  description: 'Template for website projects',
  category: 'Web Development',
  content: `Dear {{customer_name}},

Thank you for your interest in our website development services.

Project: {{project_name}}
Estimated Timeline: {{timeline}} weeks
Total Investment: ${{ total_cost }}

This quote is valid until {{valid_until}}.`,

  variables: [
    {
      name: 'customer_name',
      type: 'text',
      label: 'Customer Name',
      required: true,
    },
    {
      name: 'project_name',
      type: 'text',
      label: 'Project Name',
      required: true,
    },
    {
      name: 'timeline',
      type: 'number',
      label: 'Timeline (weeks)',
      default_value: 8,
      validation: { min_value: 1, max_value: 52 },
    },
    {
      name: 'total_cost',
      type: 'currency',
      label: 'Total Cost',
      required: true,
    },
  ],

  line_items: [
    {
      description: 'Website Design - {{pages}} pages',
      quantity: 1,
      unit_price: 2000,
      is_variable: true,
    },
    {
      description: 'Development',
      quantity: 1,
      unit_price: 5000,
      is_variable: false,
    },
  ],

  valid_days: 30,
  payment_terms: 'Net 30',
  default_tax_rate: '0.0825',
  tags: ['web', 'development'],
};

const created = await templateService.createTemplate(template);
```

### Using a Template

```typescript
// Render template with variables
const result = await templateService.renderTemplate({
  template_id: template.id,
  variables: {
    customer_name: 'John Doe',
    project_name: 'E-commerce Website',
    timeline: 12,
    total_cost: 15000,
    pages: 10,
  },
});

// Create quote from rendered template
const quote = {
  customer_name: result.variables.customer_name,
  content: result.content,
  items: result.line_items,
  payment_terms: result.payment_terms,
  tax_rate: result.tax_rate,
  valid_until: result.valid_until,
};
```

## Best Practices

1. **Variable Naming**: Use snake_case (customer_name, not customerName)
2. **Required Variables**: Mark essential variables as required
3. **Default Values**: Provide sensible defaults for optional variables
4. **Validation**: Add validation rules to prevent invalid input
5. **Categories**: Organize templates into logical categories
6. **Tags**: Use tags for additional filtering/organization
7. **Testing**: Preview templates before saving
8. **Versioning**: Templates auto-increment version on update
9. **Public Templates**: Mark reusable templates as public
10. **Documentation**: Add clear descriptions to variables

## API Endpoints

```
POST   /api/v1/templates                - Create template
GET    /api/v1/templates/:id            - Get template
PUT    /api/v1/templates/:id            - Update template
DELETE /api/v1/templates/:id            - Delete template
GET    /api/v1/templates                - List templates
POST   /api/v1/templates/:id/duplicate  - Duplicate template
POST   /api/v1/templates/render         - Render template
GET    /api/v1/templates/stats          - Get statistics
GET    /api/v1/templates/export         - Export templates
POST   /api/v1/templates/import         - Import templates

POST   /api/v1/template-categories      - Create category
GET    /api/v1/template-categories      - List categories
PUT    /api/v1/template-categories/:id  - Update category
DELETE /api/v1/template-categories/:id  - Delete category
```

## Database Schema

```sql
CREATE TABLE quote_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    content TEXT NOT NULL,
    variables JSONB,
    line_items JSONB,
    valid_days INTEGER DEFAULT 30,
    payment_terms VARCHAR(500),
    delivery_info VARCHAR(500),
    warranty_info VARCHAR(500),
    default_tax_rate DECIMAL(10,4),
    tags TEXT[],
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE template_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(20),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_templates_category ON quote_templates(category);
CREATE INDEX idx_templates_created_by ON quote_templates(created_by);
CREATE INDEX idx_templates_is_active ON quote_templates(is_active);
```

## License

MIT
