package email

import "log"

// MockEmailService is a mock email service for testing and development
type MockEmailService struct{}

// SendWelcomeEmail sends a welcome email (mocked)
func (m *MockEmailService) SendWelcomeEmail(to, name string) error {
	log.Printf("Mock: Sending welcome email to %s (name: %s)", to, name)
	return nil
}

// SendPasswordResetEmail sends a password reset email (mocked)
func (m *MockEmailService) SendPasswordResetEmail(to, resetToken, resetURL string) error {
	log.Printf("Mock: Sending password reset email to %s - Token: %s, URL: %s", to, resetToken, resetURL)
	return nil
}

// SendPasswordResetConfirmationEmail sends a password reset confirmation email (mocked)
func (m *MockEmailService) SendPasswordResetConfirmationEmail(to string) error {
	log.Printf("Mock: Sending password reset confirmation email to %s", to)
	return nil
}

// SendEmailVerificationEmail sends an email verification email (mocked)
func (m *MockEmailService) SendEmailVerificationEmail(to, verificationToken, verificationURL string) error {
	fullURL := verificationURL + "?token=" + verificationToken
	log.Printf("Mock: Sending email verification to %s", to)
	log.Printf("Mock: *** VERIFICATION LINK: %s ***", fullURL)
	return nil
}

// SendEmailVerificationReminderEmail sends a verification reminder email (mocked)
func (m *MockEmailService) SendEmailVerificationReminderEmail(to, verificationToken, verificationURL string) error {
	log.Printf("Mock: Sending email verification reminder to %s - Token: %s, URL: %s", to, verificationToken, verificationURL)
	return nil
}

// SendEmailVerificationSuccessEmail sends a verification success email (mocked)
func (m *MockEmailService) SendEmailVerificationSuccessEmail(to string) error {
	log.Printf("Mock: Sending email verification success to %s", to)
	return nil
}

// SendEmail sends an email (mocked)
func (m *MockEmailService) SendEmail(to, subject, body string) error {
	log.Printf("Mock: Sending email to %s - Subject: %s", to, subject)
	return nil
}
