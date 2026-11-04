package imports

// Header aliases per source, entity and field, in priority order. Headers are
// compared after normalizeHeader (lower case, letters and digits only), so
// "Main Phone #s", "main_phone_s" and "MAIN PHONE S" are the same header.
// When several aliases of one field are present, each row takes the first
// non-empty one (e.g. Jobber's main phone, then work, then home).
//
// The Jobber and Housecall Pro layouts are documented in
// docs/api/imports.md with their sources and the assumptions made where the
// vendors don't publish their column lists. Source-specific aliases come
// are matched before the generic ones (which also cover ServicePro's own
// customer template); see AutoMap.

type aliasTable map[Entity]map[string][]string

var genericAliases = aliasTable{
	EntityCustomers: {
		FExternalID:     {"external id", "customer id", "client id", "id", "customer number", "account number"},
		FFirstName:      {"first name", "firstname", "given name", "contact first name"},
		FLastName:       {"last name", "lastname", "surname", "family name", "contact last name"},
		FFullName:       {"full name", "name", "contact name", "customer name", "client name", "display name", "customer", "client"},
		FCompanyName:    {"company name", "company", "business name", "organization", "organisation"},
		FEmail:          {"email", "email address", "e-mail", "emails", "e-mails", "primary email", "main email"},
		FPhone:          {"phone primary", "phone", "phone number", "main phone", "primary phone", "mobile", "mobile phone", "cell", "cell phone", "telephone"},
		FPhoneSecondary: {"phone secondary", "secondary phone", "alternate phone", "other phone", "work phone", "home phone"},
		FBillingStreet:  {"billing address street", "billing street", "billing street 1", "billing address", "billing address 1", "billing address line 1", "street", "street 1", "address", "address 1", "address line 1"},
		FBillingStreet2: {"billing street 2", "billing address 2", "billing address line 2", "street 2", "address 2", "address line 2"},
		FBillingCity:    {"billing address city", "billing city", "city", "town"},
		FBillingState:   {"billing address state", "billing state", "billing province", "state", "province", "region"},
		FBillingZip:     {"billing address zip", "billing zip", "billing zip code", "billing postal code", "zip", "zip code", "postal code", "postcode"},
		FSiteName:       {"site name", "property name", "location name", "service property name"},
		FStreet:         {"service address street", "service street", "service street 1", "service address", "property street", "property street 1", "property address", "site street", "site address"},
		FStreet2:        {"service street 2", "service address 2", "property street 2", "site street 2"},
		FCity:           {"service address city", "service city", "property city", "site city"},
		FState:          {"service address state", "service state", "service province", "property state", "property province", "site state"},
		FZip:            {"service address zip", "service zip", "service zip code", "service postal code", "property zip", "property zip code", "property postal code", "site zip"},
		FCustomerType:   {"customer type", "client type", "type"},
		FStatus:         {"status", "customer status", "client status"},
		FTags:           {"tags", "customer tags", "client tags"},
		FNotes:          {"notes", "note", "comments", "customer notes", "client notes"},
	},
	EntitySites: {
		FCustomerExternalID: {"customer id", "client id", "customer external id", "client j-id", "client jid"},
		FCustomerEmail:      {"customer email", "client email", "email", "email address"},
		FCustomerName:       {"customer name", "client name", "customer", "client", "company name", "company", "display name"},
		FName:               {"site name", "property name", "location name", "name"},
		FStreet:             {"street", "street 1", "address", "address 1", "address line 1", "street line 1", "property street", "property street 1", "service street", "service street 1", "site street"},
		FStreet2:            {"street 2", "address 2", "address line 2", "street line 2", "property street 2", "service street 2"},
		FCity:               {"city", "town", "property city", "service city"},
		FState:              {"state", "province", "region", "property state", "property province", "service state"},
		FZip:                {"zip", "zip code", "postal code", "postcode", "property zip", "property zip code", "service zip", "service zip code"},
		FAccessNotes:        {"access notes", "access instructions", "gate code", "instructions", "property notes", "site notes", "notes"},
		FContactName:        {"site contact", "site contact name", "contact name", "property contact"},
		FContactPhone:       {"site contact phone", "contact phone", "property phone", "phone"},
		FContactEmail:       {"site contact email", "contact email", "property email"},
	},
	EntityJobs: {
		FExternalID:         {"job number", "job #", "job no", "job id", "work order", "work order number", "id"},
		FCustomerExternalID: {"customer id", "client id", "client j-id"},
		FCustomerEmail:      {"customer email", "client email", "email", "email address"},
		FCustomerName:       {"customer name", "client name", "customer", "client", "company name", "company"},
		FTitle:              {"title", "job title", "job name", "subject", "service"},
		FDescription:        {"description", "job description", "instructions", "details", "line items"},
		FJobType:            {"job type", "work type", "service type"},
		FStatus:             {"status", "job status"},
		FScheduledStart:     {"scheduled start", "scheduled start date", "start date", "start", "scheduled date", "visit date", "date"},
		FScheduledEnd:       {"scheduled end", "scheduled end date", "end date", "end", "end time"},
		FCompletedAt:        {"completed date", "completed", "completed at", "completion date", "closed date", "closed"},
		FStreet:             {"service street", "service street 1", "property street", "property street 1", "street", "street 1", "address", "address 1", "address line 1"},
		FStreet2:            {"service street 2", "property street 2", "street 2", "street line 2", "address 2", "address line 2"},
		FCity:               {"service city", "property city", "city"},
		FState:              {"service state", "service province", "property state", "property province", "state", "province"},
		FZip:                {"service zip", "service zip code", "service postal code", "property zip", "property zip code", "property postal code", "zip", "zip code", "postal code"},
		FPONumber:           {"po number", "po #", "po", "purchase order", "purchase order number"},
		FNotes:              {"notes", "note", "internal notes", "comments"},
	},
	EntityInvoices: {
		FInvoiceNumber:      {"invoice number", "invoice #", "invoice no", "invoice", "number", "invoice id"},
		FCustomerExternalID: {"customer id", "client id", "client j-id"},
		FCustomerEmail:      {"customer email", "client email", "email", "email address", "billing email"},
		FCustomerName:       {"customer name", "client name", "customer", "client", "company name", "company", "bill to"},
		FIssueDate:          {"issue date", "issued date", "issued", "invoice date", "date", "created date", "date issued", "sent date"},
		FDueDate:            {"due date", "date due", "payment due"},
		FTotal:              {"total", "total amount", "invoice total", "amount", "grand total"},
		FAmountPaid:         {"amount paid", "paid", "paid amount", "payments", "total paid"},
		FBalance:            {"balance", "balance due", "amount due", "outstanding", "open balance", "due", "remaining"},
		FStatus:             {"status", "invoice status", "payment status"},
		FSubject:            {"subject", "title", "description", "memo"},
		FPONumber:           {"po number", "po #", "po", "purchase order", "purchase order number"},
		FNotes:              {"notes", "note", "comments", "message"},
	},
}

