package email

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

// EmailService handles email sending operations
type EmailService struct {
	sesClient *ses.SES
	fromEmail string
	awsRegion string
}

// EmailServiceInterface defines the interface for email operations
type EmailServiceInterface interface {
	SendWelcomeEmail(to, name string) error
	SendPasswordResetEmail(to, resetToken, resetURL string) error
	SendPasswordResetConfirmationEmail(to string) error
	SendEmailVerificationEmail(to, verificationToken, verificationURL string) error
	SendEmailVerificationReminderEmail(to, verificationToken, verificationURL string) error
	SendEmailVerificationSuccessEmail(to string) error
	SendEmail(to, subject, body string) error
}

// NewEmailService creates a new email service
func NewEmailService(awsRegion, fromEmail string) (*EmailService, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create SES client
	sesClient := ses.New(sess)

	return &EmailService{
		sesClient: sesClient,
		fromEmail: fromEmail,
		awsRegion: awsRegion,
	}, nil
}

// SendWelcomeEmail sends a welcome email to a new user
func (e *EmailService) SendWelcomeEmail(to, name string) error {
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

	return e.SendEmail(to, subject, body)
}

// SendPasswordResetEmail sends a password reset email with token
func (e *EmailService) SendPasswordResetEmail(to, resetToken, resetURL string) error {
	subject := "Password Reset Request - ServicePro"

	// Construct the full reset URL with token
	fullResetURL := fmt.Sprintf("%s?token=%s", resetURL, resetToken)

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
				.token {
					background-color: #e8e8e8;
					padding: 10px;
					font-family: monospace;
					word-break: break-all;
					margin: 10px 0;
				}
				.warning {
					color: #d32f2f;
					font-weight: bold;
					margin-top: 20px;
				}
				.footer {
					margin-top: 20px;
					padding-top: 20px;
					border-top: 1px solid #ddd;
					font-size: 12px;
					color: #666;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Password Reset Request</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>We received a request to reset your password for your ServicePro account. If you didn't make this request, you can safely ignore this email.</p>

					<p>To reset your password, click the button below:</p>
					<p style="text-align: center;">
						<a href="%s" class="button">Reset Password</a>
					</p>

					<p>Or copy and paste this link into your browser:</p>
					<div class="token">%s</div>

					<p class="warning">⚠️ This link will expire in 1 hour for security reasons.</p>

					<p>If you didn't request a password reset, please ignore this email or contact support if you have concerns.</p>

					<p>Best regards,<br>The ServicePro Team</p>
				</div>
				<div class="footer">
					<p>This is an automated message, please do not reply to this email.</p>
					<p>© 2025 ServicePro. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullResetURL, fullResetURL)

	return e.SendEmail(to, subject, body)
}

