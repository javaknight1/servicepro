# User Registration API

Complete implementation of the user registration endpoint with email validation, password strength requirements, AWS SES integration, and comprehensive testing.

## Features

- Email format validation
- Strong password requirements enforcement
- Bcrypt password hashing
- UUID-based user IDs
- Role-based access control (default: user)
- AWS SES email notifications
- GORM ORM for database operations
- Comprehensive unit tests

## API Endpoint

### POST /api/v1/auth/register

Register a new user account.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

**Success Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "createdAt": "2025-01-23T10:30:00Z"
}
```

**Error Responses:**

**400 Bad Request - Invalid Email:**

```json
{
  "error": "invalid_email",
  "message": "Invalid email format"
}
```

**400 Bad Request - Weak Password:**

```json
{
  "error": "weak_password",
  "message": "Password must be at least 8 characters with uppercase, lowercase, number, and special character"
}
```

**409 Conflict - Email Exists:**

```json
{
  "error": "email_exists",
  "message": "An account with this email already exists"
}
```

**500 Internal Server Error:**

```json
{
  "error": "internal_error",
  "message": "An error occurred while processing your request"
}
```

## Password Requirements

Passwords must meet ALL of the following criteria:

1. **Minimum Length**: 8 characters
2. **Uppercase Letter**: At least one (A-Z)
3. **Lowercase Letter**: At least one (a-z)
4. **Number**: At least one digit (0-9)
5. **Special Character**: At least one (!@#$%^&\*()\_+-=[]{}|;:,.<>?)

**Valid Examples:**

- `SecurePass123!`
- `MyP@ssw0rd`
- `Test#1234Abc`

**Invalid Examples:**

- `short1!` (too short)
- `nouppercasehere123!` (no uppercase)
- `NOLOWERCASE123!` (no lowercase)
- `NoNumbers!` (no number)
- `NoSpecial123` (no special character)

## Email Validation

Emails are validated using RFC 5322 compliant regex:

**Valid Examples:**

- `user@example.com`
- `john.doe@company.co.uk`
- `test+tag@domain.com`
- `user_name@sub.domain.com`

**Invalid Examples:**

- `notanemail`
- `@example.com`
- `user@`
- `user@.com`
- `user@example` (no TLD)

## Database Schema

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    last_failed_login_at TIMESTAMP,
    locked_until TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

## AWS SES Email Integration

Upon successful registration, a welcome email is sent asynchronously to the user.

**Email Template:**

- **Subject**: Welcome to ServicePro!
- **Content**: HTML-formatted welcome message

**Configuration:**

```env
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
```

**Development Mode:**

- Uses `MockEmailService` that logs emails instead of sending
- Set `ENV=development` in `.env` to use mock service

**Production Mode:**

- Uses real AWS SES service
- Set `ENV=production` in `.env`

## Implementation Details

### Components

**1. Password Validator** (`pkg/auth/validator.go`)

- Validates password strength
- Validates email format
- Returns specific error messages

**2. User Repository** (`internal/repository/user_repository_gorm.go`)

- GORM-based database operations
- UUID support
- Email uniqueness check

**3. Email Service** (`pkg/email/ses.go`)

- AWS SES integration
- Welcome email template
- Mock service for development

**4. Registration Service** (`internal/services/registration_service.go`)

- Business logic orchestration
- Email/password validation
- Duplicate email check
- Password hashing
- Async email sending

**5. Registration Handler** (`internal/api/handlers/registration_handler.go`)

- HTTP request handling
- Input validation
- Error response mapping

### Security Features

**Password Hashing:**

- Bcrypt algorithm
- Cost factor: 12
- Salted automatically

**Data Protection:**

- Password hash never returned in responses
- Sensitive fields marked with `json:"-"`
- SQL injection prevention via GORM

**UUID vs Sequential IDs:**

- UUIDs prevent enumeration attacks
- No predictable user ID patterns

## Setup Instructions

### 1. Install Dependencies

```bash
cd backend
go mod tidy
```

### 2. Run Database Migration

```bash
psql -U postgres -d servicepro -f migrations/001_create_users_table.sql
psql -U postgres -d servicepro -f migrations/002_update_users_table_uuid.sql
```

### 3. Configure Environment

```bash
cp .env.example .env
# Edit .env with your settings
```

**Required Variables:**

```env
DATABASE_URL=postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable
AWS_REGION=us-east-1
ENV=development  # or production
```

### 4. Start Server

```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:8080`

## Testing

### Run All Tests

```bash
go test ./... -v
```

### Run Specific Test Suites

