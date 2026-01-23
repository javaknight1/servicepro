# Templates API

REST API for managing invoice and document templates.

## Overview

The Templates API provides:

- Invoice template management
- PDF generation with templates
- Template versioning
- Variable substitution
- Custom branding options

## Endpoints

| Method | Endpoint                 | Description          |
| ------ | ------------------------ | -------------------- |
| GET    | `/templates`             | List templates       |
| GET    | `/templates/:id`         | Get template         |
| POST   | `/templates`             | Create template      |
| PUT    | `/templates/:id`         | Update template      |
| DELETE | `/templates/:id`         | Delete template      |
| POST   | `/templates/:id/preview` | Preview with data    |
| POST   | `/templates/:id/clone`   | Clone template       |
| GET    | `/templates/default`     | Get default template |

## List Templates

Get all available templates.

```http
GET /api/v1/templates
```

### Query Parameters

| Parameter | Type    | Description                             |
| --------- | ------- | --------------------------------------- |
| type      | string  | Filter by type: invoice, quote, receipt |
| active    | boolean | Filter by active status                 |
| page      | integer | Page number                             |
| page_size | integer | Items per page                          |

### Response

```json
{
  "templates": [
    {
      "id": "uuid",
      "name": "Professional Invoice",
      "type": "invoice",
      "description": "Clean, professional invoice template",
      "is_default": true,
      "is_active": true,
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

## Get Template

Get a template with full details.

```http
GET /api/v1/templates/:id
```

### Response

```json
{
  "id": "uuid",
  "name": "Professional Invoice",
  "type": "invoice",
  "description": "Clean, professional invoice template",
  "html_content": "<html>...</html>",
  "css_content": "body { ... }",
  "header_content": "<header>...</header>",
  "footer_content": "<footer>...</footer>",
  "variables": [
    { "name": "company_name", "type": "string", "required": true },
    { "name": "logo_url", "type": "string", "required": false }
  ],
  "settings": {
    "page_size": "letter",
    "orientation": "portrait",
    "margin_top": "0.5in",
    "margin_bottom": "0.5in",
    "margin_left": "0.5in",
    "margin_right": "0.5in"
  },
  "is_default": true,
  "is_active": true,
  "version": 3,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-20T14:30:00Z"
}
```

## Create Template

Create a new template.

```http
POST /api/v1/templates
```

### Request Body

```json
{
  "name": "Custom Invoice",
  "type": "invoice",
  "description": "Custom branded invoice template",
  "html_content": "<html><body>{{company_name}}...</body></html>",
  "css_content": "body { font-family: Arial, sans-serif; }",
  "header_content": "<div class=\"header\">{{logo}}</div>",
  "footer_content": "<div class=\"footer\">Page {{page_number}}</div>",
  "settings": {
    "page_size": "letter",
    "orientation": "portrait"
  },
  "is_active": true
}
```

### Template Types

- `invoice` - Invoice documents
- `quote` - Quote/estimate documents
- `receipt` - Payment receipt documents
- `statement` - Account statement documents

### Response (201 Created)

Returns the created template.

## Update Template

Update an existing template.

```http
PUT /api/v1/templates/:id
```

### Request Body

All fields are optional.

```json
{
  "name": "Updated Template Name",
  "html_content": "<html>...</html>",
  "is_active": false
}
```

Note: Updates create a new version. Previous versions are preserved.

## Delete Template

Delete a template (soft delete).

```http
DELETE /api/v1/templates/:id
```

### Response

```
HTTP 204 No Content
```

Note: Cannot delete the default template.

## Preview Template

Preview a template with sample data.

```http
POST /api/v1/templates/:id/preview
```

### Request Body

```json
{
  "data": {
    "company_name": "ACME Corp",
    "logo_url": "https://example.com/logo.png",
    "invoice_number": "INV-2024-00001",
    "customer_name": "John Doe",
    "items": [{ "description": "Service", "amount": "500.00" }],
    "total": "500.00"
  },
  "format": "pdf"
}
```

### Response

Returns the rendered document (PDF or HTML).

## Clone Template

Create a copy of an existing template.

```http
POST /api/v1/templates/:id/clone
```

### Request Body

```json
{
  "name": "Cloned Template"
}
```

### Response

Returns the new template.

## Template Variables

Templates support variable substitution using Mustache syntax.

### Built-in Variables

| Variable               | Description          |
| ---------------------- | -------------------- |
| `{{company_name}}`     | Your company name    |
| `{{company_address}}`  | Your company address |
| `{{company_phone}}`    | Your company phone   |
| `{{company_email}}`    | Your company email   |
| `{{logo_url}}`         | Company logo URL     |
| `{{invoice_number}}`   | Invoice number       |
| `{{invoice_date}}`     | Invoice date         |
| `{{due_date}}`         | Payment due date     |
| `{{customer_name}}`    | Customer name        |
| `{{customer_email}}`   | Customer email       |
| `{{customer_address}}` | Customer address     |
| `{{subtotal}}`         | Subtotal amount      |
| `{{tax_amount}}`       | Tax amount           |
| `{{total_amount}}`     | Total amount         |
| `{{amount_paid}}`      | Amount paid          |
| `{{amount_due}}`       | Amount due           |
| `{{page_number}}`      | Current page number  |
| `{{total_pages}}`      | Total pages          |

### Line Item Loop

```html
{{#items}}
<tr>
  <td>{{description}}</td>
  <td>{{quantity}}</td>
  <td>{{unit_price}}</td>
  <td>{{line_total}}</td>
</tr>
{{/items}}
```

### Conditional Sections

```html
{{#has_tax}}
<div class="tax">Tax: {{tax_amount}}</div>
{{/has_tax}} {{#is_overdue}}
<div class="overdue-notice">This invoice is overdue!</div>
{{/is_overdue}}
```

## Settings

### Page Settings

| Setting       | Options              |
| ------------- | -------------------- |
| page_size     | letter, a4, legal    |
| orientation   | portrait, landscape  |
| margin_top    | e.g., "0.5in", "1cm" |
| margin_bottom | e.g., "0.5in"        |
| margin_left   | e.g., "0.5in"        |
| margin_right  | e.g., "0.5in"        |

### Style Settings

| Setting         | Description           |
| --------------- | --------------------- |
| primary_color   | Brand primary color   |
| secondary_color | Brand secondary color |
| font_family     | Main font family      |
| font_size       | Base font size        |

## Error Responses

### Template Not Found (404)

```json
{
  "error": "template_not_found",
  "message": "Template not found"
}
```

### Invalid Template (400)

```json
{
  "error": "invalid_template",
  "message": "Template HTML contains invalid syntax"
}
```

### Cannot Delete Default (400)

```json
{
  "error": "cannot_delete_default",
  "message": "Cannot delete the default template"
}
```

## Examples

### Create Invoice Template

```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Modern Invoice",
    "type": "invoice",
    "html_content": "<html><body><h1>{{company_name}}</h1>...</body></html>",
    "is_active": true
  }'
```

### Preview Template

```bash
curl -X POST http://localhost:8080/api/v1/templates/{id}/preview \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "company_name": "ACME Corp",
      "invoice_number": "INV-001",
      "total": "500.00"
    },
    "format": "html"
  }'
```

## Database Schema

### templates table

| Column         | Type         | Description          |
| -------------- | ------------ | -------------------- |
| id             | UUID         | Primary key          |
| name           | VARCHAR(200) | Template name        |
| type           | VARCHAR(50)  | Template type        |
| description    | TEXT         | Description          |
| html_content   | TEXT         | HTML template        |
| css_content    | TEXT         | CSS styles           |
| header_content | TEXT         | Header HTML          |
| footer_content | TEXT         | Footer HTML          |
| variables      | JSONB        | Variable definitions |
| settings       | JSONB        | Template settings    |
| is_default     | BOOLEAN      | Is default template  |
| is_active      | BOOLEAN      | Is active            |
| version        | INTEGER      | Version number       |
| created_by     | UUID         | Creator user         |
| created_at     | TIMESTAMP    | Creation time        |
| updated_at     | TIMESTAMP    | Last update          |
| deleted_at     | TIMESTAMP    | Soft delete          |

### template_versions table

| Column       | Type      | Description          |
| ------------ | --------- | -------------------- |
| id           | UUID      | Primary key          |
| template_id  | UUID      | Parent template      |
| version      | INTEGER   | Version number       |
| html_content | TEXT      | HTML at this version |
| css_content  | TEXT      | CSS at this version  |
| created_by   | UUID      | Who created version  |
| created_at   | TIMESTAMP | When created         |
