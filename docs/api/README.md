# ServicePro API Reference

REST API documentation for the ServicePro platform.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All API endpoints (except registration and login) require JWT authentication.

### Headers

```
Authorization: Bearer <jwt_token>
```

### Obtaining a Token

1. Register a new account: `POST /api/v1/auth/register`
2. Login: `POST /api/v1/auth/login`
3. Use the returned token in subsequent requests

### Token Format

```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "role": "admin",
  "exp": 1234567890
}
```

## API Endpoints Overview

### Authentication

| Method | Endpoint               | Description               |
| ------ | ---------------------- | ------------------------- |
| POST   | `/auth/register`       | Register new account      |
| POST   | `/auth/login`          | Login and get token       |
| POST   | `/auth/logout`         | Logout (invalidate token) |
| POST   | `/auth/reset-request`  | Request password reset    |
| POST   | `/auth/reset-password` | Reset password with token |
| POST   | `/auth/verify-email`   | Verify email address      |

See [Authentication API](./authentication.md) for details.

### Customers

| Method | Endpoint         | Description          |
| ------ | ---------------- | -------------------- |
| GET    | `/customers`     | List customers       |
| GET    | `/customers/:id` | Get customer details |
| POST   | `/customers`     | Create customer      |
| PUT    | `/customers/:id` | Update customer      |
| DELETE | `/customers/:id` | Delete customer      |

See [Customers API](./customers.md) for details.

### Jobs

| Method | Endpoint           | Description                |
| ------ | ------------------ | -------------------------- |
| GET    | `/jobs`            | List jobs                  |
| GET    | `/jobs/:id`        | Get job details            |
| POST   | `/jobs`            | Create job                 |
| PUT    | `/jobs/:id`        | Update job                 |
| DELETE | `/jobs/:id`        | Delete job                 |
| POST   | `/jobs/:id/assign` | Assign technician          |
| GET    | `/jobs/conflicts`  | Check scheduling conflicts |

See [Jobs API](./jobs.md) for details.

### Invoices

| Method | Endpoint                 | Description         |
| ------ | ------------------------ | ------------------- |
| GET    | `/invoices`              | List invoices       |
| GET    | `/invoices/:id`          | Get invoice details |
| POST   | `/invoices`              | Create invoice      |
| PUT    | `/invoices/:id`          | Update invoice      |
| DELETE | `/invoices/:id`          | Delete invoice      |
| POST   | `/invoices/:id/send`     | Send invoice        |
| POST   | `/invoices/:id/payments` | Record payment      |
| POST   | `/invoices/:id/cancel`   | Cancel invoice      |

See [Invoices API](./invoices.md) for details.

### Payments

| Method | Endpoint                | Description               |
| ------ | ----------------------- | ------------------------- |
| POST   | `/payments/intent`      | Create payment intent     |
| POST   | `/payments/confirm`     | Confirm payment           |
| GET    | `/payments/methods`     | Get saved payment methods |
| POST   | `/payments/refund`      | Create refund             |
| GET    | `/payments/receipt/:id` | Get payment receipt       |

See [Payments API](./payments.md) for details.

### Quotes

| Method | Endpoint              | Description        |
| ------ | --------------------- | ------------------ |
| GET    | `/quotes`             | List quotes        |
| GET    | `/quotes/:id`         | Get quote details  |
| POST   | `/quotes`             | Create quote       |
| PUT    | `/quotes/:id`         | Update quote       |
| POST   | `/quotes/:id/send`    | Send quote         |
| POST   | `/quotes/:id/accept`  | Accept quote       |
| POST   | `/quotes/:id/convert` | Convert to invoice |

See [Quotes API](./quotes.md) for details.

## Request/Response Format

### Content Type

All requests and responses use JSON:

```
Content-Type: application/json
```

### Success Responses

```json
{
  "data": { ... },
  "message": "Operation successful"
}
```

### Paginated Responses

```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

### Error Responses

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": { "field": "Additional context" }
}
```

## HTTP Status Codes

| Code | Status                | Description                               |
| ---- | --------------------- | ----------------------------------------- |
| 200  | OK                    | Request successful                        |
| 201  | Created               | Resource created                          |
| 204  | No Content            | Request successful, no content            |
| 400  | Bad Request           | Invalid request data                      |
| 401  | Unauthorized          | Authentication required                   |
| 403  | Forbidden             | Insufficient permissions                  |
| 404  | Not Found             | Resource not found                        |
| 409  | Conflict              | Resource conflict (e.g., duplicate email) |
| 429  | Too Many Requests     | Rate limit exceeded                       |
| 500  | Internal Server Error | Server error                              |

## Rate Limiting

Rate limits are applied to prevent abuse:

| Endpoint Type  | Limit        | Window              |
| -------------- | ------------ | ------------------- |
| Authentication | 5 requests   | per minute          |
| List endpoints | 100 requests | per minute          |
| Create/Update  | 30 requests  | per minute          |
| Global         | 60 requests  | per minute per user |

### Rate Limit Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1234567890
Retry-After: 45 (when limit exceeded)
```

## Filtering and Pagination

### Query Parameters

```
GET /api/v1/invoices?status=sent&customer_id=uuid&page=1&page_size=20&sort_by=created_at&sort_order=desc
```

| Parameter  | Type    | Description                            |
| ---------- | ------- | -------------------------------------- |
| page       | integer | Page number (default: 1)               |
| page_size  | integer | Items per page (default: 20, max: 100) |
| sort_by    | string  | Field to sort by                       |
| sort_order | string  | `asc` or `desc`                        |

### Date Filters

```
GET /api/v1/invoices?from_date=2024-01-01&to_date=2024-12-31
```

### Search

```
GET /api/v1/customers?search=john
```

## Common Data Types

### UUID

All resource IDs use UUID v4 format:

```
550e8400-e29b-41d4-a716-446655440000
```

### Dates

Dates use ISO 8601 format:

```
2024-01-15T10:30:00Z
```

### Money

Monetary values are strings with 2 decimal places:

```json
{
  "amount": "1500.00",
  "currency": "usd"
}
```

## Testing the API

### Health Check

```bash
curl http://localhost:8080/health
```

### Example: Create Customer

```bash
curl -X POST http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "phone": "+1234567890"
  }'
```

## API Versioning

The API is versioned via URL path (`/api/v1/`). Breaking changes will increment the version number.

## Support

For API issues:

- Check the specific endpoint documentation
- Review error messages and status codes
- Open a GitHub issue with request/response details
