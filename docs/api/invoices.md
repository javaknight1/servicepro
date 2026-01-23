# Invoice API

Complete REST API for invoice management including creation, updates, payments, and lifecycle management.

## Overview

The Invoice API provides endpoints for:

- Creating and managing invoices
- Line item management with automatic calculations
- Payment recording (full and partial)
- Status lifecycle management
- Tax and discount handling

## Endpoints

| Method | Endpoint                 | Description                  |
| ------ | ------------------------ | ---------------------------- |
| GET    | `/invoices`              | List invoices with filters   |
| GET    | `/invoices/:id`          | Get invoice details          |
| POST   | `/invoices`              | Create invoice               |
| PUT    | `/invoices/:id`          | Update invoice               |
| DELETE | `/invoices/:id`          | Delete invoice (soft delete) |
| POST   | `/invoices/:id/send`     | Send invoice to customer     |
| POST   | `/invoices/:id/payments` | Record payment               |
| POST   | `/invoices/:id/cancel`   | Cancel invoice               |

## List Invoices

Get a paginated list of invoices with optional filters.

```http
GET /api/v1/invoices
```

### Query Parameters

| Parameter   | Type    | Required | Description                      |
| ----------- | ------- | -------- | -------------------------------- |
| customer_id | UUID    | No       | Filter by customer               |
| status      | string  | No       | Filter by status                 |
| from_date   | date    | No       | Issue date from (YYYY-MM-DD)     |
| to_date     | date    | No       | Issue date to (YYYY-MM-DD)       |
| min_amount  | decimal | No       | Minimum amount                   |
| max_amount  | decimal | No       | Maximum amount                   |
| is_overdue  | boolean | No       | Filter overdue invoices          |
| search      | string  | No       | Search invoice number or notes   |
| page        | integer | No       | Page number (default: 1)         |
| page_size   | integer | No       | Items per page (default: 20)     |
| sort_by     | string  | No       | Sort field (default: created_at) |
| sort_order  | string  | No       | asc or desc (default: desc)      |

### Response

```json
{
  "invoices": [
    {
      "id": "uuid",
      "invoice_number": "INV-2024-00001",
      "customer_id": "uuid",
      "status": "sent",
      "issue_date": "2024-01-15",
      "due_date": "2024-02-14",
      "total_amount": "1082.50",
      "amount_paid": "0.00",
      "amount_due": "1082.50",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

## Get Invoice

Get a single invoice with full details.

```http
GET /api/v1/invoices/:id
```

### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "customer_id": "uuid",
  "status": "sent",
  "issue_date": "2024-01-15",
  "due_date": "2024-02-14",
  "subtotal": "1000.00",
  "tax_amount": "82.50",
  "discount_amount": "0.00",
  "total_amount": "1082.50",
  "amount_paid": "0.00",
  "amount_due": "1082.50",
  "po_number": "PO-12345",
  "notes": "Thank you for your business",
  "terms_and_conditions": "Net 30",
  "lines": [
    {
      "id": "uuid",
      "description": "Consulting Services - 10 hours",
      "quantity": "10.00",
      "unit_price": "100.00",
      "line_total": "1000.00",
      "taxable": true,
      "tax_amount": "82.50",
      "sort_order": 0
    }
  ],
  "payments": [],
  "is_overdue": false,
  "days_overdue": 0,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

## Create Invoice

Create a new invoice with line items.

```http
POST /api/v1/invoices
```

### Request Body

```json
{
  "customer_id": "uuid",
  "job_id": "uuid",
  "quote_id": "uuid",
  "issue_date": "2024-01-15",
  "due_date": "2024-02-14",
  "payment_term_id": "uuid",
  "tax_rate_id": "uuid",
  "po_number": "PO-12345",
  "notes": "Invoice notes",
  "discount_amount": "0.00",
  "lines": [
    {
      "description": "Consulting Services - 10 hours",
      "quantity": "10.00",
      "unit_price": "150.00",
      "unit_of_measure": "hours",
      "discount_percentage": "0.00",
      "discount_amount": "0.00",
      "taxable": true,
      "product_id": "uuid",
      "service_id": "uuid"
    }
  ]
}
```

### Required Fields

- `customer_id`: Customer UUID
- `lines`: At least one line item

### Response (201 Created)

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "draft",
  ...
}
```

