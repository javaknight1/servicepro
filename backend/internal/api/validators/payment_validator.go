package validators

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// Currency code regex (ISO 4217 format)
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)

	// Email regex
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// Supported currencies
	supportedCurrencies = map[string]bool{
		"USD": true,
		"EUR": true,
		"GBP": true,
		"CAD": true,
		"AUD": true,
		"JPY": true,
	}

	// Minimum amounts by currency (in base units, e.g., cents for USD)
	minimumAmounts = map[string]float64{
		"USD": 0.50,
		"EUR": 0.50,
		"GBP": 0.30,
		"CAD": 0.50,
		"AUD": 0.50,
		"JPY": 50.00,
	}

	// Maximum payment amount (in USD equivalent)
	maximumAmountUSD = 999999.99
)

// PaymentValidator handles payment validation
type PaymentValidator struct{}

// NewPaymentValidator creates a new payment validator
func NewPaymentValidator() *PaymentValidator {
	return &PaymentValidator{}
}

// ValidateCreatePaymentIntent validates payment intent creation request
func (v *PaymentValidator) ValidateCreatePaymentIntent(req *models.CreatePaymentIntentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate amount
	if err := v.ValidateAmount(req.Amount, req.Currency); err != nil {
		return err
	}

	// Validate currency
	if err := v.ValidateCurrency(req.Currency); err != nil {
		return err
	}

	// Validate invoice ID if provided
	if req.InvoiceID != nil {
		if *req.InvoiceID == uuid.Nil {
			return errors.New("invalid invoice_id")
		}
	}

	// Validate order ID if provided
	if req.OrderID != nil {
		if *req.OrderID == uuid.Nil {
			return errors.New("invalid order_id")
		}
	}

	// Validate description length
	if len(req.Description) > 500 {
		return errors.New("description too long (max 500 characters)")
	}

	// Validate statement descriptor
	if req.StatementDescriptor != "" {
		if err := v.ValidateStatementDescriptor(req.StatementDescriptor); err != nil {
			return err
		}
	}

	// Validate receipt email
	if req.ReceiptEmail != "" {
		if err := v.ValidateEmail(req.ReceiptEmail); err != nil {
			return err
		}
	}

	// Validate capture method
	if req.CaptureMethod != "" {
		if err := v.ValidateCaptureMethod(req.CaptureMethod); err != nil {
			return err
		}
	}

	// Validate metadata
	if err := v.ValidateMetadata(req.Metadata); err != nil {
		return err
	}

	return nil
}

// ValidateConfirmPayment validates payment confirmation request
func (v *PaymentValidator) ValidateConfirmPayment(req *models.ConfirmPaymentRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate payment ID
	if req.PaymentID == uuid.Nil {
		return errors.New("payment_id is required")
	}

	// Validate return URL if provided
	if req.ReturnURL != nil && *req.ReturnURL != "" {
		if err := v.ValidateURL(*req.ReturnURL); err != nil {
			return fmt.Errorf("invalid return_url: %w", err)
		}
	}

	return nil
}

// ValidateCreateRefund validates refund creation request
func (v *PaymentValidator) ValidateCreateRefund(req *models.CreateRefundRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Validate payment ID
	if req.PaymentID == uuid.Nil {
		return errors.New("payment_id is required")
	}

	// Validate amount if provided (partial refund)
	if req.Amount != nil {
		if *req.Amount <= 0 {
			return errors.New("refund amount must be greater than zero")
		}
		// We'll validate against original payment amount in the handler
	}

	// Validate reason
	if req.Reason != "" {
		if err := v.ValidateRefundReason(req.Reason); err != nil {
			return err
		}
	}

	// Validate metadata
	if err := v.ValidateMetadata(req.Metadata); err != nil {
		return err
	}

	return nil
}

// ValidateAmount validates payment amount
func (v *PaymentValidator) ValidateAmount(amount float64, currency string) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}

	// Check maximum amount
	if amount > maximumAmountUSD {
		return fmt.Errorf("amount exceeds maximum allowed ($%.2f)", maximumAmountUSD)
	}

	// Check minimum amount for currency
	currency = strings.ToUpper(currency)
	if minAmount, ok := minimumAmounts[currency]; ok {
		if amount < minAmount {
			return fmt.Errorf("amount must be at least %.2f %s", minAmount, currency)
		}
	}

	// Validate decimal places
	if !v.IsValidDecimalPlaces(amount, currency) {
		return fmt.Errorf("invalid decimal places for currency %s", currency)
	}

	return nil
}

