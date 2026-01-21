# =============================================================================
# SES Module
# =============================================================================

resource "aws_ses_domain_identity" "main" {
  count  = var.domain != "" ? 1 : 0
  domain = var.domain
}

resource "aws_route53_record" "verification" {
  count   = var.domain != "" && var.route53_zone_id != "" ? 1 : 0
  zone_id = var.route53_zone_id
  name    = "_amazonses.${var.domain}"
  type    = "TXT"
  ttl     = 600
  records = [aws_ses_domain_identity.main[0].verification_token]
}

resource "aws_ses_domain_identity_verification" "main" {
  count  = var.domain != "" && var.route53_zone_id != "" ? 1 : 0
  domain = aws_ses_domain_identity.main[0].id

  depends_on = [aws_route53_record.verification]
}

resource "aws_ses_domain_dkim" "main" {
  count  = var.domain != "" ? 1 : 0
  domain = aws_ses_domain_identity.main[0].domain
}

resource "aws_route53_record" "dkim" {
  count   = var.domain != "" && var.route53_zone_id != "" ? 3 : 0
  zone_id = var.route53_zone_id
  name    = "${aws_ses_domain_dkim.main[0].dkim_tokens[count.index]}._domainkey.${var.domain}"
  type    = "CNAME"
  ttl     = 600
  records = ["${aws_ses_domain_dkim.main[0].dkim_tokens[count.index]}.dkim.amazonses.com"]
}

resource "aws_ses_domain_mail_from" "main" {
  count            = var.domain != "" ? 1 : 0
  domain           = aws_ses_domain_identity.main[0].domain
  mail_from_domain = "mail.${var.domain}"
}

resource "aws_route53_record" "mail_from_mx" {
  count   = var.domain != "" && var.route53_zone_id != "" ? 1 : 0
  zone_id = var.route53_zone_id
  name    = "mail.${var.domain}"
  type    = "MX"
  ttl     = 600
  records = ["10 feedback-smtp.${var.aws_region}.amazonses.com"]
}

resource "aws_route53_record" "mail_from_spf" {
  count   = var.domain != "" && var.route53_zone_id != "" ? 1 : 0
  zone_id = var.route53_zone_id
  name    = "mail.${var.domain}"
  type    = "TXT"
  ttl     = 600
  records = ["v=spf1 include:amazonses.com ~all"]
}

resource "aws_route53_record" "dmarc" {
  count   = var.domain != "" && var.route53_zone_id != "" ? 1 : 0
  zone_id = var.route53_zone_id
  name    = "_dmarc.${var.domain}"
  type    = "TXT"
  ttl     = 600
  records = ["v=DMARC1; p=quarantine; rua=mailto:dmarc-reports@${var.domain}"]
}

resource "aws_ses_configuration_set" "main" {
  count = var.domain != "" ? 1 : 0
  name  = "${var.name_prefix}-config-set"

  delivery_options {
    tls_policy = "Require"
  }

  reputation_metrics_enabled = true
  sending_enabled            = true
}

resource "aws_ses_event_destination" "cloudwatch" {
  count                  = var.domain != "" ? 1 : 0
  name                   = "${var.name_prefix}-cloudwatch"
  configuration_set_name = aws_ses_configuration_set.main[0].name
  enabled                = true
  matching_types         = ["bounce", "complaint", "delivery", "reject", "send"]

  cloudwatch_destination {
    default_value  = "default"
    dimension_name = "ses:source-ip"
    value_source   = "messageTag"
  }
}

resource "aws_sns_topic" "notifications" {
  count = var.domain != "" ? 1 : 0
  name  = "${var.name_prefix}-ses-notifications"

  tags = var.tags
}

resource "aws_ses_identity_notification_topic" "bounce" {
  count                    = var.domain != "" ? 1 : 0
  topic_arn                = aws_sns_topic.notifications[0].arn
  notification_type        = "Bounce"
  identity                 = aws_ses_domain_identity.main[0].domain
  include_original_headers = true
}

resource "aws_ses_identity_notification_topic" "complaint" {
  count                    = var.domain != "" ? 1 : 0
  topic_arn                = aws_sns_topic.notifications[0].arn
  notification_type        = "Complaint"
  identity                 = aws_ses_domain_identity.main[0].domain
  include_original_headers = true
}

# Email Templates
resource "aws_ses_template" "welcome" {
  count   = var.domain != "" ? 1 : 0
  name    = "${var.name_prefix}-welcome"
  subject = "Welcome to ${var.project_name}!"
  html    = <<-HTML
    <!DOCTYPE html>
    <html>
    <head><meta charset="utf-8"></head>
    <body>
      <h1>Welcome, {{name}}!</h1>
      <p>Thank you for signing up for ${var.project_name}.</p>
      <p><a href="{{verification_url}}">Verify Email</a></p>
    </body>
    </html>
  HTML
  text    = <<-TEXT
    Welcome, {{name}}!
    Thank you for signing up for ${var.project_name}.
    Verify: {{verification_url}}
  TEXT
}

resource "aws_ses_template" "password_reset" {
  count   = var.domain != "" ? 1 : 0
  name    = "${var.name_prefix}-password-reset"
  subject = "Reset Your ${var.project_name} Password"
  html    = <<-HTML
    <!DOCTYPE html>
    <html>
    <head><meta charset="utf-8"></head>
    <body>
      <h1>Password Reset Request</h1>
      <p>Hi {{name}},</p>
      <p><a href="{{reset_url}}">Reset Password</a></p>
      <p>This link expires in 1 hour.</p>
    </body>
    </html>
  HTML
  text    = <<-TEXT
    Password Reset Request
    Hi {{name}},
    Reset: {{reset_url}}
    This link expires in 1 hour.
  TEXT
}

# CloudWatch Alarms
resource "aws_cloudwatch_metric_alarm" "bounce_rate" {
  count               = var.domain != "" ? 1 : 0
  alarm_name          = "${var.name_prefix}-ses-bounce-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Reputation.BounceRate"
  namespace           = "AWS/SES"
  period              = 300
  statistic           = "Average"
  threshold           = 0.05
  alarm_description   = "SES bounce rate is above 5%"
  alarm_actions       = var.alarm_actions
  ok_actions          = var.ok_actions

  dimensions = {
    ConfigurationSetName = aws_ses_configuration_set.main[0].name
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "complaint_rate" {
  count               = var.domain != "" ? 1 : 0
  alarm_name          = "${var.name_prefix}-ses-complaint-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "Reputation.ComplaintRate"
  namespace           = "AWS/SES"
  period              = 300
  statistic           = "Average"
  threshold           = 0.001
  alarm_description   = "SES complaint rate is above 0.1%"
  alarm_actions       = var.critical_alarm_actions
  ok_actions          = var.ok_actions

  dimensions = {
    ConfigurationSetName = aws_ses_configuration_set.main[0].name
  }

  tags = var.tags
}
