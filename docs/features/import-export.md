# Import/Export

Data import and export functionality for customers, jobs, and invoices.

## Overview

The Import/Export system provides:

- CSV import for bulk data operations
- CSV and JSON export formats
- Data validation during import
- Error reporting and handling
- Background processing for large files

## Supported Entities

| Entity    | Import | Export         |
| --------- | ------ | -------------- |
| Customers | CSV    | CSV, JSON      |
| Jobs      | CSV    | CSV, JSON      |
| Invoices  | -      | CSV, JSON, PDF |
| Quotes    | -      | CSV, JSON, PDF |
| Payments  | -      | CSV, JSON      |

## Export Endpoints

| Method | Endpoint                | Description      |
| ------ | ----------------------- | ---------------- |
| GET    | `/export/customers`     | Export customers |
| GET    | `/export/jobs`          | Export jobs      |
| GET    | `/export/invoices`      | Export invoices  |
| GET    | `/export/payments`      | Export payments  |
| GET    | `/export/reports/:type` | Export reports   |

## Import Endpoints

| Method | Endpoint                 | Description         |
| ------ | ------------------------ | ------------------- |
| POST   | `/import/customers`      | Import customers    |
| POST   | `/import/jobs`           | Import jobs         |
| GET    | `/import/template/:type` | Get import template |
| GET    | `/import/status/:id`     | Check import status |

## Export Customers

```http
GET /api/v1/export/customers?format=csv
```

### Query Parameters

| Parameter | Type   | Default | Description              |
| --------- | ------ | ------- | ------------------------ |
| format    | string | csv     | Export format: csv, json |
| status    | string | all     | Filter by status         |
| type      | string | all     | Filter by customer type  |
| from_date | date   | -       | Created after date       |
| to_date   | date   | -       | Created before date      |

### CSV Response

```csv
id,first_name,last_name,email,phone_primary,customer_type,status,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,created_at
550e8400-...,John,Doe,john@example.com,555-123-4567,residential,active,123 Main St,Springfield,IL,62701,2024-01-15T10:00:00Z
```

### JSON Response

```json
{
  "customers": [
    {
      "id": "550e8400-...",
      "first_name": "John",
      "last_name": "Doe",
      "email": "john@example.com",
      "phone_primary": "555-123-4567",
      "customer_type": "residential",
      "status": "active"
    }
  ],
  "total": 100,
  "exported_at": "2024-01-15T10:00:00Z"
}
```

## Export Invoices

```http
GET /api/v1/export/invoices?format=csv&from_date=2024-01-01&to_date=2024-12-31
```

### Query Parameters

| Parameter   | Type   | Description        |
| ----------- | ------ | ------------------ |
| format      | string | csv, json, pdf     |
| status      | string | Filter by status   |
| customer_id | UUID   | Filter by customer |
| from_date   | date   | Invoice date from  |
| to_date     | date   | Invoice date to    |

### CSV Response

```csv
invoice_number,customer_name,customer_email,status,issue_date,due_date,subtotal,tax_amount,total_amount,amount_paid,amount_due
INV-2024-00001,John Doe,john@example.com,sent,2024-01-15,2024-02-14,1000.00,82.50,1082.50,0.00,1082.50
```

## Import Customers

```http
POST /api/v1/import/customers
Content-Type: multipart/form-data
```

### Request

Upload a CSV file with the required columns.

### Get Import Template

```http
GET /api/v1/import/template/customers
```

Returns a CSV template with required headers.

### Template Format

```csv
first_name,last_name,email,phone_primary,customer_type,status,billing_address_street,billing_address_city,billing_address_state,billing_address_zip
```

### Required Fields

- first_name
- last_name
- email
- phone_primary
- billing_address_street
- billing_address_city
- billing_address_state
- billing_address_zip

### Optional Fields

- phone_secondary
- company_name
- customer_type (default: residential)
- status (default: active)
- service*address*\* (all or none)
- notes

### Response

```json
{
  "import_id": "import-uuid",
  "status": "processing",
  "total_rows": 100,
  "message": "Import started. Check status at /import/status/import-uuid"
}
```

## Check Import Status

```http
GET /api/v1/import/status/:import_id
```

### Response (In Progress)

```json
{
  "import_id": "import-uuid",
  "status": "processing",
  "total_rows": 100,
  "processed_rows": 45,
  "success_count": 43,
  "error_count": 2,
  "progress_percent": 45
}
```

### Response (Completed)

```json
{
  "import_id": "import-uuid",
  "status": "completed",
  "total_rows": 100,
  "success_count": 97,
  "error_count": 3,
  "errors": [
    { "row": 15, "field": "email", "error": "Invalid email format" },
    { "row": 42, "field": "email", "error": "Email already exists" },
    { "row": 78, "field": "phone_primary", "error": "Invalid phone format" }
  ],
  "completed_at": "2024-01-15T10:05:00Z"
}
```

## Validation Rules

### Email

- Valid email format required
- Must be unique (not already in system)

### Phone

- Valid US phone format
- Formats accepted: (555) 123-4567, 555-123-4567, 5551234567

### State

- Valid 2-letter US state code
- Case insensitive (converted to uppercase)

### ZIP Code

- 5-digit format: 12345
- 9-digit format: 12345-6789

### Customer Type

- residential
- commercial

### Status

- active
- inactive
- prospect

## Error Handling

### Validation Errors

```json
{
  "errors": [
    {
      "row": 15,
      "field": "email",
      "value": "invalid-email",
      "error": "Invalid email format"
    }
  ]
}
```

### Duplicate Handling

By default, duplicates (by email) are skipped. Use `update_existing=true` to update.

```http
POST /api/v1/import/customers?update_existing=true
```

## Export Reports

```http
GET /api/v1/export/reports/revenue?from_date=2024-01-01&to_date=2024-12-31
```

### Report Types

| Type      | Description               |
| --------- | ------------------------- |
| revenue   | Revenue by period         |
| customers | Customer acquisition      |
| jobs      | Job completion statistics |
| payments  | Payment summary           |

### Revenue Report

```json
{
  "report_type": "revenue",
  "period": {
    "from": "2024-01-01",
    "to": "2024-12-31"
  },
  "summary": {
    "total_revenue": "150000.00",
    "total_invoices": 500,
    "total_paid": "125000.00",
    "total_outstanding": "25000.00"
  },
  "by_month": [
    { "month": "2024-01", "revenue": "12000.00", "invoices": 45 },
    { "month": "2024-02", "revenue": "14500.00", "invoices": 52 }
  ]
}
```

## Best Practices

### For Imports

1. **Download template first** - Use the exact column format
2. **Validate data locally** - Check for issues before uploading
3. **Start with small batches** - Test with 10-20 rows first
4. **Review errors** - Fix and re-import failed rows
5. **Backup existing data** - Before large imports

### For Exports

1. **Use date filters** - Avoid exporting all data
2. **Choose appropriate format** - CSV for spreadsheets, JSON for systems
3. **Schedule regular exports** - For backup purposes
4. **Secure exported files** - Contains sensitive data
