# Staging Environment Configuration

terraform {
  backend "s3" {
    bucket         = "servicepro-terraform-state"
    key            = "staging/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "servicepro-terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = "staging"
      Project     = "servicepro"
      ManagedBy   = "terraform"
    }
  }
}

locals {
  environment = "staging"
  common_tags = {
    Environment = local.environment
    Project     = "servicepro"
    ManagedBy   = "terraform"
  }
}

# VPC Module
module "vpc" {
  source = "../../modules/vpc"

  environment         = local.environment
  vpc_cidr           = var.vpc_cidr
  availability_zones = var.availability_zones
  tags               = local.common_tags
}

# EKS Module
module "eks" {
  source = "../../modules/eks"

  environment        = local.environment
  cluster_name       = "${var.project_name}-${local.environment}"
  cluster_version    = var.eks_cluster_version
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  node_instance_types = var.eks_node_instance_types
  node_desired_size  = var.eks_node_desired_size
  node_min_size      = var.eks_node_min_size
  node_max_size      = var.eks_node_max_size
  tags               = local.common_tags
}

# RDS Module
module "rds" {
  source = "../../modules/rds"

  environment          = local.environment
  identifier           = "${var.project_name}-${local.environment}"
  engine_version       = var.rds_engine_version
  instance_class       = var.rds_instance_class
  allocated_storage    = var.rds_allocated_storage
  database_name        = var.rds_database_name
  master_username      = var.rds_master_username
  vpc_id               = module.vpc.vpc_id
  private_subnet_ids   = module.vpc.private_subnet_ids
  allowed_cidr_blocks  = [module.vpc.vpc_cidr]
  backup_retention_period = 7
  tags                 = local.common_tags
}

# ElastiCache Module
module "elasticache" {
  source = "../../modules/elasticache"

  environment         = local.environment
  cluster_id          = "${var.project_name}-${local.environment}"
  engine_version      = var.redis_engine_version
  node_type           = var.redis_node_type
  num_cache_nodes     = var.redis_num_cache_nodes
  vpc_id              = module.vpc.vpc_id
  private_subnet_ids  = module.vpc.private_subnet_ids
  allowed_cidr_blocks = [module.vpc.vpc_cidr]
  tags                = local.common_tags
}

# S3 Module
module "s3" {
  source = "../../modules/s3"

  environment = local.environment
  bucket_name = "${var.project_name}-${local.environment}-assets"
  tags        = local.common_tags
}
