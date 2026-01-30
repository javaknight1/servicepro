# Cloudflare R2 (Object Storage)

## Overview

**Cloudflare R2** is an S3-compatible object storage service with zero egress fees, making it significantly cheaper than AWS S3 for high-traffic applications.

### Why We Use R2

| Feature                | Benefit                                       |
| ---------------------- | --------------------------------------------- |
| **Zero Egress Fees**   | No cost to serve files (S3 charges ~$0.09/GB) |
| **S3 Compatible**      | Works with existing S3 SDKs                   |
| **Free Tier**          | 10GB storage, 10M reads, 1M writes/month      |
| **Global CDN**         | Cloudflare's edge network                     |
| **No Minimum Storage** | Pay only for what you use                     |

### How ServicePro Uses R2

- **File Uploads**: Customer documents, attachments
- **Invoice PDFs**: Generated invoice/quote PDFs
- **Profile Images**: User and customer avatars
- **Import Files**: CSV/Excel import staging

---

## Free Tier Limits

| Resource           | Limit                             |
| ------------------ | --------------------------------- |
| Storage            | 10 GB                             |
| Class A Operations | 1 million/month (PUT, POST, LIST) |
| Class B Operations | 10 million/month (GET, HEAD)      |
| Egress             | Always free                       |

**When to Upgrade**: Around 500 MAU or when you exceed 10GB storage.

---

## Setup

### Option A: Web Browser Setup (Recommended)

