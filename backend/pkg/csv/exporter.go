package csv

// GetCSVTemplate generates a CSV template for import
func GetCSVTemplate() []byte {
	template := `first_name,last_name,company_name,email,phone_primary,phone_secondary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,service_address_street,service_address_city,service_address_state,service_address_zip,customer_type,status,notes
John,Doe,Acme Corp,john.doe@example.com,555-123-4567,555-123-4568,123 Main St,Springfield,IL,62701,123 Main St,Springfield,IL,62701,commercial,active,Important client
Jane,Smith,,jane.smith@example.com,555-987-6543,,456 Oak Ave,Austin,TX,78701,,,,,residential,prospect,New lead from website`
	return []byte(template)
}
