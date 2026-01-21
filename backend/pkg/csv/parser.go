package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Parser handles CSV parsing for customer imports
type Parser struct {
	reader *csv.Reader
}

// NewParser creates a new CSV parser
func NewParser(r io.Reader) *Parser {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	return &Parser{reader: reader}
}

// ParseCustomers parses CSV data into customer import rows
func (p *Parser) ParseCustomers() ([]models.CSVImportRow, []ParseError, error) {
	var rows []models.CSVImportRow
	var errors []ParseError

	// Read header
	header, err := p.reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	headerMap, headerErrors := validateHeader(header)
	if len(headerErrors) > 0 {
		return nil, headerErrors, nil
	}

	rowNum := 1 // Start at 1 for data rows (header is row 0)
	for {
		record, err := p.reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, ParseError{
				Row:   rowNum,
				Field: "",
				Error: fmt.Sprintf("failed to read row: %v", err),
			})
			rowNum++
			continue
		}

		row, rowErrors := parseRow(record, headerMap, rowNum)
		if len(rowErrors) > 0 {
			errors = append(errors, rowErrors...)
		} else {
			rows = append(rows, row)
		}

		rowNum++
	}

	return rows, errors, nil
}

// ParseError represents a parsing error
type ParseError struct {
	Row   int
	Field string
	Error string
}

// String returns a string representation of the parse error
func (e ParseError) String() string {
	if e.Field != "" {
		return fmt.Sprintf("row %d, field '%s': %s", e.Row, e.Field, e.Error)
	}
	return fmt.Sprintf("row %d: %s", e.Row, e.Error)
}

// Required CSV headers
var requiredHeaders = map[string]bool{
	"first_name":             true,
	"last_name":              true,
	"email":                  true,
	"phone_primary":          true,
	"billing_address_street": true,
	"billing_address_city":   true,
	"billing_address_state":  true,
	"billing_address_zip":    true,
	"customer_type":          true,
}

// Optional CSV headers
var optionalHeaders = map[string]bool{
	"company_name":           true,
	"phone_secondary":        true,
	"service_address_street": true,
	"service_address_city":   true,
	"service_address_state":  true,
	"service_address_zip":    true,
	"status":                 true,
	"notes":                  true,
}

// validateHeader validates the CSV header row
func validateHeader(header []string) (map[string]int, []ParseError) {
	var errors []ParseError
	headerMap := make(map[string]int)

	// Normalize and map headers to column indices
	for i, h := range header {
		normalized := strings.ToLower(strings.TrimSpace(h))
		if normalized == "" {
			continue
		}
		headerMap[normalized] = i
	}

	// Check for required headers
	for required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			errors = append(errors, ParseError{
				Row:   0,
				Field: required,
				Error: fmt.Sprintf("required header '%s' is missing", required),
			})
		}
	}

	return headerMap, errors
}

// parseRow parses a single CSV row
func parseRow(record []string, headerMap map[string]int, rowNum int) (models.CSVImportRow, []ParseError) {
	var errors []ParseError

	row := models.CSVImportRow{
		FirstName:            getField(record, headerMap, "first_name"),
		LastName:             getField(record, headerMap, "last_name"),
		CompanyName:          getField(record, headerMap, "company_name"),
		Email:                getField(record, headerMap, "email"),
		PhonePrimary:         getField(record, headerMap, "phone_primary"),
		PhoneSecondary:       getField(record, headerMap, "phone_secondary"),
		BillingAddressStreet: getField(record, headerMap, "billing_address_street"),
		BillingAddressCity:   getField(record, headerMap, "billing_address_city"),
		BillingAddressState:  getField(record, headerMap, "billing_address_state"),
		BillingAddressZip:    getField(record, headerMap, "billing_address_zip"),
		ServiceAddressStreet: getField(record, headerMap, "service_address_street"),
		ServiceAddressCity:   getField(record, headerMap, "service_address_city"),
		ServiceAddressState:  getField(record, headerMap, "service_address_state"),
		ServiceAddressZip:    getField(record, headerMap, "service_address_zip"),
		CustomerType:         getField(record, headerMap, "customer_type"),
		Status:               getField(record, headerMap, "status"),
		Notes:                getField(record, headerMap, "notes"),
	}

	// Validate the row
	validationErrors := row.Validate()
	for _, err := range validationErrors {
		errors = append(errors, ParseError{
			Row:   rowNum,
			Field: extractFieldName(err),
			Error: err,
		})
	}

	return row, errors
}

