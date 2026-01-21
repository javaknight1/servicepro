# Customer Import/Export Documentation

## Overview

The customer import/export feature allows bulk operations for customer data using CSV and JSON formats. It includes background job processing, progress tracking, validation, and error reporting.

## Features

- **CSV Import** with validation and duplicate handling
- **CSV/JSON Export** with filtering options
- **Background Processing** for large imports
- **Progress Tracking** with real-time status updates
- **Error Reporting** with detailed validation errors
- **Template Download** for import format reference

---

## Import Functionality

### Endpoint: POST /api/v1/customers/import

Import customers from a CSV file.

#### Authentication

- Requires JWT authentication
- Requires `customers.create` permission

#### Request Parameters

| Parameter            | Type    | Required | Default | Description                                          |
| -------------------- | ------- | -------- | ------- | ---------------------------------------------------- |
| `file`               | file    | Yes      | -       | CSV file to import                                   |
| `format`             | string  | Yes      | -       | File format (`csv`)                                  |
| `duplicate_handling` | string  | No       | `error` | How to handle duplicates (`skip`, `update`, `error`) |
| `dry_run`            | boolean | No       | `false` | Test import without saving                           |

#### Duplicate Handling Strategies

- **`skip`**: Skip duplicate records (based on email)
- **`update`**: Update existing records with new data
- **`error`**: Return error for duplicate emails (default)

#### CSV Format

Download the template from `/api/v1/customers/import/template`

**Required Fields:**

- `first_name`
- `last_name`
- `email`
- `phone_primary`
- `billing_address_street`
- `billing_address_city`
- `billing_address_state` (2-letter code, e.g., TX, CA)
- `billing_address_zip`
- `customer_type` (`residential` or `commercial`)

**Optional Fields:**

- `company_name`
- `phone_secondary`
- `service_address_street`
- `service_address_city`
- `service_address_state`
- `service_address_zip`
- `status` (`active`, `inactive`, `prospect`)
- `notes`

#### Validation Rules

**Email:**

- Must be a valid email format
- Must contain `@` and `.`

**Phone:**

- Must contain 10-15 digits
- Supports formats: `555-123-4567`, `(555) 123-4567`, `555.123.4567`

**State:**

- Must be exactly 2 characters
- Automatically converted to uppercase

**ZIP Code:**

- Must be 5 or 9 digits
- Supports formats: `12345`, `12345-6789`

**Customer Type:**

- Must be `residential` or `commercial`

#### File Limits

- Maximum 10,000 rows per file
- File must contain at least 1 data row

#### Response

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Import job created successfully",
  "total_rows": 100
}
```

#### Example cURL

```bash
curl -X POST http://localhost:8080/api/v1/customers/import \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@customers.csv" \
  -F "format=csv" \
  -F "duplicate_handling=update"
```

---

### Endpoint: GET /api/v1/customers/import/:job_id

Get the status of an import job.

#### Authentication

- Requires JWT authentication
- Requires `customers.list` permission

#### Response

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "total_rows": 100,
  "processed_rows": 100,
  "successful_rows": 95,
  "failed_rows": 5,
  "progress": 100.0,
  "errors": [
    {
      "row_number": 10,
      "field": "email",
      "error": "email format is invalid"
    }
  ],
  "started_at": "2024-01-21T10:00:00Z",
  "completed_at": "2024-01-21T10:01:30Z",
  "created_at": "2024-01-21T09:59:00Z"
}
```

#### Job Status Values

- `pending`: Job is queued and waiting to be processed
- `processing`: Job is currently being processed
- `completed`: Job finished successfully (may have some errors)
- `failed`: Job failed completely
- `cancelled`: Job was cancelled

#### Example cURL

