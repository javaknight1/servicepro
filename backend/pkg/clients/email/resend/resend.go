package resend

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/resend/resend-go/v2"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

func init() {
	email.RegisterProvider(email.ProviderResend, func(ctx context.Context, cfg *config.Config) (email.Client, error) {
		resendCfg := &Config{
			APIKey:    cfg.Resend.APIKey,
			FromEmail: cfg.Resend.FromEmail,
			FromName:  cfg.Resend.FromName,
		}
		return NewClient(resendCfg)
	})
}

// Config holds configuration for the Resend email client
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	ReplyTo   string
}

// Client implements the email.Client interface using Resend
type Client struct {
	client   *resend.Client
	config   *Config
	fromAddr string
	replyTo  string
}

// NewClient creates a new Resend email client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Resend API key is required")
	}

	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}

	// Create Resend client
	client := resend.NewClient(cfg.APIKey)

	// Build from address
	fromAddr := cfg.FromEmail
	if cfg.FromName != "" {
		fromAddr = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	}

	// Build reply-to address
	replyTo := cfg.ReplyTo
	if replyTo == "" {
		replyTo = cfg.FromEmail
	}

	return &Client{
		client:   client,
		config:   cfg,
		fromAddr: fromAddr,
		replyTo:  replyTo,
	}, nil
}

// Send implements email.Client
func (c *Client) Send(ctx context.Context, msg *email.EmailMessage) (*email.SendResult, error) {
	// Determine from address
	from := c.fromAddr
	if msg.From != "" {
		if msg.FromName != "" {
			from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
		} else {
			from = msg.From
		}
	}

	// Build send email request
	params := &resend.SendEmailRequest{
		From:    from,
		To:      msg.To,
		Subject: msg.Subject,
	}

	if msg.HTMLBody != "" {
		params.Html = msg.HTMLBody
	}
	if msg.TextBody != "" {
		params.Text = msg.TextBody
	}

	if len(msg.CC) > 0 {
		params.Cc = msg.CC
	}
	if len(msg.BCC) > 0 {
		params.Bcc = msg.BCC
	}

	// Set reply-to
	replyTo := c.replyTo
	if msg.ReplyTo != "" {
		replyTo = msg.ReplyTo
	}
	if replyTo != "" {
		params.ReplyTo = replyTo
	}

	// Add tags if present
	if len(msg.Tags) > 0 {
		tags := make([]resend.Tag, 0, len(msg.Tags))
		for k, v := range msg.Tags {
			tags = append(tags, resend.Tag{Name: k, Value: v})
		}
		params.Tags = tags
	}

	// Add attachments if present
	if len(msg.Attachments) > 0 {
		attachments := make([]*resend.Attachment, 0, len(msg.Attachments))
		for _, att := range msg.Attachments {
			attachments = append(attachments, &resend.Attachment{
				Filename: att.Filename,
				Content:  att.Content,
			})
		}
		params.Attachments = attachments
	}

	// Send the email
	result, err := c.client.Emails.Send(params)
	if err != nil {
		return &email.SendResult{
			Success:  false,
			Provider: email.ProviderResend,
			SentAt:   time.Now(),
			Error:    err.Error(),
		}, fmt.Errorf("failed to send email via Resend: %w", err)
	}

	return &email.SendResult{
		Success:   true,
		MessageID: result.Id,
		Provider:  email.ProviderResend,
		SentAt:    time.Now(),
		Metadata: map[string]string{
			"resend_id": result.Id,
		},
	}, nil
}

