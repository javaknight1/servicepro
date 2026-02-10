package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

// MockEmailService is a mock implementation of email.Client interface
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) Send(ctx context.Context, msg *email.EmailMessage) (*email.SendResult, error) {
	args := m.Called(ctx, msg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*email.SendResult), args.Error(1)
}

func (m *MockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	args := m.Called(ctx, to, name)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetEmail(ctx context.Context, to, resetToken, resetURL string) error {
	args := m.Called(ctx, to, resetToken, resetURL)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetConfirmationEmail(ctx context.Context, to string) error {
	args := m.Called(ctx, to)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	args := m.Called(ctx, to, verificationToken, verificationURL)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationReminderEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	args := m.Called(ctx, to, verificationToken, verificationURL)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationSuccessEmail(ctx context.Context, to string) error {
	args := m.Called(ctx, to)
	return args.Error(0)
}

func (m *MockEmailService) SendOrganizationInviteEmail(ctx context.Context, to, orgName, inviterName, roleName, actionURL string, userExists bool) error {
	args := m.Called(ctx, to, orgName, inviterName, roleName, actionURL, userExists)
	return args.Error(0)
}

func (m *MockEmailService) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEmailService) SendQuoteEmail(ctx context.Context, to string, quote *models.Quote, pdfAttachment *email.Attachment, downloadURL string) error {
	args := m.Called(ctx, to, quote, pdfAttachment, downloadURL)
	return args.Error(0)
}

func (m *MockEmailService) SendInvoiceEmail(ctx context.Context, to string, invoice *models.Invoice, paymentURL string, pdfAttachment *email.Attachment, downloadURL string) error {
	args := m.Called(ctx, to, invoice, paymentURL, pdfAttachment, downloadURL)
	return args.Error(0)
}

func (m *MockEmailService) SendPaymentReceiptEmail(ctx context.Context, to string, invoice *models.Invoice, pdfAttachment *email.Attachment, downloadURL string) error {
	args := m.Called(ctx, to, invoice, pdfAttachment, downloadURL)
	return args.Error(0)
}

// MockSMSClient is a mock implementation of sms.Client interface
type MockSMSClient struct {
	mock.Mock
}

func (m *MockSMSClient) Send(ctx context.Context, msg *sms.SMSMessage) (*sms.SendResult, error) {
	args := m.Called(ctx, msg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sms.SendResult), args.Error(1)
}

func (m *MockSMSClient) SendBatch(ctx context.Context, messages []*sms.SMSMessage) []sms.SendResult {
	args := m.Called(ctx, messages)
	return args.Get(0).([]sms.SendResult)
}

func (m *MockSMSClient) SendOTP(ctx context.Context, phoneNumber, otp string, expiryMinutes int) error {
	args := m.Called(ctx, phoneNumber, otp, expiryMinutes)
	return args.Error(0)
}

func (m *MockSMSClient) SendNotification(ctx context.Context, phoneNumber, message string) error {
	args := m.Called(ctx, phoneNumber, message)
	return args.Error(0)
}

func (m *MockSMSClient) SendJobUpdate(ctx context.Context, phoneNumber, jobNumber, status, message string) error {
	args := m.Called(ctx, phoneNumber, jobNumber, status, message)
	return args.Error(0)
}

func (m *MockSMSClient) SendAppointmentReminder(ctx context.Context, phoneNumber, appointmentTime, jobDescription string) error {
	args := m.Called(ctx, phoneNumber, appointmentTime, jobDescription)
	return args.Error(0)
}

func (m *MockSMSClient) SendInvoiceNotification(ctx context.Context, phoneNumber, invoiceNumber, amount, paymentURL string) error {
	args := m.Called(ctx, phoneNumber, invoiceNumber, amount, paymentURL)
	return args.Error(0)
}

func (m *MockSMSClient) SendPaymentConfirmation(ctx context.Context, phoneNumber, invoiceNumber, amount string) error {
	args := m.Called(ctx, phoneNumber, invoiceNumber, amount)
	return args.Error(0)
}

func (m *MockSMSClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSMSClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSMSClient) GetProviderInfo() sms.ProviderInfo {
	args := m.Called()
	return args.Get(0).(sms.ProviderInfo)
}

// TestNewNotificationService tests service creation
func TestNewNotificationService(t *testing.T) {
	mockEmailService := new(MockEmailService)

	service, err := NewNotificationService(mockEmailService, nil)

	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.NotNil(t, service.emailClient)
	assert.Nil(t, service.smsClient) // SMS client is optional
	assert.NotNil(t, service.templates)
}

// TestNewNotificationService_WithSMS tests service creation with SMS client
func TestNewNotificationService_WithSMS(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)

	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.NotNil(t, service.emailClient)
	assert.NotNil(t, service.smsClient)
	assert.NotNil(t, service.templates)
}

// TestSendNotification_Email tests sending email notification
func TestSendNotification_Email(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "tech@example.com",
		Channel:        ChannelEmail,
		Priority:       PriorityNormal,
		Subject:        "Test Subject",
		TemplateData: map[string]interface{}{
			"Message": "Test message",
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.AnythingOfType("*email.EmailMessage")).Return(&email.SendResult{Success: true}, nil)

	result, err := service.SendNotification(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, ChannelEmail, result.Channel)
	mockEmailService.AssertExpectations(t)
}

// TestSendNotification_EmailError tests email notification error
func TestSendNotification_EmailError(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "tech@example.com",
		Channel:        ChannelEmail,
		Priority:       PriorityNormal,
		Subject:        "Test Subject",
		TemplateData: map[string]interface{}{
			"Message": "Test message",
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.AnythingOfType("*email.EmailMessage")).Return(nil, errors.New("email error"))

	result, err := service.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	mockEmailService.AssertExpectations(t)
}

// TestSendNotification_SMS tests sending SMS notification
func TestSendNotification_SMS(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "tech@example.com",
		RecipientPhone: "+1234567890",
		Channel:        ChannelSMS,
		Priority:       PriorityHigh,
		Subject:        "Test SMS",
		TemplateData: map[string]interface{}{
			"Message": "Test SMS message",
		},
	}

	mockSMSClient.On("SendNotification", ctx, "+1234567890", mock.Anything).Return(nil)

	result, err := service.SendNotification(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, ChannelSMS, result.Channel)
	mockSMSClient.AssertExpectations(t)
}

