# Resend (Transactional Email)

## Overview

**Resend** is a modern email API built for developers, offering reliable transactional email delivery with excellent developer experience.

### Why We Use Resend

| Feature                 | Benefit                       |
| ----------------------- | ----------------------------- |
| **Developer-First**     | Simple API, great SDKs        |
| **Free Tier**           | 3,000 emails/month            |
| **High Deliverability** | Built by email experts        |
| **React Email**         | Build emails with React       |
| **Webhooks**            | Track delivery, opens, clicks |
| **No Cold Start**       | Instant delivery              |

### How ServicePro Uses Resend

- **Email Verification**: Account verification emails
- **Password Reset**: Password reset links
- **Invoice Notifications**: Invoice sent/paid notifications
- **Job Updates**: Assignment notifications
- **Customer Communications**: Custom email templates

---

## Free Tier Limits

| Resource    | Limit           |
| ----------- | --------------- |
| Emails      | 3,000/month     |
| Domains     | 1               |
| API Keys    | 2               |
| Daily Limit | 100/day         |
| Rate Limit  | 2 emails/second |

**When to Upgrade**: Around 100 MAU or when you need more than 100 emails/day.

**Pricing**: $20/month for 50K emails, then $0.40 per 1K emails.

---

## Setup

### Option A: Web Browser Setup (Recommended)

