package email

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/smithy-go"
)

// Standard errors
var (
	ErrClientClosed      = errors.New("ses client is closed")
	ErrNoRecipients      = errors.New("no recipients specified")
	ErrEmptySubject      = errors.New("email subject is empty")
	ErrEmptyBody         = errors.New("email body is empty")
	ErrInvalidEmail      = errors.New("invalid email address")
	ErrTemplateRequired  = errors.New("template name is required for templated emails")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrQuotaExceeded     = errors.New("sending quota exceeded")
	ErrDomainNotVerified = errors.New("domain not verified")
	ErrEmailNotVerified  = errors.New("email address not verified")
	ErrConfigurationSet  = errors.New("configuration set error")
	ErrTemplate          = errors.New("template error")
	ErrMessageRejected   = errors.New("message rejected")
	ErrMailFromDomain    = errors.New("mail from domain error")
	ErrSandboxMode       = errors.New("account is in sandbox mode")
)

// SESError represents an SES-specific error with additional context
type SESError struct {
	Code       string
	Message    string
	Retryable  bool
	StatusCode int
	RequestID  string
	Cause      error
}

func (e *SESError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("SES error [%s]: %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *SESError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for errors.Is
func (e *SESError) Is(target error) bool {
	if t, ok := target.(*SESError); ok {
		return e.Code == t.Code
	}
	return false
}

// ErrorCode constants for SES errors
const (
	CodeAccountSendingPaused          = "AccountSendingPausedException"
	CodeConfigurationSetDoesNotExist  = "ConfigurationSetDoesNotExistException"
	CodeConfigurationSetSendingPaused = "ConfigurationSetSendingPausedException"
	CodeInvalidParameterValue         = "InvalidParameterValue"
	CodeMailFromDomainNotVerified     = "MailFromDomainNotVerifiedException"
	CodeMessageRejected               = "MessageRejected"
	CodeTemplateDoesNotExist          = "TemplateDoesNotExistException"
	CodeLimitExceeded                 = "LimitExceededException"
	CodeThrottling                    = "Throttling"
	CodeServiceUnavailable            = "ServiceUnavailable"
	CodeInternalFailure               = "InternalFailure"
)

// SESErrorHandler handles and classifies SES errors
type SESErrorHandler struct {
	config *SESConfig
}

// NewSESErrorHandler creates a new error handler
func NewSESErrorHandler(config *SESConfig) *SESErrorHandler {
	return &SESErrorHandler{
		config: config,
	}
}

// Wrap wraps an error with SES-specific context
func (h *SESErrorHandler) Wrap(err error) error {
	if err == nil {
		return nil
	}

	// Already wrapped
	if sesErr, ok := err.(*SESError); ok {
		return sesErr
	}

	sesErr := &SESError{
		Cause:     err,
		Retryable: h.IsRetryable(err),
	}

	// Extract AWS error details
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		sesErr.Code = apiErr.ErrorCode()
		sesErr.Message = apiErr.ErrorMessage()
	}

	// Extract HTTP response details
	var httpErr interface {
		HTTPStatusCode() int
	}
	if errors.As(err, &httpErr) {
		sesErr.StatusCode = httpErr.HTTPStatusCode()
	}

	// Set message if not already set
	if sesErr.Message == "" {
		sesErr.Message = err.Error()
	}

	return sesErr
}

// IsRetryable determines if an error is retryable
func (h *SESErrorHandler) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific non-retryable SES errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		// Non-retryable errors
		case CodeMessageRejected,
			CodeInvalidParameterValue,
			CodeMailFromDomainNotVerified,
			CodeTemplateDoesNotExist,
			CodeConfigurationSetDoesNotExist,
			CodeAccountSendingPaused,
			CodeConfigurationSetSendingPaused:
			return false

		// Retryable errors
		case CodeThrottling,
			CodeLimitExceeded,
			CodeServiceUnavailable,
			CodeInternalFailure:
			return true
		}
	}

	// Check for network errors (generally retryable)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Check for specific AWS SES types
	var messageRejected *types.MessageRejected
	if errors.As(err, &messageRejected) {
		return false
	}

	var accountSendingPaused *types.AccountSendingPausedException
	if errors.As(err, &accountSendingPaused) {
		return false
	}

	var limitExceeded *types.LimitExceededException
	if errors.As(err, &limitExceeded) {
		return true // Throttling errors are retryable
	}

	// Default: retry on unknown errors
	return true
}

// IsThrottling checks if the error is a throttling error
func (h *SESErrorHandler) IsThrottling(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == CodeThrottling || code == CodeLimitExceeded
	}
	return false
}

// IsConfigurationError checks if the error is a configuration error
func (h *SESErrorHandler) IsConfigurationError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == CodeConfigurationSetDoesNotExist ||
			code == CodeConfigurationSetSendingPaused ||
			code == CodeInvalidParameterValue
	}
	return false
}

// IsVerificationError checks if the error is related to email/domain verification
func (h *SESErrorHandler) IsVerificationError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == CodeMailFromDomainNotVerified
	}

	// Check error message for verification issues
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not verified") ||
		strings.Contains(msg, "verification")
}

// IsTemplateError checks if the error is a template error
func (h *SESErrorHandler) IsTemplateError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == CodeTemplateDoesNotExist
	}
	return false
}

