# =============================================================================
# Monitoring Module
# =============================================================================

# -----------------------------------------------------------------------------
# SNS Topics
# -----------------------------------------------------------------------------
resource "aws_sns_topic" "critical" {
  name         = "${var.name_prefix}-alerts-critical"
  display_name = "Critical Alerts"

  tags = var.tags
}

resource "aws_sns_topic" "warning" {
  name         = "${var.name_prefix}-alerts-warning"
  display_name = "Warning Alerts"

  tags = var.tags
}

resource "aws_sns_topic" "info" {
  name         = "${var.name_prefix}-alerts-info"
  display_name = "Info Alerts"

  tags = var.tags
}

resource "aws_sns_topic_subscription" "email" {
  for_each  = toset(var.alert_emails)
  topic_arn = aws_sns_topic.critical.arn
  protocol  = "email"
  endpoint  = each.value
}

resource "aws_sns_topic_subscription" "slack" {
  count     = var.slack_webhook_url != "" ? 1 : 0
  topic_arn = aws_sns_topic.critical.arn
  protocol  = "https"
  endpoint  = var.slack_webhook_url
}

# -----------------------------------------------------------------------------
# CloudWatch Log Groups
# -----------------------------------------------------------------------------
resource "aws_cloudwatch_log_group" "application" {
  name              = "/aws/eks/${var.name_prefix}/application"
  retention_in_days = var.log_retention_days

  tags = var.tags
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/eks/${var.name_prefix}/api"
  retention_in_days = var.log_retention_days

  tags = var.tags
}

# -----------------------------------------------------------------------------
# Local values for dashboard templates
# -----------------------------------------------------------------------------
locals {
  dashboard_template_vars = {
    ENVIRONMENT      = var.environment
    AWS_REGION       = var.aws_region
    AWS_ACCOUNT_ID   = var.aws_account_id
    PROJECT          = var.project_name
    CLUSTER_NAME     = var.eks_cluster_name
    SERVICE_NAME     = var.service_name
    DB_INSTANCE_ID   = var.rds_identifier
    REDIS_CLUSTER_ID = var.elasticache_cluster_id
    TIMESTAMP        = "Updated by Terraform"
  }

  # List of dashboards to create
  dashboards = var.enable_dashboards ? {
    "application" = {
      name = "${var.name_prefix}-application"
      file = "${path.module}/dashboards/application.json"
    }
    "database" = {
      name = "${var.name_prefix}-database"
      file = "${path.module}/dashboards/database.json"
    }
    "api" = {
      name = "${var.name_prefix}-api"
      file = "${path.module}/dashboards/api.json"
    }
    "api-metrics" = {
      name = "${var.name_prefix}-api-metrics"
      file = "${path.module}/dashboards/api-metrics.json"
    }
  } : {}
}

# -----------------------------------------------------------------------------
# CloudWatch Dashboards from JSON Templates
# -----------------------------------------------------------------------------
resource "aws_cloudwatch_dashboard" "dashboards" {
  for_each = local.dashboards

  dashboard_name = each.value.name
  dashboard_body = templatefile(each.value.file, local.dashboard_template_vars)
}

# -----------------------------------------------------------------------------
# ALB Alarms (only if ALB ARN suffix is provided)
# -----------------------------------------------------------------------------
resource "aws_cloudwatch_metric_alarm" "alb_5xx" {
  count               = var.alb_arn_suffix != "" ? 1 : 0
  alarm_name          = "${var.name_prefix}-alb-5xx"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "HTTPCode_ELB_5XX_Count"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Sum"
  threshold           = 10
  alarm_description   = "ALB returning 5xx errors"
  alarm_actions       = [aws_sns_topic.critical.arn]
  ok_actions          = [aws_sns_topic.info.arn]
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "alb_latency" {
  count               = var.alb_arn_suffix != "" && var.target_group_arn_suffix != "" ? 1 : 0
  alarm_name          = "${var.name_prefix}-alb-latency"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "TargetResponseTime"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  extended_statistic  = "p95"
  threshold           = 2.0
  alarm_description   = "API p95 latency above 2s"
  alarm_actions       = [aws_sns_topic.warning.arn]
  ok_actions          = [aws_sns_topic.info.arn]
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "alb_unhealthy_hosts" {
  count               = var.alb_arn_suffix != "" && var.target_group_arn_suffix != "" ? 1 : 0
  alarm_name          = "${var.name_prefix}-unhealthy-hosts"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "UnHealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Average"
  threshold           = 0
  alarm_description   = "Unhealthy targets detected"
  alarm_actions       = [aws_sns_topic.critical.arn]
  ok_actions          = [aws_sns_topic.info.arn]
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  tags = var.tags
}
