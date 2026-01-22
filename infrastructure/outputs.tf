# =============================================================================
# ServicePro Infrastructure - Outputs
# =============================================================================

# -----------------------------------------------------------------------------
# General
# -----------------------------------------------------------------------------
output "aws_region" {
  description = "AWS region"
  value       = var.aws_region
}

output "aws_account_id" {
  description = "AWS account ID"
  value       = data.aws_caller_identity.current.account_id
}

output "name_prefix" {
  description = "Resource naming prefix"
  value       = local.name_prefix
}

output "resource_group_name" {
  description = "AWS Resource Group name"
  value       = module.resource_group.name
}

output "resource_group_arn" {
  description = "AWS Resource Group ARN"
  value       = module.resource_group.arn
}

# -----------------------------------------------------------------------------
# VPC
# -----------------------------------------------------------------------------
output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "vpc_cidr" {
  description = "VPC CIDR block"
  value       = module.vpc.vpc_cidr
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = module.vpc.public_subnet_ids
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

output "database_subnet_ids" {
  description = "Database subnet IDs"
  value       = module.vpc.database_subnet_ids
}

output "nat_gateway_ips" {
  description = "NAT Gateway public IPs"
  value       = module.vpc.nat_gateway_ips
}

# -----------------------------------------------------------------------------
# EKS
# -----------------------------------------------------------------------------
output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_certificate_authority" {
  description = "EKS cluster CA certificate"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "eks_oidc_provider_arn" {
  description = "EKS OIDC provider ARN"
  value       = module.eks.oidc_provider_arn
}

output "eks_node_security_group_id" {
  description = "EKS node security group ID"
  value       = module.eks.node_security_group_id
}

output "kubectl_config" {
  description = "kubectl configuration command"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

# -----------------------------------------------------------------------------
# RDS
# -----------------------------------------------------------------------------
output "rds_endpoint" {
  description = "RDS instance endpoint"
  value       = module.rds.endpoint
}

output "rds_address" {
  description = "RDS instance address"
  value       = module.rds.address
}

output "rds_port" {
  description = "RDS instance port"
  value       = module.rds.port
}

output "rds_database_name" {
  description = "RDS database name"
  value       = module.rds.database_name
}

output "rds_secret_arn" {
  description = "RDS credentials secret ARN"
  value       = module.rds.secret_arn
}

# -----------------------------------------------------------------------------
# ElastiCache
# -----------------------------------------------------------------------------
output "redis_endpoint" {
  description = "Redis primary endpoint"
  value       = module.elasticache.endpoint
}

output "redis_port" {
  description = "Redis port"
  value       = module.elasticache.port
}

output "redis_secret_arn" {
  description = "Redis auth token secret ARN"
  value       = module.elasticache.secret_arn
}

# -----------------------------------------------------------------------------
# S3
# -----------------------------------------------------------------------------
output "uploads_bucket_name" {
  description = "Uploads S3 bucket name"
  value       = module.s3.uploads_bucket_name
}

output "uploads_bucket_arn" {
  description = "Uploads S3 bucket ARN"
  value       = module.s3.uploads_bucket_arn
}

output "frontend_bucket_name" {
  description = "Frontend S3 bucket name"
  value       = module.s3.frontend_bucket_name
}

# -----------------------------------------------------------------------------
# CloudFront
# -----------------------------------------------------------------------------
output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID"
  value       = module.cdn.distribution_id
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name"
  value       = module.cdn.domain_name
}

# -----------------------------------------------------------------------------
# ALB
# -----------------------------------------------------------------------------
output "alb_dns_name" {
  description = "ALB DNS name"
  value       = module.alb.dns_name
}

output "alb_arn" {
  description = "ALB ARN"
  value       = module.alb.arn
}

output "alb_security_group_id" {
  description = "ALB security group ID"
  value       = module.security.alb_security_group_id
}

# -----------------------------------------------------------------------------
# Security Groups
# -----------------------------------------------------------------------------
output "database_security_group_id" {
  description = "Database security group ID"
  value       = module.security.database_security_group_id
}

output "cache_security_group_id" {
  description = "Cache security group ID"
  value       = module.security.cache_security_group_id
}

# -----------------------------------------------------------------------------
# Bastion
# -----------------------------------------------------------------------------
output "bastion_public_ip" {
  description = "Bastion public IP"
  value       = module.security.bastion_public_ip
}

# -----------------------------------------------------------------------------
# Monitoring
# -----------------------------------------------------------------------------
output "sns_topic_critical_arn" {
  description = "Critical alerts SNS topic ARN"
  value       = module.monitoring.sns_topic_critical_arn
}

output "sns_topic_warning_arn" {
  description = "Warning alerts SNS topic ARN"
  value       = module.monitoring.sns_topic_warning_arn
}

output "cloudwatch_dashboards" {
  description = "CloudWatch dashboard names"
  value       = module.monitoring.dashboard_names
}

# -----------------------------------------------------------------------------
# Application Configuration
# -----------------------------------------------------------------------------
output "backend_config" {
  description = "Backend application configuration"
  value = {
    database_host    = module.rds.address
    database_port    = module.rds.port
    database_name    = module.rds.database_name
    database_secret  = module.rds.secret_name
    redis_host       = module.elasticache.endpoint
    redis_port       = module.elasticache.port
    redis_secret     = module.elasticache.secret_name
    s3_bucket        = module.s3.uploads_bucket_name
    ses_config_set   = module.ses.configuration_set_name
    backend_irsa_arn = module.eks.backend_irsa_role_arn
  }
}

output "frontend_config" {
  description = "Frontend application configuration"
  value = {
    s3_bucket         = module.s3.frontend_bucket_name
    cloudfront_id     = module.cdn.distribution_id
    cloudfront_domain = module.cdn.domain_name
  }
}