1. **Create Account**
   - Go to [resend.com](https://resend.com)
   - Click "Start for free"
   - Sign up with GitHub, Google, or email
   - Verify your email

2. **Get API Key**
   - After signup, you're taken to the dashboard
   - Your first API key is shown (or go to API Keys)
   - Copy and save the key (starts with `re_`)
   - **Important**: Save it now, it won't be shown again

3. **Verify Domain** (for production)
   - Click "Domains" in sidebar
   - Click "Add Domain"
   - Enter your domain: `servicepro.com`
   - Add the DNS records shown:
     - SPF record (TXT)
     - DKIM records (TXT)
     - Optional: DMARC record
   - Click "Verify"
   - Wait for verification (usually minutes)

4. **Test Email**
   - Use the dashboard "Send Test Email"
   - Or use the API with your key

### Option B: CLI Setup

Resend doesn't have an official CLI, but you can use cURL:

```bash
# Set your API key
export RESEND_API_KEY="re_xxxxx"

# List domains
curl -X GET "https://api.resend.com/domains" \
  -H "Authorization: Bearer $RESEND_API_KEY"

# Send test email
curl -X POST "https://api.resend.com/emails" \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "onboarding@resend.dev",
    "to": "your-email@example.com",
    "subject": "Test from ServicePro",
    "html": "<p>Hello from ServicePro!</p>"
  }'
```

---

## Configuration

### Environment Variables

```bash
# Resend API key
RESEND_API_KEY=re_xxxxx

# Email settings
EMAIL_FROM=noreply@servicepro.com
EMAIL_FROM_NAME=ServicePro
EMAIL_REPLY_TO=support@servicepro.com

# Feature flag (to switch between providers)
EMAIL_PROVIDER=resend
```

### Setting in Fly.io

```bash
# Set as secret (encrypted)
fly secrets set RESEND_API_KEY="re_xxxxx" --app servicepro-api
fly secrets set EMAIL_FROM="noreply@servicepro.com" --app servicepro-api
```

### Go Integration

```go
import "github.com/resend/resend-go/v2"

client := resend.NewClient(os.Getenv("RESEND_API_KEY"))

params := &resend.SendEmailRequest{
    From:    "ServicePro <noreply@servicepro.com>",
    To:      []string{"customer@example.com"},
    Subject: "Your Invoice is Ready",
    Html:    "<p>Your invoice #123 is ready...</p>",
}

sent, err := client.Emails.Send(params)
```

---

## Common Operations

### Send Email via API

```bash
# Basic email
curl -X POST "https://api.resend.com/emails" \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "ServicePro <noreply@servicepro.com>",
    "to": ["customer@example.com"],
    "subject": "Your Invoice #123",
    "html": "<h1>Invoice Ready</h1><p>Your invoice is attached.</p>"
  }'

# With attachment (base64 encoded)
curl -X POST "https://api.resend.com/emails" \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "ServicePro <noreply@servicepro.com>",
    "to": ["customer@example.com"],
    "subject": "Your Invoice #123",
    "html": "<p>Please find your invoice attached.</p>",
    "attachments": [
      {
        "filename": "invoice-123.pdf",
        "content": "base64-encoded-content-here"
      }
    ]
  }'

# With CC and BCC
curl -X POST "https://api.resend.com/emails" \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "from": "ServicePro <noreply@servicepro.com>",
    "to": ["customer@example.com"],
    "cc": ["manager@company.com"],
    "bcc": ["records@servicepro.com"],
    "subject": "Job Completed",
    "html": "<p>Your job has been completed.</p>"
  }'
```

### Check Email Status

```bash
# Get email by ID (returned when sending)
curl -X GET "https://api.resend.com/emails/{email_id}" \
  -H "Authorization: Bearer $RESEND_API_KEY"

# Response includes: status, last_event, created_at
```

### List Domains

```bash
curl -X GET "https://api.resend.com/domains" \
  -H "Authorization: Bearer $RESEND_API_KEY"
```

### Add Domain

```bash
curl -X POST "https://api.resend.com/domains" \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "servicepro.com"
  }'
```

### Verify Domain

```bash
curl -X POST "https://api.resend.com/domains/{domain_id}/verify" \
  -H "Authorization: Bearer $RESEND_API_KEY"
```

---

## Management

### View Usage (Web Dashboard)

1. Go to [resend.com/emails](https://resend.com/emails)
2. View:
   - **Emails sent**: Total count
   - **Delivery rate**: Success percentage
   - **Open rate**: Tracking (if enabled)
   - **Bounce rate**: Failed deliveries

### Email Logs

1. Go to Dashboard → Emails
2. Filter by:
   - Status (delivered, bounced, complained)
   - Date range
   - Recipient

### API Keys Management

1. Go to Dashboard → API Keys
2. Create/revoke keys
3. Set permissions (full access or sending only)

---

## Troubleshooting

### Email Not Delivered

**Symptom**: Email sent but not received

**Debugging Steps**:

1. **Check email status**

   ```bash
   curl -X GET "https://api.resend.com/emails/{email_id}" \
     -H "Authorization: Bearer $RESEND_API_KEY"
   ```

2. **Check spam folder**
   - First emails often land in spam

3. **Check domain verification**
   - Unverified domains have poor deliverability
   - Only `onboarding@resend.dev` works without verification

4. **Check DNS records**
   - SPF, DKIM must be configured correctly

### Domain Verification Failed

**Symptom**: Domain stuck in "pending" status

**Solutions**:

1. **Verify DNS records**

   ```bash
   # Check SPF
   dig TXT servicepro.com

   # Check DKIM
   dig TXT resend._domainkey.servicepro.com
   ```

2. **Wait for propagation**
   - DNS changes can take up to 48 hours
   - Usually complete in 15 minutes

3. **Re-verify**
   - Click "Verify" again in dashboard

### Rate Limited

**Symptom**: `429 Too Many Requests`

**Solutions**:

1. **Implement retry with backoff**

   ```go
   // Wait and retry after rate limit
   time.Sleep(time.Second)
   ```

2. **Queue emails**
   - Use a job queue for bulk sending
   - Process slowly (1-2 per second)

3. **Upgrade plan**
   - Higher tiers have higher rate limits

### Invalid API Key

**Symptom**: `401 Unauthorized`

**Solutions**:

1. **Verify key format**
   - Should start with `re_`
   - No extra spaces

2. **Check key permissions**
   - Ensure key has sending permission

3. **Generate new key**
   - Old key might be revoked

### Bounce/Complaint

**Symptom**: High bounce rate

**Solutions**:

1. **Check recipient addresses**
   - Validate email format
   - Remove invalid addresses

2. **Check domain reputation**
   - New domains need warmup
   - Start with low volume

3. **Review email content**
   - Avoid spam trigger words
   - Include unsubscribe link

---

## Email Templates

### Verification Email

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Verify Your Email</title>
  </head>
  <body style="font-family: Arial, sans-serif; line-height: 1.6;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
      <h1 style="color: #333;">Welcome to ServicePro!</h1>
      <p>Please verify your email address by clicking the button below:</p>
      <a
        href="{{verification_url}}"
        style="display: inline-block; background: #007bff; color: white;
              padding: 12px 24px; text-decoration: none; border-radius: 4px;"
      >
        Verify Email
      </a>
      <p style="color: #666; font-size: 14px; margin-top: 20px;">
        This link expires in 24 hours. If you didn't create an account, you can
        safely ignore this email.
      </p>
    </div>
  </body>
</html>
```

### Password Reset Email

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Reset Your Password</title>
  </head>
  <body style="font-family: Arial, sans-serif; line-height: 1.6;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
      <h1 style="color: #333;">Reset Your Password</h1>
      <p>
        We received a request to reset your password. Click the button below:
      </p>
      <a
        href="{{reset_url}}"
        style="display: inline-block; background: #dc3545; color: white;
              padding: 12px 24px; text-decoration: none; border-radius: 4px;"
      >
        Reset Password
      </a>
      <p style="color: #666; font-size: 14px; margin-top: 20px;">
        This link expires in 1 hour. If you didn't request this, please ignore
        this email or contact support.
      </p>
    </div>
  </body>
</html>
```

### Invoice Email

```html
<!DOCTYPE html>
<html>
  <head>
    <title>Invoice #{{invoice_number}}</title>
  </head>
  <body style="font-family: Arial, sans-serif; line-height: 1.6;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
      <h1 style="color: #333;">Invoice #{{invoice_number}}</h1>
      <p>Hi {{customer_name}},</p>
      <p>Your invoice is ready. Here are the details:</p>
      <table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
        <tr style="background: #f4f4f4;">
          <td style="padding: 10px; border: 1px solid #ddd;">
            <strong>Amount Due</strong>
          </td>
          <td style="padding: 10px; border: 1px solid #ddd;">{{amount}}</td>
        </tr>
        <tr>
          <td style="padding: 10px; border: 1px solid #ddd;">
            <strong>Due Date</strong>
          </td>
          <td style="padding: 10px; border: 1px solid #ddd;">{{due_date}}</td>
        </tr>
      </table>
      <a
        href="{{invoice_url}}"
        style="display: inline-block; background: #28a745; color: white;
              padding: 12px 24px; text-decoration: none; border-radius: 4px;"
      >
        View & Pay Invoice
      </a>
    </div>
  </body>
</html>
```

---

## Webhooks

Set up webhooks to track email events:

### Configure Webhook

1. Go to Dashboard → Webhooks
2. Add endpoint: `https://servicepro-api.fly.dev/webhooks/resend`
3. Select events:
   - `email.sent`
   - `email.delivered`
   - `email.bounced`
   - `email.complained`

### Webhook Payload

```json
{
  "type": "email.delivered",
  "created_at": "2024-01-15T10:00:00.000Z",
  "data": {
    "email_id": "49a3999c-0ce1-4ea6-ab68-afcd6dc2e794",
    "to": ["customer@example.com"],
    "from": "noreply@servicepro.com",
    "subject": "Your Invoice"
  }
}
```

### Verify Webhook Signature

```go
// Verify webhook signature
signature := r.Header.Get("svix-signature")
timestamp := r.Header.Get("svix-timestamp")
// Use Svix SDK to verify
```

---

## Security

### API Key Security

- Never commit API keys to git
- Use Fly.io secrets
- Rotate keys periodically

### Sender Authentication

1. **SPF Record**: Authorizes Resend to send on your behalf
2. **DKIM**: Signs emails cryptographically
3. **DMARC**: Policy for failed authentication

### Required DNS Records

```
# SPF (TXT record on root domain)
v=spf1 include:_spf.resend.com ~all

# DKIM (TXT record)
resend._domainkey.servicepro.com -> (provided by Resend)

# DMARC (TXT record, optional but recommended)
_dmarc.servicepro.com -> v=DMARC1; p=quarantine; rua=mailto:dmarc@servicepro.com
```

---

## Local Development

For local development, use one of these approaches:

### Option 1: Use Resend (with test domain)

```bash
# Use the default test domain
EMAIL_FROM=onboarding@resend.dev
```

### Option 2: Use Mailpit (local SMTP)

The docker-compose already includes Mailpit:

```bash
# In docker-compose.yml, Mailpit is on port 1025 (SMTP) and 8025 (Web UI)
SMTP_HOST=localhost
SMTP_PORT=1025
EMAIL_PROVIDER=smtp

# View emails at http://localhost:8025
```

---

## Useful Links

- [Resend Documentation](https://resend.com/docs)
- [Resend API Reference](https://resend.com/docs/api-reference)
- [React Email](https://react.email) (build emails with React)
- [Resend Go SDK](https://github.com/resend/resend-go)
- [Email Deliverability Guide](https://resend.com/docs/knowledge-base/deliverability)
- [Resend Status](https://status.resend.com)
