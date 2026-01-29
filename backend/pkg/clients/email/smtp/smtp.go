package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

func init() {
	email.RegisterProvider(email.ProviderSMTP, func(ctx context.Context, cfg *config.Config) (email.Client, error) {
		smtpCfg := &Config{
			Host:      cfg.SMTP.Host,
			Port:      cfg.SMTP.Port,
			Username:  cfg.SMTP.Username,
			Password:  cfg.SMTP.Password,
			FromEmail: cfg.SMTP.FromEmail,
			FromName:  cfg.SMTP.FromName,
			UseTLS:    cfg.SMTP.UseTLS,
			StartTLS:  cfg.SMTP.StartTLS,
		}
		return NewClient(smtpCfg)
	})
}

// Config holds configuration for the SMTP email client
type Config struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool   // Use implicit TLS (port 465)
	StartTLS  bool   // Use STARTTLS (port 587)
	LocalName string // HELO/EHLO hostname (optional)
}

// Client implements the email.Client interface using SMTP
type Client struct {
	config   *Config
	fromAddr string
}

// NewClient creates a new SMTP email client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}

	if cfg.Port == 0 {
		cfg.Port = 1025 // Default Mailpit port
	}

	if cfg.FromEmail == "" {
		cfg.FromEmail = "noreply@localhost"
	}

	// Build from address
	fromAddr := cfg.FromEmail
	if cfg.FromName != "" {
		fromAddr = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	}

	return &Client{
		config:   cfg,
		fromAddr: fromAddr,
	}, nil
}