// SendWelcomeEmail implements email.Client
func (c *Client) SendWelcomeEmail(ctx context.Context, to, name string) error {
	subject := "Welcome to ServicePro!"
	body := fmt.Sprintf(`
		<html>
		<head></head>
		<body>
			<h1>Welcome to ServicePro, %s!</h1>
			<p>Thank you for registering with us. We're excited to have you on board.</p>
			<p>You can now log in to your account and start using our services.</p>
			<p>If you have any questions, feel free to reach out to our support team.</p>
			<p>Best regards,<br>The ServicePro Team</p>
		</body>
		</html>
	`, name)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendPasswordResetEmail implements email.Client
func (c *Client) SendPasswordResetEmail(ctx context.Context, to, resetToken, resetURL string) error {
	fullResetURL := fmt.Sprintf("%s?token=%s", resetURL, resetToken)
	subject := "Password Reset Request - ServicePro"
	body := fmt.Sprintf(`
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.button {
					display: inline-block;
					padding: 12px 24px;
					background-color: #4CAF50;
					color: white;
					text-decoration: none;
					border-radius: 4px;
					margin: 20px 0;
				}
				.warning { color: #d32f2f; font-weight: bold; margin-top: 20px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Password Reset Request</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>We received a request to reset your password for your ServicePro account.</p>
					<p style="text-align: center;">
						<a href="%s" class="button">Reset Password</a>
					</p>
					<p class="warning">This link will expire in 1 hour.</p>
					<p>If you didn't request this, you can safely ignore this email.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`, fullResetURL)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendPasswordResetConfirmationEmail implements email.Client
func (c *Client) SendPasswordResetConfirmationEmail(ctx context.Context, to string) error {
	subject := "Password Successfully Reset - ServicePro"
	body := `
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.success-icon { font-size: 48px; text-align: center; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Password Reset Successful</h1>
				</div>
				<div class="content">
					<div class="success-icon">&#10003;</div>
					<p>Hello,</p>
					<p>Your password has been successfully reset for your ServicePro account.</p>
					<p>If you did not perform this action, please contact support immediately.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationEmail implements email.Client
func (c *Client) SendEmailVerificationEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	fullVerificationURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)
	subject := "Verify Your Email Address - ServicePro"
	body := fmt.Sprintf(`
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #2196F3; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.button {
					display: inline-block;
					padding: 12px 24px;
					background-color: #2196F3;
					color: white;
					text-decoration: none;
					border-radius: 4px;
					margin: 20px 0;
				}
				.warning { color: #d32f2f; font-weight: bold; margin-top: 20px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Verify Your Email Address</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>Thank you for registering with ServicePro! Please verify your email address:</p>
					<p style="text-align: center;">
						<a href="%s" class="button">Verify Email Address</a>
					</p>
					<p class="warning">This verification link will expire in 24 hours.</p>
					<p>If you didn't create an account, you can safely ignore this email.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`, fullVerificationURL)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationReminderEmail implements email.Client
func (c *Client) SendEmailVerificationReminderEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	fullVerificationURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)
	subject := "Reminder: Verify Your Email Address - ServicePro"
	body := fmt.Sprintf(`
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #FF9800; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.button {
					display: inline-block;
					padding: 12px 24px;
					background-color: #FF9800;
					color: white;
					text-decoration: none;
					border-radius: 4px;
					margin: 20px 0;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Reminder: Verify Your Email</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>We noticed you haven't verified your email address yet.</p>
					<p style="text-align: center;">
						<a href="%s" class="button">Verify Email Address</a>
					</p>
					<p>If you didn't create an account, you can safely ignore this email.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`, fullVerificationURL)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationSuccessEmail implements email.Client
func (c *Client) SendEmailVerificationSuccessEmail(ctx context.Context, to string) error {
	subject := "Email Verified Successfully - ServicePro"
	body := `
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.success-icon { font-size: 48px; text-align: center; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Email Verified Successfully!</h1>
				</div>
				<div class="content">
					<div class="success-icon">&#10003;</div>
					<p>Hello,</p>
					<p>Congratulations! Your email address has been successfully verified.</p>
					<p>You now have full access to all ServicePro features.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendOrganizationInviteEmail implements email.Client
func (c *Client) SendOrganizationInviteEmail(ctx context.Context, to, orgName, inviterName, roleName, actionURL string, userExists bool) error {
	var subject, headline, bodyText, buttonText string

	if userExists {
		subject = fmt.Sprintf("You've been invited to join %s on ServicePro", orgName)
		headline = fmt.Sprintf("Join %s on ServicePro", orgName)
		bodyText = fmt.Sprintf(`
			<p>Hello,</p>
			<p><strong>%s</strong> has invited you to join <strong>%s</strong> as a <strong>%s</strong>.</p>
			<p>Click the button below to accept this invitation and join the organization:</p>
		`, inviterName, orgName, roleName)
		buttonText = "Accept Invitation"
	} else {
		subject = fmt.Sprintf("You've been invited to join %s on ServicePro", orgName)
		headline = fmt.Sprintf("Join %s on ServicePro", orgName)
		bodyText = fmt.Sprintf(`
			<p>Hello,</p>
			<p><strong>%s</strong> has invited you to join <strong>%s</strong> as a <strong>%s</strong>.</p>
			<p>To accept this invitation, you'll need to create a ServicePro account first. Click the button below to get started:</p>
		`, inviterName, orgName, roleName)
		buttonText = "Create Account & Join"
	}

	body := fmt.Sprintf(`
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #2196F3; color: white; padding: 20px; text-align: center; }
				.content { padding: 20px; background-color: #f9f9f9; }
				.button {
					display: inline-block;
					padding: 12px 24px;
					background-color: #2196F3;
					color: white;
					text-decoration: none;
					border-radius: 4px;
					margin: 20px 0;
				}
				.info { color: #666; font-size: 14px; margin-top: 20px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>%s</h1>
				</div>
				<div class="content">
					%s
					<p style="text-align: center;">
						<a href="%s" class="button">%s</a>
					</p>
					<p class="info">This invitation will expire in 7 days.</p>
					<p class="info">If you didn't expect this invitation, you can safely ignore this email.</p>
					<p>Best regards,<br>The ServicePro Team</p>
				</div>
			</div>
		</body>
		</html>
	`, headline, bodyText, actionURL, buttonText)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements email.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	// Resend doesn't have a dedicated health check endpoint
	// We could try to get the API key info or just return nil
	log.Printf("Resend health check: API key configured")
	return nil
}

// Close implements email.Client
func (c *Client) Close() error {
	// Resend client doesn't require explicit cleanup
	return nil
}

// Ensure Client implements email.Client
var _ email.Client = (*Client)(nil)
