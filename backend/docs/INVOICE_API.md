# Invoice API Documentation

REST API for invoice management using Gin framework.

## Table of Contents

- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [Endpoints](#endpoints)
- [Request/Response Examples](#requestresponse-examples)
- [Error Handling](#error-handling)
- [Status Codes](#status-codes)

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All invoice endpoints require JWT authentication.

### Headers

```
Authorization: Bearer <jwt_token>
```

### JWT Token Format

```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "role": "admin",
  "exp": 1234567890
}
```

## Rate Limiting

Rate limits apply to prevent abuse:

| Endpoint Type  | Limit        | Window              |
| -------------- | ------------ | ------------------- |
| List Invoices  | 100 requests | per minute          |
| Create Invoice | 20 requests  | per minute          |
| Update Invoice | 30 requests  | per minute          |
| Global API     | 60 requests  | per minute per user |

### Rate Limit Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1234567890
Retry-After: 45 (when limit exceeded)
```

## Endpoints

### 1. List Invoices

Get a paginated list of invoices with optional filters.

```http
GET /api/v1/invoices
```

#### Query Parameters

| Parameter   | Type    | Required | Description                                        |
| ----------- | ------- | -------- | -------------------------------------------------- |
| customer_id | UUID    | No       | Filter by customer ID                              |
| status      | string  | No       | Filter by status (draft, sent, viewed, paid, etc.) |
| from_date   | date    | No       | Filter by issue date from (YYYY-MM-DD)             |
| to_date     | date    | No       | Filter by issue date to (YYYY-MM-DD)               |
| min_amount  | decimal | No       | Filter by minimum amount                           |
| max_amount  | decimal | No       | Filter by maximum amount                           |
| is_overdue  | boolean | No       | Filter overdue invoices                            |
| search      | string  | No       | Search by invoice number or notes                  |
| page        | integer | No       | Page number (default: 1)                           |
| page_size   | integer | No       | Page size (default: 20)                            |
| sort_by     | string  | No       | Sort field (default: created_at)                   |
| sort_order  | string  | No       | Sort order: asc/desc (default: desc)               |

#### Response

```json
{
  "invoices": [...],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

---

### 2. Get Invoice

Get a single invoice by ID.

```http
GET /api/v1/invoices/:id
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Response

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
  "total_amount": "1082.50",
  "amount_paid": "0.00",
  "amount_due": "1082.50",
  "lines": [...],
  "payments": [...],
  "is_overdue": false,
  "days_overdue": 0,
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

---

### 3. Create Invoice

Create a new invoice.

```http
POST /api/v1/invoices
```

#### Request Body

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

#### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "customer_id": "uuid",
  "status": "draft",
  "issue_date": "2024-01-15",
  "due_date": "2024-02-14",
  "subtotal": "1500.00",
  "tax_amount": "123.75",
  "total_amount": "1623.75",
  "amount_paid": "0.00",
  "amount_due": "1623.75",
  "lines": [...],
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

---

### 4. Update Invoice

Update an existing invoice.

```http
PUT /api/v1/invoices/:id
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Request Body

All fields are optional. Only include fields you want to update.

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

#### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "sent",
  ...
}
```

---

### 5. Delete Invoice

Soft delete an invoice.

```http
DELETE /api/v1/invoices/:id
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Response

```
HTTP 204 No Content
```

---

### 6. Send Invoice

Mark invoice as sent and update sent_date.

```http
POST /api/v1/invoices/:id/send
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "sent",
  "sent_date": "2024-01-15T10:00:00Z",
  ...
}
```

---

### 7. Record Payment

Record a payment for an invoice.

```http
POST /api/v1/invoices/:id/payments
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Request Body

```json
{
  "amount": "1082.50",
  "payment_date": "2024-01-20",
  "payment_method": "bank_transfer",
  "reference_number": "TXN-2024-001",
  "notes": "Payment received via wire transfer"
}
```

#### Response

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

---

### 8. Cancel Invoice

Cancel an invoice with a reason.

```http
POST /api/v1/invoices/:id/cancel
```

#### Path Parameters

| Parameter | Type | Required | Description |
| --------- | ---- | -------- | ----------- |
| id        | UUID | Yes      | Invoice ID  |

#### Request Body

```json
{
  "reason": "Customer requested cancellation"
}
```

#### Response

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "status": "cancelled",
  "notes": "Original notes\n\nCancellation reason: Customer requested cancellation",
  ...
}
```

## Request/Response Examples

### Example 1: Create Invoice with Multiple Line Items

**Request:**

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
        "description": "Frontend Development - 60 hours",
        "quantity": "60.00",
        "unit_price": "150.00",
        "taxable": true
      },
      {
        "description": "Backend Development - 50 hours",
        "quantity": "50.00",
        "unit_price": "150.00",
        "taxable": true
      }
    ]
  }'
```

**Response:**

```json
{
  "id": "850e8400-e29b-41d4-a716-446655440003",
  "invoice_number": "INV-2024-00001",
  "customer_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "draft",
  "issue_date": "2024-01-15",
  "due_date": "2024-02-14",
  "subtotal": "17500.00",
  "tax_amount": "1443.75",
  "discount_amount": "0.00",
  "total_amount": "18943.75",
  "amount_paid": "0.00",
  "amount_due": "18943.75",
  "lines": [
    {
      "id": "950e8400-e29b-41d4-a716-446655440004",
      "description": "Website Design - 40 hours",
      "quantity": "40.00",
      "unit_price": "125.00",
      "line_total": "5000.00",
      "taxable": true,
      "tax_amount": "412.50",
      "line_total_with_tax": "5412.50",
      "sort_order": 0
    },
    {
      "id": "a50e8400-e29b-41d4-a716-446655440005",
      "description": "Frontend Development - 60 hours",
      "quantity": "60.00",
      "unit_price": "150.00",
      "line_total": "9000.00",
      "taxable": true,
      "tax_amount": "742.50",
      "line_total_with_tax": "9742.50",
      "sort_order": 1
    },
    {
      "id": "b50e8400-e29b-41d4-a716-446655440006",
      "description": "Backend Development - 50 hours",
      "quantity": "50.00",
      "unit_price": "150.00",
      "line_total": "7500.00",
      "taxable": true,
      "tax_amount": "618.75",
      "line_total_with_tax": "8118.75",
      "sort_order": 2
    }
  ],
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

### Example 2: List Overdue Invoices

**Request:**

```bash
curl -X GET "http://localhost:8080/api/v1/invoices?is_overdue=true&page=1&page_size=10" \
  -H "Authorization: Bearer <token>"
```

**Response:**

```json
{
  "invoices": [
    {
      "id": "uuid",
      "invoice_number": "INV-2024-00042",
      "customer_id": "uuid",
      "status": "overdue",
      "issue_date": "2023-12-01",
      "due_date": "2023-12-31",
      "total_amount": "5000.00",
      "amount_paid": "0.00",
      "amount_due": "5000.00",
      "created_at": "2023-12-01T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1
}
```

### Example 3: Record Partial Payment

**Request:**

```bash
curl -X POST http://localhost:8080/api/v1/invoices/850e8400-e29b-41d4-a716-446655440003/payments \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": "9000.00",
    "payment_date": "2024-01-20",
    "payment_method": "credit_card",
    "reference_number": "CC-2024-001",
    "notes": "Partial payment received"
  }'
```

**Response:**

```json
{
  "id": "c50e8400-e29b-41d4-a716-446655440007",
  "invoice_id": "850e8400-e29b-41d4-a716-446655440003",
  "amount": "9000.00",
  "payment_date": "2024-01-20",
  "payment_method": "credit_card",
  "reference_number": "CC-2024-001",
  "notes": "Partial payment received",
  "created_by": "uuid",
  "created_at": "2024-01-20T10:00:00Z",
  "updated_at": "2024-01-20T10:00:00Z"
}
```

## Error Handling

### Error Response Format

```json
{
  "error": "Error message",
  "message": "Detailed error description",
  "details": {
    "field": "additional context"
  }
}
```

### Validation Error Response

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

| Error                 | Status Code | Description                  |
| --------------------- | ----------- | ---------------------------- |
| Invalid UUID          | 400         | Invalid invoice ID format    |
| Validation failed     | 400         | Request validation errors    |
| Unauthorized          | 401         | Missing or invalid JWT token |
| Forbidden             | 403         | Insufficient permissions     |
| Invoice not found     | 404         | Invoice doesn't exist        |
| Rate limit exceeded   | 429         | Too many requests            |
| Internal server error | 500         | Unexpected server error      |

## Status Codes

| Code | Status                | Description                              |
| ---- | --------------------- | ---------------------------------------- |
| 200  | OK                    | Request successful                       |
| 201  | Created               | Resource created successfully            |
| 204  | No Content            | Request successful, no content to return |
| 400  | Bad Request           | Invalid request data                     |
| 401  | Unauthorized          | Authentication required                  |
| 403  | Forbidden             | Insufficient permissions                 |
| 404  | Not Found             | Resource not found                       |
| 429  | Too Many Requests     | Rate limit exceeded                      |
| 500  | Internal Server Error | Server error                             |

## Webhook Events (Future)

Future webhook events that can be subscribed to:

- `invoice.created`
- `invoice.sent`
- `invoice.viewed`
- `invoice.paid`
- `invoice.payment_received`
- `invoice.overdue`
- `invoice.cancelled`

## SDKs and Client Libraries

Coming soon:

- Go client library
- JavaScript/TypeScript client
- Python client
- cURL examples collection

## Support

For API support:

- Email: api@servicepro.com
- Documentation: https://docs.servicepro.com
- GitHub Issues: https://github.com/servicepro/api/issues
