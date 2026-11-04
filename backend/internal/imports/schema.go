// Package imports turns exported spreadsheets (Jobber, Housecall Pro or any
// CSV/XLSX) into normalised, validated records for ServicePro's import runner
// (T193). It is pure: no database, no tenant. The runner in
// internal/services matches the records against the tenant's data and writes
// them.
//
// Flow: ReadTable (size/row caps) → AutoMap or ApplyMapping (headers → target
// fields) → Extract (one Record per data row) → Validate (normalises values
// in place and reports per-field errors).
package imports

import "fmt"

// Source is where an export came from. It picks the header aliases used to
// map columns automatically and is stored with each imported record's
// external ID, so the same file imported twice is recognised.
type Source string

const (
	SourceGeneric      Source = "generic"
	SourceJobber       Source = "jobber"
	SourceHousecallPro Source = "housecall_pro"
)

// Sources lists the sources the API accepts.
var Sources = []Source{SourceGeneric, SourceJobber, SourceHousecallPro}

// Label is the source's display name.
func (s Source) Label() string {
	switch s {
	case SourceJobber:
		return "Jobber"
	case SourceHousecallPro:
		return "Housecall Pro"
	default:
		return "CSV"
	}
}

// ParseSource validates a source name.
func ParseSource(v string) (Source, error) {
	for _, s := range Sources {
		if string(s) == v {
			return s, nil
		}
	}
	return "", fmt.Errorf("unknown source %q (use generic, jobber or housecall_pro)", v)
}

// Entity is what one file imports.
type Entity string

const (
	EntityCustomers Entity = "customers"
	EntitySites     Entity = "sites"
	EntityJobs      Entity = "jobs"
	EntityInvoices  Entity = "invoices"
)

// Entities lists the entities the API accepts, in the order they should be
// imported (sites, jobs and invoices are matched to existing customers).
var Entities = []Entity{EntityCustomers, EntitySites, EntityJobs, EntityInvoices}

// ParseEntity validates an entity name.
func ParseEntity(v string) (Entity, error) {
	for _, e := range Entities {
		if string(e) == v {
			return e, nil
		}
	}
	return "", fmt.Errorf("unknown entity %q (use customers, sites, jobs or invoices)", v)
}

// Field is one target field an import column can map to.
type Field struct {
	Key      string `json:"key" example:"email"`
	Label    string `json:"label" example:"Email"`
	Required bool   `json:"required" example:"true"`
	// Group names fields of which at least one must be mapped (e.g. a
	// customer needs a person or company name). Empty for most fields.
	Group string `json:"group,omitempty" example:"name"`
	Help  string `json:"help,omitempty" example:"First address if the cell lists several"`
}

// Field keys shared by several entities.
const (
	FExternalID         = "external_id"
	FCustomerExternalID = "customer_external_id"
	FCustomerEmail      = "customer_email"
	FCustomerName       = "customer_name"
	FStreet             = "street"
	FStreet2            = "street2"
	FCity               = "city"
	FState              = "state"
	FZip                = "zip"
	FNotes              = "notes"
)

// Customer field keys.
const (
	FFirstName      = "first_name"
	FLastName       = "last_name"
	FFullName       = "full_name"
	FCompanyName    = "company_name"
	FEmail          = "email"
	FPhone          = "phone"
	FPhoneSecondary = "phone_secondary"
	FBillingStreet  = "billing_street"
	FBillingStreet2 = "billing_street2"
	FBillingCity    = "billing_city"
	FBillingState   = "billing_state"
	FBillingZip     = "billing_zip"
	FSiteName       = "site_name"
	FCustomerType   = "customer_type"
	FStatus         = "status"
	FTags           = "tags"
)

// Site field keys.
const (
	FName         = "name"
	FAccessNotes  = "access_notes"
	FContactName  = "contact_name"
	FContactPhone = "contact_phone"
	FContactEmail = "contact_email"
)

// Job field keys.
const (
	FTitle          = "title"
	FDescription    = "description"
	FJobType        = "job_type"
	FScheduledStart = "scheduled_start"
	FScheduledEnd   = "scheduled_end"
	FCompletedAt    = "completed_at"
	FPONumber       = "po_number"
)

// Invoice field keys.
const (
	FInvoiceNumber = "invoice_number"
	FIssueDate     = "issue_date"
	FDueDate       = "due_date"
	FTotal         = "total"
	FAmountPaid    = "amount_paid"
	FBalance       = "balance"
	FSubject       = "subject"
)

var customerRef = []Field{
	{Key: FCustomerExternalID, Label: "Customer ID (from the old system)", Group: "customer", Help: "Matches customers imported earlier from the same source"},
	{Key: FCustomerEmail, Label: "Customer email", Group: "customer"},
	{Key: FCustomerName, Label: "Customer name", Group: "customer", Help: "Company name, or first and last name"},
}

