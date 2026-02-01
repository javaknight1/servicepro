// Package sns provides an SMS client using AWS SNS (Simple Notification Service).
//
// AWS SNS supports sending SMS messages directly to phone numbers. This provider
// uses the direct SMS publishing feature, not topic-based publishing.
//
// Note: AWS SNS SMS is not free, but pricing varies by region and destination.
// See: https://aws.amazon.com/sns/sms-pricing/
package sns

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

const (
	// DefaultTimeout for AWS API calls
	DefaultTimeout = 30 * time.Second
)

func init() {
	sms.RegisterProvider(sms.ProviderSNS, func(ctx context.Context, cfg *config.Config) (sms.Client, error) {
		if !cfg.AWS.SNS.Enabled {
			return nil, fmt.Errorf("AWS SNS is not enabled (set AWS_SNS_ENABLED=true)")
		}

		if cfg.AWS.Region == "" {
			return nil, fmt.Errorf("AWS region is required (set AWS_REGION)")
		}

		snsConfig := &Config{
			Region:          cfg.AWS.Region,
			AccessKeyID:     cfg.AWS.AccessKeyID,
			SecretAccessKey: cfg.AWS.SecretAccessKey,
			SenderID:        cfg.SMS.SenderID,
		}

		// Override region if SNS-specific region is set
		if cfg.AWS.SNS.Region != "" {
			snsConfig.Region = cfg.AWS.SNS.Region
		}

		return NewClient(ctx, snsConfig)
	})
}

// Config holds configuration for the AWS SNS SMS client
type Config struct {
	// AWS Region
	Region string

	// AWS credentials
	AccessKeyID     string
	SecretAccessKey string

	// SenderID is shown as the sender (where supported by carrier/region)
	SenderID string

	// SMSType is the type of SMS message: "Promotional" or "Transactional"
	// Transactional messages have higher delivery priority but may cost more
	SMSType string

	// Custom endpoint (for LocalStack or testing)
	Endpoint string
}

// Client implements the sms.Client interface using AWS SNS
type Client struct {
	config    *Config
	snsClient *sns.Client
}

// NewClient creates a new AWS SNS SMS client
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Build AWS config options
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	// Add credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	// Load AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SNS client options
	snsOpts := []func(*sns.Options){}
	if cfg.Endpoint != "" {
		snsOpts = append(snsOpts, func(o *sns.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	// Create SNS client
	snsClient := sns.NewFromConfig(awsCfg, snsOpts...)

	return &Client{
		config:    cfg,
		snsClient: snsClient,
	}, nil
}

// Send implements sms.Client
func (c *Client) Send(ctx context.Context, msg *sms.SMSMessage) (*sms.SendResult, error) {
	// Build message attributes
	attrs := make(map[string]types.MessageAttributeValue)

	// Set SMS type (Transactional or Promotional)
	smsType := c.config.SMSType
	if smsType == "" {
		// Use message type to determine SMS type
		switch msg.MessageType {
		case sms.MessageTypeOTP, sms.MessageTypeTransactional, sms.MessageTypeAlert:
			smsType = "Transactional"
		default:
			smsType = "Promotional"
		}
	}
	attrs["AWS.SNS.SMS.SMSType"] = types.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(smsType),
	}

	// Set sender ID if provided
	senderID := msg.SenderID
	if senderID == "" {
		senderID = c.config.SenderID
	}
	if senderID != "" {
		attrs["AWS.SNS.SMS.SenderID"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(senderID),
		}
	}

	// Publish SMS
	input := &sns.PublishInput{
		PhoneNumber:       aws.String(msg.PhoneNumber),
		Message:           aws.String(msg.Content),
		MessageAttributes: attrs,
	}

	output, err := c.snsClient.Publish(ctx, input)
	if err != nil {
		log.Printf("[SMS-SNS] Failed to send to %s: %v", msg.PhoneNumber, err)
		return &sms.SendResult{
			Success:     false,
			Provider:    sms.ProviderSNS,
			PhoneNumber: msg.PhoneNumber,
			Error:       err.Error(),
		}, err
	}

	messageID := ""
	if output.MessageId != nil {
		messageID = *output.MessageId
	}

	log.Printf("[SMS-SNS] Sent to %s (MessageID: %s)", msg.PhoneNumber, messageID)

	return &sms.SendResult{
		Success:      true,
		MessageID:    messageID,
		Provider:     sms.ProviderSNS,
		SentAt:       time.Now(),
		PhoneNumber:  msg.PhoneNumber,
		SegmentCount: msg.CalculateSegments(),
		Metadata: map[string]string{
			"sms_type": smsType,
		},
	}, nil
}

// SendBatch implements sms.Client
func (c *Client) SendBatch(ctx context.Context, messages []*sms.SMSMessage) []sms.SendResult {
	results := make([]sms.SendResult, len(messages))
	for i, msg := range messages {
		result, err := c.Send(ctx, msg)
		if err != nil {
			results[i] = sms.SendResult{
				Success:     false,
				Provider:    sms.ProviderSNS,
				PhoneNumber: msg.PhoneNumber,
				Error:       err.Error(),
			}
		} else {
			results[i] = *result
		}
	}
	return results
}

// SendOTP implements sms.Client
func (c *Client) SendOTP(ctx context.Context, phoneNumber, otp string, expiryMinutes int) error {
	msg := sms.NewSMSMessage(phoneNumber, fmt.Sprintf("Your verification code is: %s. Valid for %d minutes.", otp, expiryMinutes)).
		WithMessageType(sms.MessageTypeOTP).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendNotification implements sms.Client
func (c *Client) SendNotification(ctx context.Context, phoneNumber, message string) error {
	msg := sms.NewSMSMessage(phoneNumber, message).
		WithMessageType(sms.MessageTypeNotification)
	_, err := c.Send(ctx, msg)
	return err
}

// SendJobUpdate implements sms.Client
func (c *Client) SendJobUpdate(ctx context.Context, phoneNumber, jobNumber, status, message string) error {
	content := fmt.Sprintf("Job #%s - %s: %s", jobNumber, status, message)
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeNotification).
		WithMetadata("job_number", jobNumber).
		WithMetadata("status", status)
	_, err := c.Send(ctx, msg)
	return err
}

// SendAppointmentReminder implements sms.Client
func (c *Client) SendAppointmentReminder(ctx context.Context, phoneNumber, appointmentTime, jobDescription string) error {
	content := fmt.Sprintf("Reminder: Your appointment for %s is scheduled at %s.", jobDescription, appointmentTime)
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeReminder).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendInvoiceNotification implements sms.Client
func (c *Client) SendInvoiceNotification(ctx context.Context, phoneNumber, invoiceNumber, amount, paymentURL string) error {
	content := fmt.Sprintf("Invoice #%s for $%s is ready. Pay online: %s", invoiceNumber, amount, paymentURL)
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeTransactional).
		WithMetadata("invoice_number", invoiceNumber).
		WithMetadata("amount", amount)
	_, err := c.Send(ctx, msg)
	return err
}