## Update Invoice

Update an existing invoice.

```http
PUT /api/v1/invoices/:id
```

### Request Body

All fields are optional. Only include fields to update.

```json
{
  "status": "sent",
  "due_date": "2024-02-20",
  "payment_term_id": "uuid",
  "tax_rate_id": "uuid",
  "po_number": "PO-12345",
  "notes": "Updated notes",
  "discount_amount": "50.00",
  "terms_and_conditions": "Net 30",
  "lines": [...]
}
```

### Restrictions

- Cannot update invoices with status `paid` or `cancelled`
- Invoice number cannot be changed

## Delete Invoice

Soft delete an invoice.

```http
DELETE /api/v1/invoices/:id
```

### Response

```
HTTP 204 No Content
```

### Restrictions

- Cannot delete invoices with status `paid`

## Send Invoice

Mark an invoice as sent and update the sent date.

```http
POST /api/v1/invoices/:id/send
```

### Validation

Before sending, the invoice is validated:

- Must have at least one line item
- Customer must have a valid email
- Due date must be in the future

### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "sent",
  "sent_date": "2024-01-15T10:00:00Z",
  ...
}
```

## Record Payment

Record a payment against an invoice.

```http
POST /api/v1/invoices/:id/payments
```

### Request Body

```json
{
  "amount": "1082.50",
  "payment_date": "2024-01-20",
  "payment_method": "bank_transfer",
  "reference_number": "TXN-2024-001",
  "notes": "Payment received via wire transfer"
}
```

### Payment Methods

- `cash`
- `check`
- `credit_card`
- `debit_card`
- `bank_transfer`
- `paypal`
- `stripe`
- `other`

### Response

```json
{
  "id": "uuid",
  "invoice_id": "uuid",
  "amount": "1082.50",
  "payment_date": "2024-01-20",
  "payment_method": "bank_transfer",
  "reference_number": "TXN-2024-001",
  "notes": "Payment received via wire transfer",
  "created_at": "2024-01-20T10:00:00Z"
}
```

### Status Updates

- Full payment: Status changes to `paid`
- Partial payment: Status changes to `partially_paid`

## Cancel Invoice

Cancel an invoice with a reason.

```http
POST /api/v1/invoices/:id/cancel
```

### Request Body

```json
{
  "reason": "Customer requested cancellation"
}
```

### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "cancelled",
  ...
}
```

### Restrictions

- Cannot cancel invoices with status `paid`

## Invoice Status

| Status         | Description                 |
| -------------- | --------------------------- |
| draft          | Initial state, editable     |
| sent           | Sent to customer            |
| viewed         | Customer viewed the invoice |
| partially_paid | Partial payment received    |
| paid           | Fully paid                  |
| overdue        | Past due date               |
| cancelled      | Cancelled                   |
| refunded       | Payment refunded            |

### Status Transitions

```
draft → sent → viewed → partially_paid → paid
                     ↘                  ↗
                       → paid
draft → cancelled
sent → cancelled
overdue → paid (when payment received)
```

## Automatic Calculations

The following fields are calculated automatically:

- `subtotal`: Sum of all line totals
- `tax_amount`: Based on taxable line items and tax rate
- `total_amount`: subtotal + tax_amount - discount_amount
- `amount_due`: total_amount - amount_paid
- `is_overdue`: true if due_date < today and not paid

## Error Responses

### Validation Error

```json
{
  "error": "Validation failed",
  "message": "One or more fields are invalid",
  "fields": {
    "customer_id": "customer_id is required",
    "lines": "lines must have at least 1 item"
  }
}
```

### Common Errors

| Error               | Status | Description               |
| ------------------- | ------ | ------------------------- |
| Invalid UUID        | 400    | Invalid invoice ID format |
| Validation failed   | 400    | Request validation errors |
| Unauthorized        | 401    | Missing or invalid JWT    |
| Forbidden           | 403    | Insufficient permissions  |
| Invoice not found   | 404    | Invoice doesn't exist     |
| Cannot modify       | 400    | Invoice is paid/cancelled |
| Rate limit exceeded | 429    | Too many requests         |