```bash
curl http://localhost:8080/api/v1/customers/import/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

### Endpoint: GET /api/v1/customers/import/template

Download a CSV template with example data.

#### Authentication

- Requires JWT authentication
- Requires `customers.list` permission

#### Response

Returns a CSV file with headers and example rows.

#### Example cURL

```bash
curl http://localhost:8080/api/v1/customers/import/template \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -O template.csv
```

---

## Export Functionality

### Endpoint: GET /api/v1/customers/export

Export customers to CSV or JSON format.

#### Authentication

- Requires JWT authentication
- Requires `customers.list` permission

#### Query Parameters

| Parameter       | Type    | Required | Description                                         |
| --------------- | ------- | -------- | --------------------------------------------------- |
| `format`        | string  | Yes      | Export format (`csv` or `json`)                     |
| `customer_type` | string  | No       | Filter by type (`residential` or `commercial`)      |
| `status`        | string  | No       | Filter by status (`active`, `inactive`, `prospect`) |
| `city`          | string  | No       | Filter by city                                      |
| `state`         | string  | No       | Filter by state (2-letter code)                     |
| `include_notes` | boolean | No       | Include notes in CSV export (default: false)        |

#### CSV Export

```bash
curl "http://localhost:8080/api/v1/customers/export?format=csv&include_notes=true" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -O customers.csv
```

**CSV Fields:**

- `id`
- `first_name`
- `last_name`
- `company_name`
- `email`
- `phone_primary`
- `phone_secondary`
- `billing_address_street`
- `billing_address_city`
- `billing_address_state`
- `billing_address_zip`
- `service_address_street`
- `service_address_city`
- `service_address_state`
- `service_address_zip`
- `customer_type`
- `status`
- `notes` (if `include_notes=true`)
- `created_at`
- `updated_at`

#### JSON Export

```bash
curl "http://localhost:8080/api/v1/customers/export?format=json&status=active" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -O customers.json
```

Returns an array of customer objects in JSON format with all fields.

#### Export with Filters

```bash
# Export all commercial customers in Texas
curl "http://localhost:8080/api/v1/customers/export?format=csv&customer_type=commercial&state=TX" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -O texas_commercial.csv