// ValidateCurrency validates currency code
func (v *PaymentValidator) ValidateCurrency(currency string) error {
	if currency == "" {
		return errors.New("currency is required")
	}

	currency = strings.ToUpper(currency)

	// Check format
	if !currencyRegex.MatchString(currency) {
		return errors.New("invalid currency format (must be 3-letter ISO code)")
	}

	// Check if supported
	if !supportedCurrencies[currency] {
		return fmt.Errorf("currency %s is not supported", currency)
	}

	return nil
}

// ValidateStatementDescriptor validates statement descriptor
func (v *PaymentValidator) ValidateStatementDescriptor(descriptor string) error {
	if len(descriptor) > 22 {
		return errors.New("statement descriptor too long (max 22 characters)")
	}

	// Statement descriptor should not contain special characters
	if !regexp.MustCompile(`^[a-zA-Z0-9\s\-\.]+$`).MatchString(descriptor) {
		return errors.New("statement descriptor contains invalid characters")
	}

	return nil
}

// ValidateEmail validates email address
func (v *PaymentValidator) ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}

	if len(email) > 255 {
		return errors.New("email too long (max 255 characters)")
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// ValidateCaptureMethod validates capture method
func (v *PaymentValidator) ValidateCaptureMethod(method string) error {
	validMethods := map[string]bool{
		"automatic": true,
		"manual":    true,
	}

	if !validMethods[method] {
		return errors.New("capture_method must be 'automatic' or 'manual'")
	}

	return nil
}

// ValidateRefundReason validates refund reason
func (v *PaymentValidator) ValidateRefundReason(reason string) error {
	validReasons := map[string]bool{
		"duplicate":             true,
		"fraudulent":            true,
		"requested_by_customer": true,
		"":                      true, // Empty is allowed
	}

	if !validReasons[reason] {
		return errors.New("invalid refund reason")
	}

	return nil
}

// ValidateMetadata validates metadata
func (v *PaymentValidator) ValidateMetadata(metadata map[string]string) error {
	if metadata == nil {
		return nil
	}

	// Stripe allows up to 50 key-value pairs
	if len(metadata) > 50 {
		return errors.New("too many metadata entries (max 50)")
	}

	for key, value := range metadata {
		// Validate key
		if len(key) == 0 {
			return errors.New("metadata key cannot be empty")
		}
		if len(key) > 40 {
			return errors.New("metadata key too long (max 40 characters)")
		}

		// Validate value
		if len(value) > 500 {
			return errors.New("metadata value too long (max 500 characters)")
		}
	}

	return nil
}

// ValidateURL validates a URL
func (v *PaymentValidator) ValidateURL(urlStr string) error {
	if urlStr == "" {
		return errors.New("URL cannot be empty")
	}

	if len(urlStr) > 2048 {
		return errors.New("URL too long (max 2048 characters)")
	}

	// Basic URL validation
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return errors.New("URL must start with http:// or https://")
	}

	return nil
}

// IsValidDecimalPlaces checks if amount has valid decimal places for currency
func (v *PaymentValidator) IsValidDecimalPlaces(amount float64, currency string) bool {
	currency = strings.ToUpper(currency)

	// Zero-decimal currencies (like JPY)
	zeroDecimalCurrencies := map[string]bool{
		"JPY": true,
		"KRW": true,
	}

	if zeroDecimalCurrencies[currency] {
		// Should be a whole number
		return amount == float64(int64(amount))
	}

	// Most currencies use 2 decimal places
	// Check if amount has at most 2 decimal places
	multiplied := amount * 100
	return multiplied == float64(int64(multiplied))
}

// SanitizeDescription sanitizes a description string
func (v *PaymentValidator) SanitizeDescription(description string) string {
	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)

	// Remove control characters
	description = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(description, "")

	// Limit length
	if len(description) > 500 {
		description = description[:500]
	}

	return description
}

// SanitizeMetadata sanitizes metadata
func (v *PaymentValidator) SanitizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}

	sanitized := make(map[string]string)
	count := 0

	for key, value := range metadata {
		if count >= 50 {
			break // Limit to 50 entries
		}

		// Sanitize key
		key = strings.TrimSpace(key)
		if len(key) == 0 || len(key) > 40 {
			continue
		}

		// Sanitize value
		value = strings.TrimSpace(value)
		if len(value) > 500 {
			value = value[:500]
		}

		sanitized[key] = value
		count++
	}

	return sanitized
}

// GetMinimumAmount returns the minimum amount for a currency
func (v *PaymentValidator) GetMinimumAmount(currency string) float64 {
	currency = strings.ToUpper(currency)
	if min, ok := minimumAmounts[currency]; ok {
		return min
	}
	return 0.50 // Default minimum
}

// IsSupportedCurrency checks if a currency is supported
func (v *PaymentValidator) IsSupportedCurrency(currency string) bool {
	return supportedCurrencies[strings.ToUpper(currency)]
}
