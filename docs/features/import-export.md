# Import/Export

Customer import (CSV) and async exports for customers, jobs, invoices, quotes and reports.

## Overview

The Import/Export system provides:

- CSV import of customers (with dry run)
- Async export jobs in CSV, JSON, XLSX and PDF
- Data validation during import
- Error reporting and handling
- Background processing for large files

## Supported Entities

| Entity          | Import | Export (async export jobs) |
| --------------- | ------ | -------------------------- |
| Customers       | CSV    | CSV, JSON, XLSX, PDF       |
| Jobs            | -      | CSV, JSON, XLSX, PDF       |
| Invoices        | -      | CSV, JSON, XLSX, PDF       |
| Quotes          | -      | CSV, JSON, XLSX, PDF       |
| Revenue report  | -      | CSV, JSON, XLSX, PDF       |
| Customer report | -      | CSV, JSON, XLSX, PDF       |

`GET /api/v1/exports/formats` returns the live list. Single invoice and quote
PDFs are `GET /api/v1/invoices/:id/pdf` and `GET /api/v1/quotes/:id/pdf`.

## Export Endpoints

All under `/api/v1`, authenticated and tenant-scoped.

| Method | Endpoint                                                                           | Description                                 |
| ------ | ---------------------------------------------------------------------------------- | ------------------------------------------- |
| GET    | `/exports/formats`                                                                 | Supported formats and export types (public) |
| POST   | `/exports`                                                                         | Start an async export job                   |
| POST   | `/exports/quick`                                                                   | Small export, returned directly             |
| GET    | `/exports`                                                                         | List recent export jobs (`?limit=20`)       |
| GET    | `/exports/:id`                                                                     | Export job status                           |
| GET    | `/exports/:id/progress`                                                            | Progress                                    |
| GET    | `/exports/:id/stream`                                                              | Progress as a server-sent event stream      |
| GET    | `/exports/:id/download`                                                            | Download the finished file                  |
| POST   | `/exports/:id/cancel`                                                              | Cancel a running export                     |
| POST   | `/reports/revenue/export`, `/reports/customers/export`, `/reports/ar-aging/export` | Report exports                              |

`GET /customers/export` is deprecated and returns **410 Gone** — use
`POST /exports` with `export_type: "customers"`.

## Start an Export

```http
POST /api/v1/exports
Content-Type: application/json
```

```json
{
  "format": "csv",
  "export_type": "customers",
  "fields": ["first_name", "last_name", "email"],
  "query": { "status": "active" }
}
```

`format` is one of `csv`, `json`, `xlsx`, `pdf`; `export_type` one of
`customers`, `jobs`, `invoices`, `quotes`, `revenue_report`, `customer_report`.
Optional: `include_sections`, `encoding`, `csv_options`, `pdf_options`,
`excel_options`. Poll `/exports/:id/progress` (or subscribe to `/stream`) and
then fetch `/exports/:id/download`. CSV/XLSX cells are protected against
formula injection.

## Import Endpoints

| Method | Endpoint                     | Description                  |
| ------ | ---------------------------- | ---------------------------- |
| POST   | `/customers/import`          | Import customers (CSV)       |
| GET    | `/customers/import/template` | Download the CSV template    |
| GET    | `/customers/import/:job_id`  | Import job status and errors |

Only customers can be imported today.

## Import Customers

```http
POST /api/v1/customers/import
Content-Type: multipart/form-data
```

Form fields:

| Field                | Required | Description                                              |
| -------------------- | -------- | -------------------------------------------------------- |
| `file`               | yes      | The CSV file                                             |
| `format`             | yes      | `csv`                                                    |
| `duplicate_handling` | no       | `error` (default), `skip` or `update` — matched by email |
| `dry_run`            | no       | `true` to validate and preview without saving            |

### Get Import Template

```http
GET /api/v1/customers/import/template
```

Returns `customer_import_template.csv` with the headers and two example rows.

### Template Format

```csv
first_name,last_name,company_name,email,phone_primary,phone_secondary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,service_address_street,service_address_city,service_address_state,service_address_zip,customer_type,status,notes
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

### Status

`GET /api/v1/customers/import/:job_id` returns the import job: status, row
counts and per-row errors (row, field, message). Exact shape: Swagger UI at
`/api/docs` (Import/Export tag).

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

Duplicates are matched by email. `duplicate_handling=error` (default) reports
them as row errors, `skip` ignores them, `update` updates the existing customer.

## Best Practices

### For Imports

1. **Download template first** - Use the exact column format
2. **Validate data locally** - Check for issues before uploading
3. **Start with small batches** - Test with 10-20 rows first
4. **Review errors** - Fix and re-import failed rows
5. **Backup existing data** - Before large imports

### For Exports

1. **Use date filters** - Avoid exporting all data
2. **Choose appropriate format** - CSV/XLSX for spreadsheets, JSON for systems, PDF for sharing
3. **Schedule regular exports** - For backup purposes
4. **Secure exported files** - Contains sensitive data
