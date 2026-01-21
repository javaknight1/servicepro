//go:build integration

package mocks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// MailHog Client for Email Verification
// =============================================================================

// MailHogClient provides utilities for interacting with MailHog
type MailHogClient struct {
	apiURL     string
	httpClient *http.Client
}

// MailHogMessage represents an email in MailHog
type MailHogMessage struct {
	ID      string         `json:"ID"`
	From    MailHogPath    `json:"From"`
	To      []MailHogPath  `json:"To"`
	Content MailHogContent `json:"Content"`
	Created time.Time      `json:"Created"`
	MIME    MailHogMIME    `json:"MIME"`
	Raw     MailHogRaw     `json:"Raw"`
}

// MailHogPath represents an email address in MailHog
type MailHogPath struct {
	Relays  []string `json:"Relays"`
	Mailbox string   `json:"Mailbox"`
	Domain  string   `json:"Domain"`
	Params  string   `json:"Params"`
}

// MailHogContent represents email content
type MailHogContent struct {
	Headers map[string][]string `json:"Headers"`
	Body    string              `json:"Body"`
	Size    int                 `json:"Size"`
	MIME    interface{}         `json:"MIME"`
}

// MailHogMIME represents MIME information
type MailHogMIME struct {
	Parts []MailHogMIMEPart `json:"Parts"`
}

// MailHogMIMEPart represents a MIME part
type MailHogMIMEPart struct {
	Headers map[string][]string `json:"Headers"`
	Body    string              `json:"Body"`
	Size    int                 `json:"Size"`
	MIME    interface{}         `json:"MIME"`
}

// MailHogRaw represents raw email data
type MailHogRaw struct {
	From string   `json:"From"`
	To   []string `json:"To"`
	Data string   `json:"Data"`
	Helo string   `json:"Helo"`
}

// MailHogResponse represents the API response
type MailHogResponse struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Items []MailHogMessage `json:"items"`
}

// NewMailHogClient creates a new MailHog client
func NewMailHogClient(apiURL string) *MailHogClient {
	return &MailHogClient{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// =============================================================================
// Message Retrieval
// =============================================================================

// GetMessages retrieves all messages from MailHog
func (c *MailHogClient) GetMessages(t *testing.T) []MailHogMessage {
	t.Helper()

	url := fmt.Sprintf("%s/api/v2/messages", c.apiURL)
	resp, err := c.httpClient.Get(url)
	require.NoError(t, err, "Failed to get messages from MailHog")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result MailHogResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Failed to parse MailHog response")

	return result.Items
}

// GetLatestMessage retrieves the most recent message
func (c *MailHogClient) GetLatestMessage(t *testing.T) *MailHogMessage {
	t.Helper()

	messages := c.GetMessages(t)
	if len(messages) == 0 {
		return nil
	}
	return &messages[0]
}

// GetMessageByID retrieves a specific message by ID
func (c *MailHogClient) GetMessageByID(t *testing.T, id string) *MailHogMessage {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/messages/%s", c.apiURL, id)
	resp, err := c.httpClient.Get(url)
	require.NoError(t, err, "Failed to get message from MailHog")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var message MailHogMessage
	err = json.Unmarshal(body, &message)
	require.NoError(t, err)

	return &message
}

// =============================================================================
// Message Search
// =============================================================================

// SearchMessages searches for messages matching criteria
func (c *MailHogClient) SearchMessages(t *testing.T, kind, query string) []MailHogMessage {
	t.Helper()

	url := fmt.Sprintf("%s/api/v2/search?kind=%s&query=%s", c.apiURL, kind, query)
	resp, err := c.httpClient.Get(url)
	require.NoError(t, err, "Failed to search messages in MailHog")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result MailHogResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	return result.Items
}

// SearchByRecipient finds messages sent to a specific email
func (c *MailHogClient) SearchByRecipient(t *testing.T, email string) []MailHogMessage {
	t.Helper()
	return c.SearchMessages(t, "to", email)
}

// SearchBySubject finds messages with a specific subject
func (c *MailHogClient) SearchBySubject(t *testing.T, subject string) []MailHogMessage {
	t.Helper()
	return c.SearchMessages(t, "containing", subject)
}

// =============================================================================
// Message Deletion
// =============================================================================

// DeleteAllMessages deletes all messages from MailHog
func (c *MailHogClient) DeleteAllMessages(t *testing.T) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/messages", c.apiURL)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := c.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to delete messages")
}