// SendPasswordResetConfirmationEmail sends confirmation after successful password reset
func (e *EmailService) SendPasswordResetConfirmationEmail(to string) error {
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
				.alert {
					background-color: #fff3cd;
					border-left: 4px solid #ffc107;
					padding: 15px;
					margin: 20px 0;
				}
				.footer {
					margin-top: 20px;
					padding-top: 20px;
					border-top: 1px solid #ddd;
					font-size: 12px;
					color: #666;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Password Reset Successful</h1>
				</div>
				<div class="content">
					<div class="success-icon">✅</div>
					<p>Hello,</p>
					<p>Your password has been successfully reset for your ServicePro account.</p>

					<div class="alert">
						<p><strong>⚠️ Security Notice:</strong></p>
						<p>If you did not perform this action, please contact our support team immediately as your account may be compromised.</p>
					</div>

					<p>You can now log in with your new password.</p>

					<p>Best regards,<br>The ServicePro Team</p>
				</div>
				<div class="footer">
					<p>This is an automated message, please do not reply to this email.</p>
					<p>© 2025 ServicePro. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`

	return e.SendEmail(to, subject, body)
}

// SendEmailVerificationEmail sends an email verification link to the user
func (e *EmailService) SendEmailVerificationEmail(to, verificationToken, verificationURL string) error {
	subject := "Verify Your Email Address - ServicePro"

	// Construct the full verification URL with token
	fullVerificationURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)

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
				.token {
					background-color: #e8e8e8;
					padding: 10px;
					font-family: monospace;
					word-break: break-all;
					margin: 10px 0;
				}
				.warning {
					color: #d32f2f;
					font-weight: bold;
					margin-top: 20px;
				}
				.footer {
					margin-top: 20px;
					padding-top: 20px;
					border-top: 1px solid #ddd;
					font-size: 12px;
					color: #666;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Verify Your Email Address</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>Thank you for registering with ServicePro! To complete your registration, please verify your email address by clicking the button below:</p>

					<p style="text-align: center;">
						<a href="%s" class="button">Verify Email Address</a>
					</p>

					<p>Or copy and paste this link into your browser:</p>
					<div class="token">%s</div>

					<p class="warning">⚠️ This verification link will expire in 24 hours.</p>

					<p>If you didn't create an account with ServicePro, you can safely ignore this email.</p>

					<p>Best regards,<br>The ServicePro Team</p>
				</div>
				<div class="footer">
					<p>This is an automated message, please do not reply to this email.</p>
					<p>© 2025 ServicePro. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullVerificationURL, fullVerificationURL)

	return e.SendEmail(to, subject, body)
}

// SendEmailVerificationReminderEmail sends a reminder to verify email
func (e *EmailService) SendEmailVerificationReminderEmail(to, verificationToken, verificationURL string) error {
	subject := "Reminder: Verify Your Email Address - ServicePro"

	// Construct the full verification URL with token
	fullVerificationURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)

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
				.token {
					background-color: #e8e8e8;
					padding: 10px;
					font-family: monospace;
					word-break: break-all;
					margin: 10px 0;
				}
				.reminder-box {
					background-color: #fff3e0;
					border-left: 4px solid #FF9800;
					padding: 15px;
					margin: 20px 0;
				}
				.footer {
					margin-top: 20px;
					padding-top: 20px;
					border-top: 1px solid #ddd;
					font-size: 12px;
					color: #666;
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
					<p>We noticed that you haven't verified your email address yet.</p>

					<div class="reminder-box">
						<p><strong>⏰ Why verify your email?</strong></p>
						<ul>
							<li>Secure your account</li>
							<li>Receive important updates</li>
							<li>Enable password recovery</li>
							<li>Access all features</li>
						</ul>
					</div>

					<p>To verify your email address, click the button below:</p>
					<p style="text-align: center;">
						<a href="%s" class="button">Verify Email Address</a>
					</p>

					<p>Or copy and paste this link into your browser:</p>
					<div class="token">%s</div>

					<p>If you didn't create an account with ServicePro, you can safely ignore this email.</p>

					<p>Best regards,<br>The ServicePro Team</p>
				</div>
				<div class="footer">
					<p>This is an automated message, please do not reply to this email.</p>
					<p>© 2025 ServicePro. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, fullVerificationURL, fullVerificationURL)

	return e.SendEmail(to, subject, body)
}

// SendEmailVerificationSuccessEmail sends confirmation after successful email verification
func (e *EmailService) SendEmailVerificationSuccessEmail(to string) error {
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
				.features {
					background-color: #e8f5e9;
					border-left: 4px solid #4CAF50;
					padding: 15px;
					margin: 20px 0;
				}
				.footer {
					margin-top: 20px;
					padding-top: 20px;
					border-top: 1px solid #ddd;
					font-size: 12px;
					color: #666;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Email Verified Successfully!</h1>
				</div>
				<div class="content">
					<div class="success-icon">✅</div>
					<p>Hello,</p>
					<p>Congratulations! Your email address has been successfully verified.</p>

					<div class="features">
						<p><strong>🎉 You now have full access to:</strong></p>
						<ul>
							<li>All ServicePro features</li>
							<li>Password recovery options</li>
							<li>Important account notifications</li>
							<li>Enhanced security features</li>
						</ul>
					</div>

					<p>Thank you for completing your account setup. We're excited to have you as part of the ServicePro community!</p>

					<p>If you have any questions or need assistance, feel free to contact our support team.</p>

					<p>Best regards,<br>The ServicePro Team</p>
				</div>
				<div class="footer">
					<p>This is an automated message, please do not reply to this email.</p>
					<p>© 2025 ServicePro. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`

	return e.SendEmail(to, subject, body)
}

// SendEmail sends an email using AWS SES
func (e *EmailService) SendEmail(to, subject, body string) error {
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(to),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(e.fromEmail),
	}

	_, err := e.sesClient.SendEmail(input)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
