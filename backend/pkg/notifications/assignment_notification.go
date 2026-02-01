package notifications

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

var (
	// ErrInvalidRecipient indicates invalid recipient
	ErrInvalidRecipient = errors.New("invalid recipient")

	// ErrTemplateParsing indicates template parsing error
	ErrTemplateParsing = errors.New("error parsing template")

	// ErrNotificationFailed indicates notification send failed
	ErrNotificationFailed = errors.New("notification failed to send")
)

// NotificationChannel represents the type of notification channel
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
	ChannelPush  NotificationChannel = "push"
)

// NotificationPriority represents notification priority
type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

// NotificationRequest represents a notification request
type NotificationRequest struct {
	RecipientID    uuid.UUID              `json:"recipient_id"`
	RecipientEmail string                 `json:"recipient_email"`
	RecipientPhone string                 `json:"recipient_phone,omitempty"`
	Channel        NotificationChannel    `json:"channel"`
	Priority       NotificationPriority   `json:"priority"`
	Subject        string                 `json:"subject"`
	TemplateData   map[string]interface{} `json:"template_data"`
	SendAt         *time.Time             `json:"send_at,omitempty"` // For scheduled notifications
}

// AssignmentNotificationData contains data for assignment notifications
type AssignmentNotificationData struct {
	TechnicianName    string
	JobNumber         string
	JobTitle          string
	JobDescription    string
	CustomerName      string
	ServiceAddress    string
	ScheduledStartAt  string
	ScheduledEndAt    string
	Priority          string
	Role              string
	EstimatedDuration string
	Notes             string
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	Channel   NotificationChannel `json:"channel"`
	Success   bool                `json:"success"`
	Error     error               `json:"error,omitempty"`
	SentAt    time.Time           `json:"sent_at"`
	MessageID string              `json:"message_id,omitempty"`
}

// NotificationServiceInterface defines the interface for notification service
type NotificationServiceInterface interface {
	// Send single notification
	SendNotification(ctx context.Context, req *NotificationRequest) (*NotificationResult, error)

	// Send to multiple channels concurrently
	SendMultiChannelNotification(ctx context.Context, reqs []*NotificationRequest) ([]*NotificationResult, error)

	// Assignment-specific notifications
	SendAssignmentCreatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User) error
	SendAssignmentUpdatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User, changes string) error
	SendAssignmentRemovedNotification(ctx context.Context, technicianEmail, technicianName, jobNumber, jobTitle string) error

	// Bulk notifications
	SendBulkNotifications(ctx context.Context, reqs []*NotificationRequest) ([]*NotificationResult, []error)
}

// NotificationService handles sending notifications
type NotificationService struct {
	emailClient email.Client
	smsClient   sms.Client // SMS client from pkg/clients/sms
	templates   *template.Template
	logger      *log.Logger
	mu          sync.RWMutex
}

// NewNotificationService creates a new notification service
// smsClient can be nil if SMS notifications are not needed
func NewNotificationService(emailClient email.Client, smsClient sms.Client) (*NotificationService, error) {
	service := &NotificationService{
		emailClient: emailClient,
		smsClient:   smsClient,
		logger:      log.New(log.Writer(), "[NotificationService] ", log.LstdFlags),
	}

	// Parse templates
	if err := service.initTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return service, nil
}