// IsAccountPaused checks if sending is paused for the account
func (h *SESErrorHandler) IsAccountPaused(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == CodeAccountSendingPaused ||
			code == CodeConfigurationSetSendingPaused
	}
	return false
}

// GetErrorCode extracts the error code from an error
func (h *SESErrorHandler) GetErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}

	var sesErr *SESError
	if errors.As(err, &sesErr) {
		return sesErr.Code
	}

	return ""
}

// GetErrorMessage extracts a user-friendly error message
func (h *SESErrorHandler) GetErrorMessage(err error) string {
	code := h.GetErrorCode(err)

	switch code {
	case CodeMessageRejected:
		return "The email was rejected. Please check the recipients and content."
	case CodeMailFromDomainNotVerified:
		return "The sender domain is not verified. Please verify the domain in SES."
	case CodeTemplateDoesNotExist:
		return "The email template does not exist."
	case CodeConfigurationSetDoesNotExist:
		return "The configuration set does not exist."
	case CodeAccountSendingPaused:
		return "Email sending is currently paused for this account."
	case CodeLimitExceeded:
		return "The sending limit has been exceeded. Please try again later."
	case CodeThrottling:
		return "Too many requests. Please try again later."
	default:
		return "An error occurred while sending the email."
	}
}

// ClassifyError returns a classification of the error for metrics/logging
func (h *SESErrorHandler) ClassifyError(err error) string {
	if h.IsThrottling(err) {
		return "throttling"
	}
	if h.IsConfigurationError(err) {
		return "configuration"
	}
	if h.IsVerificationError(err) {
		return "verification"
	}
	if h.IsTemplateError(err) {
		return "template"
	}
	if h.IsAccountPaused(err) {
		return "account_paused"
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network"
	}

	return "unknown"
}

// ErrorMiddleware provides middleware functions for error handling
type ErrorMiddleware struct {
	handler *SESErrorHandler
	onError func(err error, classification string)
}

// NewErrorMiddleware creates a new error middleware
func NewErrorMiddleware(handler *SESErrorHandler) *ErrorMiddleware {
	return &ErrorMiddleware{
		handler: handler,
	}
}

// SetErrorCallback sets a callback function for errors
func (m *ErrorMiddleware) SetErrorCallback(fn func(err error, classification string)) {
	m.onError = fn
}

// Handle processes an error and calls the callback if set
func (m *ErrorMiddleware) Handle(err error) error {
	if err == nil {
		return nil
	}

	classification := m.handler.ClassifyError(err)

	if m.onError != nil {
		m.onError(err, classification)
	}

	return m.handler.Wrap(err)
}

// RecoverableError wraps an error with recovery suggestions
type RecoverableError struct {
	Err         error
	Suggestion  string
	RecoveryURL string
}

func (e *RecoverableError) Error() string {
	return fmt.Sprintf("%s - Suggestion: %s", e.Err.Error(), e.Suggestion)
}

func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// NewRecoverableError creates an error with recovery suggestions
func NewRecoverableError(err error, suggestion string) *RecoverableError {
	return &RecoverableError{
		Err:        err,
		Suggestion: suggestion,
	}
}

// GetRecoverySuggestion returns a recovery suggestion for an error
func (h *SESErrorHandler) GetRecoverySuggestion(err error) string {
	code := h.GetErrorCode(err)

	switch code {
	case CodeMessageRejected:
		return "Check that all recipient email addresses are valid and the email content complies with SES policies."
	case CodeMailFromDomainNotVerified:
		return "Verify your domain in the AWS SES console. Go to Verified identities and add your domain."
	case CodeTemplateDoesNotExist:
		return "Create the email template in SES or check the template name for typos."
	case CodeConfigurationSetDoesNotExist:
		return "Create the configuration set in SES or update your configuration to use an existing set."
	case CodeAccountSendingPaused:
		return "Check your AWS SES console for account status. You may need to request production access."
	case CodeLimitExceeded:
		return "Wait and retry with exponential backoff, or request a sending limit increase from AWS."
	case CodeThrottling:
		return "Implement rate limiting in your application and use exponential backoff for retries."
	default:
		return "Check the AWS SES documentation for more information on this error."
	}
}

// BulkErrorResult represents errors from a bulk operation
type BulkErrorResult struct {
	SuccessCount int
	FailureCount int
	Errors       []BulkErrorItem
}

// BulkErrorItem represents a single error in a bulk operation
type BulkErrorItem struct {
	Index   int
	Email   string
	Error   error
	Code    string
	Message string
}

// HasErrors returns true if there are any errors
func (r *BulkErrorResult) HasErrors() bool {
	return r.FailureCount > 0
}

// GetRetryableErrors returns only the retryable errors
func (r *BulkErrorResult) GetRetryableErrors(handler *SESErrorHandler) []BulkErrorItem {
	var retryable []BulkErrorItem
	for _, item := range r.Errors {
		if handler.IsRetryable(item.Error) {
			retryable = append(retryable, item)
		}
	}
	return retryable
}

// Summary returns a summary string of the bulk operation result
func (r *BulkErrorResult) Summary() string {
	return fmt.Sprintf("Bulk operation: %d succeeded, %d failed",
		r.SuccessCount, r.FailureCount)
}