1. **Create Cloudflare Account**
   - Go to [cloudflare.com](https://cloudflare.com)
   - Sign up with email
   - Verify your email

2. **Enable R2**
   - Go to Cloudflare Dashboard
   - Click "R2" in left sidebar
   - Click "Purchase R2" (it's free to start)
   - Add payment method (required but won't be charged)

3. **Create Bucket**
   - Click "Create bucket"
   - Bucket name: `servicepro-uploads`
   - Location: `Automatic` or `North America East`
   - Click "Create bucket"

4. **Create API Token**
   - Click "Manage R2 API Tokens"
   - Click "Create API Token"
   - Permissions: `Object Read & Write`
   - Specify bucket: `servicepro-uploads`
   - TTL: `Forever` (or set expiration)
   - Click "Create API Token"
   - **Save these immediately** (shown only once):
     - Access Key ID
     - Secret Access Key

5. **Get Endpoint URL**
   - Go to bucket settings
   - Find "S3 API" endpoint:
     ```
     https://<account-id>.r2.cloudflarestorage.com
     ```

### Option B: CLI Setup (Wrangler)

Cloudflare's CLI is called Wrangler:

```bash
# Install Wrangler
npm install -g wrangler

# Or with Homebrew
brew install cloudflare/cloudflare/wrangler

# Login (opens browser)
wrangler login

# Create bucket
wrangler r2 bucket create servicepro-uploads

# List buckets
wrangler r2 bucket list

# Get bucket info
wrangler r2 bucket info servicepro-uploads
```

**Note**: API tokens must still be created via web dashboard.

---

## Configuration

### Environment Variables

```bash
# R2 uses S3-compatible API
S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
S3_BUCKET_NAME=servicepro-uploads
AWS_ACCESS_KEY_ID=<your-r2-access-key>
AWS_SECRET_ACCESS_KEY=<your-r2-secret-key>
AWS_REGION=auto

# Important: R2 requires path-style URLs
S3_FORCE_PATH_STYLE=true

# Public URL for serving files (optional)
# Set up via Cloudflare dashboard or custom domain
S3_PUBLIC_URL=https://files.servicepro.com
```

### Setting in Fly.io

```bash
# Set as secrets (encrypted)
fly secrets set AWS_ACCESS_KEY_ID="your-r2-access-key" --app servicepro-api
fly secrets set AWS_SECRET_ACCESS_KEY="your-r2-secret-key" --app servicepro-api
fly secrets set S3_ENDPOINT="https://xxx.r2.cloudflarestorage.com" --app servicepro-api
fly secrets set S3_BUCKET_NAME="servicepro-uploads" --app servicepro-api
```

### Go SDK Configuration

The S3-compatible SDK (AWS SDK) works with R2. Key differences from native AWS S3:

```go
// R2 configuration
cfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("auto"),
    config.WithEndpointResolverWithOptions(
        aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL: "https://xxx.r2.cloudflarestorage.com",
            }, nil
        }),
    ),
)

// Important: Use path-style addressing
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.UsePathStyle = true
})
```

---

## Common Operations

### Using Wrangler CLI

```bash
# Upload a file
wrangler r2 object put servicepro-uploads/path/to/file.pdf --file ./local-file.pdf

# Download a file
wrangler r2 object get servicepro-uploads/path/to/file.pdf --file ./downloaded.pdf

# List objects
wrangler r2 object list servicepro-uploads

# Delete a file
wrangler r2 object delete servicepro-uploads/path/to/file.pdf
```

### Using AWS CLI (S3-compatible)

```bash
# Configure AWS CLI for R2
aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID
aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY

# List buckets
aws s3 ls --endpoint-url $S3_ENDPOINT

# List objects in bucket
aws s3 ls s3://servicepro-uploads --endpoint-url $S3_ENDPOINT

# Upload file
aws s3 cp ./file.pdf s3://servicepro-uploads/documents/ --endpoint-url $S3_ENDPOINT

# Download file
aws s3 cp s3://servicepro-uploads/documents/file.pdf ./downloaded.pdf --endpoint-url $S3_ENDPOINT

# Sync directory
aws s3 sync ./local-folder s3://servicepro-uploads/folder/ --endpoint-url $S3_ENDPOINT

# Delete file
aws s3 rm s3://servicepro-uploads/documents/file.pdf --endpoint-url $S3_ENDPOINT

# Delete all files (DANGER)
aws s3 rm s3://servicepro-uploads --recursive --endpoint-url $S3_ENDPOINT
```

### Using cURL

```bash
# R2 supports pre-signed URLs from your application
# Example pre-signed URL usage:
curl -X PUT "https://presigned-url..." --data-binary @./file.pdf
```

---

## Management

### View Usage (Web Dashboard)

1. Go to Cloudflare Dashboard
2. Click "R2" in sidebar
3. View:
   - **Current Storage**: Total data stored
   - **Operations**: Read/write counts
   - **Bandwidth**: Data transferred (free)

### Bucket Settings

1. **Public Access** (not recommended for user files)
   - Go to bucket → Settings
   - Enable "Public access"
   - Get public URL

2. **Custom Domain** (recommended)
   - Go to bucket → Settings → Custom Domains
   - Add `files.servicepro.com`
   - Configure DNS

3. **CORS Configuration**
   - Go to bucket → Settings → CORS
   - Add allowed origins for direct uploads

### CORS Configuration Example

```json
[
  {
    "AllowedOrigins": [
      "https://servicepro.com",
      "https://app.servicepro.com",
      "http://localhost:3000"
    ],
    "AllowedMethods": ["GET", "PUT", "POST", "DELETE", "HEAD"],
    "AllowedHeaders": ["*"],
    "MaxAgeSeconds": 3600
  }
]
```

---

## Troubleshooting

### Access Denied

**Symptom**: `AccessDenied` error

**Causes & Solutions**:

1. **Wrong credentials**
   - Verify Access Key ID and Secret
   - Ensure token hasn't expired

2. **Wrong bucket name**
   - Bucket names are case-sensitive
   - Check for typos

3. **Token permissions**
   - Ensure token has read/write for the bucket
   - Check if token is bucket-specific

### Invalid Endpoint

**Symptom**: `Could not connect to endpoint`

**Solutions**:

1. **Check endpoint format**
   - Must be `https://<account-id>.r2.cloudflarestorage.com`
   - No trailing slash
   - No bucket name in endpoint

2. **Use path-style URLs**
   - Set `S3_FORCE_PATH_STYLE=true`
   - R2 doesn't support virtual-hosted style

### Signature Mismatch

**Symptom**: `SignatureDoesNotMatch`

**Solutions**:

1. **Check credentials**
   - Copy/paste fresh from dashboard
   - No extra whitespace

2. **Check region setting**
   - Use `auto` or leave empty
   - Don't use AWS regions like `us-east-1`

3. **Clock sync**
   - Ensure server time is accurate
   - Signature includes timestamp

### Upload Fails

**Symptom**: Upload hangs or fails

**Solutions**:

1. **Check file size**
   - Max single upload: 5GB
   - Use multipart for larger files

2. **Check CORS** (browser uploads)
   - Configure CORS in bucket settings
   - Include your domain

3. **Check content type**
   - Set appropriate Content-Type header

### Object Not Found

**Symptom**: `NoSuchKey` error

**Solutions**:

1. **Check exact path**
   - Paths are case-sensitive
   - Include full path with slashes

2. **List objects to verify**
   ```bash
   aws s3 ls s3://servicepro-uploads/path/ --endpoint-url $S3_ENDPOINT
   ```

---

## File Organization

Recommended folder structure for ServicePro:

```
servicepro-uploads/
├── tenants/
│   └── {tenant-id}/
│       ├── customers/
│       │   └── {customer-id}/
│       │       └── documents/
│       ├── invoices/
│       │   └── {invoice-id}.pdf
│       ├── quotes/
│       │   └── {quote-id}.pdf
│       └── imports/
│           └── {import-id}/
└── users/
    └── {user-id}/
        └── avatar.jpg
```

### Naming Conventions

```bash
# Good: UUID-based, unique
tenants/550e8400-e29b-41d4-a716-446655440000/invoices/invoice-123.pdf

# Good: Timestamped
imports/2024-01-15/customers-batch-1.csv

# Avoid: User-provided names (security risk)
uploads/../../etc/passwd  # Path traversal attack
```

---

## Security

### Access Control

1. **Never make bucket public** for user files
2. **Use pre-signed URLs** for temporary access
3. **Validate file types** before upload
4. **Scan for malware** (future enhancement)

### Pre-signed URLs

Generate time-limited URLs for secure access:

```go
// Generate pre-signed URL for download (15 minutes)
presignClient := s3.NewPresignClient(client)
req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
    Bucket: aws.String("servicepro-uploads"),
    Key:    aws.String("path/to/file.pdf"),
}, s3.WithPresignExpires(15*time.Minute))
// req.URL is the pre-signed URL
```

### Rotate Credentials

1. Go to Cloudflare Dashboard → R2 → API Tokens
2. Create new token with same permissions
3. Update Fly.io secrets
4. Delete old token

```bash
# Update secrets
fly secrets set AWS_ACCESS_KEY_ID="new-key" --app servicepro-api
fly secrets set AWS_SECRET_ACCESS_KEY="new-secret" --app servicepro-api

# Restart to pick up new secrets
fly apps restart servicepro-api
```

---

## Migration from S3

If migrating from AWS S3:

```bash
# Using rclone (recommended for large migrations)
rclone sync s3:old-bucket r2:servicepro-uploads --progress

# Using AWS CLI
aws s3 sync s3://old-bucket s3://servicepro-uploads \
  --source-region us-east-1 \
  --endpoint-url $S3_ENDPOINT
```

---

## Cost Comparison

| Scenario                       | AWS S3   | Cloudflare R2 | Savings |
| ------------------------------ | -------- | ------------- | ------- |
| 10GB storage + 100GB egress/mo | ~$11.30  | ~$0.15        | 98%     |
| 50GB storage + 500GB egress/mo | ~$56.25  | ~$0.75        | 98%     |
| 100GB storage + 1TB egress/mo  | ~$116.50 | ~$1.50        | 98%     |

R2's zero egress fees make it dramatically cheaper for serving files.

---

## Useful Links

- [Cloudflare R2 Documentation](https://developers.cloudflare.com/r2/)
- [R2 S3 API Compatibility](https://developers.cloudflare.com/r2/api/s3/api/)
- [R2 Pricing](https://developers.cloudflare.com/r2/pricing/)
- [Wrangler CLI](https://developers.cloudflare.com/workers/wrangler/)
- [Cloudflare Status](https://www.cloudflarestatus.com/)