// SendPaymentConfirmation implements sms.Client
func (c *Client) SendPaymentConfirmation(ctx context.Context, phoneNumber, invoiceNumber, amount string) error {
	content := fmt.Sprintf("Payment received! Invoice #%s - $%s. Thank you!", invoiceNumber, amount)
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeTransactional).
		WithMetadata("invoice_number", invoiceNumber).
		WithMetadata("amount", amount)
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements sms.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	// Check SMS spending limits to verify we can access SNS
	_, err := c.snsClient.GetSMSAttributes(ctx, &sns.GetSMSAttributesInput{
		Attributes: []string{"MonthlySpendLimit"},
	})
	if err != nil {
		return fmt.Errorf("AWS SNS health check failed: %w", err)
	}
	return nil
}

// Close implements sms.Client
func (c *Client) Close() error {
	// AWS SDK clients don't require explicit cleanup
	return nil
}

// GetProviderInfo implements sms.Client
func (c *Client) GetProviderInfo() sms.ProviderInfo {
	return sms.ProviderInfo{
		Provider:    sms.ProviderSNS,
		DisplayName: "AWS SNS",
		IsMock:      false,
		HasFreeTier: false, // AWS SNS SMS is paid
		DailyLimit:  0,     // No hard limit, but spending limits apply
		RateLimit:   20,    // Default soft limit per second (can be increased)
	}
}

// GetSMSAttributes returns the current SMS attributes from AWS
func (c *Client) GetSMSAttributes(ctx context.Context) (map[string]string, error) {
	output, err := c.snsClient.GetSMSAttributes(ctx, &sns.GetSMSAttributesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SMS attributes: %w", err)
	}
	return output.Attributes, nil
}

// SetSMSAttributes sets SMS attributes in AWS
func (c *Client) SetSMSAttributes(ctx context.Context, attributes map[string]string) error {
	_, err := c.snsClient.SetSMSAttributes(ctx, &sns.SetSMSAttributesInput{
		Attributes: attributes,
	})
	if err != nil {
		return fmt.Errorf("failed to set SMS attributes: %w", err)
	}
	return nil
}

// Ensure Client implements sms.Client
var _ sms.Client = (*Client)(nil)
