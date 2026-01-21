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
