# Quote Template Components

React components for the quote template system with variable substitution, CRUD operations, and import/export functionality.

## Components

### TemplateEditor

Main component for creating and editing quote templates.

```tsx
import { TemplateEditor } from './components/templates';

<TemplateEditor
  templateId="uuid-here" // Optional: for editing existing template
  onSave={(template) => console.log('Saved:', template)}
  onCancel={() => console.log('Cancelled')}
/>;
```

**Features:**

- Template metadata (name, description, category)
- Content editor with variable syntax highlighting
- Variable management
- Line items configuration
- Payment terms, delivery, and warranty editors
- Real-time validation
- Tabbed interface for organization

### VariableEditor

Standalone variable management component with validation rules.

```tsx
import { VariableEditor } from './components/templates';

<VariableEditor
  variables={variables}
  onChange={(newVariables) => setVariables(newVariables)}
  undefinedVariables={['customer_name', 'total_cost']}
/>;
```

**Features:**

- Add/edit/delete variables
- Type selection (text, number, currency, date)
- Validation rules (min/max length, min/max value, regex patterns)
- Default values
- Required field configuration
- Quick-add undefined variables

### TemplatePreview

Live preview component with variable substitution.

```tsx
import { TemplatePreview } from './components/templates';

<TemplatePreview
  template={template}
  variables={initialVariables}
  onVariableChange={(vars) => console.log('Variables:', vars)}
/>;
```

**Features:**

- Variable input form with validation
- Real-time preview rendering
- Line items display
- Terms and conditions display
- Quote metadata (valid until, tax rate)
- Error handling

### TemplateList

Browse, search, and filter templates.

```tsx
import { TemplateList } from './components/templates';

<TemplateList
  onSelect={(template) => console.log('Selected:', template)}
  onEdit={(template) => console.log('Edit:', template)}
  filter={{ category: 'Service', is_active: true }}
  selectable={true}
  multiSelect={true}
  selectedIds={selectedIds}
  onSelectionChange={(ids) => setSelectedIds(ids)}
/>;
```

**Features:**

- Search by name/description
- Filter by category, tags, status
- Sort by multiple criteria
- Pagination
- Quick actions (use, edit, duplicate, delete)
- Usage statistics display
- Selection mode for bulk operations

### TemplateImportExportComponent

Import and export templates as JSON.

```tsx
import { TemplateImportExportComponent } from './components/templates';

<TemplateImportExportComponent
  selectedTemplateIds={selectedIds}
  onImportComplete={(count) => console.log(`Imported ${count} templates`)}
/>;
```

**Features:**

- Export all or selected templates
- Import from JSON file
- Preview import data before confirming
- Overwrite mode option
- Validation and error handling
- Progress feedback

### CategoryManager

Manage template categories.

```tsx
import { CategoryManager } from './components/templates';

<CategoryManager />;
```

**Features:**

- Create/edit/delete categories
- Icon selection (emoji support)
- Color customization
- Sort order management
- Live preview
- Category list with template counts

## Variable Syntax

Templates use `{{variable_name}}` syntax for variable substitution:

```
Dear {{customer_name}},

Thank you for your interest in {{service_name}}.
This quote is valid for {{valid_days}} days.

Total Cost: ${{total_cost}}

Best regards,
{{company_name}}
```

## Variable Types

- **text**: String values with optional length validation
- **number**: Numeric values with min/max validation
- **currency**: Monetary values (formatted with $ sign)
- **date**: Date values (formatted as "Month Day, Year")

## Validation Rules

Variables can have the following validation rules:

```typescript
{
  min_length: 2,        // For text
  max_length: 200,      // For text
  min_value: 0,         // For number/currency
  max_value: 999999,    // For number/currency
  pattern: "^[A-Z].*",  // Regex pattern for text
  options: ["A", "B"]   // Allowed values for text
}
```

## Example Usage

### Creating a Template

```tsx
function CreateTemplate() {
  const [showEditor, setShowEditor] = useState(false);

  return (
    <>
      <button onClick={() => setShowEditor(true)}>Create Template</button>

      {showEditor && (
        <TemplateEditor
          onSave={(template) => {
            console.log('Template created:', template);
            setShowEditor(false);
          }}
          onCancel={() => setShowEditor(false)}
        />
      )}
    </>
  );
}
```

### Using a Template

```tsx
function UseTemplate() {
  const [selectedTemplate, setSelectedTemplate] = useState(null);

  return (
    <>
      <TemplateList onSelect={(template) => setSelectedTemplate(template)} />

      {selectedTemplate && (
        <TemplatePreview
          template={selectedTemplate}
          onVariableChange={(vars) => {
            console.log('Variables updated:', vars);
          }}
        />
      )}
    </>
  );
}
```

### Import/Export Templates

```tsx
function TemplateManager() {
  const [selectedIds, setSelectedIds] = useState([]);

  return (
    <>
      <TemplateList
        selectable={true}
        multiSelect={true}
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
      />

      <TemplateImportExportComponent
        selectedTemplateIds={selectedIds}
        onImportComplete={(count) => {
          alert(`Imported ${count} templates`);
          setSelectedIds([]);
        }}
      />
    </>
  );
}
```

## Dependencies

Required packages:

- `react` and `react-dom` (18+)
- `react-hook-form` - Form state management
- `@hookform/resolvers/zod` - Zod integration for validation
- `zod` - Schema validation
- `@tanstack/react-query` - Data fetching and caching
- `axios` - HTTP client

## Styling

Components use Tailwind CSS classes. Ensure your project has Tailwind CSS configured.

## API Integration

All components use the `templateService` from `src/services/templateService.ts` which provides:

- `createTemplate(template)`
- `getTemplate(id)`
- `updateTemplate(id, updates)`
- `deleteTemplate(id)`
- `listTemplates(filter)`
- `duplicateTemplate(id)`
- `renderTemplate(request)`
- `exportTemplates(ids)`
- `importTemplates(data, overwrite)`
- `downloadTemplatesJSON(ids)`
- `uploadTemplatesJSON(file, overwrite)`

See `TEMPLATE_SYSTEM.md` for complete API documentation.
