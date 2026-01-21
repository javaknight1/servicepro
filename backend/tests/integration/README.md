# ServicePro Integration Tests

This directory contains integration tests for the ServicePro backend API.

## Directory Structure

```
tests/
├── integration/
│   ├── api/                    # API endpoint tests
│   │   ├── auth_test.go       # Authentication tests
│   │   ├── customer_test.go   # Customer API tests
│   │   └── rate_limit_test.go # Rate limiting tests
│   ├── database/              # Database operation tests
│   │   ├── crud_test.go       # CRUD operations
│   │   └── transaction_test.go # Transaction handling
│   ├── fixtures/              # Test data fixtures
│   │   └── fixtures.go        # Fixture manager
│   ├── helpers/               # Test utilities
│   │   ├── auth.go            # Authentication helpers
│   │   ├── database.go        # Database helpers
│   │   └── http_client.go     # HTTP client helpers
│   ├── mocks/                 # External service mocks
│   │   ├── aws_mock.go        # AWS/LocalStack mock
│   │   ├── email_mock.go      # MailHog client
│   │   └── stripe_mock.go     # Stripe mock server
│   ├── config.go              # Test configuration
│   ├── setup_test.go          # Test suite setup
│   └── README.md              # This file
├── localstack/                # LocalStack initialization
│   └── init-aws.sh            # AWS resource setup
├── Makefile                   # Test commands
└── .env.test.example          # Environment template
```

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- Make (optional, for using Makefile)

## Quick Start

### 1. Start Test Environment

```bash
# Using Make
make docker-up

# Or using Docker Compose directly
docker compose -f docker-compose.test.yml up -d
```

### 2. Run Tests

```bash
# Run all integration tests
make test-integration

# Run specific test suites
make run-api-tests
make run-db-tests

# Run with coverage
make test-coverage
```

### 3. Stop Environment

```bash
make docker-down
```

## Test Environment

The test environment includes:

| Service     | Port  | Description       |
| ----------- | ----- | ----------------- |
| PostgreSQL  | 5433  | Test database     |
| Redis       | 6380  | Test cache        |
| Stripe Mock | 12111 | Stripe API mock   |
| LocalStack  | 4566  | AWS services mock |
| MailHog     | 8025  | Email testing     |

## Writing Tests

### Test Structure

All integration tests use the `//go:build integration` build tag:

```go
//go:build integration

package api

import (
    "testing"
    "github.com/javaknight1/servicepro/backend/tests/integration"
)

func TestExample(t *testing.T) {
    suite := integration.SetupTest(t)
    // Your test code
}
```

### Using Test Suite

```go
func TestCustomerAPI(t *testing.T) {
    suite := integration.SetupTest(t)

    // Create authenticated client
    client, user := suite.CreateAuthenticatedClient(t, "admin")

    // Create test data using fixtures
    customer := suite.CreateTestCustomer(t)

    // Make API requests
    resp := client.Get(t, "/api/v1/customers/" + customer.ID)

    // Assert results
    helpers.AssertOK(t, resp)
}
```

### Creating Fixtures

```go
// Create a customer
customer := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
    Email:     "test@example.com",
    FirstName: "John",
    LastName:  "Doe",
})

// Create a job
job := suite.Fixtures.CreateJob(t, &fixtures.Job{
    CustomerID: customer.ID,
    Title:      "Test Job",
})

// Create an invoice with line items
invoice := suite.Fixtures.CreateInvoiceWithLineItems(t, customer.ID)
```

### Using Mocks

```go
// Stripe mock
customer := suite.Stripe.CreateCustomer("test@example.com", "Test User")
pi := suite.Stripe.CreatePaymentIntent(10000, customer.ID)
charge := suite.Stripe.ConfirmPaymentIntent(pi.ID)

// Email verification
suite.MailHog.DeleteAllMessages(t)
// ... trigger email ...
email := suite.MailHog.AssertEmailSent(t, "user@example.com", 5*time.Second)
assert.Contains(t, email.Subject(), "Welcome")

// AWS mock
suite.AWS.UploadFile(t, "bucket", "key", []byte("content"), "text/plain")
content := suite.AWS.GetFile(t, "bucket", "key")
```

### HTTP Client Helpers

```go
// Simple requests
resp := client.Get(t, "/api/v1/customers")
resp := client.Post(t, "/api/v1/customers", data)
resp := client.Put(t, "/api/v1/customers/id", data)
resp := client.Delete(t, "/api/v1/customers/id")

// Complex requests with builder
resp := client.NewRequest("GET", "/api/v1/customers").
    WithQuery("page", "1").
    WithQuery("limit", "10").
    WithHeader("X-Custom", "value").
    Do(t)

// Response assertions
helpers.AssertOK(t, resp)
helpers.AssertCreated(t, resp)
helpers.AssertNotFound(t, resp)
helpers.AssertUnauthorized(t, resp)
helpers.AssertBadRequest(t, resp)

// Parse response
var result map[string]interface{}
resp.JSON(&result)
```

### Database Tests

```go
func TestTransactions(t *testing.T) {
    suite := integration.SetupTest(t)

    // Use transaction for test isolation
    suite.DB.WithTransaction(t, func(tx *gorm.DB) {
        // Test code - automatically rolled back
    })
}

func TestConcurrency(t *testing.T) {
    suite := integration.SetupTest(t)

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Concurrent operations
        }()
    }
    wg.Wait()
}
```

## Configuration

### Environment Variables

Create `.env.test` from the template:

```bash
cp .env.test.example .env.test
source .env.test
```

Key variables:

| Variable          | Default         | Description   |
| ----------------- | --------------- | ------------- |
| TEST_DATABASE_URL | localhost:5433  | Test database |
| REDIS_URL         | localhost:6380  | Test Redis    |
| STRIPE_API_BASE   | localhost:12111 | Stripe mock   |
| AWS_ENDPOINT_URL  | localhost:4566  | LocalStack    |
| MAILHOG_API       | localhost:8025  | MailHog API   |

### Test Configuration

Modify `config.go` to adjust:

- Timeouts
- Rate limits for testing
- Mock service URLs

## CI/CD Integration

### GitHub Actions

```yaml
test:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:16
      env:
        POSTGRES_DB: servicepro_test
        POSTGRES_USER: test_user
        POSTGRES_PASSWORD: test_password
    redis:
      image: redis:7
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - name: Run tests
      run: make ci-test
```

### Running in CI

```bash
make ci-test
```

## Troubleshooting

### Container Issues

```bash
# Check container status
docker compose -f docker-compose.test.yml ps

# View logs
docker compose -f docker-compose.test.yml logs postgres
docker compose -f docker-compose.test.yml logs redis

# Restart specific service
docker compose -f docker-compose.test.yml restart postgres
```

### Database Issues

```bash
# Connect to test database
docker exec -it servicepro-test-postgres psql -U test_user -d servicepro_test

# Reset database
docker compose -f docker-compose.test.yml down -v
docker compose -f docker-compose.test.yml up -d
```

### Test Failures

1. **Flaky tests**: Use `-count=1` to disable test caching
2. **Timeouts**: Increase `TEST_TIMEOUT` or specific test timeouts
3. **Port conflicts**: Check if ports 5433, 6380, etc. are available