**Password Validation Tests:**

```bash
go test ./pkg/auth/... -v -run TestValidate
```

**Registration Service Tests:**

```bash
go test ./internal/services/... -v -run TestRegister
```

**Registration Handler Tests:**

```bash
go test ./internal/api/handlers/... -v -run TestRegisterHandler
```

### Test Coverage

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Scenarios Covered

**Password Validation:**

- ✓ Valid passwords
- ✓ Too short
- ✓ Missing uppercase
- ✓ Missing lowercase
- ✓ Missing number
- ✓ Missing special character

**Email Validation:**

- ✓ Valid email formats
- ✓ Invalid email formats

**Registration Service:**

- ✓ Successful registration
- ✓ Invalid email format
- ✓ Weak password
- ✓ Duplicate email
- ✓ All password validation errors

**Registration Handler:**

- ✓ Successful registration (201)
- ✓ Invalid request body (400)
- ✓ Invalid email (400)
- ✓ Weak password (400)
- ✓ Email already exists (409)
- ✓ Internal server error (500)

## Usage Examples

### cURL

```bash
# Successful registration
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!"
  }'

# Response:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "email": "newuser@example.com",
#   "role": "user",
#   "createdAt": "2025-01-23T10:30:00Z"
# }
```

### JavaScript (Fetch)

```javascript
const response = await fetch('http://localhost:8080/api/v1/auth/register', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    email: 'newuser@example.com',
    password: 'SecurePass123!',
  }),
});

const data = await response.json();

if (response.ok) {
  console.log('Registration successful:', data);
} else {
  console.error('Registration failed:', data.error);
}
```

### Python (requests)

```python
import requests

response = requests.post(
    'http://localhost:8080/api/v1/auth/register',
    json={
        'email': 'newuser@example.com',
        'password': 'SecurePass123!'
    }
)

if response.status_code == 201:
    data = response.json()
    print(f"User created: {data['id']}")
else:
    error = response.json()
    print(f"Error: {error['error']}")
```

## Architecture

```
Registration Flow:
┌─────────┐    ┌─────────────┐    ┌──────────────────┐    ┌──────────┐
│ Client  │───▶│   Handler   │───▶│ Registration Svc │───▶│  Repo    │
└─────────┘    └─────────────┘    └──────────────────┘    └──────────┘
                      │                     │                     │
                      │                     ▼                     ▼
                      │             ┌──────────────┐      ┌──────────┐
                      │             │  Validator   │      │ Database │
                      │             └──────────────┘      └──────────┘
                      │                     │
                      │                     ▼
                      │             ┌──────────────┐
                      │             │ Email Service│
                      │             └──────────────┘
                      │                     │
                      ▼                     ▼
                ┌──────────┐          ┌─────────┐
                │  Client  │          │ AWS SES │
                └──────────┘          └─────────┘
```

## File Structure

```
backend/
├── pkg/
│   ├── auth/
│   │   ├── validator.go           # Password & email validation
│   │   ├── validator_test.go      # Validation tests
│   │   ├── password.go            # Bcrypt hashing
│   │   └── jwt.go                 # JWT utilities (updated for UUID)
│   ├── email/
│   │   ├── ses.go                 # AWS SES implementation
│   │   └── mock.go                # Mock email service
│   └── database/
│       └── gorm.go                # GORM database connection
├── internal/
│   ├── models/
│   │   └── user.go                # User model with UUID & role
│   ├── repository/
│   │   ├── interface.go           # Repository interfaces
│   │   └── user_repository_gorm.go # GORM repository implementation
│   ├── services/
│   │   ├── registration_service.go      # Registration logic
│   │   ├── registration_service_test.go # Service tests
│   │   └── registration_interface.go    # Service interface
│   └── api/
│       ├── handlers/
│       │   ├── registration_handler.go       # HTTP handler
│       │   └── registration_handler_test.go  # Handler tests
│       └── routes/
│           └── routes_gorm.go            # Route setup with GORM
└── migrations/
    ├── 001_create_users_table.sql
    └── 002_update_users_table_uuid.sql
```

## Troubleshooting

**Issue: Email already exists**

- Check if user already registered
- Use different email address

**Issue: Weak password error**

- Ensure password meets all requirements
- Check for at least one of each: uppercase, lowercase, number, special char

**Issue: AWS SES errors in production**

- Verify AWS credentials
- Check AWS region configuration
- Ensure SES is verified/out of sandbox mode

**Issue: Database connection failed**

- Verify DATABASE_URL in .env
- Ensure PostgreSQL is running
- Check database exists

## License

MIT
