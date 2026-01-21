# Invoice API Implementation

Complete Go REST API for invoice management using Gin framework with comprehensive middleware, business logic, and testing.

## 📦 Files Created

### Service Layer

**`internal/services/invoice_service.go`** (481 lines)

- Complete business logic for invoice management
- CRUD operations with validation
- Payment recording and processing
- Invoice lifecycle management (send, cancel)
- Automatic calculations and validations
- Line item management
- Due date calculation from payment terms

### API Handlers

**`internal/api/handlers/invoice_handler.go`** (545 lines)

- RESTful HTTP handlers for all invoice operations
- Request/response DTOs with validation
- Proper error handling and status codes
- Swagger/OpenAPI annotations
- Request parsing and transformation

### Middleware Components

**`internal/api/middleware/auth.go`** (86 lines)

- JWT authentication middleware
- Role-based access control
- Optional authentication support
- Token validation and parsing
- User context management

**`internal/api/middleware/error_handler.go`** (119 lines)

- Centralized error handling
- Validation error formatting
- Panic recovery
- Standardized error responses
- Not found and method not allowed handlers

**`internal/api/middleware/logger.go`** (155 lines)

- Request logging middleware
- Structured logging support
- Access log generation
- Request ID tracking
- Latency measurement

**`internal/api/middleware/ratelimit_invoice.go`** (229 lines)

- Token bucket rate limiter
- IP-based rate limiting
- User-based rate limiting
- Endpoint-specific limits
- Automatic cleanup routine
- Rate limit headers (X-RateLimit-\*)

### Routes

**`internal/api/routes/invoice_routes.go`** (93 lines)

- Route configuration
- Middleware application
- Rate limit setup
- Public vs authenticated routes

### Tests

**`internal/services/invoice_service_test.go`** (528 lines)

- Comprehensive unit tests
- Test database setup
- 15+ test cases covering:
  - Invoice creation
  - Updates and deletions
  - Listing and filtering
  - Payment recording
  - Validation
  - Status transitions

### Documentation

**`docs/INVOICE_API.md`** (700+ lines)

- Complete API documentation
- All endpoints with examples
- Request/response formats
- Error handling guide
- Rate limiting documentation
- cURL examples

## ✨ Features Implemented

### HTTP Handlers

- ✅ **ListInvoices** - Paginated list with extensive filtering
- ✅ **GetInvoice** - Single invoice retrieval with relationships
- ✅ **CreateInvoice** - Invoice creation with line items
- ✅ **UpdateInvoice** - Update invoice and line items
- ✅ **DeleteInvoice** - Soft delete implementation
- ✅ **SendInvoice** - Mark as sent with validation
- ✅ **RecordPayment** - Payment tracking (full/partial)
- ✅ **CancelInvoice** - Cancellation with reason

### Middleware

- ✅ **Authentication** - JWT validation and user context
- ✅ **Authorization** - Role-based access control
- ✅ **Validation** - Request body validation with detailed errors
- ✅ **Error Handling** - Standardized error responses
- ✅ **Logging** - Request/response logging
- ✅ **Rate Limiting** - Multiple strategies (IP, user, endpoint)
- ✅ **Recovery** - Panic recovery
- ✅ **Request ID** - Request tracking

### Business Logic

- ✅ **Invoice Number Generation** - Auto-generated via DB trigger
- ✅ **Tax Calculation** - Automatic tax calculation on line items
- ✅ **Total Calculation** - Automatic totals via DB triggers
- ✅ **Due Date Setting** - Calculated from payment terms
- ✅ **Payment Processing** - Full and partial payments
- ✅ **Status Management** - Invoice lifecycle state machine
- ✅ **Validation** - Comprehensive validation rules

### Testing

- ✅ **Unit Tests** - 15+ test cases
- ✅ **Test Database** - In-memory SQLite
- ✅ **Test Fixtures** - Helper functions for test data
- ✅ **Coverage** - All major code paths
- ✅ **Assertions** - Comprehensive assertions using testify