## Rate Limits

| Endpoint       | Limit        | Window     |
| -------------- | ------------ | ---------- |
| List invoices  | 100 requests | per minute |
| Create invoice | 20 requests  | per minute |
| Update invoice | 30 requests  | per minute |

## Examples

### Create Invoice with Multiple Line Items

```bash
curl -X POST http://localhost:8080/api/v1/invoices \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "payment_term_id": "650e8400-e29b-41d4-a716-446655440001",
    "tax_rate_id": "750e8400-e29b-41d4-a716-446655440002",
    "notes": "Website development project",
    "lines": [
      {
        "description": "Website Design - 40 hours",
        "quantity": "40.00",
        "unit_price": "125.00",
        "taxable": true
      },
      {
        "description": "Development - 60 hours",
        "quantity": "60.00",
        "unit_price": "150.00",
        "taxable": true
      }
    ]
  }'
```

### List Overdue Invoices

```bash
curl -X GET "http://localhost:8080/api/v1/invoices?is_overdue=true&page=1&page_size=10" \
  -H "Authorization: Bearer <token>"
```

### Record Partial Payment

```bash
curl -X POST http://localhost:8080/api/v1/invoices/850e8400.../payments \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": "500.00",
    "payment_date": "2024-01-20",
    "payment_method": "credit_card",
    "reference_number": "CC-2024-001",
    "notes": "Partial payment received"
  }'
```

## Database Schema

### invoices table

| Column          | Type          | Description                     |
| --------------- | ------------- | ------------------------------- |
| id              | UUID          | Primary key                     |
| invoice_number  | VARCHAR       | Auto-generated (INV-YYYY-NNNNN) |
| customer_id     | UUID          | Foreign key to users            |
| job_id          | UUID          | Optional, links to job          |
| quote_id        | UUID          | Optional, links to quote        |
| status          | VARCHAR       | Invoice status                  |
| issue_date      | DATE          | Invoice issue date              |
| due_date        | DATE          | Payment due date                |
| sent_date       | TIMESTAMP     | When invoice was sent           |
| subtotal        | DECIMAL(12,2) | Sum of line items               |
| tax_amount      | DECIMAL(12,2) | Calculated tax                  |
| discount_amount | DECIMAL(12,2) | Total discount                  |
| total_amount    | DECIMAL(12,2) | Final amount                    |
| amount_paid     | DECIMAL(12,2) | Total payments received         |
| amount_due      | DECIMAL(12,2) | Remaining balance               |
| created_at      | TIMESTAMP     | Creation timestamp              |
| updated_at      | TIMESTAMP     | Last update timestamp           |
| deleted_at      | TIMESTAMP     | Soft delete timestamp           |

### invoice_lines table

| Column      | Type          | Description             |
| ----------- | ------------- | ----------------------- |
| id          | UUID          | Primary key             |
| invoice_id  | UUID          | Foreign key to invoices |
| description | TEXT          | Line item description   |
| quantity    | DECIMAL(10,2) | Quantity                |
| unit_price  | DECIMAL(10,2) | Price per unit          |
| line_total  | DECIMAL(12,2) | quantity \* unit_price  |
| taxable     | BOOLEAN       | Whether item is taxable |
| tax_amount  | DECIMAL(12,2) | Calculated tax          |
| sort_order  | INTEGER       | Display order           |

### invoice_payments table

| Column           | Type          | Description               |
| ---------------- | ------------- | ------------------------- |
| id               | UUID          | Primary key               |
| invoice_id       | UUID          | Foreign key to invoices   |
| amount           | DECIMAL(12,2) | Payment amount            |
| payment_date     | DATE          | Date of payment           |
| payment_method   | VARCHAR       | Payment method used       |
| reference_number | VARCHAR       | Transaction reference     |
| notes            | TEXT          | Payment notes             |
| created_by       | UUID          | User who recorded payment |
| created_at       | TIMESTAMP     | Creation timestamp        |
