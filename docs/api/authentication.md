# Authentication API

User authentication endpoints including registration, login, password reset, and email verification.

## Overview

The Authentication API provides:

- User registration with email validation
- JWT-based authentication
- Password reset via email
- Email verification
- Strong password enforcement

## Endpoints

| Method | Endpoint               | Description               |
| ------ | ---------------------- | ------------------------- |
| POST   | `/auth/register`       | Register new account      |
| POST   | `/auth/login`          | Login and get JWT token   |
| POST   | `/auth/logout`         | Logout (invalidate token) |
| POST   | `/auth/reset-request`  | Request password reset    |
| POST   | `/auth/reset-password` | Reset password with token |
| POST   | `/auth/verify-email`   | Verify email address      |
| POST   | `/auth/refresh`        | Refresh JWT token         |

## Register

Create a new user account.

```http
POST /api/v1/auth/register
```

### Request Body

```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

### Password Requirements

Passwords must meet ALL criteria:

- Minimum 8 characters
- At least one uppercase letter (A-Z)
- At least one lowercase letter (a-z)
- At least one number (0-9)
- At least one special character (!@#$%^&\*()\_+-=[]{}|;:,.<>?)

### Response (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "createdAt": "2025-01-23T10:30:00Z"
}
```

### Error Responses

**Invalid Email (400):**

```json
{
  "error": "invalid_email",
  "message": "Invalid email format"
}
```

**Weak Password (400):**

```json
{
  "error": "weak_password",
  "message": "Password must be at least 8 characters with uppercase, lowercase, number, and special character"
}
```

**Email Exists (409):**

```json
{
  "error": "email_exists",
  "message": "An account with this email already exists"
}
```

## Login

Authenticate and receive a JWT token.

```http
POST /api/v1/auth/login
```

### Request Body

```json
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

### Response (200 OK)

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "role": "user"
  },
  "expires_at": "2025-01-24T10:30:00Z"
}
```

### Error Responses

**Invalid Credentials (401):**

```json
{
  "error": "invalid_credentials",
  "message": "Invalid email or password"
}
```

**Account Locked (423):**

```json
{
  "error": "account_locked",
  "message": "Account is locked. Try again in 15 minutes"
}
```

**Email Not Verified (403):**

```json
{
  "error": "email_not_verified",
  "message": "Please verify your email before logging in"
}
```

## Logout

Invalidate the current JWT token.

```http
POST /api/v1/auth/logout
Authorization: Bearer <token>
```

### Response (200 OK)

```json
{
  "message": "Successfully logged out"
}
```

## Request Password Reset

Initiate a password reset request.

```http
POST /api/v1/auth/reset-request
```

### Request Body

```json
{
  "email": "user@example.com"
}
```

### Response (200 OK)

```json
{
  "message": "If an account with that email exists, a password reset link has been sent."
}
```

**Note:** The response is intentionally generic to prevent email enumeration attacks.

### Security Features

- Rate limited: 3 requests per 15 minutes per email/IP
- Token expires in 1 hour
- Tokens are single-use

## Reset Password

Reset password using the token from email.

```http
POST /api/v1/auth/reset-password
```

### Request Body

```json
{
  "token": "secure-reset-token-from-email",
  "newPassword": "NewStrongP@ssw0rd"
}
```

### Response (200 OK)

```json
{
  "message": "Password has been successfully reset. You can now log in with your new password."
}
```

### Error Responses

**Invalid Token (401):**

```json
{
  "error": "invalid_token",
  "message": "Invalid or expired reset token"
}
```

**Weak Password (400):**

```json
{
  "error": "weak_password",
  "message": "Password must be at least 8 characters with uppercase, lowercase, number, and special character"
}
```

## Verify Email

Verify email address using the token from verification email.

```http
POST /api/v1/auth/verify-email
```

### Request Body

```json
{
  "token": "verification-token-from-email"
}
```

### Response (200 OK)

```json
{
  "message": "Email verified successfully"
}
```

## Using the JWT Token

Include the token in the Authorization header for authenticated requests:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Token Format

The JWT contains:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "exp": 1234567890
}
```

### Token Expiration

- Access tokens expire in 1 hour (configurable)
- Use the refresh endpoint to get a new token

## Email Validation

Emails are validated using RFC 5322 compliant regex:

**Valid Examples:**

- `user@example.com`
- `john.doe@company.co.uk`
- `test+tag@domain.com`

**Invalid Examples:**

- `notanemail`
- `@example.com`
- `user@.com`

## Rate Limiting

| Endpoint       | Limit       | Window                   |
| -------------- | ----------- | ------------------------ |
| Register       | 5 requests  | per minute per IP        |
| Login          | 10 requests | per minute per IP        |
| Password Reset | 3 requests  | per 15 minutes per email |

## Security Best Practices

### Password Security

- Passwords are hashed using bcrypt (cost factor 12)
- Password hash is never returned in responses
- Failed login attempts are tracked
- Account locks after 5 failed attempts

### Token Security

- Use HTTPS in production
- Tokens are stateless JWT
- Short expiration times
- Secure storage in frontend

### Account Protection

- Email verification required for login
- Account lockout after failed attempts
- Password reset confirmation email
- Rate limiting on all auth endpoints

## Examples

### Register New User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

### Request Password Reset

```bash
curl -X POST http://localhost:8080/api/v1/auth/reset-request \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

### Use Token

```bash
curl http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Frontend Integration

### JavaScript Example

```javascript
// Register
const register = async (email, password) => {
  const response = await fetch('/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  return response.json();
};

// Login
const login = async (email, password) => {
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  const data = await response.json();
  localStorage.setItem('token', data.token);
  return data;
};

// Authenticated request
const fetchCustomers = async () => {
  const token = localStorage.getItem('token');
  const response = await fetch('/api/v1/customers', {
    headers: { Authorization: `Bearer ${token}` },
  });
  return response.json();
};
```

## Configuration

### Environment Variables

```bash
# JWT Configuration
JWT_SECRET=your-super-secret-key
JWT_EXPIRY=3600  # seconds (1 hour)

# AWS Configuration (shared credentials)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYxxxxxxxxxx

# AWS SES Email Configuration
AWS_SES_FROM_EMAIL=noreply@servicepro.com
AWS_SES_FROM_NAME=ServicePro

# Password Reset
RESET_PASSWORD_URL=http://localhost:3000/reset-password

# Redis (for token storage)
REDIS_URL=redis://localhost:6379/0
```

## Database Schema

### users table

| Column             | Type         | Description               |
| ------------------ | ------------ | ------------------------- |
| id                 | UUID         | Primary key               |
| email              | VARCHAR(255) | Unique email address      |
| password_hash      | VARCHAR(255) | Bcrypt hash               |
| role               | VARCHAR(50)  | User role (user, admin)   |
| email_verified     | BOOLEAN      | Email verification status |
| failed_login_count | INTEGER      | Failed login attempts     |
| locked_until       | TIMESTAMP    | Account lock expiry       |
| created_at         | TIMESTAMP    | Account creation time     |
| updated_at         | TIMESTAMP    | Last update time          |
