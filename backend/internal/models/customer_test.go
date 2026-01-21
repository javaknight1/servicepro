package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomer_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		customer Customer
		want     string
	}{
		{
			name: "Residential customer without company",
			customer: Customer{
				FirstName:   "John",
				LastName:    "Doe",
				CompanyName: nil,
			},
			want: "John Doe",
		},
		{
			name: "Commercial customer with company",
			customer: Customer{
				FirstName:   "Jane",
				LastName:    "Smith",
				CompanyName: stringPtr("Acme Corp"),
			},
			want: "Acme Corp (Jane Smith)",
		},
		{
			name: "Customer with empty company name",
			customer: Customer{
				FirstName:   "Bob",
				LastName:    "Johnson",
				CompanyName: stringPtr(""),
			},
			want: "Bob Johnson",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.customer.GetDisplayName()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCustomer_GetBillingAddress(t *testing.T) {
	customer := Customer{
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
	}

	expected := "123 Main St, Springfield, IL 62701"
	got := customer.GetBillingAddress()
	assert.Equal(t, expected, got)
}

func TestCustomer_GetServiceAddress(t *testing.T) {
	tests := []struct {
		name     string
		customer Customer
		want     string
	}{
		{
			name: "With separate service address",
			customer: Customer{
				BillingAddressStreet: "123 Main St",
				BillingAddressCity:   "Springfield",
				BillingAddressState:  "IL",
				BillingAddressZip:    "62701",
				ServiceAddressStreet: stringPtr("456 Oak Ave"),
				ServiceAddressCity:   stringPtr("Chicago"),
				ServiceAddressState:  stringPtr("IL"),
				ServiceAddressZip:    stringPtr("60601"),
			},
			want: "456 Oak Ave, Chicago, IL 60601",
		},
		{
			name: "Without service address - uses billing",
			customer: Customer{
				BillingAddressStreet: "123 Main St",
				BillingAddressCity:   "Springfield",
				BillingAddressState:  "IL",
				BillingAddressZip:    "62701",
				ServiceAddressStreet: nil,
				ServiceAddressCity:   nil,
				ServiceAddressState:  nil,
				ServiceAddressZip:    nil,
			},
			want: "123 Main St, Springfield, IL 62701",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.customer.GetServiceAddress()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCustomer_HasSeparateServiceAddress(t *testing.T) {
	tests := []struct {
		name     string
		customer Customer
		want     bool
	}{
		{
			name: "Has separate service address",
			customer: Customer{
				ServiceAddressStreet: stringPtr("456 Oak Ave"),
			},
			want: true,
		},
		{
			name: "No separate service address",
			customer: Customer{
				ServiceAddressStreet: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.customer.HasSeparateServiceAddress()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCustomer_ToResponse(t *testing.T) {
	customerID := uuid.New()
	companyName := "Tech Solutions LLC"
	phoneSecondary := "555-987-6543"
	notes := "Preferred customer"

	customer := Customer{
		ID:                   customerID,
		FirstName:            "Alice",
		LastName:             "Williams",
		CompanyName:          &companyName,
		Email:                "alice@techsolutions.com",
		PhonePrimary:         "555-123-4567",
		PhoneSecondary:       &phoneSecondary,
		BillingAddressStreet: "789 Business Blvd",
		BillingAddressCity:   "Chicago",
		BillingAddressState:  "IL",
		BillingAddressZip:    "60602",
		ServiceAddressStreet: stringPtr("321 Service Lane"),
		ServiceAddressCity:   stringPtr("Naperville"),
		ServiceAddressState:  stringPtr("IL"),
		ServiceAddressZip:    stringPtr("60540"),
		CustomerType:         CustomerTypeCommercial,
		Status:               CustomerStatusActive,
		Notes:                &notes,
	}

	resp := customer.ToResponse()

	// Basic fields
	assert.Equal(t, customerID, resp.ID)
	assert.Equal(t, "Alice", resp.FirstName)
	assert.Equal(t, "Williams", resp.LastName)
	assert.Equal(t, "Tech Solutions LLC (Alice Williams)", resp.DisplayName)
	assert.Equal(t, &companyName, resp.CompanyName)
	assert.Equal(t, "alice@techsolutions.com", resp.Email)
	assert.Equal(t, "555-123-4567", resp.PhonePrimary)
	assert.Equal(t, &phoneSecondary, resp.PhoneSecondary)

	// Billing address
	assert.Equal(t, "789 Business Blvd", resp.BillingAddressStreet)
	assert.Equal(t, "Chicago", resp.BillingAddressCity)
	assert.Equal(t, "IL", resp.BillingAddressState)
	assert.Equal(t, "60602", resp.BillingAddressZip)
	assert.Equal(t, "789 Business Blvd, Chicago, IL 60602", resp.BillingAddressFull)

	// Service address
	assert.Equal(t, "321 Service Lane", *resp.ServiceAddressStreet)
	assert.Equal(t, "Naperville", *resp.ServiceAddressCity)
	assert.Equal(t, "IL", *resp.ServiceAddressState)
	assert.Equal(t, "60540", *resp.ServiceAddressZip)
	assert.Equal(t, "321 Service Lane, Naperville, IL 60540", resp.ServiceAddressFull)

	// Business info
	assert.Equal(t, CustomerTypeCommercial, resp.CustomerType)
	assert.Equal(t, CustomerStatusActive, resp.Status)
	assert.Equal(t, &notes, resp.Notes)
}

func TestCustomer_ToResponse_WithoutServiceAddress(t *testing.T) {
	customerID := uuid.New()

	customer := Customer{
		ID:                   customerID,
		FirstName:            "Bob",
		LastName:             "Smith",
		CompanyName:          nil,
		Email:                "bob@example.com",
		PhonePrimary:         "555-111-2222",
		PhoneSecondary:       nil,
		BillingAddressStreet: "456 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		ServiceAddressStreet: nil,
		ServiceAddressCity:   nil,
		ServiceAddressState:  nil,
		ServiceAddressZip:    nil,
		CustomerType:         CustomerTypeResidential,
		Status:               CustomerStatusProspect,
		Notes:                nil,
	}

	resp := customer.ToResponse()

	// Check that display name doesn't include company
	assert.Equal(t, "Bob Smith", resp.DisplayName)

	// Check that service address falls back to billing
	assert.Nil(t, resp.ServiceAddressStreet)
	assert.Equal(t, "456 Main St, Springfield, IL 62701", resp.ServiceAddressFull)
}

func TestCustomerType_Constants(t *testing.T) {
	assert.Equal(t, CustomerType("residential"), CustomerTypeResidential)
	assert.Equal(t, CustomerType("commercial"), CustomerTypeCommercial)
}

func TestCustomerStatus_Constants(t *testing.T) {
	assert.Equal(t, CustomerStatus("active"), CustomerStatusActive)
	assert.Equal(t, CustomerStatus("inactive"), CustomerStatusInactive)
	assert.Equal(t, CustomerStatus("prospect"), CustomerStatusProspect)
}

func TestCustomer_TableName(t *testing.T) {
	customer := Customer{}
	assert.Equal(t, "customers", customer.TableName())
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