// getField safely retrieves a field value from a CSV record
func getField(record []string, headerMap map[string]int, fieldName string) string {
	idx, exists := headerMap[fieldName]
	if !exists || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

// extractFieldName extracts the field name from a validation error message
func extractFieldName(errMsg string) string {
	if strings.Contains(errMsg, "first_name") {
		return "first_name"
	}
	if strings.Contains(errMsg, "last_name") {
		return "last_name"
	}
	if strings.Contains(errMsg, "email") {
		return "email"
	}
	if strings.Contains(errMsg, "phone_primary") {
		return "phone_primary"
	}
	if strings.Contains(errMsg, "billing_address_street") {
		return "billing_address_street"
	}
	if strings.Contains(errMsg, "billing_address_city") {
		return "billing_address_city"
	}
	if strings.Contains(errMsg, "billing_address_state") {
		return "billing_address_state"
	}
	if strings.Contains(errMsg, "billing_address_zip") {
		return "billing_address_zip"
	}
	if strings.Contains(errMsg, "customer_type") {
		return "customer_type"
	}
	return ""
}

// ValidateCustomerData validates customer data for import
func ValidateCustomerData(row *models.CSVImportRow) []string {
	var errors []string

	// Email format validation
	if row.Email != "" && !isValidEmail(row.Email) {
		errors = append(errors, "email format is invalid")
	}

	// Phone format validation
	if row.PhonePrimary != "" && !isValidPhone(row.PhonePrimary) {
		errors = append(errors, "phone_primary format is invalid")
	}

	if row.PhoneSecondary != "" && !isValidPhone(row.PhoneSecondary) {
		errors = append(errors, "phone_secondary format is invalid")
	}

	// State validation (must be 2 uppercase letters)
	if row.BillingAddressState != "" {
		row.BillingAddressState = strings.ToUpper(row.BillingAddressState)
		if len(row.BillingAddressState) != 2 {
			errors = append(errors, "billing_address_state must be 2 characters")
		}
	}

	if row.ServiceAddressState != "" {
		row.ServiceAddressState = strings.ToUpper(row.ServiceAddressState)
		if len(row.ServiceAddressState) != 2 {
			errors = append(errors, "service_address_state must be 2 characters")
		}
	}

	// ZIP code validation
	if row.BillingAddressZip != "" && !isValidZip(row.BillingAddressZip) {
		errors = append(errors, "billing_address_zip format is invalid")
	}

	if row.ServiceAddressZip != "" && !isValidZip(row.ServiceAddressZip) {
		errors = append(errors, "service_address_zip format is invalid")
	}

	return errors
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// isValidPhone performs basic phone validation
func isValidPhone(phone string) bool {
	// Remove common formatting characters
	cleaned := strings.ReplaceAll(phone, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	// Check if it's at least 10 digits
	digitCount := 0
	for _, c := range cleaned {
		if c >= '0' && c <= '9' {
			digitCount++
		}
	}

	return digitCount >= 10 && digitCount <= 15
}

// isValidZip validates ZIP code format
func isValidZip(zip string) bool {
	// US ZIP: 5 digits or 5+4 format
	cleaned := strings.ReplaceAll(zip, "-", "")
	return len(cleaned) == 5 || len(cleaned) == 9
}

// CountRows counts the number of data rows in a CSV file (excluding header)
func CountRows(r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	count := 0

	// Skip header
	if _, err := reader.Read(); err != nil {
		return 0, fmt.Errorf("failed to read header: %w", err)
	}

	// Count data rows
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("error reading row %d: %w", count+1, err)
		}
		count++
	}

	return count, nil
}
