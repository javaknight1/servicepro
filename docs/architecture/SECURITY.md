# ServicePro Security Architecture

> **Classification**: Internal
> **Version**: 1.0
> **Last Updated**: January 2026

## Table of Contents

1. [Security Overview](#1-security-overview)
2. [Authentication](#2-authentication)
3. [Authorization](#3-authorization)
4. [Data Protection](#4-data-protection)
5. [Network Security](#5-network-security)
6. [Application Security](#6-application-security)
7. [Infrastructure Security](#7-infrastructure-security)
8. [Compliance](#8-compliance)
9. [Security Monitoring](#9-security-monitoring)
10. [Incident Response](#10-incident-response)

---

## 1. Security Overview

### 1.1 Security Principles

ServicePro follows defense-in-depth security principles:

```mermaid
graph TB
    subgraph "Defense in Depth"
        L1[Layer 1: Edge Security<br/>WAF, DDoS Protection, TLS]
        L2[Layer 2: Network Security<br/>VPC, Security Groups, NACLs]
        L3[Layer 3: Application Security<br/>Authentication, Authorization, Validation]
        L4[Layer 4: Data Security<br/>Encryption, Masking, Access Control]
        L5[Layer 5: Operational Security<br/>Logging, Monitoring, Alerting]
    end

    L1 --> L2 --> L3 --> L4 --> L5
```

### 1.2 Security Domains

| Domain                      | Responsibility             | Key Controls                |
| --------------------------- | -------------------------- | --------------------------- |
| **Identity & Access**       | Who can access what        | JWT, RBAC, MFA              |
| **Data Protection**         | Protecting sensitive data  | Encryption, Masking, Backup |
| **Network Security**        | Secure communications      | TLS, VPC, Firewalls         |
| **Application Security**    | Secure code practices      | Input validation, OWASP     |
| **Infrastructure Security** | Secure hosting environment | AWS security, Patching      |
| **Operations Security**     | Secure operations          | Logging, Monitoring, IR     |

---

## 2. Authentication

### 2.1 Authentication Architecture

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant A as API Server
    participant R as Redis
    participant DB as Database

    Note over U,DB: Initial Login
    U->>F: Enter email/password
    F->>A: POST /auth/login
    A->>R: Check rate limit (IP + email)

    alt Rate Limited
        A-->>F: 429 Too Many Requests
    else Within Limit
        A->>DB: SELECT user WHERE email = ?
        DB-->>A: User record

        alt User Not Found
            A-->>F: 401 Invalid credentials
        else User Found
            A->>A: bcrypt.Compare(password, hash)

            alt Password Invalid
                A->>DB: INCREMENT failed_login_count
                A->>A: Check if should lock

                alt Should Lock
                    A->>DB: SET locked_until
                    A-->>F: 401 Account locked
                else Not Locked Yet
                    A-->>F: 401 Invalid credentials
                end
            else Password Valid
                A->>A: Check email_verified

                alt Not Verified
                    A-->>F: 403 Email not verified
                else Verified
                    A->>A: Generate JWT (15m) + Refresh (7d)
                    A->>DB: RESET failed_login_count
                    A->>R: Store refresh token hash
                    A-->>F: 200 { access_token, refresh_token }
                end
            end
        end
    end
```

### 2.2 JWT Token Structure

```
Header:
{
  "alg": "HS256",
  "typ": "JWT"
}

Payload:
{
  "sub": "user-uuid",           // Subject (user ID)
  "email": "user@example.com",  // User email
  "role": "manager",            // Primary role
  "iat": 1700000000,            // Issued at
  "exp": 1700000900,            // Expires (15 minutes)
  "iss": "servicepro",          // Issuer
  "jti": "unique-token-id"      // Token ID for revocation
}

Signature:
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  JWT_SECRET
)
```

### 2.3 Token Lifecycle

```mermaid
stateDiagram-v2
    [*] --> NoToken: User arrives
    NoToken --> Login: Submit credentials
    Login --> HasAccessToken: Login success
    Login --> NoToken: Login failure

    HasAccessToken --> ValidToken: Token valid
    HasAccessToken --> ExpiredToken: Token expired

    ValidToken --> HasAccessToken: Use API
    ExpiredToken --> RefreshAttempt: Auto-refresh

    RefreshAttempt --> HasAccessToken: Refresh success
    RefreshAttempt --> NoToken: Refresh failed (logout)

    HasAccessToken --> NoToken: User logout
    NoToken --> [*]
```

### 2.4 Password Security

| Control               | Implementation                             |
| --------------------- | ------------------------------------------ |
| **Hashing Algorithm** | bcrypt with cost factor 12                 |
| **Minimum Length**    | 8 characters                               |
| **Complexity**        | Must contain: uppercase, lowercase, number |
| **History**           | Cannot reuse last 5 passwords              |
| **Expiration**        | Optional, configurable by admin            |
| **Reset**             | Token-based, 1-hour expiry                 |

### 2.5 Account Lockout

```mermaid
flowchart TD
    A[Login Attempt] --> B{Credentials Valid?}
    B -->|Yes| C[Reset Counter]
    C --> D[Grant Access]

    B -->|No| E[Increment Counter]
    E --> F{Counter >= 5?}
    F -->|No| G[401 Invalid Credentials]
    F -->|Yes| H[Lock Account 15 min]
    H --> I[401 Account Locked]

    J[Wait 15 minutes] --> K[Auto Unlock]
    K --> A
```

---

## 3. Authorization

### 3.1 Role-Based Access Control (RBAC)

```mermaid
graph TB
    subgraph "Role Hierarchy"
        SA[super_admin<br/>Level 100]
        A[admin<br/>Level 80]
        M[manager<br/>Level 60]
        U[user<br/>Level 40]
        G[guest<br/>Level 20]
    end

    SA -->|inherits| A
    A -->|inherits| M
    M -->|inherits| U
    U -->|inherits| G
```

### 3.2 Permission Model

Permissions follow the format: `resource.action`

**Resources:**

- `users` - User accounts
- `roles` - Role definitions
- `permissions` - Permission definitions
- `customers` - Customer records
- `jobs` - Job/work orders
- `quotes` - Quotes
- `invoices` - Invoices
- `payments` - Payments
- `reports` - Analytics reports
- `settings` - System settings

**Actions:**

- `create` - Create new records
- `read` - View records
- `update` - Modify records
- `delete` - Delete records
- `list` - List/search records
- `manage` - Full control
- `assign` - Assign resources
- `approve` - Approve workflows

### 3.3 Default Role Permissions

| Permission       | super_admin | admin | manager | user | guest |
| ---------------- | :---------: | :---: | :-----: | :--: | :---: |
| users.manage     |      ✓      |   ✓   |         |      |       |
| roles.manage     |      ✓      |       |         |      |       |
| customers.create |      ✓      |   ✓   |    ✓    |      |       |
| customers.read   |      ✓      |   ✓   |    ✓    |  ✓   |   ✓   |
| customers.update |      ✓      |   ✓   |    ✓    |      |       |
| customers.delete |      ✓      |   ✓   |         |      |       |
| jobs.create      |      ✓      |   ✓   |    ✓    |      |       |
| jobs.read        |      ✓      |   ✓   |    ✓    |  ✓   |   ✓   |
| jobs.update      |      ✓      |   ✓   |    ✓    |  ✓   |       |
| jobs.delete      |      ✓      |   ✓   |         |      |       |
| invoices.create  |      ✓      |   ✓   |    ✓    |      |       |
| invoices.read    |      ✓      |   ✓   |    ✓    |  ✓   |       |
| invoices.manage  |      ✓      |   ✓   |         |      |       |
| reports.view     |      ✓      |   ✓   |    ✓    |      |       |
| settings.manage  |      ✓      |   ✓   |         |      |       |

### 3.4 Permission Check Flow

```mermaid
sequenceDiagram
    participant R as Request
    participant M as Auth Middleware
    participant P as Permission Middleware
    participant C as Redis Cache
    participant DB as Database
    participant H as Handler

    R->>M: Request with JWT
    M->>M: Validate JWT
    M->>M: Extract user_id, role

    M->>P: Pass to Permission Check
    P->>C: Get cached permissions

    alt Cache Hit
        C-->>P: Permissions
    else Cache Miss
        P->>DB: SELECT permissions for user
        DB-->>P: Permissions
        P->>C: Cache for 5 min
    end

    P->>P: Check required permission

    alt Has Permission
        P->>H: Proceed to handler
        H-->>R: Response
    else No Permission
        P-->>R: 403 Forbidden
    end
```

---

## 4. Data Protection

### 4.1 Encryption

#### At Rest

| Data Type  | Encryption       | Key Management |
| ---------- | ---------------- | -------------- |
| Database   | AES-256 (RDS)    | AWS KMS        |
| S3 Objects | AES-256 (SSE-S3) | AWS Managed    |
| Redis      | In-transit only  | N/A            |
| Backups    | AES-256          | AWS KMS        |

#### In Transit

| Connection   | Protocol | Cipher Suites          |
| ------------ | -------- | ---------------------- |
| Client → CDN | TLS 1.3  | ECDHE+AESGCM           |
| CDN → ALB    | TLS 1.2+ | AWS ALB default        |
| ALB → API    | TLS 1.2+ | AWS internal           |
| API → RDS    | TLS 1.2+ | AWS RDS CA             |
| API → Redis  | TLS 1.2+ | ElastiCache encryption |

### 4.2 Sensitive Data Handling

```mermaid
flowchart TB
    subgraph "Input"
        RAW[Raw Input]
    end

    subgraph "Processing"
        VAL[Validation]
        SAN[Sanitization]
        MASK[Masking]
    end

    subgraph "Storage"
        HASH[Hashed<br/>Passwords, Tokens]
        ENC[Encrypted<br/>PII, Financial]
        PLAIN[Plain<br/>Non-sensitive]
    end

    subgraph "Output"
        REDACT[Redacted<br/>Logs, Errors]
        FULL[Full Data<br/>Authorized Only]
    end

    RAW --> VAL --> SAN

    SAN -->|Passwords| HASH
    SAN -->|PII| ENC
    SAN -->|Other| PLAIN

    HASH --> REDACT
    ENC --> REDACT
    ENC -->|With Permission| FULL
    PLAIN --> FULL
```

### 4.3 PII Data Classification

| Classification         | Examples              | Controls                                |
| ---------------------- | --------------------- | --------------------------------------- |
| **High Sensitivity**   | SSN, Payment Cards    | Encrypted, Limited access, Audit logged |
| **Medium Sensitivity** | Email, Phone, Address | Encrypted at rest, Role-based access    |
| **Low Sensitivity**    | Names, Company        | Standard protection                     |
| **Public**             | Invoice numbers       | No special protection                   |

### 4.4 Data Retention

| Data Type     | Retention                   | Deletion Method            |
| ------------- | --------------------------- | -------------------------- |
| User Accounts | Until deletion requested    | Soft delete + 30-day purge |
| Customers     | 7 years after last activity | Anonymization              |
| Invoices      | 7 years (legal requirement) | Archive to cold storage    |
| Jobs          | 5 years                     | Archive                    |
| Audit Logs    | 2 years                     | Rotate to archive          |
| Session Data  | 24 hours                    | Auto-expire                |

---

## 5. Network Security

### 5.1 Network Architecture

```mermaid
graph TB
    subgraph "Internet"
        USERS[Users]
        ATTACKERS[Potential Attackers]
    end

    subgraph "AWS Edge"
        SHIELD[AWS Shield<br/>DDoS Protection]
        WAF[AWS WAF<br/>Web Application Firewall]
        CF[CloudFront<br/>CDN]
    end

    subgraph "VPC"
        subgraph "Public Subnets"
            ALB[Application<br/>Load Balancer]
            NAT[NAT Gateway]
        end

        subgraph "Private Subnets"
            ECS[ECS Fargate<br/>Containers]
            RDS[(RDS)]
            REDIS[(ElastiCache)]
        end
    end

    USERS --> SHIELD
    ATTACKERS --> SHIELD
    SHIELD --> WAF
    WAF --> CF
    CF --> ALB
    ALB --> ECS
    ECS --> RDS
    ECS --> REDIS
    ECS --> NAT
    NAT --> USERS
```

### 5.2 Security Groups

```
SG: ALB (sg-alb)
┌────────────────────────────────────────┐
│ Inbound:                               │
│   - 443/tcp from 0.0.0.0/0 (HTTPS)    │
│ Outbound:                              │
│   - 8080/tcp to sg-api                │
└────────────────────────────────────────┘

SG: API (sg-api)
┌────────────────────────────────────────┐
│ Inbound:                               │
│   - 8080/tcp from sg-alb only         │
│ Outbound:                              │
│   - 5432/tcp to sg-db                 │
│   - 6379/tcp to sg-redis              │
│   - 443/tcp to 0.0.0.0/0 (AWS APIs)   │
└────────────────────────────────────────┘

SG: Database (sg-db)
┌────────────────────────────────────────┐
│ Inbound:                               │
│   - 5432/tcp from sg-api only         │
│ Outbound:                              │
│   - None                               │
└────────────────────────────────────────┘

SG: Redis (sg-redis)
┌────────────────────────────────────────┐
│ Inbound:                               │
│   - 6379/tcp from sg-api only         │
│ Outbound:                              │
│   - None                               │
└────────────────────────────────────────┘
```

### 5.3 WAF Rules

| Rule                                     | Action | Description                                |
| ---------------------------------------- | ------ | ------------------------------------------ |
| AWS-AWSManagedRulesCommonRuleSet         | Block  | Common vulnerabilities                     |
| AWS-AWSManagedRulesSQLiRuleSet           | Block  | SQL injection                              |
| AWS-AWSManagedRulesKnownBadInputsRuleSet | Block  | Known bad patterns                         |
| RateLimit-1000                           | Block  | > 1000 requests/5min per IP                |
| GeoBlock                                 | Block  | Block non-target countries (if applicable) |
| Custom-APIAbuse                          | Block  | Specific API abuse patterns                |

---

## 6. Application Security

### 6.1 OWASP Top 10 Mitigations

| Vulnerability                        | Mitigation                                         |
| ------------------------------------ | -------------------------------------------------- |
| **A01: Broken Access Control**       | RBAC, permission checks on every endpoint          |
| **A02: Cryptographic Failures**      | TLS 1.3, AES-256 encryption, secure key management |
| **A03: Injection**                   | Parameterized queries (GORM), input validation     |
| **A04: Insecure Design**             | Threat modeling, security reviews                  |
| **A05: Security Misconfiguration**   | Infrastructure as code, security scanning          |
| **A06: Vulnerable Components**       | Dependency scanning, auto-updates                  |
| **A07: Authentication Failures**     | JWT, rate limiting, account lockout                |
| **A08: Software Integrity Failures** | Signed artifacts, code review                      |
| **A09: Logging Failures**            | Comprehensive audit logging                        |
| **A10: SSRF**                        | Allowlist external calls, disable redirects        |

### 6.2 Input Validation

```go
// Example: Customer creation validation
type CreateCustomerRequest struct {
    FirstName    string `json:"first_name" validate:"required,min=1,max=100"`
    LastName     string `json:"last_name" validate:"required,min=1,max=100"`
    Email        string `json:"email" validate:"required,email"`
    Phone        string `json:"phone" validate:"omitempty,e164"`
    CustomerType string `json:"customer_type" validate:"required,oneof=residential commercial"`
}

// Validation occurs at:
// 1. JSON binding (type checking)
// 2. Struct validation (business rules)
// 3. Database constraints (uniqueness, foreign keys)
```

### 6.3 Rate Limiting

| Endpoint Category     | Limit        | Window    | Key        |
| --------------------- | ------------ | --------- | ---------- |
| Login                 | 10 requests  | 1 minute  | IP + Email |
| Password Reset        | 5 requests   | 1 minute  | IP + Email |
| API (authenticated)   | 100 requests | 1 minute  | User ID    |
| API (unauthenticated) | 30 requests  | 1 minute  | IP         |
| Invoice PDF           | 10 requests  | 1 minute  | User ID    |
| Export                | 5 requests   | 5 minutes | User ID    |

### 6.4 Security Headers

```
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' https://js.stripe.com
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

---

## 7. Infrastructure Security

### 7.1 AWS Security Services

```mermaid
graph TB
    subgraph "Detection & Response"
        GD[GuardDuty<br/>Threat Detection]
        SEC[Security Hub<br/>Compliance]
        DET[Detective<br/>Investigation]
    end

    subgraph "Protection"
        WAF[WAF<br/>Web Firewall]
        SHIELD[Shield<br/>DDoS Protection]
        FW[Network Firewall]
    end

    subgraph "Identity"
        IAM[IAM<br/>Access Management]
        ORG[Organizations<br/>Multi-account]
        SSO[IAM Identity Center]
    end

    subgraph "Data Protection"
        KMS[KMS<br/>Key Management]
        SM[Secrets Manager]
        MACIE[Macie<br/>Data Discovery]
    end

    subgraph "Logging"
        CT[CloudTrail<br/>API Logging]
        CW[CloudWatch<br/>Monitoring]
        VPC_FL[VPC Flow Logs]
    end

    GD --> SEC
    DET --> SEC
    CT --> GD
    VPC_FL --> GD
    KMS --> SM
```

### 7.2 IAM Policies

```json
// ECS Task Role - Minimum Required Permissions
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::servicepro-uploads/*"
    },
    {
      "Effect": "Allow",
      "Action": ["ses:SendEmail", "ses:SendRawEmail"],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ses:FromAddress": "noreply@servicepro.com"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": ["secretsmanager:GetSecretValue"],
      "Resource": "arn:aws:secretsmanager:*:*:secret:servicepro/*"
    }
  ]
}
```

### 7.3 Secrets Management

| Secret               | Storage         | Rotation         |
| -------------------- | --------------- | ---------------- |
| Database credentials | Secrets Manager | 90 days (auto)   |
| JWT signing key      | Secrets Manager | Manual           |
| Stripe API keys      | Secrets Manager | Manual           |
| AWS SES credentials  | IAM Role        | N/A (role-based) |
| Redis password       | Secrets Manager | 90 days          |

---

## 8. Compliance

### 8.1 Compliance Framework

| Framework     | Applicability                 | Status         |
| ------------- | ----------------------------- | -------------- |
| SOC 2 Type II | Service organization controls | In progress    |
| GDPR          | EU customer data              | Compliant      |
| PCI DSS       | Payment card handling         | Stripe handles |
| CCPA          | California consumer privacy   | Compliant      |

### 8.2 Data Subject Rights (GDPR/CCPA)

| Right             | Implementation            |
| ----------------- | ------------------------- |
| **Access**        | Export user data via API  |
| **Rectification** | Update profile endpoint   |
| **Erasure**       | Account deletion workflow |
| **Portability**   | JSON/CSV data export      |
| **Restriction**   | Account suspension        |
| **Object**        | Marketing opt-out         |

---

## 9. Security Monitoring

### 9.1 Monitoring Architecture

```mermaid
graph LR
    subgraph "Sources"
        APP[Application Logs]
        AWS[AWS CloudTrail]
        VPC[VPC Flow Logs]
        WAF_LOG[WAF Logs]
    end

    subgraph "Collection"
        CW[CloudWatch Logs]
    end

    subgraph "Analysis"
        GD[GuardDuty]
        SIEM[Security Hub]
    end

    subgraph "Response"
        SNS[SNS Alerts]
        LAMBDA[Auto-remediation]
        ONCALL[On-call Team]
    end

    APP --> CW
    AWS --> CW
    VPC --> CW
    WAF_LOG --> CW

    CW --> GD
    CW --> SIEM

    GD --> SNS
    SIEM --> SNS
    SNS --> LAMBDA
    SNS --> ONCALL
```

### 9.2 Security Alerts

| Alert                             | Severity | Response                           |
| --------------------------------- | -------- | ---------------------------------- |
| Multiple failed logins            | Medium   | Monitor, potential account lockout |
| Successful login from new country | Low      | Email notification to user         |
| Privilege escalation attempt      | High     | Immediate investigation            |
| SQL injection detected            | High     | Block IP, investigate              |
| Data exfiltration pattern         | Critical | Immediate response                 |
| Unusual API usage pattern         | Medium   | Review and investigate             |

### 9.3 Audit Logging

All security-relevant events are logged:

```json
{
  "timestamp": "2026-01-17T10:30:00Z",
  "event_type": "authentication.login_success",
  "user_id": "uuid",
  "email": "user@example.com",
  "ip_address": "1.2.3.4",
  "user_agent": "Mozilla/5.0...",
  "geo_location": "US",
  "session_id": "session-uuid",
  "request_id": "request-uuid"
}
```

---

## 10. Incident Response

### 10.1 Incident Severity Levels

| Level           | Description                  | Response Time | Examples                           |
| --------------- | ---------------------------- | ------------- | ---------------------------------- |
| **P1 Critical** | Service down or data breach  | 15 minutes    | Data leak, complete outage         |
| **P2 High**     | Major functionality impacted | 1 hour        | Auth system down, payment failures |
| **P3 Medium**   | Minor functionality impacted | 4 hours       | Feature degradation                |
| **P4 Low**      | No user impact               | 24 hours      | Security scan finding              |

### 10.2 Incident Response Process

```mermaid
stateDiagram-v2
    [*] --> Detection: Alert Triggered

    Detection --> Triage: Acknowledge Alert
    Triage --> Containment: Severity Assessed

    Containment --> Eradication: Threat Contained
    Eradication --> Recovery: Root Cause Fixed

    Recovery --> PostIncident: Service Restored
    PostIncident --> [*]: Lessons Learned

    state Containment {
        [*] --> Isolate
        Isolate --> Block
        Block --> Preserve
    }

    state Eradication {
        [*] --> RootCause
        RootCause --> Remediate
        Remediate --> Verify
    }
```

### 10.3 Contact Information

| Role                      | Contact                        | Escalation Path |
| ------------------------- | ------------------------------ | --------------- |
| Security On-Call          | security-oncall@servicepro.com | PagerDuty       |
| Security Lead             | security-lead@servicepro.com   | Direct          |
| Legal (for breaches)      | legal@servicepro.com           | Email           |
| PR (for public incidents) | pr@servicepro.com              | Email           |

---

## Document History

| Version | Date    | Author        | Changes               |
| ------- | ------- | ------------- | --------------------- |
| 1.0     | 2026-01 | Security Team | Initial documentation |

---

_This document is reviewed quarterly and updated as security controls evolve._