// TestSendNotification_SMS_NoClient tests SMS notification when client is nil
func TestSendNotification_SMS_NoClient(t *testing.T) {
	mockEmailService := new(MockEmailService)

	service, err := NewNotificationService(mockEmailService, nil) // No SMS client
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "tech@example.com",
		RecipientPhone: "+1234567890",
		Channel:        ChannelSMS,
		Priority:       PriorityHigh,
		Subject:        "Test SMS",
		TemplateData: map[string]interface{}{
			"Message": "Test SMS message",
		},
	}

	// Should succeed silently when SMS client is not configured
	result, err := service.SendNotification(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success) // Succeeds by skipping
}

// TestSendNotification_InvalidChannel tests invalid notification channel
func TestSendNotification_InvalidChannel(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "tech@example.com",
		Channel:        NotificationChannel("invalid"),
		Priority:       PriorityNormal,
		Subject:        "Test Subject",
		TemplateData:   map[string]interface{}{},
	}

	result, err := service.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestSendMultiChannelNotification_Success tests multi-channel notification
func TestSendMultiChannelNotification_Success(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	reqs := []*NotificationRequest{
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Email Notification",
			TemplateData:   map[string]interface{}{"Message": "Email message"},
		},
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech@example.com",
			RecipientPhone: "+1234567890",
			Channel:        ChannelSMS,
			Priority:       PriorityNormal,
			Subject:        "SMS Notification",
			TemplateData:   map[string]interface{}{"Message": "SMS message"},
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com" && msg.Subject == "Email Notification"
	})).Return(&email.SendResult{Success: true}, nil)
	mockSMSClient.On("SendNotification", ctx, "+1234567890", mock.Anything).Return(nil)

	results, err := service.SendMultiChannelNotification(ctx, reqs)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)
	mockEmailService.AssertExpectations(t)
	mockSMSClient.AssertExpectations(t)
}

