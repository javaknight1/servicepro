# Invoice Template System - Complete Documentation

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Architecture](#architecture)
4. [Installation & Setup](#installation--setup)
5. [Quick Start](#quick-start)
6. [Template Syntax](#template-syntax)
7. [API Reference](#api-reference)
8. [Integration Guide](#integration-guide)
9. [Configuration Options](#configuration-options)
10. [Best Practices](#best-practices)
11. [Troubleshooting](#troubleshooting)
12. [Performance Optimization](#performance-optimization)

---

## Overview

The Invoice Template System provides a comprehensive solution for creating, managing, and generating PDF invoices with customizable HTML/CSS templates. Built with Go, GORM, and wkhtmltopdf, it offers enterprise-grade features including version control, asset management, and usage analytics.

### Key Capabilities

- **Template Management**: Full CRUD operations for invoice templates
- **PDF Generation**: High-quality PDF generation using wkhtmltopdf
- **Dynamic Content**: Handlebars templating with custom helpers
- **Version Control**: Automatic versioning with restore capability
- **Asset Management**: Logo, image, and font file handling
- **Usage Analytics**: Track template usage and performance
- **Watermarks**: Configurable watermarks for draft invoices
- **Page Numbers**: Custom page number formatting and positioning

---

## Features

### Template Structure

- **HTML Templates**: Full HTML5 support with Handlebars syntax
- **CSS Styling**: Custom CSS with print media queries
- **Header/Footer**: Configurable header and footer HTML
- **Dynamic Fields**: Insert data using {{field_name}} syntax
- **Conditional Content**: Show/hide content based on data
- **Loops**: Iterate over arrays (line items, etc.)

### PDF Generation

- **wkhtmltopdf Integration**: Production-ready PDF generation
- **Page Sizing**: Support for A4, Letter, Legal, A5
- **Orientation**: Portrait and landscape modes
- **Custom Margins**: Configurable page margins
- **Custom Fonts**: Load and embed custom fonts
- **Page Numbers**: Automatic page numbering with custom format
- **Watermarks**: Text watermarks with opacity and rotation

### Template Management

- **CRUD Operations**: Create, read, update, delete templates
- **Preview**: Generate preview PDFs with sample data
- **Version Control**: Automatic version snapshots on changes
- **Default Template**: Set one template as default
- **Clone**: Duplicate templates for quick variations
- **Archive**: Soft delete with archive status
- **Search & Filter**: Find templates by name, status, tags

### Asset Management

- **Upload**: Upload logos, images, fonts
- **Storage**: Organized file storage per template
- **Dimensions**: Automatic image dimension detection
- **Mime Types**: Automatic mime type detection
- **Delete**: Remove assets with file cleanup

---

## Architecture

### Database Schema

```
invoice_templates
├── id (uuid, primary key)
├── name (varchar)
├── description (text)
├── html_content (text)
├── css_content (text)
├── header_html (text)
├── footer_html (text)
├── status (enum: draft, active, archived, deprecated)
├── is_default (boolean)
├── version (integer)
├── page_size (varchar)
├── page_orientation (varchar)
├── margins (decimal)
├── logo settings
├── watermark settings
├── page number settings
├── custom_fonts (jsonb)
├── field_mappings (jsonb)
├── preview_data (jsonb)
├── tags (text[])
└── timestamps

invoice_template_assets
├── id (uuid, primary key)
├── template_id (uuid, foreign key)
├── asset_type (varchar)
├── asset_name (varchar)
├── file_path (text)
├── file_size (bigint)
├── mime_type (varchar)
├── dimensions (width, height)
└── timestamps

invoice_template_usage
├── id (uuid, primary key)
├── template_id (uuid, foreign key)
├── invoice_id (uuid)
├── generated_at (timestamp)
├── generated_by (uuid)
├── pdf_file_path (text)
├── pdf_file_size (bigint)
├── generation_time_ms (integer)
├── success (boolean)
└── error_message (text)

invoice_template_versions
├── id (uuid, primary key)
├── template_id (uuid, foreign key)
├── version_number (integer)
├── html_content (text)
├── css_content (text)
├── header_html (text)
├── footer_html (text)
├── configuration (jsonb)
├── version_notes (text)
└── timestamps
```

### Service Layer

```
PDFService
├── GeneratePDFFromTemplate()
├── RenderTemplate()
├── ApplyCSS()
├── AddWatermark()
├── BuildFooter()
├── GeneratePDF()
├── ProcessImage()
└── ValidateTemplate()

InvoiceTemplateService
├── CreateTemplate()
├── GetTemplate()
├── ListTemplates()
├── UpdateTemplate()
├── DeleteTemplate()
├── CloneTemplate()
├── GetDefaultTemplate()
├── SetDefaultTemplate()
├── GetTemplateStatistics()
├── GetVersionHistory()
├── RestoreVersion()
├── UploadAsset()
├── DeleteAsset()
├── GeneratePDF()
├── GeneratePreview()
└── GeneratePreviewFromContent()
```

---

## Installation & Setup

### Prerequisites

1. **Go 1.21+**
2. **PostgreSQL 14+**
3. **wkhtmltopdf 0.12.6+**

### Install wkhtmltopdf

**Ubuntu/Debian:**

```bash
sudo apt-get update
sudo apt-get install wkhtmltopdf
```

**macOS:**

```bash
brew install wkhtmltopdf
```

**CentOS/RHEL:**

```bash
sudo yum install wkhtmltopdf
```

**From Source:**

```bash
wget https://github.com/wkhtmltopdf/packaging/releases/download/0.12.6-1/wkhtmltox_0.12.6-1.bionic_amd64.deb
sudo dpkg -i wkhtmltox_0.12.6-1.bionic_amd64.deb
sudo apt-get install -f
```

### Install Go Dependencies

```bash
go get github.com/aymerick/raymond
go get github.com/shopspring/decimal
go get gorm.io/gorm
```

### Database Setup

Run the migration:

```bash
psql -U postgres -d servicepro < migrations/008_create_invoice_templates.sql
```

This creates:

- All required tables
- Enums for status, page size, orientation
- Database triggers for version control
- Functions for cloning and statistics
- Default "Classic Invoice" template

### Application Configuration

Add to your `config/config.go`:

```go
type Config struct {
    // ... existing config

    // PDF Configuration
    WkhtmltopdfPath string
    PDFTempDir      string
    PDFOutputDir    string

    // Asset Configuration
    AssetStorageDir string
}
```

Load from environment:

```go
config := &Config{
    WkhtmltopdfPath: getEnv("WKHTMLTOPDF_PATH", "/usr/bin/wkhtmltopdf"),
    PDFTempDir:      getEnv("PDF_TEMP_DIR", "/tmp/servicepro/pdf"),
    PDFOutputDir:    getEnv("PDF_OUTPUT_DIR", "/var/servicepro/pdfs"),
    AssetStorageDir: getEnv("ASSET_STORAGE_DIR", "/var/servicepro/assets"),
}
```

### Setup Routes

In your `cmd/api/main.go`:

```go
import (
    "github.com/javaknight1/servicepro/backend/internal/api/routes"
)

func main() {
    // ... existing setup

    v1 := router.Group("/api/v1")
    routes.SetupInvoiceTemplateRoutes(v1, db, jwtSecret, config.WkhtmltopdfPath)
}
```

### Create Required Directories

```bash
sudo mkdir -p /var/servicepro/pdfs
sudo mkdir -p /var/servicepro/assets
sudo mkdir -p /tmp/servicepro/pdf

sudo chown -R youruser:yourgroup /var/servicepro
sudo chown -R youruser:yourgroup /tmp/servicepro
```

---

## Quick Start

### 1. Get the Default Template

```bash
curl -X GET http://localhost:8080/api/v1/templates/default \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Classic Invoice",
  "status": "active",
  "is_default": true,
  "page_size": "A4",
  "page_orientation": "portrait"
}
```

### 2. Generate a PDF Invoice

```bash
curl -X POST http://localhost:8080/api/v1/templates/550e8400-e29b-41d4-a716-446655440000/generate-pdf \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_id": "123e4567-e89b-12d3-a456-426614174000",
    "data": {
      "invoice_number": "INV-2024-001",
      "issue_date": "2024-01-15",
      "due_date": "2024-02-15",
      "company_name": "Acme Corp",
      "company_address": "123 Main St, City, State 12345",
      "customer_name": "John Doe",
      "customer_email": "john@example.com",
      "line_items": [
        {
          "description": "Web Development",
          "quantity": 40,
          "unit_price": 100.00,
          "amount": 4000.00
        },
        {
          "description": "Consulting",
          "quantity": 10,
          "unit_price": 150.00,
          "amount": 1500.00
        }
      ],
      "subtotal": 5500.00,
      "tax": 550.00,
      "total": 6050.00,
      "notes": "Payment due within 30 days"
    }
  }'
```

Response:

```json
{
  "success": true,
  "file_path": "/var/servicepro/pdfs/invoice_123e4567-e89b-12d3-a456-426614174000.pdf",
  "file_size": 45678,
  "generation_time_ms": 1234,
  "generated_at": "2024-01-15T10:30:00Z"
}
```

### 3. Create a Custom Template

```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Modern Invoice",
    "description": "A modern, minimal invoice design",
    "html_content": "<html><body><h1>Invoice {{invoice_number}}</h1>...</body></html>",
    "css_content": "body { font-family: Helvetica; color: #333; }",
    "status": "active",
    "page_size": "A4",
    "page_orientation": "portrait"
  }'
```

---

## Template Syntax

### Handlebars Basics

The system uses Handlebars for dynamic content insertion.

#### Variables

```html
<h1>Invoice #{{invoice_number}}</h1>
<p>Date: {{issue_date}}</p>
<p>Customer: {{customer_name}}</p>
```

#### Conditionals

```html
{{#if company_logo}}
<img src="{{company_logo}}" alt="Logo" class="logo" />
{{/if}} {{#if notes}}
<div class="notes">
  <h3>Notes</h3>
  <p>{{notes}}</p>
</div>
{{/if}}
```

#### Loops

```html
<table>
  <thead>
    <tr>
      <th>Description</th>
      <th>Qty</th>
      <th>Price</th>
      <th>Amount</th>
    </tr>
  </thead>
  <tbody>
    {{#each line_items}}
    <tr>
      <td>{{description}}</td>
      <td>{{quantity}}</td>
      <td>${{unit_price}}</td>
      <td>${{amount}}</td>
    </tr>
    {{/each}}
  </tbody>
</table>
```

#### Nested Data

```html
<div class="customer">
  <h2>Bill To:</h2>
  <p>{{customer.name}}</p>
  <p>{{customer.address.street}}</p>
  <p>
    {{customer.address.city}}, {{customer.address.state}}
    {{customer.address.zip}}
  </p>
</div>
```

### Custom Helpers

#### format_currency

Formats numbers as currency:

```html
<p>Total: {{format_currency total}}</p>
<!-- Output: Total: 1234.56 -->
```

#### format_date

Formats dates:

```html
<p>Date: {{format_date issue_date "01/02/2006"}}</p>
<!-- Output: Date: 01/15/2024 -->

<p>Date: {{format_date issue_date "January 2, 2006"}}</p>
<!-- Output: Date: January 15, 2024 -->
```

### Complete Template Example

```html
<!doctype html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>Invoice {{invoice_number}}</title>
    <style>
      body {
        font-family: Arial, sans-serif;
        margin: 0;
        padding: 20px;
        color: #333;
      }
      .header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 30px;
        border-bottom: 2px solid #2c3e50;
        padding-bottom: 20px;
      }
      .company-info h1 {
        margin: 0;
        color: #2c3e50;
      }
      .invoice-info {
        text-align: right;
      }
      table {
        width: 100%;
        border-collapse: collapse;
        margin: 20px 0;
      }
      th {
        background: #2c3e50;
        color: white;
        padding: 10px;
        text-align: left;
      }
      td {
        padding: 10px;
        border-bottom: 1px solid #ddd;
      }
      .totals {
        margin-left: auto;
        width: 300px;
      }
      .totals tr td:first-child {
        text-align: right;
        font-weight: bold;
      }
      .total-row {
        font-size: 1.2em;
        color: #2c3e50;
      }
    </style>
  </head>
  <body>
    <div class="header">
      <div class="company-info">
        {{#if company_logo}}
        <img src="{{company_logo}}" alt="Logo" style="max-width: 200px;" />
        {{/if}}
        <h1>{{company_name}}</h1>
        <p>{{company_address}}</p>
        {{#if company_phone}}
        <p>Phone: {{company_phone}}</p>
        {{/if}}
      </div>

      <div class="invoice-info">
        <h2>INVOICE</h2>
        <p><strong>Invoice #:</strong> {{invoice_number}}</p>
        <p><strong>Date:</strong> {{format_date issue_date "01/02/2006"}}</p>
        <p><strong>Due Date:</strong> {{format_date due_date "01/02/2006"}}</p>
      </div>
    </div>

    <div class="customer-info">
      <h3>Bill To:</h3>
      <p><strong>{{customer_name}}</strong></p>
      {{#if customer_address}}
      <p>{{customer_address}}</p>
      {{/if}} {{#if customer_email}}
      <p>Email: {{customer_email}}</p>
      {{/if}}
    </div>

    <table>
      <thead>
        <tr>
          <th>Description</th>
          <th style="text-align: center;">Quantity</th>
          <th style="text-align: right;">Unit Price</th>
          <th style="text-align: right;">Amount</th>
        </tr>
      </thead>
      <tbody>
        {{#each line_items}}
        <tr>
          <td>{{description}}</td>
          <td style="text-align: center;">{{quantity}}</td>
          <td style="text-align: right;">${{format_currency unit_price}}</td>
          <td style="text-align: right;">${{format_currency amount}}</td>
        </tr>
        {{/each}}
      </tbody>
    </table>

    <table class="totals">
      <tr>
        <td>Subtotal:</td>
        <td style="text-align: right;">${{format_currency subtotal}}</td>
      </tr>
      {{#if tax}}
      <tr>
        <td>Tax:</td>
        <td style="text-align: right;">${{format_currency tax}}</td>
      </tr>
      {{/if}} {{#if discount}}
      <tr>
        <td>Discount:</td>
        <td style="text-align: right;">-${{format_currency discount}}</td>
      </tr>
      {{/if}}
      <tr class="total-row">
        <td>Total:</td>
        <td style="text-align: right;">${{format_currency total}}</td>
      </tr>
      {{#if amount_paid}}
      <tr>
        <td>Amount Paid:</td>
        <td style="text-align: right;">-${{format_currency amount_paid}}</td>
      </tr>
      <tr class="total-row">
        <td>Balance Due:</td>
        <td style="text-align: right;">${{format_currency balance_due}}</td>
      </tr>
      {{/if}}
    </table>

    {{#if notes}}
    <div class="notes">
      <h3>Notes</h3>
      <p>{{notes}}</p>
    </div>
    {{/if}} {{#if payment_terms}}
    <div class="terms">
      <h3>Payment Terms</h3>
      <p>{{payment_terms}}</p>
    </div>
    {{/if}}
  </body>
</html>
```

---

## API Reference

### Base URL

```
http://localhost:8080/api/v1/templates
```

All endpoints require authentication with JWT Bearer token.

### Endpoints

#### 1. Create Template

**POST** `/templates`

Create a new invoice template.

**Request Body:**

```json
{
  "name": "string (required)",
  "description": "string",
  "html_content": "string (required)",
  "css_content": "string",
  "header_html": "string",
  "footer_html": "string",
  "status": "draft|active|archived|deprecated",
  "is_default": false,
  "page_size": "A4|Letter|Legal|A5",
  "page_orientation": "portrait|landscape",
  "margin_top": 10.0,
  "margin_right": 10.0,
  "margin_bottom": 10.0,
  "margin_left": 10.0,
  "logo_url": "string",
  "logo_position": "top-left|top-center|top-right|...",
  "logo_width": 100.0,
  "logo_height": 50.0,
  "watermark_enabled": false,
  "watermark_text": "string",
  "watermark_opacity": 0.3,
  "watermark_rotation": 45,
  "show_page_numbers": true,
  "page_number_format": "Page {page} of {total}",
  "page_number_position": "bottom-center",
  "custom_fonts": {},
  "field_mappings": {},
  "preview_data": {},
  "tags": ["tag1", "tag2"]
}
```

**Response:** `201 Created`

```json
{
  "id": "uuid",
  "name": "string",
  "version": 1,
  ...
}
```

#### 2. List Templates

**GET** `/templates`

Get a list of templates with optional filtering.

**Query Parameters:**

- `status` - Filter by status
- `is_default` - Filter by default (true/false)
- `search` - Search in name and description
- `page` - Page number (default: 1)
- `page_size` - Page size (default: 20, max: 100)

**Response:** `200 OK`

```json
{
  "templates": [...],
  "total": 10,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

#### 3. Get Template

**GET** `/templates/:id`

Get a specific template by ID.

**Response:** `200 OK`

```json
{
  "id": "uuid",
  "name": "string",
  "html_content": "string",
  ...
}
```

#### 4. Update Template

**PUT** `/templates/:id`

Update an existing template. Creates a version snapshot automatically.

**Request Body:**

```json
{
  "name": "string",
  "description": "string",
  "html_content": "string",
  ...
  "version_notes": "What changed in this version"
}
```

**Response:** `200 OK`

#### 5. Delete Template

**DELETE** `/templates/:id`

Delete or archive a template.

**Query Parameters:**

- `hard_delete` - Permanently delete (true/false, default: false)

**Response:** `200 OK`

```json
{
  "message": "Template successfully archived",
  "id": "uuid"
}
```

#### 6. Clone Template

**POST** `/templates/:id/clone`

Create a copy of an existing template.

**Request Body:**

```json
{
  "new_name": "string (required)"
}
```

**Response:** `201 Created`

#### 7. Get Default Template

**GET** `/templates/default`

Get the template marked as default.

**Response:** `200 OK`

#### 8. Set Default Template

**PUT** `/templates/:id/set-default`

Mark a template as the default. Unsets any previously default template.

**Response:** `200 OK`

#### 9. Generate PDF

**POST** `/templates/:id/generate-pdf`

Generate a PDF invoice using the template.

**Request Body:**

```json
{
  "invoice_id": "uuid",
  "data": {
    "invoice_number": "INV-001",
    "customer_name": "John Doe",
    ...
  },
  "output_path": "/optional/custom/path.pdf"
}
```

**Response:** `200 OK`

```json
{
  "success": true,
  "file_path": "/path/to/generated.pdf",
  "file_size": 45678,
  "generation_time_ms": 1234,
  "generated_at": "2024-01-15T10:30:00Z"
}
```

#### 10. Generate Preview

**POST** `/templates/:id/preview`

Generate a preview PDF using the template's preview data.

**Response:** `200 OK`

#### 11. Generate Preview from Content

**POST** `/templates/preview`

Generate a preview PDF from template content without saving.

**Request Body:**

```json
{
  "html_content": "string (required)",
  "css_content": "string",
  "data": {}
}
```

**Response:** `200 OK`

#### 12. Get Statistics

**GET** `/templates/:id/statistics`

Get usage statistics for a template.

**Response:** `200 OK`

```json
{
  "total_usage": 150,
  "successful_generations": 145,
  "failed_generations": 5,
  "avg_generation_time_ms": 1234.56,
  "last_used": "2024-01-15T10:30:00Z",
  "total_pdf_size_mb": 45.67
}
```

#### 13. Get Version History

**GET** `/templates/:id/versions`

Get all versions of a template.

**Response:** `200 OK`

```json
{
  "versions": [
    {
      "id": "uuid",
      "version_number": 3,
      "html_content": "...",
      "version_notes": "Updated footer",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 3
}
```

#### 14. Restore Version

**POST** `/templates/:id/restore/:version`

Restore a template to a specific version.

**Response:** `200 OK`

#### 15. Upload Asset

**POST** `/templates/:id/assets`

Upload an asset (logo, image, font) for a template.

**Request:** `multipart/form-data`

- `asset_type` - Type: logo, image, font, css
- `asset_name` - Display name
- `file` - File upload

**Response:** `201 Created`

```json
{
  "id": "uuid",
  "template_id": "uuid",
  "asset_type": "logo",
  "asset_name": "Company Logo",
  "file_path": "/path/to/asset.png",
  "file_size": 12345,
  "mime_type": "image/png",
  "width": 200,
  "height": 100
}
```

#### 16. Delete Asset

**DELETE** `/templates/:id/assets/:asset_id`

Delete an asset from a template.

**Response:** `200 OK`

---

## Integration Guide

### Basic Integration

```go
package main

import (
    "context"
    "log"

    "github.com/javaknight1/servicepro/backend/internal/services"
    "gorm.io/gorm"
)

func GenerateInvoicePDF(db *gorm.DB, invoiceData map[string]interface{}) error {
    ctx := context.Background()

    // Initialize services
    pdfService := services.NewPDFService(
        "/usr/bin/wkhtmltopdf",
        "/tmp/servicepro/pdf",
        "/var/servicepro/pdfs",
    )

    templateService := services.NewInvoiceTemplateService(
        db,
        pdfService,
        "/var/servicepro/assets",
    )

    // Get default template
    template, err := templateService.GetDefaultTemplate(ctx)
    if err != nil {
        return err
    }

    // Generate PDF
    request := &models.GeneratePDFRequest{
        TemplateID: template.ID,
        Data:       invoiceData,
    }

    response, err := templateService.GeneratePDF(ctx, request)
    if err != nil {
        return err
    }

    log.Printf("PDF generated: %s (%d bytes, %dms)",
        response.FilePath,
        response.FileSize,
        response.GenerationTimeMs)

    return nil
}
```

### With Custom Template

```go
func GenerateWithCustomTemplate(db *gorm.DB, templateID uuid.UUID, data map[string]interface{}) error {
    ctx := context.Background()

    pdfService := services.NewPDFService(...)
    templateService := services.NewInvoiceTemplateService(db, pdfService, "/var/servicepro/assets")

    // Get specific template
    template, err := templateService.GetTemplate(ctx, templateID)
    if err != nil {
        return err
    }

    // Add custom output path
    outputPath := fmt.Sprintf("/invoices/invoice_%s.pdf", data["invoice_number"])

    request := &models.GeneratePDFRequest{
        TemplateID: template.ID,
        Data:       data,
        OutputPath: &outputPath,
    }

    response, err := templateService.GeneratePDF(ctx, request)
    if err != nil {
        return err
    }

    return nil
}
```

### Batch PDF Generation

```go
func GenerateMultiplePDFs(db *gorm.DB, invoices []Invoice) error {
    ctx := context.Background()

    pdfService := services.NewPDFService(...)
    templateService := services.NewInvoiceTemplateService(db, pdfService, "/var/servicepro/assets")

    template, err := templateService.GetDefaultTemplate(ctx)
    if err != nil {
        return err
    }

    // Process invoices concurrently
    sem := make(chan struct{}, 5) // Limit to 5 concurrent
    errors := make(chan error, len(invoices))

    for _, invoice := range invoices {
        sem <- struct{}{}

        go func(inv Invoice) {
            defer func() { <-sem }()

            request := &models.GeneratePDFRequest{
                TemplateID: template.ID,
                InvoiceID:  &inv.ID,
                Data:       inv.ToMap(),
            }

            _, err := templateService.GeneratePDF(ctx, request)
            if err != nil {
                errors <- err
            }
        }(invoice)
    }

    // Wait for all to complete
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }

    close(errors)

    // Check for errors
    for err := range errors {
        if err != nil {
            return err
        }
    }

    return nil
}
```

---

## Configuration Options

### Page Settings

```go
PageSize: models.PageSizeA4        // A4, Letter, Legal, A5
PageOrientation: models.PageOrientationPortrait  // portrait, landscape

// Margins in millimeters
MarginTop:    decimal.NewFromFloat(15.0)
MarginRight:  decimal.NewFromFloat(15.0)
MarginBottom: decimal.NewFromFloat(15.0)
MarginLeft:   decimal.NewFromFloat(15.0)
```

### Logo Configuration

```go
LogoURL:      "https://example.com/logo.png"
LogoPosition: "top-left"  // top-left, top-center, top-right, etc.
LogoWidth:    decimal.NewFromFloat(150.0)  // mm
LogoHeight:   decimal.NewFromFloat(50.0)   // mm
```

### Watermark Settings

```go
WatermarkEnabled:  true
WatermarkText:     "DRAFT"
WatermarkOpacity:  decimal.NewFromFloat(0.3)  // 0.0 to 1.0
WatermarkRotation: 45  // degrees
```

### Page Numbers

```go
ShowPageNumbers:    true
PageNumberFormat:   "Page {page} of {total}"
PageNumberPosition: "bottom-center"  // bottom-left, bottom-center, bottom-right
```

### Custom Fonts

```json
{
  "fonts": [
    {
      "name": "Roboto",
      "url": "https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;700",
      "weights": [300, 400, 700]
    }
  ]
}
```

---

## Best Practices

### Template Design

1. **Use Semantic HTML**: Structure content with proper HTML5 tags
2. **Mobile-First CSS**: Design for smaller sizes, scale up
3. **Print Media Queries**: Use `@media print` for print-specific styles
4. **Avoid Floats**: Use flexbox or grid for layout
5. **Test Thoroughly**: Preview with real data before deploying

### Performance

1. **Cache Templates**: Load templates once, reuse
2. **Optimize Images**: Compress and resize images before upload
3. **Limit Concurrent Generation**: Use semaphores to limit parallel PDF generation
4. **Clean Up PDFs**: Implement cleanup job for old generated PDFs
5. **Use CDN for Assets**: Host logos and images on CDN

### Security

1. **Sanitize Input**: Validate all template content
2. **Validate URLs**: Check asset URLs for safety
3. **Limit File Sizes**: Set max size for uploads
4. **Access Control**: Implement proper permissions
5. **Audit Trail**: Use version history for audit

### Data Structure

```go
// Recommended invoice data structure
type InvoiceData struct {
    InvoiceNumber   string
    IssueDate       time.Time
    DueDate         time.Time

    Company struct {
        Name    string
        Address string
        Phone   string
        Email   string
        Logo    string
    }

    Customer struct {
        Name    string
        Address string
        Email   string
        Phone   string
    }

    LineItems []struct {
        Description string
        Quantity    int
        UnitPrice   decimal.Decimal
        Amount      decimal.Decimal
    }

    Subtotal    decimal.Decimal
    Tax         decimal.Decimal
    Discount    decimal.Decimal
    Total       decimal.Decimal
    AmountPaid  decimal.Decimal
    BalanceDue  decimal.Decimal

    Notes        string
    PaymentTerms string
}
```

---

## Troubleshooting

### PDF Generation Fails

**Problem:** PDF generation returns error

**Solutions:**

1. Check wkhtmltopdf is installed: `which wkhtmltopdf`
2. Verify wkhtmltopdf path in config
3. Check file permissions on output directory
4. Review HTML/CSS for syntax errors
5. Check logs for specific error message

### Fonts Not Rendering

**Problem:** Custom fonts don't appear in PDF

**Solutions:**

1. Ensure font URLs are accessible
2. Use absolute URLs, not relative
3. Include font weights in @font-face
4. Test with Google Fonts first
5. Check wkhtmltopdf font support

### Images Not Displaying

**Problem:** Images missing in generated PDF

**Solutions:**

1. Use absolute URLs for images
2. Verify image URLs are accessible
3. Check image format (PNG, JPG supported)
4. Ensure images aren't too large
5. Try data URLs for small images

### Watermark Not Visible

**Problem:** Watermark doesn't appear

**Solutions:**

1. Check `watermark_enabled` is true
2. Increase opacity (0.3-0.5 recommended)
3. Verify z-index in CSS
4. Check text color contrast
5. Ensure text isn't too long

### Version Not Incrementing

**Problem:** Template version stays at 1

**Solutions:**

1. Verify database trigger is created
2. Check content actually changed
3. Review trigger function logs
4. Manually check `invoice_template_versions` table

### Performance Issues

**Problem:** PDF generation is slow

**Solutions:**

1. Optimize template HTML/CSS
2. Reduce image sizes
3. Limit concurrent generations
4. Use faster server
5. Consider PDF caching

### Memory Issues

**Problem:** Out of memory errors

**Solutions:**

1. Limit concurrent PDF generations
2. Clean up temp files regularly
3. Increase available memory
4. Optimize template complexity
5. Use streaming for large batches

---

## Performance Optimization

### Template Caching

```go
type TemplateCacheService struct {
    cache map[uuid.UUID]*models.InvoiceTemplate
    mu    sync.RWMutex
}

func (s *TemplateCacheService) GetTemplate(ctx context.Context, id uuid.UUID) (*models.InvoiceTemplate, error) {
    s.mu.RLock()
    if template, ok := s.cache[id]; ok {
        s.mu.RUnlock()
        return template, nil
    }
    s.mu.RUnlock()

    // Load from database
    template, err := s.service.GetTemplate(ctx, id)
    if err != nil {
        return nil, err
    }

    // Cache it
    s.mu.Lock()
    s.cache[id] = template
    s.mu.Unlock()

    return template, nil
}
```

### Concurrent Generation

```go
func GenerateConcurrent(requests []*models.GeneratePDFRequest, maxConcurrent int) error {
    sem := make(chan struct{}, maxConcurrent)
    errors := make(chan error, len(requests))

    for _, req := range requests {
        sem <- struct{}{}

        go func(r *models.GeneratePDFRequest) {
            defer func() { <-sem }()

            _, err := templateService.GeneratePDF(ctx, r)
            if err != nil {
                errors <- err
            }
        }(req)
    }

    // Wait and collect errors
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }
    close(errors)

    for err := range errors {
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Cleanup Old PDFs

```go
func CleanupOldPDFs(dir string, maxAge time.Duration) error {
    cutoff := time.Now().Add(-maxAge)

    return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() && info.ModTime().Before(cutoff) {
            return os.Remove(path)
        }

        return nil
    })
}

// Run daily
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        CleanupOldPDFs("/var/servicepro/pdfs", 30*24*time.Hour) // 30 days
    }
}()
```

---

## Support

For issues, questions, or contributions:

- **GitHub Issues**: https://github.com/javaknight1/servicepro/issues
- **Documentation**: See `docs/` directory
- **Email**: support@servicepro.com

---

## License

Copyright (c) 2024 ServicePro. All rights reserved.
