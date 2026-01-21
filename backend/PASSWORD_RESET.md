# Password Reset API Documentation

## Overview

The Password Reset API provides secure endpoints for users to reset their passwords when they forget them. The system uses cryptographically secure tokens stored in Redis with a 1-hour expiration, and sends reset links via AWS SES.

## Technology Stack

- **Framework**: Go 1.21 with Gin
- **Token Storage**: Redis (1-hour expiry)
- **Email Service**: AWS SES
- **Token Generation**: crypto/rand (base64 URL-safe encoding)
- **Rate Limiting**: Redis-based (3 attempts per 15 minutes)
- **Password Security**: bcrypt hashing

## Endpoints

### 1. Request Password Reset

**Endpoint**: `POST /api/v1/auth/reset-request`

**Description**: Initiates a password reset request by generating a secure token and sending a reset link via email.

**Request Headers**:

```
Content-Type: application/json
```

**Request Body**:

```json
{
  "email": "user@example.com"
}
```

**Validation Rules**:

- `email`: Required, valid email format

**Success Response** (200 OK):

```json
{
  "message": "If an account with that email exists, a password reset link has been sent."
}
```

**Note**: The response is intentionally generic to prevent email enumeration attacks. The same response is returned whether the email exists or not.

**Error Responses**:

- **400 Bad Request** - Invalid request format:

```json
{
  "error": "invalid_request",
  "message": "Invalid email format"
}
```

- **429 Too Many Requests** - Rate limit exceeded:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many password reset requests. Please try again in 14m30s"
}
```

**Security Features**:

- Email enumeration prevention (always returns success)
- Rate limiting: 3 requests per 15 minutes per email/IP
- Secure token generation using crypto/rand
- Tokens stored in Redis with automatic 1-hour expiration
- Async email sending to prevent blocking

**Example Request**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset-request \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

---

### 2. Reset Password

**Endpoint**: `POST /api/v1/auth/reset-password`

**Description**: Resets the user's password using a valid reset token.

**Request Headers**:

```
Content-Type: application/json
```

**Request Body**:

```json
{
  "token": "secure-reset-token-from-email",
  "newPassword": "NewStrongP@ssw0rd"
}
```

**Validation Rules**:

- `token`: Required
- `newPassword`: Required, minimum 8 characters, must contain:
  - At least one uppercase letter (A-Z)
  - At least one lowercase letter (a-z)
  - At least one number (0-9)
  - At least one special character (!@#$%^&\*()\_+-=[]{}|;:,.<>?)

**Success Response** (200 OK):

```json
{
  "message": "Password has been successfully reset. You can now log in with your new password."
}
```

**Error Responses**:

- **400 Bad Request** - Invalid request format:

```json
{
  "error": "invalid_request",
  "message": "Token and password are required"
}
```

- **400 Bad Request** - Weak password:

```json
{
  "error": "weak_password",
  "message": "Password must be at least 8 characters with uppercase, lowercase, number, and special character"
}
```

- **401 Unauthorized** - Invalid or expired token:

```json
{
  "error": "invalid_token",
  "message": "Invalid or expired reset token"
}
```

- **500 Internal Server Error** - Server error:

```json
{
  "error": "internal_error",
  "message": "An error occurred while processing your request"
}
```

**Security Features**:

- Token validation from Redis
- Strong password enforcement
- Automatic token deletion after use
- Failed login counter reset on successful password change
- Account unlock on successful password reset
- Confirmation email sent after successful reset

**Example Request**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token":"abc123xyz789",
    "newPassword":"NewStrongP@ssw0rd123"
  }'
```

---

## Email Templates

### Reset Request Email

**Subject**: Password Reset Request

**Content**:

- Personalized greeting
- Reset link with embedded token
- Token expiration notice (1 hour)
- Security warning if request wasn't initiated by user
- Styled HTML with responsive design

**Example**:

