output "sns_topic_critical_arn" {
  description = "Critical alerts SNS topic ARN"
  value       = aws_sns_topic.critical.arn
}

output "sns_topic_warning_arn" {
  description = "Warning alerts SNS topic ARN"
  value       = aws_sns_topic.warning.arn
}

output "sns_topic_info_arn" {
  description = "Info alerts SNS topic ARN"
  value       = aws_sns_topic.info.arn
}

output "log_group_application_name" {
  description = "Application log group name"
  value       = aws_cloudwatch_log_group.application.name
}

output "log_group_api_name" {
  description = "API log group name"
  value       = aws_cloudwatch_log_group.api.name
}

output "dashboard_names" {
  description = "CloudWatch dashboard names"
  value       = { for k, v in aws_cloudwatch_dashboard.dashboards : k => v.dashboard_name }
}

output "dashboard_arns" {
  description = "CloudWatch dashboard ARNs"
  value       = { for k, v in aws_cloudwatch_dashboard.dashboards : k => v.dashboard_arn }
}
