# Quotes API

REST API for quote management with state machine workflow and real-time notifications.

## Overview

The Quotes API provides:

- Quote creation and management
- State machine for status transitions
- WebSocket notifications for real-time updates
- Status history tracking
- Automatic expiration handling

## Endpoints

| Method | Endpoint              | Description        |
| ------ | --------------------- | ------------------ |
| GET    | `/quotes`             | List quotes        |
| GET    | `/quotes/:id`         | Get quote details  |
| POST   | `/quotes`             | Create quote       |
| PUT    | `/quotes/:id`         | Update quote       |
| DELETE | `/quotes/:id`         | Delete quote       |
| POST   | `/quotes/:id/send`    | Send to customer   |
| GET    | `/quotes/:id/view`    | Mark as viewed     |
| POST   | `/quotes/:id/accept`  | Accept quote       |
| POST   | `/quotes/:id/decline` | Decline quote      |
| GET    | `/quotes/:id/history` | Get status history |
| POST   | `/quotes/:id/convert` | Convert to invoice |

## Quote Status Flow

```
┌─────────┐     send      ┌─────────┐     view      ┌─────────┐
│  DRAFT  │──────────────>│  SENT   │──────────────>│  VIEWED │
└─────────┘               └─────────┘               └─────────┘
                              │  │                       │  │
                          accept│  │decline          accept│  │decline
                              │  │                       │  │
                              ▼  ▼                       ▼  ▼
                          ┌──────────┐              ┌──────────┐
                          │ ACCEPTED │              │ DECLINED │
                          └──────────┘              └──────────┘
                              │
                              │ expire
                              ▼
                          ┌──────────┐
                          │ EXPIRED  │
                          └──────────┘
```

### Status Definitions

| Status   | Description          | Terminal |
| -------- | -------------------- | -------- |
| draft    | Being created/edited | No       |
| sent     | Sent to customer     | No       |
| viewed   | Customer viewed      | No       |
| accepted | Customer accepted    | Yes\*    |
| declined | Customer declined    | Yes      |
| expired  | Past valid date      | Yes      |

\*Accepted quotes can only transition to expired.

## Create Quote

```http
POST /api/v1/quotes
```

### Request Body

```json
{
  "customer_id": "uuid",
  "valid_until": "2024-02-15",
  "notes": "Quote for HVAC installation",
  "terms_and_conditions": "Valid for 30 days",
  "items": [
    {
      "description": "HVAC System Installation",
      "quantity": 1,
      "unit_price": "2500.00",
      "taxable": true
    },
    {
      "description": "Labor (8 hours)",
      "quantity": 8,
      "unit_price": "75.00",
      "taxable": true
    }
  ]
}
```

### Response (201 Created)

```json
{
  "id": "uuid",
  "quote_number": "QTE-2024-00001",
  "customer_id": "uuid",
  "status": "draft",
  "subtotal": "3100.00",
  "tax_amount": "255.75",
  "total_amount": "3355.75",
  "valid_until": "2024-02-15",
  "items": [...],
  "created_at": "2024-01-15T10:00:00Z"
}
```

## Send Quote

Send a quote to the customer via email.

```http
POST /api/v1/quotes/:id/send
```

### Validation

Before sending, the quote is validated:

- Must have at least one line item
- Customer must have valid email
- Valid until date must be in the future

### Response

```json
{
  "id": "uuid",
  "quote_number": "QTE-2024-00001",
  "status": "sent",
  "sent_at": "2024-01-15T10:30:00Z"
}
```

## Mark as Viewed

Called when customer opens the quote.

```http
GET /api/v1/quotes/:id/view
```

### Response

```json
{
  "id": "uuid",
  "status": "viewed",
  "viewed_at": "2024-01-16T09:00:00Z"
}
```

## Accept Quote

Customer accepts the quote.

```http
POST /api/v1/quotes/:id/accept
```

### Request Body

```json
{
  "acceptance_notes": "Looks good, proceed with installation"
}
```

### Response

```json
{
  "id": "uuid",
  "status": "accepted",
  "accepted_at": "2024-01-17T14:00:00Z"
}
```

## Decline Quote

Customer declines the quote.

```http
POST /api/v1/quotes/:id/decline
```

### Request Body

```json
{
  "reason": "Found a better price elsewhere"
}
```

### Response

```json
{
  "id": "uuid",
  "status": "declined",
  "declined_at": "2024-01-17T14:00:00Z"
}
```

## Get Status History

Get the complete status change history for a quote.

```http
GET /api/v1/quotes/:id/history
```

### Response

```json
[
  {
    "id": "uuid",
    "from_status": null,
    "to_status": "draft",
    "event": "quote.created",
    "changed_by_id": "user-uuid",
    "changed_by_type": "user",
    "created_at": "2024-01-15T10:00:00Z"
  },
  {
    "id": "uuid",
    "from_status": "draft",
    "to_status": "sent",
    "event": "quote.sent",
    "changed_by_id": "user-uuid",
    "changed_by_type": "user",
    "created_at": "2024-01-15T10:30:00Z"
  },
  {
    "id": "uuid",
    "from_status": "sent",
    "to_status": "viewed",
    "event": "quote.viewed",
    "changed_by_id": "customer-uuid",
    "changed_by_type": "customer",
    "created_at": "2024-01-16T09:00:00Z"
  }
]
```