// TestSendMultiChannelNotification_PartialFailure tests partial failures
func TestSendMultiChannelNotification_PartialFailure(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	reqs := []*NotificationRequest{
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Email Notification",
			TemplateData:   map[string]interface{}{"Message": "Email message"},
		},
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech@example.com",
			RecipientPhone: "+1234567890",
			Channel:        ChannelSMS,
			Priority:       PriorityNormal,
			Subject:        "SMS Notification",
			TemplateData:   map[string]interface{}{"Message": "SMS message"},
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com" && msg.Subject == "Email Notification"
	})).Return(&email.SendResult{Success: true}, nil)
	mockSMSClient.On("SendNotification", ctx, "+1234567890", mock.Anything).Return(errors.New("SMS error"))

	results, err := service.SendMultiChannelNotification(ctx, reqs)

	require.NoError(t, err) // At least one succeeded
	assert.Len(t, results, 2)

	// Results may be in any order due to concurrent processing
	successCount := 0
	failureCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failureCount++
		}
	}
	assert.Equal(t, 1, successCount, "Expected 1 successful notification")
	assert.Equal(t, 1, failureCount, "Expected 1 failed notification")
	mockEmailService.AssertExpectations(t)
	mockSMSClient.AssertExpectations(t)
}

// TestSendMultiChannelNotification_AllFailures tests all failures
func TestSendMultiChannelNotification_AllFailures(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	reqs := []*NotificationRequest{
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Email Notification",
			TemplateData:   map[string]interface{}{"Message": "Email message"},
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com" && msg.Subject == "Email Notification"
	})).Return(nil, errors.New("email error"))

	results, err := service.SendMultiChannelNotification(ctx, reqs)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotificationFailed)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Success)
	mockEmailService.AssertExpectations(t)
}

// TestSendAssignmentCreatedNotification tests assignment created notification
func TestSendAssignmentCreatedNotification(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	assignment := &models.JobAssignment{
		ID:         uuid.New(),
		JobID:      uuid.New(),
		UserID:     uuid.New(),
		Role:       "Lead Technician",
		AssignedAt: time.Now().Unix(),
	}

	job := &models.Job{
		ID:        assignment.JobID,
		JobNumber: "JOB-001",
		Title:     "HVAC Repair",
		Status:    models.JobStatusScheduled,
	}

	technician := &models.User{
		ID:    assignment.UserID,
		Email: "tech@example.com",
		Role:  models.UserRoleTechnician,
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com"
	})).Return(&email.SendResult{Success: true}, nil)

	err = service.SendAssignmentCreatedNotification(ctx, assignment, job, technician)

	require.NoError(t, err)
	mockEmailService.AssertExpectations(t)
}

// TestSendAssignmentUpdatedNotification tests assignment updated notification
func TestSendAssignmentUpdatedNotification(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	assignment := &models.JobAssignment{
		ID:         uuid.New(),
		JobID:      uuid.New(),
		UserID:     uuid.New(),
		Role:       "Senior Technician",
		AssignedAt: time.Now().Unix(),
	}

	job := &models.Job{
		ID:        assignment.JobID,
		JobNumber: "JOB-002",
		Title:     "HVAC Installation",
		Status:    models.JobStatusScheduled,
	}

	technician := &models.User{
		ID:    assignment.UserID,
		Email: "tech@example.com",
		Role:  models.UserRoleTechnician,
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com"
	})).Return(&email.SendResult{Success: true}, nil)

	err = service.SendAssignmentUpdatedNotification(ctx, assignment, job, technician, "Role changed from Technician to Senior Technician")

	require.NoError(t, err)
	mockEmailService.AssertExpectations(t)
}

// TestSendAssignmentRemovedNotification tests assignment removed notification
func TestSendAssignmentRemovedNotification(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech@example.com"
	})).Return(&email.SendResult{Success: true}, nil)

	err = service.SendAssignmentRemovedNotification(ctx, "tech@example.com", "John Doe", "JOB-003", "HVAC Maintenance")

	require.NoError(t, err)
	mockEmailService.AssertExpectations(t)
}

