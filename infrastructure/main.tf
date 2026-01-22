# =============================================================================
# ServicePro Infrastructure - Main Configuration
# =============================================================================
#
# Usage:
#   terraform init
#   terraform plan -var-file="environments/dev.tfvars"
#   terraform apply -var-file="environments/dev.tfvars"
# =============================================================================

# -----------------------------------------------------------------------------
# Data Sources
# -----------------------------------------------------------------------------
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# -----------------------------------------------------------------------------
# Random Resources
# -----------------------------------------------------------------------------
resource "random_id" "suffix" {
  byte_length = 4
}

resource "random_password" "db_password" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# -----------------------------------------------------------------------------
# Local Values
# -----------------------------------------------------------------------------
locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# -----------------------------------------------------------------------------
# Resource Group Module
# -----------------------------------------------------------------------------
module "resource_group" {
  source = "./modules/resource-group"

  name_prefix  = local.name_prefix
  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

# -----------------------------------------------------------------------------
# VPC Module
# -----------------------------------------------------------------------------
module "vpc" {
  source = "./modules/vpc"

  name_prefix          = local.name_prefix
  vpc_cidr             = var.vpc_cidr
  az_count             = var.az_count
  enable_nat_gateway   = var.enable_nat_gateway
  single_nat_gateway   = var.single_nat_gateway
  enable_vpc_flow_logs = var.enable_vpc_flow_logs
  log_retention_days   = var.log_retention_days
  tags                 = local.common_tags
}

# -----------------------------------------------------------------------------
# S3 Module
# -----------------------------------------------------------------------------
module "s3" {
  source = "./modules/s3"

  name_prefix          = local.name_prefix
  cors_allowed_origins = var.cors_allowed_origins
  tags                 = local.common_tags
}

# -----------------------------------------------------------------------------
# Security Module (Phase 1 - ALB SG only, no EKS dependency)
# -----------------------------------------------------------------------------
module "security" {
  source = "./modules/security"

  name_prefix                = local.name_prefix
  vpc_id                     = module.vpc.vpc_id
  vpc_cidr                   = module.vpc.vpc_cidr
  aws_region                 = var.aws_region
  public_subnet_ids          = module.vpc.public_subnet_ids
  private_subnet_ids         = module.vpc.private_subnet_ids
  private_route_table_ids    = module.vpc.private_route_table_ids
  public_route_table_id      = module.vpc.public_route_table_id
  eks_node_security_group_id = module.eks.node_security_group_id
  enable_bastion             = var.enable_bastion
  bastion_instance_type      = var.bastion_instance_type
  bastion_allowed_cidrs      = var.bastion_allowed_cidrs
  enable_vpc_endpoints       = var.enable_vpc_endpoints
  tags                       = local.common_tags
}

# -----------------------------------------------------------------------------
# EKS Module
# -----------------------------------------------------------------------------
module "eks" {
  source = "./modules/eks"

  name_prefix           = local.name_prefix
  environment           = var.environment
  aws_region            = var.aws_region
  aws_account_id        = data.aws_caller_identity.current.account_id
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  alb_security_group_id = module.security.alb_security_group_id
  admin_principal_arn   = data.aws_caller_identity.current.arn
  cluster_version       = var.eks_cluster_version
  public_access         = var.eks_public_access
  node_instance_types   = var.eks_node_instance_types
  node_min_size         = var.eks_node_min_size
  node_max_size         = var.eks_node_max_size
  node_desired_size     = var.eks_node_desired_size
  node_disk_size        = var.eks_node_disk_size
  use_spot              = var.eks_use_spot
  uploads_bucket_arn    = module.s3.uploads_bucket_arn
  enable_argocd         = var.enable_argocd
  argocd_domain         = var.argocd_domain
  certificate_arn       = var.certificate_arn
  tags                  = local.common_tags

  depends_on = [module.vpc, module.s3]
}

# -----------------------------------------------------------------------------
# ALB Module
# -----------------------------------------------------------------------------
module "alb" {
  source = "./modules/alb"

  name_prefix        = local.name_prefix
  environment        = var.environment
  vpc_id             = module.vpc.vpc_id
  public_subnet_ids  = module.vpc.public_subnet_ids
  security_group_id  = module.security.alb_security_group_id
  certificate_arn    = var.certificate_arn
  enable_access_logs = var.enable_alb_access_logs
  log_retention_days = var.alb_log_retention_days
  enable_waf         = var.enable_waf
  waf_rate_limit     = var.waf_rate_limit
  tags               = local.common_tags
}

# -----------------------------------------------------------------------------
# Monitoring Module (SNS Topics, Dashboards)
# -----------------------------------------------------------------------------
module "monitoring" {
  source = "./modules/monitoring"