## 🎯 API Endpoints

| Method | Endpoint                        | Description                |
| ------ | ------------------------------- | -------------------------- |
| GET    | `/api/v1/invoices`              | List invoices with filters |
| GET    | `/api/v1/invoices/:id`          | Get single invoice         |
| POST   | `/api/v1/invoices`              | Create invoice             |
| PUT    | `/api/v1/invoices/:id`          | Update invoice             |
| DELETE | `/api/v1/invoices/:id`          | Delete invoice             |
| POST   | `/api/v1/invoices/:id/send`     | Send invoice               |
| POST   | `/api/v1/invoices/:id/payments` | Record payment             |
| POST   | `/api/v1/invoices/:id/cancel`   | Cancel invoice             |

## 🔒 Security Features

### Authentication

```go
// JWT-based authentication
Authorization: Bearer <jwt_token>

// Token contains:
{
  "user_id": "uuid",
  "email": "user@example.com",
  "role": "admin",
  "exp": 1234567890
}
```

### Rate Limiting

| Endpoint       | Limit           |
| -------------- | --------------- |
| List Invoices  | 100/min         |
| Create Invoice | 20/min          |
| Update Invoice | 30/min          |
| Global API     | 60/min per user |

### Validation

- Request body validation using struct tags
- Field-level validation with detailed error messages
- Business logic validation in service layer
- Database-level constraints

## 📊 Request/Response Examples

### Create Invoice

**Request:**

```bash
POST /api/v1/invoices
Authorization: Bearer <token>
Content-Type: application/json

{
  "customer_id": "uuid",
  "payment_term_id": "uuid",
  "tax_rate_id": "uuid",
  "lines": [
    {
      "description": "Consulting - 10 hours",
      "quantity": "10.00",
      "unit_price": "150.00",
      "taxable": true
    }
  ]
}
```

**Response:**

```json
{
  "id": "uuid",
  "invoice_number": "INV-2024-00001",
  "customer_id": "uuid",
  "status": "draft",
  "subtotal": "1500.00",
  "tax_amount": "123.75",
  "total_amount": "1623.75",
  "lines": [...]
}
```

### List with Filters

**Request:**

```bash
GET /api/v1/invoices?status=overdue&customer_id=uuid&page=1&page_size=20
Authorization: Bearer <token>
```

**Response:**

```json
{
  "invoices": [...],
  "total": 5,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

### Record Payment

**Request:**

```bash
POST /api/v1/invoices/:id/payments
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": "1623.75",
  "payment_method": "bank_transfer",
  "reference_number": "TXN-001"
}
```

**Response:**

```json
{
  "id": "uuid",
  "invoice_id": "uuid",
  "amount": "1623.75",
  "payment_date": "2024-01-15",
  "payment_method": "bank_transfer",
  "reference_number": "TXN-001"
}
```

## 🛠️ Usage

### Setup Routes in Your Application

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/javaknight1/servicepro/backend/internal/api/middleware"
    "github.com/javaknight1/servicepro/backend/internal/api/routes"
    "gorm.io/gorm"
)

func main() {
    router := gin.Default()

    // Global middleware
    router.Use(middleware.RequestLogger())
    router.Use(middleware.RecoveryMiddleware())
    router.Use(middleware.ErrorHandler())
    router.Use(middleware.RequestIDMiddleware())

    // Global rate limiting
    router.Use(middleware.GlobalRateLimiter.RateLimit())

    // API v1 group
    v1 := router.Group("/api/v1")

    // Setup invoice routes
    routes.SetupInvoiceRoutes(v1, db, jwtSecret)

    // Start server
    router.Run(":8080")
}
```

### Running Tests

```bash
# Run all invoice service tests
go test ./internal/services -v -run TestInvoiceService

# Run specific test
go test ./internal/services -v -run TestInvoiceService_CreateInvoice

# Run with coverage
go test ./internal/services -cover -coverprofile=coverage.out

# View coverage HTML
go tool cover -html=coverage.out
```

