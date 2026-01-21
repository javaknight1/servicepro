package validators

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestValidateCreatePaymentIntent_Success(t *testing.T) {
	validator := NewPaymentValidator()

	req := &models.CreatePaymentIntentRequest{
		Amount:      100.50,
		Currency:    "USD",
		Description: "Test payment",
	}

	err := validator.ValidateCreatePaymentIntent(req)
	assert.NoError(t, err)
}

func TestValidateCreatePaymentIntent_NilRequest(t *testing.T) {
	validator := NewPaymentValidator()

	err := validator.ValidateCreatePaymentIntent(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestValidateCreatePaymentIntent_InvalidAmount(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name     string
		amount   float64
		currency string
		wantErr  bool
	}{
		{"Zero amount", 0, "USD", true},
		{"Negative amount", -100, "USD", true},
		{"Too small USD", 0.40, "USD", true},
		{"Valid USD", 0.50, "USD", false},
		{"Too large amount", 1000000.00, "USD", true},
		{"Valid JPY", 50.00, "JPY", false},
		{"Too small JPY", 40.00, "JPY", true},
		{"Invalid decimal USD", 10.555, "USD", true},
		{"Valid decimal USD", 10.50, "USD", false},
		{"Invalid decimal JPY", 50.50, "JPY", true},
		{"Valid integer JPY", 50.00, "JPY", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePaymentIntentRequest{
				Amount:   tc.amount,
				Currency: tc.currency,
			}
			err := validator.ValidateCreatePaymentIntent(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreatePaymentIntent_InvalidCurrency(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{"Empty currency", "", true},
		{"Invalid format", "US", true},
		{"Invalid format long", "USDD", true},
		{"Unsupported currency", "XYZ", true},
		{"Valid USD", "USD", false},
		{"Valid lowercase usd", "usd", false},
		{"Valid EUR", "EUR", false},
		{"Valid GBP", "GBP", false},
		{"Valid CAD", "CAD", false},
		{"Valid AUD", "AUD", false},
		{"Valid JPY", "JPY", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePaymentIntentRequest{
				Amount:   100.00,
				Currency: tc.currency,
			}
			err := validator.ValidateCreatePaymentIntent(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreatePaymentIntent_InvalidInvoiceID(t *testing.T) {
	validator := NewPaymentValidator()

	nilUUID := uuid.Nil
	req := &models.CreatePaymentIntentRequest{
		Amount:    100.00,
		Currency:  "USD",
		InvoiceID: &nilUUID,
	}

	err := validator.ValidateCreatePaymentIntent(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid invoice_id")
}

func TestValidateCreatePaymentIntent_InvalidOrderID(t *testing.T) {
	validator := NewPaymentValidator()

	nilUUID := uuid.Nil
	req := &models.CreatePaymentIntentRequest{
		Amount:   100.00,
		Currency: "USD",
		OrderID:  &nilUUID,
	}

	err := validator.ValidateCreatePaymentIntent(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order_id")
}

func TestValidateCreatePaymentIntent_DescriptionTooLong(t *testing.T) {
	validator := NewPaymentValidator()

	longDesc := string(make([]byte, 501))
	for i := range longDesc {
		longDesc = longDesc[:i] + "a" + longDesc[i+1:]
	}

	req := &models.CreatePaymentIntentRequest{
		Amount:      100.00,
		Currency:    "USD",
		Description: longDesc,
	}

	err := validator.ValidateCreatePaymentIntent(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description too long")
}

func TestValidateCreatePaymentIntent_InvalidStatementDescriptor(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name       string
		descriptor string
		wantErr    bool
	}{
		{"Valid descriptor", "My Company", false},
		{"Too long", "This is a very long statement descriptor", true},
		{"Special characters", "Company@#$%", true},
		{"Valid with dash", "My-Company", false},
		{"Valid with dot", "My.Company", false},
		{"Valid with space", "My Company LLC", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePaymentIntentRequest{
				Amount:              100.00,
				Currency:            "USD",
				StatementDescriptor: tc.descriptor,
			}
			err := validator.ValidateCreatePaymentIntent(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreatePaymentIntent_InvalidEmail(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid email", "test@example.com", false},
		{"Invalid format", "invalid-email", true},
		{"No domain", "test@", true},
		{"No at sign", "testexample.com", true},
		{"Too long", string(make([]byte, 256)) + "@example.com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePaymentIntentRequest{
				Amount:       100.00,
				Currency:     "USD",
				ReceiptEmail: tc.email,
			}
			err := validator.ValidateCreatePaymentIntent(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreatePaymentIntent_InvalidCaptureMethod(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name          string
		captureMethod string
		wantErr       bool
	}{
		{"Automatic", "automatic", false},
		{"Manual", "manual", false},
		{"Invalid", "invalid", true},
		{"Empty", "", false}, // Empty is allowed (defaults to automatic)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePaymentIntentRequest{
				Amount:        100.00,
				Currency:      "USD",
				CaptureMethod: tc.captureMethod,
			}
			err := validator.ValidateCreatePaymentIntent(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCreatePaymentIntent_InvalidMetadata(t *testing.T) {
	validator := NewPaymentValidator()

	// Too many entries
	metadata := make(map[string]string)
	for i := 0; i < 51; i++ {
		metadata[string(rune(i))] = "value"
	}

	req := &models.CreatePaymentIntentRequest{
		Amount:   100.00,
		Currency: "USD",
		Metadata: metadata,
	}

	err := validator.ValidateCreatePaymentIntent(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many metadata entries")
}

func TestValidateConfirmPayment_Success(t *testing.T) {
	validator := NewPaymentValidator()

	req := &models.ConfirmPaymentRequest{
		PaymentID: uuid.New(),
	}

	err := validator.ValidateConfirmPayment(req)
	assert.NoError(t, err)
}

func TestValidateConfirmPayment_NilRequest(t *testing.T) {
	validator := NewPaymentValidator()

	err := validator.ValidateConfirmPayment(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestValidateConfirmPayment_InvalidPaymentID(t *testing.T) {
	validator := NewPaymentValidator()

	req := &models.ConfirmPaymentRequest{
		PaymentID: uuid.Nil,
	}

	err := validator.ValidateConfirmPayment(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment_id is required")
}

func TestValidateConfirmPayment_InvalidReturnURL(t *testing.T) {
	validator := NewPaymentValidator()

	invalidURL := "not-a-url"
	req := &models.ConfirmPaymentRequest{
		PaymentID: uuid.New(),
		ReturnURL: &invalidURL,
	}

	err := validator.ValidateConfirmPayment(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid return_url")
}

func TestValidateCreateRefund_Success(t *testing.T) {
	validator := NewPaymentValidator()

	req := &models.CreateRefundRequest{
		PaymentID: uuid.New(),
		Reason:    "requested_by_customer",
	}

	err := validator.ValidateCreateRefund(req)
	assert.NoError(t, err)
}

func TestValidateCreateRefund_NilRequest(t *testing.T) {
	validator := NewPaymentValidator()

	err := validator.ValidateCreateRefund(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestValidateCreateRefund_InvalidPaymentID(t *testing.T) {
	validator := NewPaymentValidator()

	req := &models.CreateRefundRequest{
		PaymentID: uuid.Nil,
	}

	err := validator.ValidateCreateRefund(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment_id is required")
}

func TestValidateCreateRefund_InvalidAmount(t *testing.T) {
	validator := NewPaymentValidator()

	amount := -10.00
	req := &models.CreateRefundRequest{
		PaymentID: uuid.New(),
		Amount:    &amount,
	}

	err := validator.ValidateCreateRefund(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be greater than zero")
}

func TestValidateCreateRefund_InvalidReason(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		reason  string
		wantErr bool
	}{
		{"Valid duplicate", "duplicate", false},
		{"Valid fraudulent", "fraudulent", false},
		{"Valid requested", "requested_by_customer", false},
		{"Valid empty", "", false},
		{"Invalid reason", "invalid_reason", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreateRefundRequest{
				PaymentID: uuid.New(),
				Reason:    tc.reason,
			}
			err := validator.ValidateCreateRefund(req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name     string
		amount   float64
		currency string
		wantErr  bool
	}{
		{"Valid USD", 100.00, "USD", false},
		{"Zero", 0, "USD", true},
		{"Negative", -50.00, "USD", true},
		{"Too large", 1000000.00, "USD", true},
		{"Below minimum USD", 0.40, "USD", true},
		{"At minimum USD", 0.50, "USD", false},
		{"Below minimum JPY", 40.00, "JPY", true},
		{"At minimum JPY", 50.00, "JPY", false},
		{"Invalid decimals USD", 10.555, "USD", true},
		{"Valid decimals USD", 10.50, "USD", false},
		{"Invalid decimals JPY", 50.50, "JPY", true},
		{"Valid integer JPY", 100.00, "JPY", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateAmount(tc.amount, tc.currency)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{"Valid USD", "USD", false},
		{"Valid lowercase", "usd", false},
		{"Valid EUR", "EUR", false},
		{"Valid GBP", "GBP", false},
		{"Valid CAD", "CAD", false},
		{"Valid AUD", "AUD", false},
		{"Valid JPY", "JPY", false},
		{"Empty", "", true},
		{"Too short", "US", true},
		{"Too long", "USDD", true},
		{"Unsupported", "XYZ", true},
		{"Numbers", "123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateCurrency(tc.currency)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStatementDescriptor(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name       string
		descriptor string
		wantErr    bool
	}{
		{"Valid simple", "MyCompany", false},
		{"Valid with space", "My Company", false},
		{"Valid with dash", "My-Company", false},
		{"Valid with dot", "My.Company", false},
		{"Too long", "This is a very long statement descriptor", true},
		{"Special chars", "Company@#$", true},
		{"At max length", "1234567890123456789012", false},
		{"Over max length", "12345678901234567890123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateStatementDescriptor(tc.descriptor)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid", "test@example.com", false},
		{"Valid with subdomain", "test@mail.example.com", false},
		{"Valid with plus", "test+tag@example.com", false},
		{"Empty (allowed)", "", false},
		{"No at sign", "testexample.com", true},
		{"No domain", "test@", true},
		{"No local part", "@example.com", true},
		{"Invalid format", "test@", true},
		{"Too long", string(make([]byte, 256)) + "@example.com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateEmail(tc.email)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCaptureMethod(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{"Automatic", "automatic", false},
		{"Manual", "manual", false},
		{"Invalid", "invalid", true},
		{"Empty", "immediate", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateCaptureMethod(tc.method)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRefundReason(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		reason  string
		wantErr bool
	}{
		{"Duplicate", "duplicate", false},
		{"Fraudulent", "fraudulent", false},
		{"Requested by customer", "requested_by_customer", false},
		{"Empty", "", false},
		{"Invalid", "some_other_reason", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateRefundReason(tc.reason)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	validator := NewPaymentValidator()

	// Valid metadata
	validMetadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	err := validator.ValidateMetadata(validMetadata)
	assert.NoError(t, err)

	// Nil metadata (allowed)
	err = validator.ValidateMetadata(nil)
	assert.NoError(t, err)

	// Too many entries
	tooMany := make(map[string]string)
	for i := 0; i < 51; i++ {
		tooMany[string(rune('a'+i))] = "value"
	}
	err = validator.ValidateMetadata(tooMany)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many metadata entries")

	// Empty key
	emptyKey := map[string]string{
		"": "value",
	}
	err = validator.ValidateMetadata(emptyKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key cannot be empty")

	// Key too long
	longKey := map[string]string{
		string(make([]byte, 41)): "value",
	}
	err = validator.ValidateMetadata(longKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key too long")

	// Value too long
	longValue := map[string]string{
		"key": string(make([]byte, 501)),
	}
	err = validator.ValidateMetadata(longValue)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value too long")
}

func TestValidateURL(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"Valid HTTP", "http://example.com", false},
		{"Valid HTTPS", "https://example.com", false},
		{"Valid with path", "https://example.com/path", false},
		{"Valid with query", "https://example.com?query=value", false},
		{"Empty", "", true},
		{"No protocol", "example.com", true},
		{"Invalid protocol", "ftp://example.com", true},
		{"Too long", "https://" + string(make([]byte, 2050)), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateURL(tc.url)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidDecimalPlaces(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		name     string
		amount   float64
		currency string
		want     bool
	}{
		{"USD two decimals", 10.50, "USD", true},
		{"USD integer", 10.00, "USD", true},
		{"USD three decimals", 10.555, "USD", false},
		{"JPY integer", 100.00, "JPY", true},
		{"JPY with decimals", 100.50, "JPY", false},
		{"EUR two decimals", 50.25, "EUR", true},
		{"EUR three decimals", 50.255, "EUR", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.IsValidDecimalPlaces(tc.amount, tc.currency)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestSanitizeDescription(t *testing.T) {
	validator := NewPaymentValidator()

	// Create a long string
	longString := ""
	for i := 0; i < 600; i++ {
		longString += "a"
	}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Trim spaces", "  test  ", "test"},
		{"Remove control chars", "test\x00\x01", "test"},
		{"Limit length", longString, ""},
		{"Normal text", "Normal description", "Normal description"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.SanitizeDescription(tc.input)
			if tc.name == "Limit length" {
				assert.Len(t, result, 500)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestSanitizeMetadata(t *testing.T) {
	validator := NewPaymentValidator()

	// Nil metadata
	result := validator.SanitizeMetadata(nil)
	assert.Nil(t, result)

	// Trim spaces
	input := map[string]string{
		"  key1  ": "  value1  ",
	}
	result = validator.SanitizeMetadata(input)
	assert.Equal(t, "value1", result["key1"])

	// Remove empty keys
	input = map[string]string{
		"":     "value",
		"key1": "value1",
	}
	result = validator.SanitizeMetadata(input)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "key1")

	// Limit to 50 entries
	input = make(map[string]string)
	for i := 0; i < 60; i++ {
		input[string(rune('a'+i))] = "value"
	}
	result = validator.SanitizeMetadata(input)
	assert.LessOrEqual(t, len(result), 50)

	// Truncate long values
	longValue := ""
	for i := 0; i < 600; i++ {
		longValue += "a"
	}
	input = map[string]string{
		"key1": longValue,
	}
	result = validator.SanitizeMetadata(input)
	assert.Len(t, result["key1"], 500)
}

func TestGetMinimumAmount(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		currency string
		expected float64
	}{
		{"USD", 0.50},
		{"EUR", 0.50},
		{"GBP", 0.30},
		{"CAD", 0.50},
		{"AUD", 0.50},
		{"JPY", 50.00},
		{"XYZ", 0.50}, // Unknown currency returns default
	}

	for _, tc := range testCases {
		t.Run(tc.currency, func(t *testing.T) {
			result := validator.GetMinimumAmount(tc.currency)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsSupportedCurrency(t *testing.T) {
	validator := NewPaymentValidator()

	testCases := []struct {
		currency string
		expected bool
	}{
		{"USD", true},
		{"usd", true},
		{"EUR", true},
		{"GBP", true},
		{"CAD", true},
		{"AUD", true},
		{"JPY", true},
		{"XYZ", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.currency, func(t *testing.T) {
			result := validator.IsSupportedCurrency(tc.currency)
			assert.Equal(t, tc.expected, result)
		})
	}
}
