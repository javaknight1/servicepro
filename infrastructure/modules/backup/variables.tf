variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "aws_account_id" {
  description = "AWS account ID"
  type        = string
}

variable "retention_days" {
  description = "Backup retention in days"
  type        = number
  default     = 30
}

variable "rds_arn" {
  description = "RDS instance ARN"
  type        = string
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}
