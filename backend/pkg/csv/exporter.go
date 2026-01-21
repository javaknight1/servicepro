package csv

import (
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Exporter handles CSV and JSON export formatting
type Exporter struct {
	writer io.Writer
}

// NewExporter creates a new exporter
func NewExporter(w io.Writer) *Exporter {
	return &Exporter{writer: w}
}

// ExportCustomersCSV exports customers to CSV format
func (e *Exporter) ExportCustomersCSV(customers []models.Customer, includeNotes bool) error {
	writer := csv.NewWriter(e.writer)
	defer writer.Flush()

	// Write header
	header := []string{
		"id",
		"first_name",
		"last_name",
		"company_name",
		"email",
		"phone_primary",
		"phone_secondary",
		"billing_address_street",
		"billing_address_city",
		"billing_address_state",
		"billing_address_zip",
		"service_address_street",
		"service_address_city",
		"service_address_state",
		"service_address_zip",
		"customer_type",
		"status",
	}

	if includeNotes {
		header = append(header, "notes")
	}

	header = append(header, "created_at", "updated_at")

	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, customer := range customers {
		row := models.CSVExportRow{}
		row.FromCustomer(&customer)

		record := []string{
			row.ID,
			row.FirstName,
			row.LastName,
			row.CompanyName,
			row.Email,
			row.PhonePrimary,
			row.PhoneSecondary,
			row.BillingAddressStreet,
			row.BillingAddressCity,
			row.BillingAddressState,
			row.BillingAddressZip,
			row.ServiceAddressStreet,
			row.ServiceAddressCity,
			row.ServiceAddressState,
			row.ServiceAddressZip,
			row.CustomerType,
			row.Status,
		}

		if includeNotes {
			record = append(record, row.Notes)
		}

		record = append(record, row.CreatedAt, row.UpdatedAt)

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportCustomersJSON exports customers to JSON format
func (e *Exporter) ExportCustomersJSON(customers []models.Customer) error {
	// Convert to response format
	responses := make([]models.CustomerResponse, len(customers))
	for i, customer := range customers {
		responses[i] = customer.ToResponse()
	}

	encoder := json.NewEncoder(e.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(responses)
}

// GetCSVTemplate generates a CSV template for import
func GetCSVTemplate() []byte {
	template := `first_name,last_name,company_name,email,phone_primary,phone_secondary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,service_address_street,service_address_city,service_address_state,service_address_zip,customer_type,status,notes
John,Doe,Acme Corp,john.doe@example.com,555-123-4567,555-123-4568,123 Main St,Springfield,IL,62701,123 Main St,Springfield,IL,62701,commercial,active,Important client
Jane,Smith,,jane.smith@example.com,555-987-6543,,456 Oak Ave,Austin,TX,78701,,,,,residential,prospect,New lead from website`
	return []byte(template)
}
