# =============================================================================
# ServicePro Infrastructure - Variables
# =============================================================================

# -----------------------------------------------------------------------------
# General
# -----------------------------------------------------------------------------
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "servicepro"
}

variable "environment" {
  description = "Deployment environment"
  type        = string

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be: dev, staging, or production."
  }
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

# -----------------------------------------------------------------------------
# Networking
# -----------------------------------------------------------------------------
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of availability zones"
  type        = number
  default     = 3
}

variable "enable_nat_gateway" {
  description = "Enable NAT Gateway"
  type        = bool
  default     = true
}

variable "single_nat_gateway" {
  description = "Use single NAT Gateway"
  type        = bool
  default     = false
}

variable "enable_vpc_flow_logs" {
  description = "Enable VPC Flow Logs"
  type        = bool
  default     = true
}

variable "enable_vpc_endpoints" {
  description = "Enable VPC endpoints"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# EKS
# -----------------------------------------------------------------------------
variable "eks_cluster_version" {
  description = "EKS cluster version"
  type        = string
  default     = "1.29"
}

variable "eks_public_access" {
  description = "Enable public access to EKS API"
  type        = bool
  default     = true
}

variable "eks_node_instance_types" {
  description = "Instance types for EKS nodes"
  type        = list(string)
  default     = ["t3.medium", "t3.large"]
}

variable "eks_node_min_size" {
  description = "Minimum number of EKS nodes"
  type        = number
  default     = 2
}

variable "eks_node_max_size" {
  description = "Maximum number of EKS nodes"
  type        = number
  default     = 10
}

variable "eks_node_desired_size" {
  description = "Desired number of EKS nodes"
  type        = number
  default     = 2
}

variable "eks_node_disk_size" {
  description = "EKS node disk size in GB"
  type        = number
  default     = 50
}

variable "eks_use_spot" {
  description = "Use spot instances"
  type        = bool
  default     = false
}

variable "enable_argocd" {
  description = "Install ArgoCD"
  type        = bool
  default     = true
}

variable "argocd_domain" {
  description = "Domain for ArgoCD"
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# RDS
# -----------------------------------------------------------------------------
variable "rds_engine_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15.4"
}

variable "rds_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "rds_allocated_storage" {
  description = "Initial storage in GB"
  type        = number
  default     = 20
}

variable "rds_max_allocated_storage" {
  description = "Maximum storage in GB"
  type        = number
  default     = 100
}

variable "rds_database_name" {
  description = "Database name"
  type        = string
  default     = "servicepro"
}

variable "rds_master_username" {
  description = "Master username"
  type        = string
  default     = "servicepro"
}

variable "rds_multi_az" {
  description = "Enable Multi-AZ"
  type        = bool
  default     = false
}

variable "rds_backup_retention_period" {
  description = "Backup retention in days"
  type        = number
  default     = 7
}

variable "rds_performance_insights" {
  description = "Enable Performance Insights"
  type        = bool
  default     = true
}

variable "rds_enhanced_monitoring" {
  description = "Enable enhanced monitoring"
  type        = bool
  default     = true
}

variable "rds_max_connections" {
  description = "Max database connections"
  type        = number
  default     = 100
}

variable "rds_kms_key_arn" {
  description = "KMS key ARN for encryption"
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# ElastiCache
# -----------------------------------------------------------------------------
variable "redis_engine_version" {
  description = "Redis version"
  type        = string
  default     = "7.0"
}

variable "redis_node_type" {
  description = "Redis node type"
  type        = string
  default     = "cache.t3.micro"
}

variable "redis_num_cache_nodes" {
  description = "Number of cache nodes"
  type        = number
  default     = 1
}

variable "redis_transit_encryption" {
  description = "Enable transit encryption"
  type        = bool
  default     = true
}

variable "redis_snapshot_retention" {
  description = "Snapshot retention in days"
  type        = number
  default     = 7
}

# -----------------------------------------------------------------------------
# S3 and CloudFront
# -----------------------------------------------------------------------------
variable "cors_allowed_origins" {
  description = "CORS allowed origins"
  type        = list(string)
  default     = ["*"]
}

variable "cloudfront_price_class" {
  description = "CloudFront price class"
  type        = string
  default     = "PriceClass_100"
}

variable "cloudfront_geo_restriction" {
  description = "Countries to block"
  type        = list(string)
  default     = []
}

variable "enable_cloudfront_logging" {
  description = "Enable CloudFront logging"
  type        = bool
  default     = true
}

variable "cloudfront_log_retention_days" {
  description = "CloudFront log retention in days"
  type        = number
  default     = 30
}

variable "cloudfront_certificate_arn" {
  description = "ACM certificate ARN for CloudFront (must be in us-east-1)"
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# SES
# -----------------------------------------------------------------------------
variable "ses_domain" {
  description = "Domain for SES"
  type        = string
  default     = ""
}

variable "ses_sender_email" {
  description = "Default sender email"
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# ALB and DNS
# -----------------------------------------------------------------------------
variable "domain_name" {
  description = "Primary domain name"
  type        = string
  default     = ""
}

variable "create_certificate" {
  description = "Create ACM certificate"
  type        = bool
  default     = true
}

variable "certificate_arn" {
  description = "Existing certificate ARN"
  type        = string
  default     = ""
}

variable "subject_alternative_names" {
  description = "Certificate SANs"
  type        = list(string)
  default     = []
}

variable "route53_zone_id" {
  description = "Route53 hosted zone ID"
  type        = string
  default     = ""
}

variable "enable_alb_access_logs" {
  description = "Enable ALB access logs"
  type        = bool
  default     = true
}

variable "alb_log_retention_days" {
  description = "ALB log retention in days"
  type        = number
  default     = 30
}

# -----------------------------------------------------------------------------
# WAF
# -----------------------------------------------------------------------------
variable "enable_waf" {
  description = "Enable WAF"
  type        = bool
  default     = true
}

variable "waf_rate_limit" {
  description = "WAF rate limit per 5 minutes"
  type        = number
  default     = 2000
}

# -----------------------------------------------------------------------------
# Bastion
# -----------------------------------------------------------------------------
variable "enable_bastion" {
  description = "Deploy bastion host"
  type        = bool
  default     = false
}

variable "bastion_instance_type" {
  description = "Bastion instance type"
  type        = string
  default     = "t3.micro"
}

variable "bastion_allowed_cidrs" {
  description = "CIDRs allowed to SSH to bastion"
  type        = list(string)
  default     = []
}

# -----------------------------------------------------------------------------
# Monitoring
# -----------------------------------------------------------------------------
variable "log_retention_days" {
  description = "CloudWatch log retention"
  type        = number
  default     = 30
}

variable "alert_emails" {
  description = "Email addresses for alerts"
  type        = list(string)
  default     = []
}

variable "slack_webhook_url" {
  description = "Slack webhook for alerts"
  type        = string
  default     = ""
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Backup
# -----------------------------------------------------------------------------
variable "backup_retention_days" {
  description = "Backup retention in days"
  type        = number
  default     = 30
}

variable "enable_cross_region_backup" {
  description = "Enable cross-region backup"
  type        = bool
  default     = false
}