// initTemplates initializes notification templates
func (s *NotificationService) initTemplates() error {
	templates := template.New("notifications")

	// Assignment Created Email Template
	_, err := templates.New("assignment_created_email").Parse(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #007bff; color: white; padding: 20px; text-align: center; }
        .content { background: #f9f9f9; padding: 20px; margin: 20px 0; }
        .details { background: white; padding: 15px; margin: 10px 0; border-left: 4px solid #007bff; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
        .priority-urgent { border-left-color: #dc3545; }
        .priority-high { border-left-color: #ffc107; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>New Job Assignment</h1>
        </div>
        <div class="content">
            <p>Hello {{.TechnicianName}},</p>
            <p>You have been assigned to a new job:</p>

            <div class="details {{if eq .Priority "urgent"}}priority-urgent{{else if eq .Priority "high"}}priority-high{{end}}">
                <h3>{{.JobTitle}}</h3>
                <p><strong>Job #:</strong> {{.JobNumber}}</p>
                <p><strong>Customer:</strong> {{.CustomerName}}</p>
                <p><strong>Address:</strong> {{.ServiceAddress}}</p>
                <p><strong>Scheduled:</strong> {{.ScheduledStartAt}} - {{.ScheduledEndAt}}</p>
                <p><strong>Priority:</strong> {{.Priority}}</p>
                <p><strong>Your Role:</strong> {{.Role}}</p>
                {{if .EstimatedDuration}}
                <p><strong>Estimated Duration:</strong> {{.EstimatedDuration}}</p>
                {{end}}
            </div>

            {{if .JobDescription}}
            <div class="details">
                <h4>Job Description:</h4>
                <p>{{.JobDescription}}</p>
            </div>
            {{end}}

            {{if .Notes}}
            <div class="details">
                <h4>Notes:</h4>
                <p>{{.Notes}}</p>
            </div>
            {{end}}

            <p>Please confirm your availability and review the job details in the system.</p>
        </div>
        <div class="footer">
            <p>This is an automated notification from ServicePro</p>
        </div>
    </div>
</body>
</html>
`)
	if err != nil {
		return err
	}

	// Assignment Created SMS Template
	_, err = templates.New("assignment_created_sms").Parse(
		"New Job Assignment: {{.JobTitle}} ({{.JobNumber}})\n" +
			"Customer: {{.CustomerName}}\n" +
			"Scheduled: {{.ScheduledStartAt}}\n" +
			"Priority: {{.Priority}}\n" +
			"Check app for details.")
	if err != nil {
		return err
	}

	// Assignment Updated Email Template
	_, err = templates.New("assignment_updated_email").Parse(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #ffc107; color: white; padding: 20px; text-align: center; }
        .content { background: #f9f9f9; padding: 20px; margin: 20px 0; }
        .details { background: white; padding: 15px; margin: 10px 0; border-left: 4px solid #ffc107; }
        .changes { background: #fff3cd; padding: 10px; margin: 10px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Job Assignment Updated</h1>
        </div>
        <div class="content">
            <p>Hello {{.TechnicianName}},</p>
            <p>Your job assignment has been updated:</p>

            <div class="details">
                <h3>{{.JobTitle}}</h3>
                <p><strong>Job #:</strong> {{.JobNumber}}</p>
                <p><strong>Customer:</strong> {{.CustomerName}}</p>
            </div>

            {{if .Changes}}
            <div class="changes">
                <h4>Changes:</h4>
                <p>{{.Changes}}</p>
            </div>
            {{end}}

            <p>Please review the updated details in the system.</p>
        </div>
        <div class="footer">
            <p>This is an automated notification from ServicePro</p>
        </div>
    </div>
</body>
</html>
`)
	if err != nil {
		return err
	}

	// Assignment Removed Email Template
	_, err = templates.New("assignment_removed_email").Parse(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #6c757d; color: white; padding: 20px; text-align: center; }
        .content { background: #f9f9f9; padding: 20px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Job Assignment Removed</h1>
        </div>
        <div class="content">
            <p>Hello {{.TechnicianName}},</p>
            <p>You have been removed from the following job:</p>
            <p><strong>Job #:</strong> {{.JobNumber}}</p>
            <p><strong>Job Title:</strong> {{.JobTitle}}</p>
            <p>If you have questions, please contact your supervisor.</p>
        </div>
        <div class="footer">
            <p>This is an automated notification from ServicePro</p>
        </div>
    </div>
</body>
</html>
`)
	if err != nil {
		return err
	}

	s.templates = templates
	return nil
}

// SendNotification sends a single notification
func (s *NotificationService) SendNotification(ctx context.Context, req *NotificationRequest) (*NotificationResult, error) {
	s.logger.Printf("Sending %s notification to %s", req.Channel, req.RecipientEmail)

	result := &NotificationResult{
		Channel: req.Channel,
		SentAt:  time.Now(),
	}

	switch req.Channel {
	case ChannelEmail:
		err := s.sendEmailNotification(ctx, req)
		result.Success = err == nil
		result.Error = err
		return result, err

	case ChannelSMS:
		err := s.sendSMSNotification(ctx, req)
		result.Success = err == nil
		result.Error = err
		return result, err

	default:
		err := fmt.Errorf("unsupported channel: %s", req.Channel)
		result.Success = false
		result.Error = err
		return result, err
	}
}

// SendMultiChannelNotification sends to multiple channels concurrently
func (s *NotificationService) SendMultiChannelNotification(ctx context.Context, reqs []*NotificationRequest) ([]*NotificationResult, error) {
	s.logger.Printf("Sending multi-channel notification (%d channels)", len(reqs))

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*NotificationResult
	)

	for _, req := range reqs {
		wg.Add(1)
		go func(r *NotificationRequest) {
			defer wg.Done()

			result, _ := s.SendNotification(ctx, r)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(req)
	}

	wg.Wait()

	// Check if at least one channel succeeded
	hasSuccess := false
	for _, r := range results {
		if r.Success {
			hasSuccess = true
			break
		}
	}

	if !hasSuccess {
		return results, ErrNotificationFailed
	}

	return results, nil
}

// SendAssignmentCreatedNotification sends notification when assignment is created
func (s *NotificationService) SendAssignmentCreatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User) error {
	s.logger.Printf("Sending assignment created notification for job %s to technician %s", job.JobNumber, technician.Email)

	// Prepare template data
	data := s.prepareAssignmentData(assignment, job, technician)

	// Convert to interface{} map
	templateData := make(map[string]interface{})
	for k, v := range data {
		templateData[k] = v
	}

	// Send email
	emailReq := &NotificationRequest{
		RecipientID:    technician.ID,
		RecipientEmail: technician.Email,
		Channel:        ChannelEmail,
		Priority:       s.mapJobPriorityToNotificationPriority(job.Priority),
		Subject:        fmt.Sprintf("New Job Assignment: %s", job.Title),
		TemplateData:   templateData,
	}

	// Send notification (email only for now)
	_, err := s.SendNotification(ctx, emailReq)
	return err
}

// SendAssignmentUpdatedNotification sends notification when assignment is updated
func (s *NotificationService) SendAssignmentUpdatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User, changes string) error {
	s.logger.Printf("Sending assignment updated notification for job %s", job.JobNumber)

	data := s.prepareAssignmentData(assignment, job, technician)
	data["Changes"] = changes

	// Convert to interface{} map
	templateData := make(map[string]interface{})
	for k, v := range data {
		templateData[k] = v
	}

	req := &NotificationRequest{
		RecipientID:    technician.ID,
		RecipientEmail: technician.Email,
		Channel:        ChannelEmail,
		Priority:       PriorityNormal,
		Subject:        fmt.Sprintf("Job Assignment Updated: %s", job.Title),
		TemplateData:   templateData,
	}

	_, err := s.SendNotification(ctx, req)
	return err
}

// SendAssignmentRemovedNotification sends notification when assignment is removed
func (s *NotificationService) SendAssignmentRemovedNotification(ctx context.Context, technicianEmail, technicianName, jobNumber, jobTitle string) error {
	s.logger.Printf("Sending assignment removed notification for job %s", jobNumber)

	data := map[string]interface{}{
		"TechnicianName": technicianName,
		"JobNumber":      jobNumber,
		"JobTitle":       jobTitle,
	}

	req := &NotificationRequest{
		RecipientEmail: technicianEmail,
		Channel:        ChannelEmail,
		Priority:       PriorityNormal,
		Subject:        fmt.Sprintf("Job Assignment Removed: %s", jobTitle),
		TemplateData:   data,
	}

	_, err := s.SendNotification(ctx, req)
	return err
}

// SendBulkNotifications sends multiple notifications concurrently
func (s *NotificationService) SendBulkNotifications(ctx context.Context, reqs []*NotificationRequest) ([]*NotificationResult, []error) {
	s.logger.Printf("Sending bulk notifications (%d total)", len(reqs))

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*NotificationResult
		errs    []error
	)

	for _, req := range reqs {
		wg.Add(1)
		go func(r *NotificationRequest) {
			defer wg.Done()

			result, err := s.SendNotification(ctx, r)

			mu.Lock()
			results = append(results, result)
			if err != nil {
				errs = append(errs, err)
			}
			mu.Unlock()
		}(req)
	}

	wg.Wait()

	s.logger.Printf("Bulk notifications completed. Success: %d, Errors: %d", len(results)-len(errs), len(errs))
	return results, errs
}

// Helper methods

func (s *NotificationService) sendEmailNotification(ctx context.Context, req *NotificationRequest) error {
	if req.RecipientEmail == "" {
		return ErrInvalidRecipient
	}

	// Determine template name based on context
	templateName := "assignment_created_email"
	if subject, ok := req.TemplateData["subject"].(string); ok {
		if subject == "updated" {
			templateName = "assignment_updated_email"
		} else if subject == "removed" {
			templateName = "assignment_removed_email"
		}
	}

	// Render template
	var body bytes.Buffer
	tmpl := s.templates.Lookup(templateName)
	if tmpl == nil {
		return fmt.Errorf("%w: template %s not found", ErrTemplateParsing, templateName)
	}

	if err := tmpl.Execute(&body, req.TemplateData); err != nil {
		return fmt.Errorf("%w: %v", ErrTemplateParsing, err)
	}

	// Send email using the new client interface
	msg := email.NewEmailMessage(req.RecipientEmail, req.Subject, body.String())
	_, sendErr := s.emailClient.Send(ctx, msg)
	return sendErr
}

func (s *NotificationService) sendSMSNotification(ctx context.Context, req *NotificationRequest) error {
	if req.RecipientPhone == "" {
		return ErrInvalidRecipient
	}

	// Check if SMS client is available
	if s.smsClient == nil {
		s.logger.Printf("SMS client not configured, skipping SMS notification to %s", req.RecipientPhone)
		return nil
	}

	// Render SMS template
	var message bytes.Buffer
	tmpl := s.templates.Lookup("assignment_created_sms")
	if tmpl == nil {
		return fmt.Errorf("%w: SMS template not found", ErrTemplateParsing)
	}

	if err := tmpl.Execute(&message, req.TemplateData); err != nil {
		return fmt.Errorf("%w: %v", ErrTemplateParsing, err)
	}

	// Use the new SMS client
	return s.smsClient.SendNotification(ctx, req.RecipientPhone, message.String())
}

func (s *NotificationService) prepareAssignmentData(assignment *models.JobAssignment, job *models.Job, technician *models.User) map[string]string {
	data := map[string]string{
		"TechnicianName": technician.Email, // Use email as name since User model doesn't have name fields
		"JobNumber":      job.JobNumber,
		"JobTitle":       job.Title,
		"JobDescription": job.Description,
		"Priority":       string(job.Priority),
		"Role":           assignment.Role,
	}

	if job.Customer != nil {
		data["CustomerName"] = job.Customer.FirstName + " " + job.Customer.LastName
	}

	data["ServiceAddress"] = fmt.Sprintf("%s, %s, %s %s",
		job.ServiceAddress.Street,
		job.ServiceAddress.City,
		job.ServiceAddress.State,
		job.ServiceAddress.Zip)

	if job.ScheduledStartAt != nil {
		data["ScheduledStartAt"] = job.ScheduledStartAt.Format("Monday, January 2, 2006 at 3:04 PM")
	}

	if job.ScheduledEndAt != nil {
		data["ScheduledEndAt"] = job.ScheduledEndAt.Format("3:04 PM")
	}

	if job.EstimatedDuration > 0 {
		hours := job.EstimatedDuration / 60
		minutes := job.EstimatedDuration % 60
		if hours > 0 {
			data["EstimatedDuration"] = fmt.Sprintf("%d hours %d minutes", hours, minutes)
		} else {
			data["EstimatedDuration"] = fmt.Sprintf("%d minutes", minutes)
		}
	}

	if assignment.Notes != nil {
		data["Notes"] = *assignment.Notes
	}

	return data
}

func (s *NotificationService) mapJobPriorityToNotificationPriority(priority models.JobPriority) NotificationPriority {
	switch priority {
	case models.JobPriorityUrgent:
		return PriorityUrgent
	case models.JobPriorityHigh:
		return PriorityHigh
	case models.JobPriorityLow:
		return PriorityLow
	default:
		return PriorityNormal
	}
}
