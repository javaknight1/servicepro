# ServicePro Infrastructure

Terraform configurations for provisioning and managing ServicePro on AWS. The infrastructure is designed for an EKS-based deployment using GitOps with ArgoCD.

## Table of Contents

- [Quick Start](#quick-start)
- [Directory Structure](#directory-structure)
- [Infrastructure Overview](#infrastructure-overview)
- [Environment Differences](#environment-differences)
- [Usage](#usage)
- [Module Reference](#module-reference)
- [Development Workflow](#development-workflow)
- [Cost Optimization](#cost-optimization)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Quick Start

```bash
# From the repository root:

# 1. Initialize terraform
make tf-init TF_ENV=dev

# 2. Review the plan
make tf-plan TF_ENV=dev

# 3. Apply infrastructure
make tf-apply TF_ENV=dev

# 4. Get outputs (database endpoints, etc.)
make tf-output TF_ENV=dev
```

## Directory Structure

```
infrastructure/
├── main.tf                 # Module orchestration
├── variables.tf            # Input variables
├── outputs.tf              # Output values
├── providers.tf            # Provider configuration
├── environments/           # Environment-specific configurations
│   ├── dev.tfvars          # Development settings
│   ├── staging.tfvars      # Staging settings
│   └── production.tfvars   # Production settings
├── tests/                  # Validation scripts
│   └── validate.sh         # Comprehensive validation suite
└── modules/                # Reusable Terraform modules
    ├── vpc/                # VPC, subnets, NAT gateways
    ├── security/           # Security groups, bastion, VPC endpoints
    ├── eks/                # EKS cluster, node groups, IRSA, Helm releases
    ├── rds/                # PostgreSQL RDS instance
    ├── elasticache/        # Redis cluster
    ├── s3/                 # S3 buckets (uploads, frontend, backups)
    ├── alb/                # Application Load Balancer, WAF
    ├── cdn/                # CloudFront distribution
    ├── ses/                # Simple Email Service
    ├── monitoring/         # CloudWatch, SNS alerts
    └── backup/             # AWS Backup for RDS
```

## Infrastructure Overview

### Architecture Diagram

```
                                    ┌─────────────────────┐
                                    │     CloudFront      │
                                    │   (CDN + Frontend)  │
                                    └──────────┬──────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              │                │                │
                              ▼                ▼                ▼
                    ┌─────────────────────────────────────────────────┐
                    │                       VPC                        │
                    │  ┌─────────────────────────────────────────┐    │
                    │  │            Public Subnets               │    │
                    │  │  ┌─────────┐  ┌─────────┐  ┌─────────┐ │    │
                    │  │  │   ALB   │  │   NAT   │  │ Bastion │ │    │
                    │  │  │  (WAF)  │  │ Gateway │  │  Host   │ │    │
                    │  │  └────┬────┘  └────┬────┘  └─────────┘ │    │
                    │  └───────┼────────────┼───────────────────┘    │
                    │          │            │                         │
                    │  ┌───────┼────────────┼───────────────────┐    │
                    │  │       │  Private Subnets               │    │
                    │  │       ▼            │                   │    │
                    │  │  ┌─────────────────┴───────┐          │    │
                    │  │  │      EKS Cluster        │          │    │
                    │  │  │  ┌───────┐ ┌───────┐   │          │    │
                    │  │  │  │Backend│ │ArgoCD │   │          │    │
                    │  │  │  │ Pods  │ │       │   │          │    │
                    │  │  │  └───┬───┘ └───────┘   │          │    │
                    │  │  └──────┼─────────────────┘          │    │
                    │  └─────────┼─────────────────────────────┘    │
                    │            │                                   │
                    │  ┌─────────┼─────────────────────────────┐    │
                    │  │         │  Database Subnets           │    │
                    │  │         ▼                             │    │
                    │  │  ┌─────────────┐  ┌─────────────┐    │    │
                    │  │  │     RDS     │  │ ElastiCache │    │    │
                    │  │  │ (PostgreSQL)│  │   (Redis)   │    │    │
                    │  │  └─────────────┘  └─────────────┘    │    │
                    │  └───────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              ▼                ▼                ▼
                    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                    │      S3      │  │     SES      │  │  CloudWatch  │
                    │  (Uploads)   │  │   (Email)    │  │ (Monitoring) │
                    └──────────────┘  └──────────────┘  └──────────────┘
```

### Components

| Component         | Service           | Purpose                                       |
| ----------------- | ----------------- | --------------------------------------------- |
| **Compute**       | EKS               | Container orchestration for backend services  |
| **Database**      | RDS PostgreSQL    | Primary relational database                   |
| **Cache**         | ElastiCache Redis | Session storage, caching                      |
| **Storage**       | S3                | File uploads, frontend static assets, backups |
| **CDN**           | CloudFront        | Global content delivery, frontend hosting     |
| **Load Balancer** | ALB               | HTTPS termination, routing to EKS             |
| **Security**      | WAF               | Web application firewall                      |
| **Email**         | SES               | Transactional emails                          |
| **Monitoring**    | CloudWatch + SNS  | Logs, metrics, alerts                         |
| **GitOps**        | ArgoCD            | Kubernetes deployments                        |
| **Backup**        | AWS Backup        | Automated RDS backups                         |

## Environment Differences

### Quick Comparison

| Feature                | Development | Staging    | Production |
| ---------------------- | ----------- | ---------- | ---------- |
| **Availability Zones** | 2           | 2          | 3          |
| **NAT Gateways**       | 1 (shared)  | 1 (shared) | 3 (per AZ) |
| **VPC Endpoints**      | Disabled    | Enabled    | Enabled    |
| **VPC Flow Logs**      | Disabled    | Disabled   | Enabled    |

### EKS

| Feature            | Development | Staging         | Production      |
| ------------------ | ----------- | --------------- | --------------- |
| **API Access**     | Public      | Public          | Private only    |
| **Instance Types** | t3.medium   | t3.medium/large | m5.large/xlarge |
| **Node Count**     | 1-3         | 2-5             | 3-20            |
| **Spot Instances** | Yes         | Yes             | No (on-demand)  |
| **Disk Size**      | 30 GB       | 50 GB           | 100 GB          |

### RDS PostgreSQL

| Feature                  | Development | Staging      | Production  |
| ------------------------ | ----------- | ------------ | ----------- |
| **Instance Class**       | db.t3.small | db.t3.medium | db.r5.large |
| **Storage**              | 20-50 GB    | 20-100 GB    | 100-500 GB  |
| **Multi-AZ**             | No          | No           | Yes         |
| **Backup Retention**     | 1 day       | 7 days       | 30 days     |
| **Performance Insights** | No          | Yes          | Yes         |
| **Enhanced Monitoring**  | No          | Yes          | Yes         |

### ElastiCache Redis

| Feature                | Development    | Staging        | Production     |
| ---------------------- | -------------- | -------------- | -------------- |
| **Node Type**          | cache.t3.micro | cache.t3.small | cache.r5.large |
| **Node Count**         | 1              | 1              | 2              |
| **Transit Encryption** | No             | Yes            | Yes            |
| **Snapshot Retention** | 0 days         | 7 days         | 7 days         |

### Security & Monitoring

| Feature                | Development | Staging | Production |
| ---------------------- | ----------- | ------- | ---------- |
| **WAF**                | Disabled    | Enabled | Enabled    |
| **ALB Logs**           | Disabled    | Enabled | Enabled    |
| **CloudFront Logs**    | Disabled    | Enabled | Enabled    |
| **Log Retention**      | 7 days      | 30 days | 90 days    |
| **CloudFront Regions** | US/EU       | US/EU   | Global     |

### Cost Comparison (Estimated)

| Environment | Monthly Cost   |
| ----------- | -------------- |
| Development | ~$200-300      |
| Staging     | ~$400-600      |
| Production  | ~$1,500-3,000+ |

## Usage

### Prerequisites

- Terraform >= 1.5
- AWS CLI configured with appropriate credentials
- kubectl (for EKS management)
- helm (for Kubernetes packages)

### Using Make Commands

All terraform operations can be run from the repository root using make commands:

```bash
# Initialize (download providers, modules)
make tf-init TF_ENV=dev

# Validate configuration
make tf-validate TF_ENV=dev

# Create and review execution plan
make tf-plan TF_ENV=dev

# Apply changes (with confirmation)
make tf-apply TF_ENV=dev

# Show outputs
make tf-output TF_ENV=dev

# Format terraform files
make tf-fmt

# Run linting
make tf-lint

# Security scan
make tf-security-scan

# Run full validation suite (format, validate, lint, security, compliance)
make tf-test

# Run quick validation (format and validate only)
make tf-test-quick

# Destroy infrastructure (use with caution!)
make tf-destroy TF_ENV=dev
```

### Direct Terraform Commands

If you prefer to run terraform directly:

```bash
cd infrastructure

# Initialize
terraform init

# Plan with environment
terraform plan -var-file=environments/dev.tfvars

# Apply
terraform apply -var-file=environments/dev.tfvars

# Outputs
terraform output
```

### Connecting to EKS

After applying, configure kubectl:

```bash
# Get the command from terraform output
make tf-output TF_ENV=dev | grep kubectl_config

# Example output:
# kubectl_config = "aws eks update-kubeconfig --region us-west-2 --name servicepro-dev"

# Run the command
aws eks update-kubeconfig --region us-west-2 --name servicepro-dev

# Verify connection
kubectl get nodes
```

### Accessing ArgoCD

ArgoCD is deployed automatically. To access:

```bash
# Port forward to ArgoCD server
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Open https://localhost:8080
# Username: admin
```

## Module Reference

### vpc

Creates VPC infrastructure with public, private, and database subnets.

**Key Resources:**

- VPC with configurable CIDR
- Public subnets (ALB, NAT, Bastion)
- Private subnets (EKS nodes)
- Database subnets (RDS, ElastiCache)
- NAT Gateway(s) for private subnet internet access
- Internet Gateway for public subnets
- Route tables and associations
- VPC Flow Logs (optional)

### security

Manages security groups and network security.

**Key Resources:**

- ALB security group (HTTPS ingress)
- Database security group (PostgreSQL from EKS)
- Cache security group (Redis from EKS)
- Bastion security group (SSH access)
- VPC endpoints for AWS services (S3, ECR, etc.)
- Bastion host (optional)

### eks

Provisions EKS cluster with managed node groups.

**Key Resources:**

- EKS cluster with managed node groups
- IAM roles for Service Accounts (IRSA):
  - VPC CNI
  - EBS CSI Driver
  - AWS Load Balancer Controller
  - Cluster Autoscaler
  - External Secrets Operator
  - Backend application
- Helm releases:
  - AWS Load Balancer Controller
  - Cluster Autoscaler
  - Metrics Server
  - External Secrets
  - ArgoCD (optional)

### rds

Creates PostgreSQL database instance.

**Key Resources:**

- RDS PostgreSQL instance
- DB parameter group
- Enhanced monitoring IAM role
- Secrets Manager secret for credentials
- CloudWatch alarms

### elasticache

Sets up Redis cluster for caching.

**Key Resources:**

- ElastiCache Redis replication group
- Parameter group
- Secrets Manager secret for auth token
- CloudWatch alarms

### s3

Creates S3 buckets with appropriate configurations.

**Key Resources:**

- Uploads bucket (user file storage)
- Frontend bucket (static assets)
- Backup reports bucket
- Lifecycle rules for cost optimization
- CORS configuration
- Encryption at rest

### alb

Configures Application Load Balancer.

**Key Resources:**

- Application Load Balancer
- Target groups
- HTTPS listener with SSL certificate
- HTTP to HTTPS redirect
- WAF Web ACL with AWS managed rules
- Access logging

### cdn

Sets up CloudFront distribution.

**Key Resources:**

- CloudFront distribution
- Origin Access Control for S3
- Cache policies (static assets, API)
- Response headers (security headers)
- Access logging

### ses

Configures Simple Email Service.

**Key Resources:**

- Domain identity with DKIM
- Mail From domain
- DMARC record
- Configuration set with engagement tracking
- Email templates

### monitoring

Sets up monitoring, alerting, and CloudWatch dashboards.

**Key Resources:**

- SNS topics (critical, warning, info)
- Email/Slack subscriptions
- CloudWatch log groups
- CloudWatch dashboards (automatically deployed from JSON templates):
  - Application dashboard - Overall application health and metrics
  - Database dashboard - PostgreSQL and Redis performance
  - API dashboard - Request distribution and endpoint metrics
  - API Metrics dashboard - Custom application metrics
- Metric alarms (ALB, RDS, ElastiCache)

**Dashboard Templates:**

The monitoring module automatically deploys CloudWatch dashboards from JSON template files located in `modules/monitoring/dashboards/`. These templates support variable substitution for environment-specific values:

- `${ENVIRONMENT}` - Environment name
- `${AWS_REGION}` - AWS region
- `${AWS_ACCOUNT_ID}` - Account ID
- `${CLUSTER_NAME}` - EKS cluster name
- `${SERVICE_NAME}` - Service name
- `${DB_INSTANCE_ID}` - RDS instance identifier
- `${REDIS_CLUSTER_ID}` - ElastiCache cluster ID

### backup

Configures AWS Backup for RDS.

**Key Resources:**

- Backup vault with KMS encryption
- Backup plan with schedule
- Backup selection for RDS
- IAM role for backup operations

## Development Workflow

### Making Infrastructure Changes

1. **Create a feature branch**

   ```bash
   git checkout -b feature/add-new-resource
   ```

2. **Make changes to terraform files**

3. **Format and validate**

   ```bash
   make tf-fmt
   make tf-validate TF_ENV=dev
   ```

4. **Run security scan**

   ```bash
   make tf-security-scan
   ```

5. **Test in development**

   ```bash
   make tf-plan TF_ENV=dev
   make tf-apply TF_ENV=dev
   ```

6. **Create PR for review**

7. **After approval, apply to staging**

   ```bash
   make tf-plan TF_ENV=staging
   make tf-apply TF_ENV=staging
   ```

8. **Apply to production**
   ```bash
   make tf-plan TF_ENV=production
   make tf-apply TF_ENV=production
   ```

### Adding a New Module

1. Create module directory:

   ```bash
   mkdir -p modules/new-module
   ```

2. Create module files:

   - `main.tf` - Resource definitions
   - `variables.tf` - Input variables
   - `outputs.tf` - Output values

3. Call module from `main.tf`:

   ```hcl
   module "new_module" {
     source = "./modules/new-module"

     name_prefix = local.name_prefix
     tags        = local.common_tags
     # ... other variables
   }
   ```

4. Add any root-level variables to `variables.tf`

5. Add outputs to `outputs.tf`

### State Management

Terraform state is stored locally by default. For team collaboration, configure remote state:

```hcl
# In providers.tf, uncomment and configure:
terraform {
  backend "s3" {
    bucket         = "servicepro-terraform-state"
    key            = "infrastructure/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "servicepro-terraform-locks"
    encrypt        = true
  }
}
```

Create the backend resources:

```bash
# Create S3 bucket
aws s3 mb s3://servicepro-terraform-state --region us-west-2
aws s3api put-bucket-versioning --bucket servicepro-terraform-state \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for locking
aws dynamodb create-table \
  --table-name servicepro-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-west-2
```

## Cost Optimization

### Development Environment

The dev environment is optimized for cost:

- Single NAT gateway instead of one per AZ
- Spot instances for EKS nodes
- Smaller instance types
- No Multi-AZ for RDS
- Minimal logging and monitoring
- WAF disabled

### Tips for Reducing Costs

1. **Destroy dev when not in use**

   ```bash
   make tf-destroy TF_ENV=dev
   ```

2. **Use spot instances in non-production**

3. **Right-size instances** based on actual usage

4. **Review NAT gateway costs** - consider NAT instances for dev

5. **Use cost allocation tags** to track spending

## Security

### Best Practices Implemented

- **Network isolation**: Database and cache in private subnets
- **Encryption at rest**: RDS, ElastiCache, S3
- **Encryption in transit**: TLS everywhere
- **Secrets management**: AWS Secrets Manager for credentials
- **IRSA**: Pod-level AWS permissions
- **WAF**: Protection against common web attacks
- **Security groups**: Least-privilege network access
- **VPC endpoints**: Private AWS API access (staging/prod)

### Secrets Handling

**Never commit secrets to git.** Sensitive values are managed via:

1. **Terraform variables** - Pass via environment or tfvars (gitignored)
2. **AWS Secrets Manager** - Credentials generated and stored securely
3. **External Secrets Operator** - Syncs secrets to Kubernetes

## Troubleshooting

### Common Issues

**State locked**

```bash
# Get lock ID from error message, then:
make tf-unlock TF_LOCK_ID=<lock-id>
```

**Resource already exists**

```bash
# Import existing resource
make tf-import TF_RESOURCE=module.vpc.aws_vpc.main TF_ID=vpc-xxxxx TF_ENV=dev
```

**EKS connection issues**

```bash
# Update kubeconfig
aws eks update-kubeconfig --region us-west-2 --name servicepro-dev

# Check cluster status
aws eks describe-cluster --name servicepro-dev --query 'cluster.status'
```

**RDS connection issues**

```bash
# Get RDS endpoint from terraform
make tf-output TF_ENV=dev | grep rds_endpoint

# Get credentials from Secrets Manager
aws secretsmanager get-secret-value --secret-id servicepro-dev-db-credentials
```

### Useful Commands

```bash
# List all resources in state
make tf-state-list TF_ENV=dev

# Show specific resource details
make tf-state-show TF_RESOURCE=module.eks.aws_eks_cluster.main TF_ENV=dev

# Refresh state from AWS
make tf-refresh TF_ENV=dev

# Generate resource graph
make tf-graph TF_ENV=dev
```

## Support

For issues or questions:

- Check the [troubleshooting](#troubleshooting) section
- Review terraform plan output carefully
- Create an issue in the repository