// DeleteMessage deletes a specific message
func (c *MailHogClient) DeleteMessage(t *testing.T, id string) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/messages/%s", c.apiURL, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := c.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
}

// =============================================================================
// Helper Methods
// =============================================================================

// Email returns the full email address from MailHogPath
func (p *MailHogPath) Email() string {
	return fmt.Sprintf("%s@%s", p.Mailbox, p.Domain)
}

// Subject returns the email subject
func (m *MailHogMessage) Subject() string {
	if subjects, ok := m.Content.Headers["Subject"]; ok && len(subjects) > 0 {
		return subjects[0]
	}
	return ""
}

// PlainTextBody returns the plain text body
func (m *MailHogMessage) PlainTextBody() string {
	// Check MIME parts for text/plain
	for _, part := range m.MIME.Parts {
		if contentType, ok := part.Headers["Content-Type"]; ok {
			for _, ct := range contentType {
				if strings.Contains(ct, "text/plain") {
					return part.Body
				}
			}
		}
	}
	// Fall back to Content.Body
	return m.Content.Body
}

// HTMLBody returns the HTML body
func (m *MailHogMessage) HTMLBody() string {
	for _, part := range m.MIME.Parts {
		if contentType, ok := part.Headers["Content-Type"]; ok {
			for _, ct := range contentType {
				if strings.Contains(ct, "text/html") {
					return part.Body
				}
			}
		}
	}
	return ""
}

// ToAddresses returns a list of recipient email addresses
func (m *MailHogMessage) ToAddresses() []string {
	addresses := make([]string, len(m.To))
	for i, to := range m.To {
		addresses[i] = to.Email()
	}
	return addresses
}

// HasRecipient checks if the email was sent to a specific address
func (m *MailHogMessage) HasRecipient(email string) bool {
	for _, to := range m.ToAddresses() {
		if strings.EqualFold(to, email) {
			return true
		}
	}
	return false
}

// ContainsBody checks if the body contains a specific string
func (m *MailHogMessage) ContainsBody(substr string) bool {
	return strings.Contains(m.PlainTextBody(), substr) || strings.Contains(m.HTMLBody(), substr)
}

// =============================================================================
// Assertion Helpers
// =============================================================================

// AssertEmailSent asserts that an email was sent to the given recipient
func (c *MailHogClient) AssertEmailSent(t *testing.T, recipient string, timeout time.Duration) *MailHogMessage {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		messages := c.SearchByRecipient(t, recipient)
		if len(messages) > 0 {
			return &messages[0]
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("No email found for recipient %s within %s", recipient, timeout)
	return nil
}

// AssertNoEmailSent asserts that no email was sent to the given recipient
func (c *MailHogClient) AssertNoEmailSent(t *testing.T, recipient string) {
	t.Helper()

	messages := c.SearchByRecipient(t, recipient)
	require.Empty(t, messages, "Expected no emails for %s, but found %d", recipient, len(messages))
}

// AssertEmailSubject asserts the email subject matches
func (c *MailHogClient) AssertEmailSubject(t *testing.T, msg *MailHogMessage, expected string) {
	t.Helper()
	require.Equal(t, expected, msg.Subject(), "Email subject mismatch")
}

// AssertEmailContains asserts the email body contains the given text
func (c *MailHogClient) AssertEmailContains(t *testing.T, msg *MailHogMessage, text string) {
	t.Helper()
	require.True(t, msg.ContainsBody(text), "Email body does not contain: %s", text)
}

// WaitForEmail waits for an email to be received
func (c *MailHogClient) WaitForEmail(t *testing.T, timeout time.Duration) *MailHogMessage {
	t.Helper()

	deadline := time.Now().Add(timeout)
	initialCount := len(c.GetMessages(t))

	for time.Now().Before(deadline) {
		messages := c.GetMessages(t)
		if len(messages) > initialCount {
			return &messages[0]
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("No new email received within timeout")
	return nil
}