```
Hello,

We received a request to reset your password. Click the button below to reset your password:

[Reset Password Button] → http://localhost:5173/reset-password?token=abc123xyz789

This link will expire in 1 hour.

If you didn't request this, please ignore this email and your password will remain unchanged.
```

### Password Reset Confirmation Email

**Subject**: Password Successfully Reset

**Content**:

- Confirmation of successful password change
- Security notice
- Recommendation to review account activity
- Contact information for unexpected changes

**Example**:

```
Hello,

Your password has been successfully reset.

If you did not make this change, please contact our support team immediately.

For security, we recommend:
- Reviewing your recent account activity
- Ensuring your email account is secure
- Using a strong, unique password
```

---

## Security Considerations

### Token Security

- **Generation**: Tokens are generated using Go's `crypto/rand` package, providing cryptographically secure random values
- **Format**: Base64 URL-safe encoded (32 bytes = ~43 characters)
- **Storage**: Tokens are stored in Redis with the user's email as the value
- **Expiration**: Automatic 1-hour TTL in Redis
- **Single-use**: Tokens are deleted from Redis immediately after successful password reset

### Rate Limiting

- **Window**: 15 minutes
- **Limit**: 3 reset requests per email/IP
- **Storage**: Redis with automatic expiration
- **Identifier**: Uses email if provided in request, otherwise falls back to IP address

### Password Requirements

- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character from: `!@#$%^&*()_+-=[]{}|;:,.<>?`

### Email Enumeration Prevention

The system prevents attackers from determining which emails are registered by:

- Always returning the same success message regardless of email existence
- Not revealing whether the email exists in the database
- Logging attempts for non-existent emails server-side for monitoring

### Account Security Features

On successful password reset:

- Failed login attempt counter is reset to 0
- Account is automatically unlocked if it was locked
- User receives confirmation email

---

## Implementation Details

### Token Storage Schema (Redis)

```
Key: password_reset:<token>
Value: user_email@example.com
TTL: 3600 seconds (1 hour)
```

### Password Reset Flow

1. **Request Phase**:

   - User submits email address
   - System validates email format (Gin binding)
   - Service validates email format (RFC 5322)
   - System checks if user exists
   - If user exists:
     - Generate secure token (32 bytes)
     - Store token in Redis with 1-hour TTL
     - Send reset email asynchronously
   - Return generic success message

2. **Reset Phase**:
   - User receives email with reset link
   - User clicks link (frontend redirects with token)
   - User submits new password with token
   - System validates password strength (Gin: min 8, Service: full strength)
   - System retrieves email from Redis using token
   - If token invalid/expired: return error
   - System retrieves user by email
   - System hashes new password (bcrypt cost 12)
   - System updates password in database
   - System deletes used token from Redis
   - System resets failed login counters
   - System sends confirmation email asynchronously
   - Return success message

---

## Testing

### Unit Tests

Comprehensive test coverage includes:

**Token Generation Tests** (`pkg/auth/token_test.go`):

- Basic token generation
- Password reset token generation
- Token uniqueness (100 iterations)
- Different token lengths

**Password Reset Service Tests** (`internal/services/password_reset_service_test.go`):

- Request password reset with valid email
- Request password reset with non-existent email
- Request password reset with invalid email
- Database errors during request
- Reset password with valid token and strong password
- Reset password with invalid/expired token
- Reset password with weak passwords (5 variants)
- Reset password when user not found
- Password update failures
- Token expiry verification

**Password Reset Handler Tests** (`internal/api/handlers/password_reset_handler_test.go`):

- Request success
- Invalid JSON request
- Invalid email format (Gin validation)
- Service errors (returns success for security)
- Reset success
- Invalid JSON for reset
- Invalid/expired token
- Weak password (various validation scenarios)
- Internal server errors

### Running Tests

```bash
# Run all password reset tests
go test ./internal/services/... -v -run "TestRequestPasswordReset|TestResetPassword"
go test ./internal/api/handlers/... -v -run "TestRequestPasswordReset|TestResetPassword"
go test ./pkg/auth/... -v -run TestGenerateSecureToken

# Run all tests with coverage
go test ./... -cover
```