// Send implements email.Client
func (c *Client) Send(ctx context.Context, msg *email.EmailMessage) (*email.SendResult, error) {
	// Build the email message
	from := c.config.FromEmail
	if msg.From != "" {
		from = msg.From
	}

	// Collect all recipients
	allRecipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	allRecipients = append(allRecipients, msg.To...)
	allRecipients = append(allRecipients, msg.CC...)
	allRecipients = append(allRecipients, msg.BCC...)

	// Build email headers and body
	var emailBuilder strings.Builder

	// From header
	fromHeader := c.fromAddr
	if msg.From != "" {
		if msg.FromName != "" {
			fromHeader = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
		} else {
			fromHeader = msg.From
		}
	}
	emailBuilder.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))

	// To header
	emailBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))

	// CC header (if any)
	if len(msg.CC) > 0 {
		emailBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}

	// Reply-To header
	replyTo := msg.ReplyTo
	if replyTo == "" {
		replyTo = from
	}
	emailBuilder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))

	// Subject
	emailBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// Date
	emailBuilder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Message-ID
	messageID := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), "smtp", c.config.Host)
	emailBuilder.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))

	// MIME headers
	emailBuilder.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for key, value := range msg.Headers {
		emailBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Content type and body
	if msg.HTMLBody != "" && msg.TextBody != "" {
		// Multipart message
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		emailBuilder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		emailBuilder.WriteString("\r\n")

		// Text part
		emailBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		emailBuilder.WriteString("\r\n")
		emailBuilder.WriteString(msg.TextBody)
		emailBuilder.WriteString("\r\n")

		// HTML part
		emailBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		emailBuilder.WriteString("\r\n")
		emailBuilder.WriteString(msg.HTMLBody)
		emailBuilder.WriteString("\r\n")

		// End boundary
		emailBuilder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.HTMLBody != "" {
		emailBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		emailBuilder.WriteString("\r\n")
		emailBuilder.WriteString(msg.HTMLBody)
	} else {
		emailBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		emailBuilder.WriteString("\r\n")
		emailBuilder.WriteString(msg.TextBody)
	}

	// Send via SMTP
	err := c.sendMail(from, allRecipients, []byte(emailBuilder.String()))
	if err != nil {
		return &email.SendResult{
			Success:  false,
			Provider: email.ProviderSMTP,
			SentAt:   time.Now(),
			Error:    err.Error(),
		}, fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return &email.SendResult{
		Success:   true,
		MessageID: messageID,
		Provider:  email.ProviderSMTP,
		SentAt:    time.Now(),
		Metadata: map[string]string{
			"smtp_host": c.config.Host,
		},
	}, nil
}

// sendMail sends the email using the configured SMTP settings
func (c *Client) sendMail(from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// For Mailpit and other dev servers, we use plain SMTP without auth
	if c.config.Username == "" && c.config.Password == "" && !c.config.UseTLS && !c.config.StartTLS {
		return smtp.SendMail(addr, nil, from, to, msg)
	}

	// For production SMTP servers with authentication
	var auth smtp.Auth
	if c.config.Username != "" && c.config.Password != "" {
		auth = smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
	}

	// Connect to the server
	var conn net.Conn
	var err error

	if c.config.UseTLS {
		// Implicit TLS (port 465)
		tlsConfig := &tls.Config{
			ServerName: c.config.Host,
		}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// STARTTLS if required
	if c.config.StartTLS && !c.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: c.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
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

// SendInvoiceEmail implements email.Client
func (c *Client) SendInvoiceEmail(ctx context.Context, to string, invoice *models.Invoice, paymentURL string) error {
	customerName := "Valued Customer"
	if invoice.Customer != nil {
		if invoice.Customer.CompanyName != nil && *invoice.Customer.CompanyName != "" {
			customerName = *invoice.Customer.CompanyName
		} else {
			customerName = invoice.Customer.FirstName + " " + invoice.Customer.LastName
		}
	}

	subject := fmt.Sprintf("Invoice %s from ServicePro", invoice.InvoiceNumber)
	body := fmt.Sprintf(`
		<html>
		<body>
			<h1>Invoice from ServicePro</h1>
			<p>Hello %s,</p>
			<p>Please find your invoice details below:</p>
			<p><strong>Invoice Number:</strong> %s</p>
			<p><strong>Total Amount:</strong> $%s</p>
			<p><strong>Due Date:</strong> %s</p>
			<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none;">Pay Now</a></p>
			<p>Best regards,<br>The ServicePro Team</p>
		</body>
		</html>
	`, customerName, invoice.InvoiceNumber, invoice.TotalAmount.StringFixed(2), invoice.DueDate.Format("January 2, 2006"), paymentURL)

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// SendPaymentReceiptEmail implements email.Client
func (c *Client) SendPaymentReceiptEmail(ctx context.Context, to string, invoice *models.Invoice) error {
	customerName := "Valued Customer"
	if invoice.Customer != nil {
		if invoice.Customer.CompanyName != nil && *invoice.Customer.CompanyName != "" {
			customerName = *invoice.Customer.CompanyName
		} else {
			customerName = invoice.Customer.FirstName + " " + invoice.Customer.LastName
		}
	}

	subject := fmt.Sprintf("Payment Receipt for Invoice %s - ServicePro", invoice.InvoiceNumber)
	body := fmt.Sprintf(`
		<html>
		<body>
			<h1>Payment Received!</h1>
			<p>Hello %s,</p>
			<p>Thank you for your payment! This confirms we have received payment for:</p>
			<p><strong>Invoice Number:</strong> %s</p>
			<p><strong>Amount Paid:</strong> $%s</p>
			<p>Best regards,<br>The ServicePro Team</p>
		</body>
		</html>
	`, customerName, invoice.InvoiceNumber, invoice.AmountPaid.StringFixed(2))

	msg := email.NewEmailMessage(to, subject, body)
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements email.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP health check failed: %w", err)
	}
	conn.Close()
	log.Printf("SMTP health check passed: %s", addr)
	return nil
}

// Close implements email.Client
func (c *Client) Close() error {
	// SMTP client doesn't maintain persistent connections
	return nil
}

// Ensure Client implements email.Client
var _ email.Client = (*Client)(nil)
