# Email Verification API

This document describes the email verification system for ServicePro, which ensures users verify their email addresses during registration.

## Overview

The email verification system:

- Sends verification emails automatically upon user registration
- Stores verification tokens in Redis with 24-hour expiry
- Tracks verification status in PostgreSQL
- Sends reminder emails after 24 hours
- Handles AWS SES bounce notifications
- Provides endpoints for verification and resending emails

## Table of Contents

- [Architecture](#architecture)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Error Codes](#error-codes)
- [Email Templates](#email-templates)
- [AWS SES Integration](#aws-ses-integration)
- [Testing](#testing)

## Architecture

### Components

1. **Email Verification Service** (`internal/services/email_verification_service.go`)

   - Generates secure verification tokens
   - Stores tokens in Redis with 24-hour TTL
   - Sends verification emails via AWS SES
   - Manages verification status in database

2. **Email Verification Handler** (`internal/api/handlers/email_verification_handler.go`)

   - Handles HTTP requests for verification endpoints
   - Validates request payloads
   - Returns appropriate HTTP status codes

3. **User Repository** (`internal/repository/user_repository_gorm.go`)

   - Provides database operations for email verification
   - Tracks `email_verified` and `verification_sent_at` fields

4. **Email Service** (`pkg/email/ses.go`)
   - Sends verification emails via AWS SES
   - Provides HTML email templates
   - Handles different email types (initial, reminder, success)

### Token Storage

- **Storage**: Redis
- **Key Format**: `email_verification:<token>`
- **Value**: User email address
- **Expiry**: 24 hours
- **Token Length**: 32 bytes (base64 URL-safe)

## Database Schema

### Users Table Additions

```sql
ALTER TABLE users
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN verification_sent_at TIMESTAMP;

CREATE INDEX idx_users_email_verified ON users(email_verified);
CREATE INDEX idx_users_verification_sent_at ON users(verification_sent_at)
WHERE email_verified = FALSE;
```

### Fields

- **email_verified**: `BOOLEAN` - Indicates if the user's email has been verified
- **verification_sent_at**: `TIMESTAMP` - Timestamp when verification email was last sent

## API Endpoints

### 1. Verify Email

Verifies a user's email address using a verification token.

**Endpoint**: `POST /api/v1/auth/verify`

**Request Body**:

```json
{
  "token": "string (required)"
}
```

**Success Response** (200 OK):

```json
{
  "message": "Your email has been successfully verified. You can now access all features."
}
```

**Error Responses**:

- **400 Bad Request** - Invalid request format

```json
{
  "error": "invalid_request",
  "message": "Verification token is required"
}
```

- **401 Unauthorized** - Invalid or expired token

```json
{
  "error": "invalid_token",
  "message": "Invalid or expired verification token"
}
```

- **500 Internal Server Error** - Server error

```json
{
  "error": "internal_error",
  "message": "An error occurred while verifying your email"
}
```

**Behavior**:

- Validates token against Redis store
- Marks user email as verified in database
- Deletes token from Redis
- Sends success confirmation email
- **Idempotent**: Re-verifying an already verified email succeeds without error

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"token":"AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"}'
```

### 2. Resend Verification Email

Resends the verification email to a user.

**Endpoint**: `POST /api/v1/auth/resend-verification`

**Request Body**:

```json
{
  "email": "string (required, valid email format)"
}
```

**Success Response** (200 OK):

```json
{
  "message": "If an account with that email exists and is unverified, a verification email has been sent."
}
```

**Error Response**:

- **400 Bad Request** - Invalid email or already verified

```json
{
  "error": "already_verified",
  "message": "Email address is already verified"
}
```

**Behavior**:

- Validates email format
- Generates new verification token
- Stores token in Redis with 24-hour expiry
- Sends verification email
- **Security**: Returns generic success message even if email doesn't exist (prevents email enumeration)

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/resend-verification \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

### 3. AWS SES Bounce Webhook

Processes AWS SES bounce notifications.

**Endpoint**: `POST /api/v1/webhooks/ses-bounce`

**Request Body**: AWS SES Bounce Notification Format

```json
{
  "notificationType": "Bounce",
  "bounce": {
    "bounceType": "Permanent",
    "bounceSubType": "General",
    "bouncedRecipients": [
      {
        "emailAddress": "bounce@example.com",
        "action": "failed",
        "status": "5.1.1",
        "diagnosticCode": "smtp; 550 5.1.1 user unknown"
      }
    ],
    "timestamp": "2025-01-01T00:00:00.000Z",
    "feedbackId": "00000000-0000-0000-0000-000000000000"
  }
}
```

**Success Response** (200 OK):

```
OK
```

**Behavior**:

- Logs bounced email addresses
- Clears `verification_sent_at` to prevent reminder emails
- **Always returns 200 OK** (AWS SES requires this)

### 4. AWS SES General Notification Webhook

Handles AWS SNS subscription confirmations and general SES notifications.

**Endpoint**: `POST /api/v1/webhooks/ses-notification`

**Request Body**: AWS SNS/SES Notification Format

**Success Response** (200 OK):

```
OK
```

**Behavior**:

- Handles SNS subscription confirmation
- Routes bounce notifications to bounce handler
- Logs subscription URLs for manual/automated confirmation

## Error Codes

| Code               | HTTP Status | Description                                       |
| ------------------ | ----------- | ------------------------------------------------- |
| `invalid_request`  | 400         | Malformed request body or missing required fields |
| `invalid_token`    | 401         | Verification token is invalid or has expired      |
| `already_verified` | 400         | Email address has already been verified           |
| `internal_error`   | 500         | Server error during verification process          |

## Email Templates

### 1. Verification Email

Sent immediately after registration.

**Subject**: "Verify Your Email Address - ServicePro"

**Contains**:

- Verification link with token
- 24-hour expiry warning
- Support contact information

**Template**: `pkg/email/ses.go:SendEmailVerificationEmail()`

### 2. Reminder Email

Sent 24 hours after registration if email is still unverified.

**Subject**: "Reminder: Verify Your Email Address - ServicePro"

**Contains**:

- Reminder message
- New verification link with token
- Benefits of verification
- 24-hour expiry warning

**Template**: `pkg/email/ses.go:SendEmailVerificationReminderEmail()`

### 3. Success Confirmation Email

Sent after successful verification.

**Subject**: "Email Verification Successful - ServicePro"

**Contains**:

- Confirmation of successful verification
- Available features
- Next steps

**Template**: `pkg/email/ses.go:SendEmailVerificationSuccessEmail()`

## AWS SES Integration

### Setup

1. **Configure AWS SES**:

   ```bash
   # Set environment variables
   export AWS_REGION=us-east-1
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

2. **Verify Sender Email**:

   - Verify `noreply@servicepro.com` in AWS SES console
   - Move out of SES sandbox for production use

3. **Configure SNS Topics**:

   - Create SNS topic for bounce notifications
   - Subscribe webhook endpoint: `https://your-domain.com/api/v1/webhooks/ses-bounce`
   - Confirm subscription via webhook response

4. **Configure SES Notifications**:
   - Set up Configuration Set in SES
   - Enable bounce and complaint tracking
   - Route notifications to SNS topic

### Bounce Handling

The system handles three types of bounces:

1. **Hard Bounces** (Permanent):

   - Invalid email addresses
   - Non-existent domains
   - **Action**: Clears `verification_sent_at`, logs event

2. **Soft Bounces** (Temporary):

   - Mailbox full
   - Temporary server issues
   - **Action**: Logs event, allows retry

3. **Complaint**:
   - User marked email as spam
   - **Action**: Logs event, may require manual review

## Reminder System

### Scheduled Job

Run the reminder job as a cron task or scheduled worker:

```go
// Example cron job (every hour)
func runVerificationReminders() {
    service := services.NewEmailVerificationService(
        userRepo,
        emailService,
        redisClient,
        verificationURL,
    )

    if err := service.SendReminderEmails(); err != nil {
        log.Printf("Failed to send verification reminders: %v", err)
    }
}
```

### Reminder Logic

- Queries users where:
  - `email_verified = FALSE`
  - `verification_sent_at < NOW() - 24 hours`
- Generates new verification tokens
- Sends reminder emails
- Updates `verification_sent_at`

## Testing

### Unit Tests

Run all email verification tests:

```bash
go test ./internal/services/... -v -run "TestSendVerificationEmail|TestVerifyEmail|TestResendVerificationEmail|TestSendReminderEmails|TestHandleBounce"
```

### Test Coverage

The test suite covers:

✓ Send verification email - success
✓ Send verification email - user not found
✓ Send verification email - already verified
✓ Verify email - success
✓ Verify email - invalid token
✓ Verify email - idempotent (already verified)
✓ Resend verification - success
✓ Resend verification - already verified
✓ Resend verification - user not found
✓ Send reminder emails - success
✓ Send reminder emails - no users
✓ Handle bounce - success
✓ Handle bounce - user not found

### Manual Testing

1. **Test Registration Flow**:

```bash
# Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123!"
  }'

# Check Redis for token
redis-cli KEYS "email_verification:*"

# Verify email (use token from email)
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"token":"<token-from-email>"}'
```

2. **Test Resend Flow**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/resend-verification \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'
```

3. **Test Webhook** (use AWS SNS):

```bash
# Simulate bounce notification
curl -X POST http://localhost:8080/api/v1/webhooks/ses-bounce \
  -H "Content-Type: application/json" \
  -d @test-bounce-notification.json
```

## Security Considerations

1. **Token Security**:

   - Tokens generated using `crypto/rand`
   - 32-byte length (256-bit security)
   - Base64 URL-safe encoding
   - One-time use (deleted after verification)

2. **Email Enumeration Prevention**:

   - Resend endpoint returns generic success message
   - Doesn't reveal if email exists in system

3. **Rate Limiting**:

   - Apply rate limiting to verification endpoints
   - Prevent abuse of resend functionality

4. **Token Expiry**:
   - 24-hour automatic expiry via Redis TTL
   - Expired tokens cannot be used

## Configuration

### Environment Variables

```bash
# Frontend URLs
VERIFICATION_URL="http://localhost:5173/verify-email"

# Redis Configuration
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""
REDIS_DB=0

# AWS SES Configuration
AWS_REGION="us-east-1"
AWS_ACCESS_KEY_ID="your-access-key"
AWS_SECRET_ACCESS_KEY="your-secret-key"
SES_FROM_EMAIL="noreply@servicepro.com"

# Server Configuration
SERVER_ENV="development" # or "production"
```

### Configuration in Code

```go
// routes_gorm.go
verificationURL := os.Getenv("VERIFICATION_URL")
if verificationURL == "" {
    verificationURL = "http://localhost:5173/verify-email"
}

emailVerificationService := services.NewEmailVerificationService(
    userRepo,
    emailService,
    redisClient,
    verificationURL,
)
```

## Troubleshooting

### Common Issues

1. **Verification email not received**:

   - Check spam folder
   - Verify AWS SES sending limits
   - Check email service logs
   - Verify sender email is verified in SES

2. **Token expired**:

   - Tokens expire after 24 hours
   - Use resend endpoint to get new token
   - Check Redis TTL configuration

3. **Bounce handling not working**:

   - Verify SNS subscription is confirmed
   - Check webhook endpoint is publicly accessible
   - Review AWS SES notification configuration
   - Check application logs for webhook errors

4. **Reminder emails not sent**:
   - Verify cron job is running
   - Check database query for unverified users
   - Review email service logs
   - Ensure `verification_sent_at` is properly set

## Future Enhancements

Potential improvements to consider:

1. **Email Validation Service Integration**:

   - Integrate with email validation API
   - Prevent disposable email addresses
   - Check email deliverability before registration

2. **Multi-channel Verification**:

   - SMS verification as alternative
   - Social login verification

3. **Advanced Bounce Handling**:

   - Add `email_bounced` flag to users table
   - Implement automatic account suspension
   - Require manual verification for bounced emails

4. **Analytics Dashboard**:

   - Track verification rates
   - Monitor bounce rates
   - Analyze reminder email effectiveness

5. **Customizable Templates**:
   - Allow template customization via admin panel
   - Support multiple languages
   - Dynamic branding

## Support

For questions or issues:

- File an issue in the project repository
- Contact the development team
- Review application logs in `logs/` directory