---

## Error Handling

### Service Layer Errors

- `ErrInvalidResetToken`: Token not found in Redis or expired
- `ErrWeakPassword`: Password doesn't meet strength requirements
- `ErrUserNotFoundForReset`: User associated with token no longer exists
- `auth.ErrInvalidEmailFormat`: Email format validation failed

### Handler Layer Error Mapping

| Service Error                | HTTP Status | Error Code     | User Message                   |
| ---------------------------- | ----------- | -------------- | ------------------------------ |
| `auth.ErrInvalidEmailFormat` | 400         | invalid_email  | Invalid email format           |
| `ErrInvalidResetToken`       | 401         | invalid_token  | Invalid or expired reset token |
| `ErrWeakPassword`            | 400         | weak_password  | Password requirements not met  |
| Password validation errors   | 400         | weak_password  | Specific requirement message   |
| Other errors                 | 500         | internal_error | Generic error message          |

---

## Configuration

### Required Environment Variables

```bash
# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# AWS SES Configuration
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
SES_FROM_EMAIL=noreply@servicepro.com

# Frontend Configuration
RESET_PASSWORD_URL=http://localhost:5173/reset-password

# Rate Limiting
RATE_LIMIT_WINDOW=15m
PASSWORD_RESET_RATE_LIMIT=3
```

### Development vs Production

- **Development**: Uses `MockEmailService` for testing (logs emails instead of sending)
- **Production**: Uses real AWS SES with configured credentials

---

## API Integration Example

### JavaScript/TypeScript (Frontend)

```typescript
// Request password reset
async function requestPasswordReset(email: string): Promise<void> {
  const response = await fetch(
    'http://localhost:8080/api/v1/auth/reset-request',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email }),
    }
  );

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }

  const data = await response.json();
  console.log(data.message);
}

// Reset password with token
async function resetPassword(
  token: string,
  newPassword: string
): Promise<void> {
  const response = await fetch(
    'http://localhost:8080/api/v1/auth/reset-password',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ token, newPassword }),
    }
  );

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }

  const data = await response.json();
  console.log(data.message);
}

// Usage
try {
  await requestPasswordReset('user@example.com');
  // User receives email and clicks link with token
  await resetPassword('token-from-email', 'NewStrongP@ssw0rd123');
} catch (error) {
  console.error('Password reset failed:', error.message);
}
```

---

## Monitoring and Logging

### Server-Side Logging

The system logs the following events:

- Password reset requests for non-existent emails (INFO level)
- Failed email sending attempts (ERROR level)
- Redis connection failures (ERROR level)
- Database errors (ERROR level)

### Metrics to Monitor

- Password reset request rate
- Token expiration rate (users not completing reset)
- Email delivery success rate
- Failed password reset attempts (invalid tokens)
- Rate limit violations

---

## Future Enhancements

Potential improvements for future versions:

1. **Multi-factor Authentication**: Require additional verification before allowing password reset
2. **Custom Token Length**: Configurable token length based on security requirements
3. **Email Templates**: Customizable HTML templates via configuration
4. **Password History**: Prevent reuse of recent passwords
5. **Notification on All Devices**: Alert user on all logged-in devices after password change
6. **Suspicious Activity Detection**: Flag reset requests from unusual locations/IPs
7. **Backup Recovery Methods**: SMS, security questions, or backup codes

---

## Support

For issues or questions:

- GitHub Issues: https://github.com/javaknight1/servicepro/issues
- Email: support@servicepro.com

---

## Changelog

### Version 1.0.0 (Current)

- Initial implementation
- Secure token generation using crypto/rand
- Redis-based token storage with 1-hour expiry
- AWS SES email integration
- Rate limiting (3 attempts / 15 minutes)
- Comprehensive test coverage
- Email enumeration prevention
- Password strength enforcement
