output "endpoint" {
  description = "Redis primary endpoint"
  value       = aws_elasticache_replication_group.main.primary_endpoint_address
}

output "port" {
  description = "Redis port"
  value       = 6379
}

output "id" {
  description = "Replication group ID"
  value       = aws_elasticache_replication_group.main.id
}

output "secret_arn" {
  description = "Auth token secret ARN"
  value       = var.transit_encryption ? aws_secretsmanager_secret.auth[0].arn : null
}

output "secret_name" {
  description = "Auth token secret name"
  value       = var.transit_encryption ? aws_secretsmanager_secret.auth[0].name : null
}