var sourceAliases = map[Source]aliasTable{
	// Jobber: Clients → Export Clients (CSV, emailed) and the Clients,
	// Properties, Jobs and Invoices reports (Reports → Export CSV).
	SourceJobber: {
		EntityCustomers: {
			FExternalID:     {"j-id", "jid", "client j-id", "client id"},
			FFullName:       {"display name", "client name"},
			FCompanyName:    {"company name"},
			FEmail:          {"e-mails", "emails", "email"},
			FPhone:          {"main phone #s", "main phone", "work phone #s", "home phone #s", "other phone #s"},
			FPhoneSecondary: {"mobile phone #s", "mobile phone"},
			FBillingStreet:  {"billing street 1", "billing street"},
			FBillingStreet2: {"billing street 2"},
			FBillingCity:    {"billing city"},
			FBillingState:   {"billing state", "billing province"},
			FBillingZip:     {"billing zip code", "billing zip", "billing postal code"},
			FSiteName:       {"service property name", "property name"},
			FStreet:         {"service street 1", "service street", "property street 1", "property street"},
			FStreet2:        {"service street 2", "property street 2"},
			FCity:           {"service city", "property city"},
			FState:          {"service state", "service province", "property state", "property province"},
			FZip:            {"service zip code", "service zip", "service postal code", "property zip code", "property zip", "property postal code"},
			FStatus:         {"status", "client status", "lead"},
			FTags:           {"tags"},
		},
		EntitySites: {
			FCustomerExternalID: {"client j-id", "j-id", "client id"},
			FCustomerEmail:      {"client email", "client e-mails", "e-mails"},
			FCustomerName:       {"client name", "client", "display name"},
			FName:               {"property name", "name"},
			FStreet:             {"street 1", "property street 1", "street"},
			FStreet2:            {"street 2", "property street 2"},
			FCity:               {"city"},
			FState:              {"state", "province"},
			FZip:                {"zip code", "postal code", "zip"},
		},
		EntityJobs: {
			FExternalID:     {"job #", "job number", "job"},
			FCustomerName:   {"client name", "client"},
			FCustomerEmail:  {"client email", "client e-mail"},
			FTitle:          {"title", "job title"},
			FDescription:    {"instructions", "description"},
			FStatus:         {"job status", "status"},
			FScheduledStart: {"scheduled start date", "start date", "schedule start", "scheduled date"},
			FScheduledEnd:   {"scheduled end date", "end date"},
			FCompletedAt:    {"closed date", "completed date", "completed"},
			FStreet:         {"service street", "property street", "service street 1", "property street 1"},
			FCity:           {"service city", "property city"},
			FState:          {"service province", "service state", "property state", "property province"},
			FZip:            {"service zip", "service postal code", "service zip code", "property zip code", "property postal code"},
		},
		EntityInvoices: {
			FInvoiceNumber: {"invoice #", "invoice number"},
			FCustomerName:  {"client name", "client"},
			FCustomerEmail: {"client email", "client e-mail"},
			FIssueDate:     {"issued date", "issue date", "created date"},
			FDueDate:       {"due date"},
			FTotal:         {"total $", "total"},
			FAmountPaid:    {"paid $", "amount paid", "paid"},
			FBalance:       {"balance $", "balance"},
			FStatus:        {"status"},
			FSubject:       {"subject"},
		},
	},
	// Housecall Pro: Customers → More → Download customer list (customers
	// report) and the Jobs data report. Housecall Pro has no separate
	// invoice export: each job row carries its invoice number, amount, paid
	// amount and amount due, so open invoices import from the jobs report.
	SourceHousecallPro: {
		EntityCustomers: {
			FExternalID:     {"id", "customer id"},
			FCompanyName:    {"company"},
			FPhone:          {"mobile number", "home number", "phone"},
			FPhoneSecondary: {"work number"},
			FStreet:         {"address_1 street line 1", "address 1 street line 1", "street line 1", "street"},
			FStreet2:        {"address_1 street line 2", "address 1 street line 2", "street line 2"},
			FCity:           {"address_1 city", "address 1 city", "city"},
			FState:          {"address_1 state", "address 1 state", "state"},
			FZip:            {"address_1 postal code", "address 1 postal code", "postal code", "zip"},
			FTags:           {"tags"},
		},
		EntitySites: {
			FCustomerExternalID: {"customer id", "id"},
			FCustomerName:       {"customer", "customer name"},
			FCustomerEmail:      {"email"},
			FStreet:             {"street line 1", "street"},
			FStreet2:            {"street line 2"},
			FCity:               {"city"},
			FState:              {"state"},
			FZip:                {"postal code", "zip"},
		},
		EntityJobs: {
			FExternalID:     {"invoice", "job", "job #", "hcp id"},
			FCustomerName:   {"customer"},
			FCustomerEmail:  {"email"},
			FDescription:    {"description"},
			FStatus:         {"job status"},
			FScheduledStart: {"date"},
			FScheduledEnd:   {"end time"},
			FStreet:         {"street"},
			FStreet2:        {"street line 2"},
			FCity:           {"city"},
			FState:          {"state"},
			FZip:            {"zip"},
			FNotes:          {"notes"},
		},
		EntityInvoices: {
			FInvoiceNumber: {"invoice"},
			FCustomerName:  {"customer"},
			FCustomerEmail: {"email"},
			FIssueDate:     {"invoice sent", "date"},
			FTotal:         {"amount"},
			FAmountPaid:    {"paid amount"},
			FBalance:       {"due"},
			FStatus:        {"payment"},
			FSubject:       {"description"},
			FNotes:         {"notes"},
		},
	},
}

// sourceAliasesFor is a field's source-specific alias list (normalised).
func sourceAliasesFor(src Source, e Entity, f Field) []string {
	return normalizeAll(sourceAliases[src][e][f.Key])
}

// genericAliasesFor is a field's generic alias list plus its key and label
// (normalised).
func genericAliasesFor(e Entity, f Field) []string {
	return normalizeAll(append(append([]string{}, genericAliases[e][f.Key]...), f.Key, f.Label))
}

func normalizeAll(list []string) []string {
	out := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, a := range list {
		if n := normalizeHeader(a); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