# Export all active customers in Austin
curl "http://localhost:8080/api/v1/customers/export?format=json&status=active&city=Austin" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -O austin_active.json
```

---

## Error Handling

### Common Error Codes

| Error Code         | HTTP Status | Description                    |
| ------------------ | ----------- | ------------------------------ |
| `invalid_request`  | 400         | Missing or invalid parameters  |
| `invalid_format`   | 400         | Unsupported file format        |
| `invalid_file`     | 400         | File parsing error             |
| `file_too_large`   | 400         | File exceeds 10,000 rows       |
| `validation_error` | 400         | CSV contains validation errors |
| `unauthorized`     | 401         | Missing or invalid JWT token   |
| `forbidden`        | 403         | Insufficient permissions       |
| `not_found`        | 404         | Import job not found           |
| `internal_error`   | 500         | Server error                   |

### Validation Error Response

```json
{
  "error": "validation_error",
  "message": "file contains 5 validation errors",
  "details": [
    "row 2, field 'email': email is required",
    "row 3, field 'customer_type': customer_type must be 'residential' or 'commercial'",
    "row 5, field 'billing_address_state': billing_address_state must be 2 characters",
    "row 7, field 'phone_primary': phone_primary format is invalid",
    "row 10, field 'email': duplicate email address"
  ]
}
```

---

## Background Job Processing

Import jobs are processed asynchronously in the background using Redis queues.

### Job Lifecycle

1. **Upload**: File is uploaded and validated
2. **Created**: Job is created with `pending` status
3. **Queued**: Job is added to Redis queue
4. **Processing**: Worker picks up job and processes rows
5. **Progress Updates**: Status updated every 10 rows
6. **Completed**: Job finished with success/error summary

### Progress Tracking

Poll the import status endpoint to track progress:

```javascript
async function trackImport(jobId) {
  while (true) {
    const response = await fetch(`/api/v1/customers/import/${jobId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const job = await response.json();

    console.log(`Progress: ${job.progress}%`);
    console.log(`Processed: ${job.processed_rows}/${job.total_rows}`);
    console.log(`Success: ${job.successful_rows}, Failed: ${job.failed_rows}`);

    if (job.status === 'completed' || job.status === 'failed') {
      console.log('Import finished!');
      if (job.errors.length > 0) {
        console.log('Errors:', job.errors);
      }
      break;
    }

    await new Promise((resolve) => setTimeout(resolve, 2000)); // Poll every 2s
  }
}
```

---

## Database Schema

### import_jobs Table

```sql
CREATE TABLE import_jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_rows INT NOT NULL DEFAULT 0,
    processed_rows INT NOT NULL DEFAULT 0,
    successful_rows INT NOT NULL DEFAULT 0,
    failed_rows INT NOT NULL DEFAULT 0,
    error_summary TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### import_errors Table

```sql
CREATE TABLE import_errors (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES import_jobs(id),
    row_number INT NOT NULL,
    field VARCHAR(100),
    error TEXT NOT NULL,
    row_data TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## Best Practices

### Importing Large Datasets

1. **Split into chunks**: Break large files into 5,000-10,000 row chunks
2. **Use dry_run**: Test import with `dry_run=true` first
3. **Handle duplicates**: Choose appropriate `duplicate_handling` strategy
4. **Monitor progress**: Poll job status regularly
5. **Review errors**: Check error summary before retrying

### Export Optimization

1. **Use filters**: Export only needed data with filters
2. **Choose format wisely**: CSV for spreadsheets, JSON for applications
3. **Exclude notes**: Set `include_notes=false` for smaller CSV files

### Error Recovery

1. **Download error report**: Get detailed errors from status endpoint
2. **Fix source data**: Correct validation errors in original file
3. **Retry with update**: Use `duplicate_handling=update` to fix errors
4. **Incremental imports**: Import only failed rows after fixing

---

## Testing

Run import/export tests:

```bash
# CSV parser tests
go test ./pkg/csv/... -v

# All tests
go test ./... -v
```

---

## Examples

### Complete Import Workflow

```bash
# 1. Download template
curl http://localhost:8080/api/v1/customers/import/template \
  -H "Authorization: Bearer $TOKEN" \
  -o template.csv

# 2. Fill in customer data in template.csv

# 3. Dry run to test
curl -X POST http://localhost:8080/api/v1/customers/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@template.csv" \
  -F "format=csv" \
  -F "dry_run=true"

# 4. Actual import
JOB=$(curl -X POST http://localhost:8080/api/v1/customers/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@customers.csv" \
  -F "format=csv" \
  -F "duplicate_handling=update" | jq -r '.job_id')

# 5. Track progress
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/customers/import/$JOB \
    -H "Authorization: Bearer $TOKEN" | jq -r '.status')
  echo "Status: $STATUS"
  [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ] && break
  sleep 2
done

# 6. Get final results
curl http://localhost:8080/api/v1/customers/import/$JOB \
  -H "Authorization: Bearer $TOKEN" | jq
```

### Export and Backup

```bash
# Daily backup of all customers
DATE=$(date +%Y%m%d)
curl "http://localhost:8080/api/v1/customers/export?format=json" \
  -H "Authorization: Bearer $TOKEN" \
  -o "backup_customers_$DATE.json"

# Export for reporting
curl "http://localhost:8080/api/v1/customers/export?format=csv&status=active&include_notes=false" \
  -H "Authorization: Bearer $TOKEN" \
  -o "active_customers.csv"
```

---

## Migration

Apply the import/export database migration:

```bash
psql -U postgres -d servicepro -f migrations/008_create_import_export_tables.sql
```

Rollback if needed:

```bash
psql -U postgres -d servicepro -f migrations/008_create_import_export_tables_rollback.sql
```
