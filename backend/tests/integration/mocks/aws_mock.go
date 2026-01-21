//go:build integration

package mocks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// LocalStack Client for AWS Mock Services
// =============================================================================

// AWSMock provides utilities for interacting with LocalStack
type AWSMock struct {
	endpointURL string
	region      string
	accessKey   string
	secretKey   string
	httpClient  *http.Client
	mu          sync.Mutex

	// Tracked operations for verification
	sentEmails    []SentEmail
	sentSMS       []SentSMS
	uploadedFiles []UploadedFile
	publishedMsgs []PublishedMessage
}

// SentEmail represents an email sent via SES
type SentEmail struct {
	From     string
	To       []string
	Subject  string
	Body     string
	HTMLBody string
	SentAt   time.Time
}

// SentSMS represents an SMS sent via SNS
type SentSMS struct {
	PhoneNumber string
	Message     string
	SentAt      time.Time
}

// UploadedFile represents a file uploaded to S3
type UploadedFile struct {
	Bucket      string
	Key         string
	ContentType string
	Size        int64
	UploadedAt  time.Time
}

// PublishedMessage represents a message published to SNS
type PublishedMessage struct {
	TopicARN   string
	Message    string
	Subject    string
	Attributes map[string]string
	SentAt     time.Time
}

// NewAWSMock creates a new AWS mock client
func NewAWSMock(endpointURL, region, accessKey, secretKey string) *AWSMock {
	return &AWSMock{
		endpointURL: endpointURL,
		region:      region,
		accessKey:   accessKey,
		secretKey:   secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// =============================================================================
// S3 Operations
// =============================================================================

// CreateBucket creates an S3 bucket
func (m *AWSMock) CreateBucket(t *testing.T, bucket string) {
	t.Helper()

	url := fmt.Sprintf("%s/%s", m.endpointURL, bucket)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	require.NoError(t, err)

	req.Header.Set("Host", fmt.Sprintf("%s.s3.%s.localhost.localstack.cloud:4566", bucket, m.region))

	resp, err := m.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 200 OK or 409 (already exists) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to create bucket: %s, body: %s", resp.Status, string(body))
	}
}

// UploadFile uploads a file to S3
func (m *AWSMock) UploadFile(t *testing.T, bucket, key string, content []byte, contentType string) {
	t.Helper()

	url := fmt.Sprintf("%s/%s/%s", m.endpointURL, bucket, key)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(content))
	require.NoError(t, err)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := m.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to upload file")

	m.mu.Lock()
	m.uploadedFiles = append(m.uploadedFiles, UploadedFile{
		Bucket:      bucket,
		Key:         key,
		ContentType: contentType,
		Size:        int64(len(content)),
		UploadedAt:  time.Now(),
	})
	m.mu.Unlock()
}

// GetFile retrieves a file from S3
func (m *AWSMock) GetFile(t *testing.T, bucket, key string) []byte {
	t.Helper()

	url := fmt.Sprintf("%s/%s/%s", m.endpointURL, bucket, key)
	resp, err := m.httpClient.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get file")

	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return content
}

// DeleteFile deletes a file from S3
func (m *AWSMock) DeleteFile(t *testing.T, bucket, key string) {
	t.Helper()

	url := fmt.Sprintf("%s/%s/%s", m.endpointURL, bucket, key)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := m.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 204 No Content or 404 (already deleted) are both acceptable
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Failed to delete file: %s", resp.Status)
	}
}

// FileExists checks if a file exists in S3
func (m *AWSMock) FileExists(t *testing.T, bucket, key string) bool {
	t.Helper()

	url := fmt.Sprintf("%s/%s/%s", m.endpointURL, bucket, key)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	require.NoError(t, err)

	resp, err := m.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetUploadedFiles returns all tracked uploaded files
func (m *AWSMock) GetUploadedFiles() []UploadedFile {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.uploadedFiles
}

// =============================================================================
// SES Operations
// =============================================================================

// SendEmail simulates sending an email via SES
func (m *AWSMock) SendEmail(t *testing.T, from string, to []string, subject, body, htmlBody string) {
	t.Helper()

	m.mu.Lock()
	m.sentEmails = append(m.sentEmails, SentEmail{
		From:     from,
		To:       to,
		Subject:  subject,
		Body:     body,
		HTMLBody: htmlBody,
		SentAt:   time.Now(),
	})
	m.mu.Unlock()
}

// GetSentEmails returns all tracked sent emails
func (m *AWSMock) GetSentEmails() []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentEmails
}

// GetLastEmail returns the most recently sent email
func (m *AWSMock) GetLastEmail() *SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sentEmails) == 0 {
		return nil
	}
	return &m.sentEmails[len(m.sentEmails)-1]
}

// ClearEmails clears all tracked emails
func (m *AWSMock) ClearEmails() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = nil
}

// =============================================================================
// SNS Operations
// =============================================================================

// PublishMessage publishes a message to an SNS topic
func (m *AWSMock) PublishMessage(t *testing.T, topicARN, message, subject string, attrs map[string]string) {
	t.Helper()

	m.mu.Lock()
	m.publishedMsgs = append(m.publishedMsgs, PublishedMessage{
		TopicARN:   topicARN,
		Message:    message,
		Subject:    subject,
		Attributes: attrs,
		SentAt:     time.Now(),
	})
	m.mu.Unlock()
}

// SendSMS sends an SMS via SNS
func (m *AWSMock) SendSMS(t *testing.T, phoneNumber, message string) {
	t.Helper()

	m.mu.Lock()
	m.sentSMS = append(m.sentSMS, SentSMS{
		PhoneNumber: phoneNumber,
		Message:     message,
		SentAt:      time.Now(),
	})
	m.mu.Unlock()
}

// GetPublishedMessages returns all tracked published messages
func (m *AWSMock) GetPublishedMessages() []PublishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishedMsgs
}

// GetSentSMS returns all tracked sent SMS messages
func (m *AWSMock) GetSentSMS() []SentSMS {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentSMS
}

// =============================================================================
// SQS Operations
// =============================================================================

// QueueMessage represents a message in an SQS queue
type QueueMessage struct {
	ID         string
	Body       string
	Attributes map[string]string
}

// SendQueueMessage sends a message to an SQS queue
func (m *AWSMock) SendQueueMessage(ctx context.Context, t *testing.T, queueURL, body string) string {
	t.Helper()
	// LocalStack handles SQS via HTTP API
	return "msg-" + fmt.Sprintf("%d", time.Now().UnixNano())
}

// ReceiveQueueMessages receives messages from an SQS queue
func (m *AWSMock) ReceiveQueueMessages(ctx context.Context, t *testing.T, queueURL string, maxMessages int) []QueueMessage {
	t.Helper()
	// Would interact with LocalStack SQS
	return nil
}

// =============================================================================
// Reset/Cleanup
// =============================================================================

// Reset clears all tracked operations
func (m *AWSMock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentEmails = nil
	m.sentSMS = nil
	m.uploadedFiles = nil
	m.publishedMsgs = nil
}
