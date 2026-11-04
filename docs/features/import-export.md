# Import/Export

Imports (customers, sites, jobs and open invoices from Jobber, Housecall Pro or any
CSV/XLSX — see [docs/api/imports.md](../api/imports.md)) and async exports for
customers, jobs, invoices, quotes and reports.

## Overview

The Import/Export system provides:

- Mapped imports from Jobber / Housecall Pro / any CSV or XLSX, with preview
  ([imports.md](../api/imports.md), T193)
- CSV import of customers in ServicePro's template layout (with dry run)
- Async export jobs in CSV, JSON, XLSX and PDF
- Data validation during import
- Error reporting and handling
- Background processing for large files

## Supported Entities

| Entity          | Import    | Export (async export jobs) |
| --------------- | --------- | -------------------------- |
| Customers       | CSV, XLSX | CSV, JSON, XLSX, PDF       |
| Sites           | CSV, XLSX | -                          |
| Jobs            | CSV, XLSX | CSV, JSON, XLSX, PDF       |
| Invoices (open) | CSV, XLSX | CSV, JSON, XLSX, PDF       |
| Quotes          | -         | CSV, JSON, XLSX, PDF       |
| Revenue report  | -         | CSV, JSON, XLSX            |

`GET /api/v1/exports/formats` returns the live list. Single invoice and quote
PDFs are `GET /api/v1/invoices/:id/pdf` and `GET /api/v1/quotes/:id/pdf`.

Every PDF (list exports, quotes, invoices, receipts) is drawn by the gofpdf
generator in `backend/internal/services/pdf` — there is no HTML-to-PDF engine
(D13, T260). A PDF export of a report type is rejected with a 400.

## Export Endpoints

All under `/api/v1`, authenticated and tenant-scoped.

| Method | Endpoint                                              | Description                                 |
| ------ | ----------------------------------------------------- | ------------------------------------------- |
| GET    | `/exports/formats`                                    | Supported formats and export types (public) |
| POST   | `/exports`                                            | Start an async export job                   |
| POST   | `/exports/quick`                                      | Small export, returned directly             |
| GET    | `/exports`                                            | List recent export jobs (`?limit=20`)       |
| GET    | `/exports/:id`                                        | Export job status                           |
| GET    | `/exports/:id/progress`                               | Progress                                    |
| GET    | `/exports/:id/stream`                                 | Progress as a server-sent event stream      |
| GET    | `/exports/:id/download`                               | Download the finished file                  |
| POST   | `/exports/:id/cancel`                                 | Cancel a running export                     |
| POST   | `/reports/revenue/export`, `/reports/ar-aging/export` | Report exports                              |

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
`customers`, `jobs`, `invoices`, `quotes`, `revenue_report`.
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

These are the customer-template endpoints. Jobber, Housecall Pro and mapped
CSV/XLSX imports of customers, sites, jobs and open invoices use
`/imports/{entity}` — see [imports.md](../api/imports.md).

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

Limits: 10 MiB and 50,000 data rows per file (413 / 400 `file_too_large` above that;
reasoning in [imports.md](../api/imports.md#limits-security-audit-f-23)).
A file with any row-level format error is rejected whole (400 `validation_error`).

How it runs: the upload creates an import job holding the parsed rows. Files of
up to 1,000 rows start importing during the request, for up to 20 seconds (the 202
response has status `completed` or `failed` when that finished them). The rest stays
`pending` and is imported in chunks (a saved row cursor, see
[imports.md](../api/imports.md#how-an-import-runs)) by
`POST /internal/jobs/imports`, which the scheduler calls every 15 minutes
(`.github/workflows/scheduled-jobs.yml`; the dev server runs it every minute with
`RUN_BACKGROUND_WORKERS`). Each job is claimed atomically before it runs, so it is
imported once even when runs overlap, and always into its own tenant. A job stuck
in `processing` for 5 minutes is marked `failed` (template imports are not
resumed: with `duplicate_handling=error` a re-run would report duplicates). The stored rows are deleted when the job finishes.

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
counts and per-row errors (row, field, message, row data). Exact shape: Swagger UI at
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
