package csv

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestParser_ParseCustomers_Success(t *testing.T) {
	csvData := `first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type
John,Doe,john@example.com,555-123-4567,123 Main St,Springfield,IL,62701,residential
Jane,Smith,jane@example.com,555-987-6543,456 Oak Ave,Austin,TX,78701,commercial`

	parser := NewParser(strings.NewReader(csvData))
	rows, errors, err := parser.ParseCustomers()

	assert.NoError(t, err)
	assert.Empty(t, errors)
	assert.Len(t, rows, 2)

	// Check first row
	assert.Equal(t, "John", rows[0].FirstName)
	assert.Equal(t, "Doe", rows[0].LastName)
	assert.Equal(t, "john@example.com", rows[0].Email)
	assert.Equal(t, "555-123-4567", rows[0].PhonePrimary)
	assert.Equal(t, "residential", rows[0].CustomerType)

	// Check second row
	assert.Equal(t, "Jane", rows[1].FirstName)
	assert.Equal(t, "commercial", rows[1].CustomerType)
}

func TestParser_ParseCustomers_MissingRequiredField(t *testing.T) {
	csvData := `first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type
John,Doe,,555-123-4567,123 Main St,Springfield,IL,62701,residential`

	parser := NewParser(strings.NewReader(csvData))
	rows, errors, err := parser.ParseCustomers()

	assert.NoError(t, err)
	assert.NotEmpty(t, errors)
	assert.Empty(t, rows)
	assert.Contains(t, errors[0].Error, "email")
}

func TestParser_ParseCustomers_MissingHeader(t *testing.T) {
	csvData := `first_name,last_name,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type
John,Doe,555-123-4567,123 Main St,Springfield,IL,62701,residential`

	parser := NewParser(strings.NewReader(csvData))
	_, errors, err := parser.ParseCustomers()

	assert.NoError(t, err)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0].Error, "email")
}

func TestParser_ParseCustomers_InvalidCustomerType(t *testing.T) {
	csvData := `first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type
John,Doe,john@example.com,555-123-4567,123 Main St,Springfield,IL,62701,invalid_type`

	parser := NewParser(strings.NewReader(csvData))
	rows, errors, err := parser.ParseCustomers()

	assert.NoError(t, err)
	assert.NotEmpty(t, errors)
	assert.Empty(t, rows)
}

func TestParser_ParseCustomers_OptionalFields(t *testing.T) {
	csvData := `first_name,last_name,company_name,email,phone_primary,phone_secondary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,service_address_street,service_address_city,service_address_state,service_address_zip,customer_type,status,notes
John,Doe,Acme Corp,john@example.com,555-123-4567,555-123-4568,123 Main St,Springfield,IL,62701,456 Service Rd,Dallas,TX,75201,commercial,active,VIP customer`

	parser := NewParser(strings.NewReader(csvData))
	rows, errors, err := parser.ParseCustomers()

	assert.NoError(t, err)
	assert.Empty(t, errors)
	assert.Len(t, rows, 1)

	row := rows[0]
	assert.Equal(t, "Acme Corp", row.CompanyName)
	assert.Equal(t, "555-123-4568", row.PhoneSecondary)
	assert.Equal(t, "456 Service Rd", row.ServiceAddressStreet)
	assert.Equal(t, "Dallas", row.ServiceAddressCity)
	assert.Equal(t, "TX", row.ServiceAddressState)
	assert.Equal(t, "75201", row.ServiceAddressZip)
	assert.Equal(t, "active", row.Status)
	assert.Equal(t, "VIP customer", row.Notes)
}

func TestValidateCustomerData_ValidEmail(t *testing.T) {
	row := &models.CSVImportRow{
		Email: "john@example.com",
	}

	errors := ValidateCustomerData(row)
	assert.Empty(t, errors)
}

func TestValidateCustomerData_InvalidEmail(t *testing.T) {
	row := &models.CSVImportRow{
		Email: "invalid-email",
	}

	errors := ValidateCustomerData(row)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "email")
}

func TestValidateCustomerData_ValidPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
	}{
		{"Dashes", "555-123-4567"},
		{"Parentheses", "(555) 123-4567"},
		{"Dots", "555.123.4567"},
		{"Plain", "5551234567"},
		{"With country code", "+1-555-123-4567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &models.CSVImportRow{
				PhonePrimary: tt.phone,
			}
			errors := ValidateCustomerData(row)
			assert.Empty(t, errors)
		})
	}
}

func TestValidateCustomerData_InvalidPhone(t *testing.T) {
	row := &models.CSVImportRow{
		PhonePrimary: "123", // Too short
	}

	errors := ValidateCustomerData(row)
	assert.NotEmpty(t, errors)
}

func TestValidateCustomerData_ValidState(t *testing.T) {
	row := &models.CSVImportRow{
		BillingAddressState: "TX",
	}

	errors := ValidateCustomerData(row)
	assert.Empty(t, errors)
	assert.Equal(t, "TX", row.BillingAddressState)
}

func TestValidateCustomerData_InvalidStateLength(t *testing.T) {
	row := &models.CSVImportRow{
		BillingAddressState: "TEX", // Too long
	}

	errors := ValidateCustomerData(row)
	assert.NotEmpty(t, errors)
}

func TestValidateCustomerData_ValidZip(t *testing.T) {
	tests := []struct {
		name string
		zip  string
	}{
		{"5 digits", "78701"},
		{"9 digits", "787011234"},
		{"5+4 format", "78701-1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &models.CSVImportRow{
				BillingAddressZip: tt.zip,
			}
			errors := ValidateCustomerData(row)
			assert.Empty(t, errors)
		})
	}
}

func TestValidateCustomerData_InvalidZip(t *testing.T) {
	row := &models.CSVImportRow{
		BillingAddressZip: "123", // Too short
	}

	errors := ValidateCustomerData(row)
	assert.NotEmpty(t, errors)
}

func TestCountRows(t *testing.T) {
	csvData := `first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type
John,Doe,john@example.com,555-123-4567,123 Main St,Springfield,IL,62701,residential
Jane,Smith,jane@example.com,555-987-6543,456 Oak Ave,Austin,TX,78701,commercial`

	count, err := CountRows(strings.NewReader(csvData))
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestCountRows_EmptyFile(t *testing.T) {
	csvData := `first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type`

	count, err := CountRows(strings.NewReader(csvData))
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}