### Example Client Usage

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func createInvoice() {
    invoice := map[string]interface{}{
        "customer_id": "uuid",
        "lines": []map[string]interface{}{
            {
                "description": "Service",
                "quantity": "10.00",
                "unit_price": "100.00",
                "taxable": true,
            },
        },
    }

    body, _ := json.Marshal(invoice)

    req, _ := http.NewRequest("POST",
        "http://localhost:8080/api/v1/invoices",
        bytes.NewBuffer(body))

    req.Header.Set("Authorization", "Bearer <token>")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, _ := client.Do(req)
    defer resp.Body.Close()

    // Handle response...
}
```

## 🧪 Testing Strategy

### Unit Tests

- Service layer business logic
- In-memory database (SQLite)
- Test fixtures and helpers
- Comprehensive assertions

### Test Coverage

```
CreateInvoice      ✅ Success case
                   ✅ Missing customer ID
                   ✅ Due date calculation

GetInvoice         ✅ Found
                   ✅ Not found

UpdateInvoice      ✅ Update fields
                   ✅ Cannot update paid invoice

DeleteInvoice      ✅ Soft delete
                   ✅ Cannot delete paid

ListInvoices       ✅ All invoices
                   ✅ Filter by customer
                   ✅ Filter by status
                   ✅ Filter by amount
                   ✅ Pagination

RecordPayment      ✅ Full payment
                   ✅ Negative amount fails
                   ✅ Exceeds balance fails

SendInvoice        ✅ Valid invoice
                   ✅ Requires line items
```

## 📈 Performance Considerations

### Database Optimization

- Indexes on frequently queried fields
- Eager loading of relationships with Preload
- Pagination to limit result sets
- Computed columns for derived values

### Rate Limiting

- In-memory token bucket algorithm
- Automatic cleanup of old buckets
- Per-user and per-IP limits
- Graceful degradation

### Caching Opportunities

- Tax rates (rarely change)
- Payment terms (rarely change)
- Invoice summaries
- Customer invoice counts

## 🔧 Configuration

### Environment Variables

```bash
# Server
PORT=8080
GIN_MODE=release

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=servicepro
DB_USER=postgres
DB_PASSWORD=password

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=1m
```

### Middleware Configuration

```go
// Custom rate limits
createLimiter := middleware.NewRateLimiter(10, time.Minute)
updateLimiter := middleware.NewRateLimiter(20, time.Minute)

// Custom authentication
auth := middleware.AuthMiddleware(jwtSecret)

// Custom error handling
errorHandler := middleware.ErrorHandler()

// Apply to routes
router.Use(auth, errorHandler, createLimiter.RateLimit())
```

## 📝 Best Practices

### Error Handling

- Use custom error types
- Provide meaningful error messages
- Log errors for debugging
- Return appropriate HTTP status codes

### Validation

- Validate at multiple layers
- Use struct tags for basic validation
- Implement business logic validation
- Return detailed validation errors

### Security

- Always validate JWT tokens
- Implement rate limiting
- Sanitize user input
- Use HTTPS in production
- Never log sensitive data

### Testing

- Write tests before implementing features
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and failure cases

## 🚀 Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/api/main.go

EXPOSE 8080

CMD ["./main"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - '8080:8080'
    environment:
      - DB_HOST=db
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=servicepro
      - POSTGRES_PASSWORD=password
```

## 📚 Additional Resources

- [Invoice System Documentation](../INVOICE_SYSTEM.md)
- [API Documentation](INVOICE_API.md)
- [SQL Reference](INVOICE_SQL_REFERENCE.md)
- [Gin Framework Docs](https://gin-gonic.com/docs/)
- [GORM Documentation](https://gorm.io/docs/)

## 🤝 Contributing

1. Write tests for new features
2. Follow Go best practices
3. Update documentation
4. Ensure all tests pass
5. Submit pull request

## 📄 License

MIT License