// TestSendBulkNotifications_Success tests bulk notifications
func TestSendBulkNotifications_Success(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	reqs := []*NotificationRequest{
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech1@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Notification 1",
			TemplateData:   map[string]interface{}{"Message": "Message 1"},
		},
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech2@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Notification 2",
			TemplateData:   map[string]interface{}{"Message": "Message 2"},
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech1@example.com"
	})).Return(&email.SendResult{Success: true}, nil)
	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech2@example.com"
	})).Return(&email.SendResult{Success: true}, nil)

	results, errs := service.SendBulkNotifications(ctx, reqs)

	assert.Len(t, results, 2)
	assert.Len(t, errs, 0)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)
	mockEmailService.AssertExpectations(t)
}

// TestSendBulkNotifications_PartialFailure tests bulk notifications with partial failures
func TestSendBulkNotifications_PartialFailure(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	reqs := []*NotificationRequest{
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech1@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Notification 1",
			TemplateData:   map[string]interface{}{"Message": "Message 1"},
		},
		{
			RecipientID:    uuid.New(),
			RecipientEmail: "tech2@example.com",
			Channel:        ChannelEmail,
			Priority:       PriorityNormal,
			Subject:        "Notification 2",
			TemplateData:   map[string]interface{}{"Message": "Message 2"},
		},
	}

	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech1@example.com"
	})).Return(&email.SendResult{Success: true}, nil)
	mockEmailService.On("Send", mock.Anything, mock.MatchedBy(func(msg *email.EmailMessage) bool {
		return len(msg.To) > 0 && msg.To[0] == "tech2@example.com"
	})).Return(nil, errors.New("email error"))

	results, errs := service.SendBulkNotifications(ctx, reqs)

	assert.Len(t, results, 2)
	assert.Len(t, errs, 1)

	// Results may be in any order due to concurrent processing
	successCount := 0
	failureCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failureCount++
		}
	}
	assert.Equal(t, 1, successCount, "Expected 1 successful notification")
	assert.Equal(t, 1, failureCount, "Expected 1 failed notification")
	mockEmailService.AssertExpectations(t)
}

// TestSendNotification_InvalidRecipient tests invalid recipient
func TestSendNotification_InvalidRecipient(t *testing.T) {
	mockEmailService := new(MockEmailService)
	mockSMSClient := new(MockSMSClient)

	service, err := NewNotificationService(mockEmailService, mockSMSClient)
	require.NoError(t, err)

	ctx := context.Background()
	req := &NotificationRequest{
		RecipientID:    uuid.New(),
		RecipientEmail: "", // Empty email
		Channel:        ChannelEmail,
		Priority:       PriorityNormal,
		Subject:        "Test Subject",
		TemplateData:   map[string]interface{}{},
	}

	result, err := service.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRecipient)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestSendNotification_PriorityLevels tests different priority levels
func TestSendNotification_PriorityLevels(t *testing.T) {
	priorities := []NotificationPriority{
		PriorityLow,
		PriorityNormal,
		PriorityHigh,
		PriorityUrgent,
	}

	for _, priority := range priorities {
		t.Run(string(priority), func(t *testing.T) {
			mockEmailService := new(MockEmailService)
			mockSMSClient := new(MockSMSClient)

			service, err := NewNotificationService(mockEmailService, mockSMSClient)
			require.NoError(t, err)

			ctx := context.Background()
			req := &NotificationRequest{
				RecipientID:    uuid.New(),
				RecipientEmail: "tech@example.com",
				Channel:        ChannelEmail,
				Priority:       priority,
				Subject:        "Test Subject",
				TemplateData:   map[string]interface{}{"Message": "Test message"},
			}

			mockEmailService.On("Send", mock.Anything, mock.AnythingOfType("*email.EmailMessage")).Return(&email.SendResult{Success: true}, nil)

			result, err := service.SendNotification(ctx, req)

			require.NoError(t, err)
			assert.True(t, result.Success)
			mockEmailService.AssertExpectations(t)
		})
	}
}
