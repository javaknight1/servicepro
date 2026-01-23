package ses

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

func init() {
	email.RegisterProvider(email.ProviderSES, func(ctx context.Context, cfg *config.Config) (email.Client, error) {
		sesCfg := &Config{
			Region:          cfg.AWS.Region,
			AccessKeyID:     cfg.AWS.AccessKeyID,
			SecretAccessKey: cfg.AWS.SecretAccessKey,
			FromEmail:       cfg.SES.FromEmail,
			FromName:        cfg.SES.FromName,
			ReplyTo:         cfg.SES.ReplyTo,
		}
		return NewClient(ctx, sesCfg)
	})
}

// Config holds configuration for the SES email client
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	FromEmail       string
	FromName        string
	ReplyTo         string
}

// Client implements the email.Client interface using AWS SES
type Client struct {
	client   *ses.Client
	config   *Config
	fromAddr string
	replyTo  string
}

// NewClient creates a new SES email client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region is required")
	}

	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}

	// Build AWS config options
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.Region))

	// Use custom credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SES client
	sesClient := ses.NewFromConfig(awsCfg)

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
		client:   sesClient,
		config:   cfg,
		fromAddr: fromAddr,
		replyTo:  replyTo,
	}, nil
}

// Send implements email.Client
func (c *Client) Send(ctx context.Context, msg *email.EmailMessage) (*email.SendResult, error) {
	// Build destination
	dest := &types.Destination{
		ToAddresses: msg.To,
	}
	if len(msg.CC) > 0 {
		dest.CcAddresses = msg.CC
	}
	if len(msg.BCC) > 0 {
		dest.BccAddresses = msg.BCC
	}

	// Build message body
	body := &types.Body{}
	if msg.HTMLBody != "" {
		body.Html = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(msg.HTMLBody),
		}
	}
	if msg.TextBody != "" {
		body.Text = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(msg.TextBody),
		}
	}

	// Determine from address
	from := c.fromAddr
	if msg.From != "" {
		if msg.FromName != "" {
			from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
		} else {
			from = msg.From
		}
	}

	// Build send email input
	input := &ses.SendEmailInput{
		Destination: dest,
		Message: &types.Message{
			Body: body,
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(msg.Subject),
			},
		},
		Source: aws.String(from),
	}

	// Set reply-to
	replyTo := c.replyTo
	if msg.ReplyTo != "" {
		replyTo = msg.ReplyTo
	}
	if replyTo != "" {
		input.ReplyToAddresses = []string{replyTo}
	}

	// Send the email
	result, err := c.client.SendEmail(ctx, input)
	if err != nil {
		return &email.SendResult{
			Success:  false,
			Provider: email.ProviderSES,
			SentAt:   time.Now(),
			Error:    err.Error(),
		}, fmt.Errorf("failed to send email via SES: %w", err)
	}

	return &email.SendResult{
		Success:   true,
		MessageID: aws.ToString(result.MessageId),
		Provider:  email.ProviderSES,
		SentAt:    time.Now(),
		Metadata: map[string]string{
			"ses_message_id": aws.ToString(result.MessageId),
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

// HealthCheck implements email.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	// Try to get account sending statistics as a health check
	_, err := c.client.GetSendQuota(ctx, &ses.GetSendQuotaInput{})
	if err != nil {
		return fmt.Errorf("SES health check failed: %w", err)
	}
	log.Printf("SES health check passed")
	return nil
}

// Close implements email.Client
func (c *Client) Close() error {
	// SES client doesn't require explicit cleanup
	return nil
}

// Ensure Client implements email.Client
var _ email.Client = (*Client)(nil)