  name_prefix        = local.name_prefix
  environment        = var.environment
  aws_region         = var.aws_region
  aws_account_id     = data.aws_caller_identity.current.account_id
  project_name       = var.project_name
  log_retention_days = var.log_retention_days
  alert_emails       = var.alert_emails
  slack_webhook_url  = var.slack_webhook_url

  # Dashboard configuration
  enable_dashboards  = var.enable_dashboards
  eks_cluster_name   = module.eks.cluster_name
  service_name       = "servicepro-api"

  # Resource IDs for dashboard metrics (empty strings initially, dashboards use custom metrics)
  alb_arn_suffix          = ""
  target_group_arn_suffix = ""
  rds_identifier          = ""
  elasticache_cluster_id  = ""

  tags = local.common_tags

  depends_on = [module.eks]
}

# -----------------------------------------------------------------------------
# RDS Module
# -----------------------------------------------------------------------------
module "rds" {
  source = "./modules/rds"

  name_prefix             = local.name_prefix
  environment             = var.environment
  engine_version          = var.rds_engine_version
  instance_class          = var.rds_instance_class
  allocated_storage       = var.rds_allocated_storage
  max_allocated_storage   = var.rds_max_allocated_storage
  database_name           = var.rds_database_name
  master_username         = var.rds_master_username
  master_password         = random_password.db_password.result
  multi_az                = var.rds_multi_az
  backup_retention_period = var.rds_backup_retention_period
  performance_insights    = var.rds_performance_insights
  enhanced_monitoring     = var.rds_enhanced_monitoring
  max_connections         = var.rds_max_connections
  kms_key_arn             = var.rds_kms_key_arn
  db_subnet_group_name    = module.vpc.db_subnet_group_name
  security_group_id       = module.security.database_security_group_id
  alarm_actions           = [module.monitoring.sns_topic_warning_arn]
  critical_alarm_actions  = [module.monitoring.sns_topic_critical_arn]
  ok_actions              = [module.monitoring.sns_topic_info_arn]
  tags                    = local.common_tags

  depends_on = [module.vpc, module.security, module.monitoring]
}

# -----------------------------------------------------------------------------
# ElastiCache Module
# -----------------------------------------------------------------------------
module "elasticache" {
  source = "./modules/elasticache"

  name_prefix            = local.name_prefix
  environment            = var.environment
  engine_version         = var.redis_engine_version
  node_type              = var.redis_node_type
  num_cache_nodes        = var.redis_num_cache_nodes
  transit_encryption     = var.redis_transit_encryption
  snapshot_retention     = var.redis_snapshot_retention
  subnet_group_name      = module.vpc.elasticache_subnet_group_name
  security_group_id      = module.security.cache_security_group_id
  notification_topic_arn = module.monitoring.sns_topic_info_arn
  alarm_actions          = [module.monitoring.sns_topic_warning_arn]
  ok_actions             = [module.monitoring.sns_topic_info_arn]
  tags                   = local.common_tags

  depends_on = [module.vpc, module.security, module.monitoring]
}

# -----------------------------------------------------------------------------
# CDN Module
# -----------------------------------------------------------------------------
module "cdn" {
  source = "./modules/cdn"

  name_prefix                 = local.name_prefix
  random_suffix               = random_id.suffix.hex
  domain_name                 = var.domain_name
  certificate_arn             = var.cloudfront_certificate_arn
  frontend_bucket_name        = module.s3.frontend_bucket_name
  frontend_bucket_arn         = module.s3.frontend_bucket_arn
  frontend_bucket_domain_name = module.s3.frontend_bucket_regional_domain_name
  alb_dns_name                = module.alb.dns_name
  web_acl_arn                 = module.alb.waf_web_acl_arn
  price_class                 = var.cloudfront_price_class
  geo_restriction             = var.cloudfront_geo_restriction
  enable_logging              = var.enable_cloudfront_logging
  log_retention_days          = var.cloudfront_log_retention_days
  tags                        = local.common_tags

  depends_on = [module.s3, module.alb]
}

# -----------------------------------------------------------------------------
# SES Module
# -----------------------------------------------------------------------------
module "ses" {
  source = "./modules/ses"

  name_prefix            = local.name_prefix
  project_name           = var.project_name
  aws_region             = var.aws_region
  domain                 = var.ses_domain
  route53_zone_id        = var.route53_zone_id
  alarm_actions          = [module.monitoring.sns_topic_warning_arn]
  critical_alarm_actions = [module.monitoring.sns_topic_critical_arn]
  ok_actions             = [module.monitoring.sns_topic_info_arn]
  tags                   = local.common_tags

  depends_on = [module.monitoring]
}

# -----------------------------------------------------------------------------
# Backup Module
# -----------------------------------------------------------------------------
module "backup" {
  source = "./modules/backup"

  name_prefix    = local.name_prefix
  aws_account_id = data.aws_caller_identity.current.account_id
  retention_days = var.backup_retention_days
  rds_arn        = module.rds.arn
  tags           = local.common_tags

  depends_on = [module.rds]
}
