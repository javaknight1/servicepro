# ServicePro Terraform Infrastructure

This directory contains Terraform configurations for provisioning and managing ServicePro infrastructure on AWS.

## Structure

````
terraform/
├── modules/              # Reusable Terraform modules
│   ├── vpc/             # VPC and networking
│   ├── eks/             # EKS cluster
│   ├── rds/             # RDS PostgreSQL
│   ├── elasticache/     # ElastiCache Redis
│   └── s3/              # S3 buckets
├── environments/        # Environment-specific configurations
│   ├── staging/
│   ├── preprod/
│   └── prod/
└── versions.tf         # Provider versions

## Prerequisites

- Terraform >= 1.5
- AWS CLI configured with appropriate credentials
- AWS account with necessary permissions

## Initial Setup

### 1. Configure Backend (Recommended)

Create an S3 bucket and DynamoDB table for state management:

```bash
# Create S3 bucket for state
aws s3 mb s3://servicepro-terraform-state --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket servicepro-terraform-state \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for state locking
aws dynamodb create-table \
  --table-name servicepro-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
  --region us-east-1
````

### 2. Initialize Environment

```bash
cd environments/staging
terraform init
```

## Usage

### Deploy Staging Environment

```bash
cd environments/staging

# Review the plan
terraform plan

# Apply changes
terraform apply

# Get outputs
terraform output
```

### Deploy PreProd Environment

```bash
cd environments/preprod

# Review the plan
terraform plan

# Apply changes
terraform apply
```

### Deploy Production Environment

```bash
cd environments/prod

# Review the plan
terraform plan

# Apply changes
terraform apply
```

## Module Documentation

### VPC Module

Creates VPC with public and private subnets across multiple availability zones.

### EKS Module

Provisions EKS cluster with managed node groups.

### RDS Module

Creates PostgreSQL RDS instance with automated backups.

### ElastiCache Module

Sets up Redis cluster for caching.

### S3 Module

Creates S3 buckets for application assets.

## State Management

Terraform state is stored remotely in S3 with DynamoDB locking to prevent concurrent modifications.

## Security Best Practices

1. Never commit `.tfvars` files containing secrets
2. Use AWS Secrets Manager or Parameter Store for sensitive values
3. Enable encryption for all resources
4. Use private subnets for databases and application workloads
5. Enable VPC Flow Logs
6. Use IAM roles with least privilege

## Troubleshooting

### Common Issues

**Issue**: Terraform state locked

```bash
# Remove stale lock (use with caution)
terraform force-unlock <lock-id>
```

**Issue**: Resource already exists

```bash
# Import existing resource
terraform import module.vpc.aws_vpc.main vpc-xxxxx
```

## Updating Infrastructure

1. Create a new branch
2. Make changes to Terraform files
3. Run `terraform plan` to review changes
4. Create PR for review
5. After approval, apply changes
6. Tag the release if needed

## Cleanup

To destroy infrastructure (use with extreme caution):

```bash
cd environments/staging
terraform destroy
```

## Support

For issues or questions, contact the infrastructure team or create an issue in the repository.