var fields = map[Entity][]Field{
	EntityCustomers: {
		{Key: FExternalID, Label: "Customer ID (from the old system)", Help: "Lets a re-import recognise customers it already created"},
		{Key: FFirstName, Label: "First name", Group: "name"},
		{Key: FLastName, Label: "Last name", Group: "name"},
		{Key: FFullName, Label: "Full name", Group: "name", Help: "Split into first and last name when those columns are missing"},
		{Key: FCompanyName, Label: "Company name", Group: "name"},
		{Key: FEmail, Label: "Email", Required: true, Help: "First address if the cell lists several"},
		{Key: FPhone, Label: "Phone"},
		{Key: FPhoneSecondary, Label: "Second phone"},
		{Key: FBillingStreet, Label: "Billing street"},
		{Key: FBillingStreet2, Label: "Billing street line 2"},
		{Key: FBillingCity, Label: "Billing city"},
		{Key: FBillingState, Label: "Billing state"},
		{Key: FBillingZip, Label: "Billing ZIP"},
		{Key: FSiteName, Label: "Site (property) name"},
		{Key: FStreet, Label: "Service street", Help: "Creates the customer's first site; also the billing address when that is empty"},
		{Key: FStreet2, Label: "Service street line 2"},
		{Key: FCity, Label: "Service city"},
		{Key: FState, Label: "Service state"},
		{Key: FZip, Label: "Service ZIP"},
		{Key: FCustomerType, Label: "Customer type", Help: "residential or commercial; defaults to commercial when there is a company name"},
		{Key: FStatus, Label: "Status", Help: "active, inactive, lead/prospect or archived"},
		{Key: FTags, Label: "Tags", Help: "Kept in the customer's notes"},
		{Key: FNotes, Label: "Notes"},
	},
	EntitySites: append(append([]Field{}, customerRef...),
		Field{Key: FName, Label: "Site name"},
		Field{Key: FStreet, Label: "Street", Required: true},
		Field{Key: FStreet2, Label: "Street line 2"},
		Field{Key: FCity, Label: "City"},
		Field{Key: FState, Label: "State"},
		Field{Key: FZip, Label: "ZIP"},
		Field{Key: FAccessNotes, Label: "Access notes"},
		Field{Key: FContactName, Label: "Site contact name"},
		Field{Key: FContactPhone, Label: "Site contact phone"},
		Field{Key: FContactEmail, Label: "Site contact email"},
	),
	EntityJobs: append(append([]Field{
		{Key: FExternalID, Label: "Job number or ID", Help: "Kept as the job number and used to recognise a re-import"},
	}, customerRef...),
		Field{Key: FTitle, Label: "Title"},
		Field{Key: FDescription, Label: "Description"},
		Field{Key: FJobType, Label: "Job type", Help: "installation, maintenance, repair, inspection or emergency; defaults to maintenance"},
		Field{Key: FStatus, Label: "Status"},
		Field{Key: FScheduledStart, Label: "Scheduled start"},
		Field{Key: FScheduledEnd, Label: "Scheduled end"},
		Field{Key: FCompletedAt, Label: "Completed date"},
		Field{Key: FStreet, Label: "Service street", Help: "Matched to (or creates) one of the customer's sites"},
		Field{Key: FStreet2, Label: "Service street line 2"},
		Field{Key: FCity, Label: "Service city"},
		Field{Key: FState, Label: "Service state"},
		Field{Key: FZip, Label: "Service ZIP"},
		Field{Key: FPONumber, Label: "PO number"},
		Field{Key: FNotes, Label: "Notes"},
	),
	EntityInvoices: append(append([]Field{
		{Key: FInvoiceNumber, Label: "Invoice number", Required: true},
	}, customerRef...),
		Field{Key: FIssueDate, Label: "Issue date"},
		Field{Key: FDueDate, Label: "Due date", Help: "Defaults to the issue date"},
		Field{Key: FTotal, Label: "Total", Group: "amount"},
		Field{Key: FAmountPaid, Label: "Amount paid"},
		Field{Key: FBalance, Label: "Balance due", Group: "amount", Help: "Only invoices with a balance are imported; defaults to total minus amount paid"},
		Field{Key: FStatus, Label: "Status"},
		Field{Key: FSubject, Label: "Subject"},
		Field{Key: FPONumber, Label: "PO number"},
		Field{Key: FNotes, Label: "Notes"},
	),
}

// Fields returns the target fields of an entity (a copy).
func Fields(e Entity) []Field {
	return append([]Field(nil), fields[e]...)
}

// groupLabels describe each "at least one of" group for error messages.
var groupLabels = map[string]string{
	"name":     "a name (first/last, full name or company)",
	"customer": "a customer column (ID, email or name)",
	"amount":   "an amount column (total or balance due)",
}