## Convert to Invoice

Convert an accepted quote to an invoice.

```http
POST /api/v1/quotes/:id/convert
```

### Validation

- Quote must have status `accepted`
- Quote must not be expired

### Response

```json
{
  "quote_id": "uuid",
  "invoice_id": "new-invoice-uuid",
  "invoice_number": "INV-2024-00001",
  "message": "Quote successfully converted to invoice"
}
```

## WebSocket Notifications

Connect to receive real-time quote status updates.

```
WS /api/v1/ws
```

### Subscribe to Quote

```json
{
  "action": "subscribe",
  "quote_id": "uuid"
}
```

### Status Change Notification

```json
{
  "type": "status_change",
  "quote_id": "uuid",
  "event": "quote.accepted",
  "status": "accepted",
  "timestamp": "2024-01-17T14:00:00Z",
  "data": {
    "from_status": "viewed",
    "reason": "Looks good!"
  }
}
```

## Events

| Event          | Trigger           | Status Change          |
| -------------- | ----------------- | ---------------------- |
| quote.created  | Quote created     | → draft                |
| quote.sent     | User sends quote  | draft → sent           |
| quote.viewed   | Customer views    | sent → viewed          |
| quote.accepted | Customer accepts  | sent/viewed → accepted |
| quote.declined | Customer declines | sent/viewed → declined |
| quote.expired  | System expires    | any → expired          |
| quote.updated  | Quote modified    | No status change       |

## Allowed Transitions

### From Draft

- → sent (requires line items, customer email)

### From Sent

- → viewed (automatic when customer opens)
- → accepted
- → declined
- → draft (back to editing)

### From Viewed

- → accepted
- → declined

### From Accepted

- → expired (only via expiration)

### Terminal States

- declined: No further transitions
- expired: No further transitions

## Error Responses

### Invalid Transition (400)

```json
{
  "error": "invalid_transition",
  "message": "Cannot accept quote with status 'draft'"
}
```

### Quote Expired (409)

```json
{
  "error": "quote_expired",
  "message": "This quote has expired"
}
```

### Missing Line Items (400)

```json
{
  "error": "validation_error",
  "message": "Cannot send quote without line items"
}
```

## Examples

### Create and Send Quote

```bash
# Create quote
curl -X POST http://localhost:8080/api/v1/quotes \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "customer-uuid",
    "valid_until": "2024-02-15",
    "items": [
      {"description": "Service", "quantity": 1, "unit_price": "500.00"}
    ]
  }'

# Send quote
curl -X POST http://localhost:8080/api/v1/quotes/{id}/send \
  -H "Authorization: Bearer <token>"
```

### Frontend WebSocket Integration

```typescript
const { lastMessage, isConnected } = useQuoteStatusUpdates(
  quoteId,
  (message) => {
    console.log('Status changed to:', message.status);
    refetchQuote();
  }
);
```

## Database Schema

### quotes table

| Column               | Type          | Description                     |
| -------------------- | ------------- | ------------------------------- |
| id                   | UUID          | Primary key                     |
| quote_number         | VARCHAR       | Auto-generated (QTE-YYYY-NNNNN) |
| customer_id          | UUID          | Foreign key to customers        |
| status               | VARCHAR(20)   | Current status                  |
| subtotal             | DECIMAL(12,2) | Sum of line items               |
| tax_amount           | DECIMAL(12,2) | Calculated tax                  |
| total_amount         | DECIMAL(12,2) | Final amount                    |
| valid_until          | DATE          | Expiration date                 |
| sent_at              | TIMESTAMP     | When sent                       |
| viewed_at            | TIMESTAMP     | When viewed                     |
| accepted_at          | TIMESTAMP     | When accepted                   |
| declined_at          | TIMESTAMP     | When declined                   |
| notes                | TEXT          | Quote notes                     |
| terms_and_conditions | TEXT          | Terms                           |
| created_at           | TIMESTAMP     | Creation timestamp              |
| updated_at           | TIMESTAMP     | Last update                     |

### quote_status_history table

| Column          | Type        | Description               |
| --------------- | ----------- | ------------------------- |
| id              | UUID        | Primary key               |
| quote_id        | UUID        | Foreign key to quotes     |
| from_status     | VARCHAR(20) | Previous status           |
| to_status       | VARCHAR(20) | New status                |
| event           | VARCHAR(50) | Event type                |
| changed_by_id   | UUID        | User/customer who changed |
| changed_by_type | VARCHAR(20) | user, customer, system    |
| reason          | TEXT        | Reason for change         |
| created_at      | TIMESTAMP   | When changed              |
