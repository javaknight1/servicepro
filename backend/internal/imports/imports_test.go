package imports

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

var testNow = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

// load reads a testdata file through the source's automatic mapping and
// validates every record.
func load(t *testing.T, file string, src Source, e Entity) (*Prepared, map[int][]FieldError) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	require.NoError(t, err)
	tbl, err := ReadTable(data, file)
	require.NoError(t, err)
	p, err := Prepare(tbl, src, e, nil)
	require.NoError(t, err)
	errs := map[int][]FieldError{}
	for i := range p.Records {
		if fe := Validate(src, e, &p.Records[i], testNow); len(fe) > 0 {
			errs[p.Records[i].Row] = fe
		}
	}
	return p, errs
}

func TestJobberClients(t *testing.T) {
	p, errs := load(t, "jobber_clients.csv", SourceJobber, EntityCustomers)
	assert.Empty(t, p.Missing)
	require.Len(t, p.Records, 5)

	riverside := p.Records[0].Values
	assert.Equal(t, "58213901", riverside[FExternalID])
	assert.Equal(t, "Dana", riverside[FFirstName])
	assert.Equal(t, "Whitfield", riverside[FLastName])
	assert.Equal(t, "Riverside Property Management", riverside[FCompanyName])
	assert.Equal(t, "ap@riversidepm.example", riverside[FEmail], "first of the listed emails")
	assert.Equal(t, "512-555-0142", riverside[FPhone], "US phones normalised")
	assert.Equal(t, "512-555-0199", riverside[FPhoneSecondary], "mobile is the second phone")
	assert.Equal(t, "400 Congress Ave, Suite 210", riverside[FBillingStreet])
	assert.Equal(t, "TX", riverside[FBillingState], "state names become codes")
	assert.Equal(t, "Riverside Tower", riverside[FSiteName])
	assert.Equal(t, "1200 Riverside Dr", riverside[FStreet])
	assert.Equal(t, "commercial", riverside[FCustomerType])
	assert.Equal(t, "active", riverside[FStatus])
	assert.Equal(t, "Tags: net30, commercial", riverside[FNotes])

	harbor := p.Records[1].Values
	assert.Equal(t, "Main", harbor[FFirstName], "company-only client gets a placeholder contact")
	assert.Equal(t, "Contact", harbor[FLastName])
	assert.Equal(t, "9000 Mopac Expy, Bldg A", harbor[FBillingStreet], "no billing address: the service address is used")
	assert.Equal(t, "78759", harbor[FBillingZip])

	sam := p.Records[2].Values
	assert.Equal(t, "sam.ortiz@example.com", sam[FEmail])
	assert.Equal(t, "737-555-0123", sam[FPhone], "mobile promoted when there is no main phone")
	assert.Equal(t, "residential", sam[FCustomerType])
	assert.Empty(t, sam[FStreet], "no property: no site")

	require.Contains(t, errs, 5, "the client with no email fails")
	assert.Equal(t, FEmail, errs[5][0].Field)
	assert.NotContains(t, errs, 2)
	assert.NotContains(t, errs, 3)
	assert.NotContains(t, errs, 6, "formula-looking text is data: it is stored as is and escaped on export")
	assert.Equal(t, "=cmd", p.Records[4].Values[FFirstName])
}

func TestJobberProperties(t *testing.T) {
	p, errs := load(t, "jobber_properties.csv", SourceJobber, EntitySites)
	assert.Empty(t, p.Missing)
	assert.Empty(t, errs)
	lamar := p.Records[1].Values
	assert.Equal(t, "Lamar Commons", lamar[FName])
	assert.Equal(t, "2500 S Lamar Blvd, Unit 100", lamar[FStreet])
	assert.Equal(t, "TX", lamar[FState])
	assert.Equal(t, "Riverside Property Management", lamar[FCustomerName])
	assert.Equal(t, "ap@riversidepm.example", lamar[FCustomerEmail])
}

func TestJobberJobs(t *testing.T) {
	p, errs := load(t, "jobber_jobs.csv", SourceJobber, EntityJobs)
	assert.Empty(t, p.Missing)
	require.Len(t, p.Records, 4)

	j101 := p.Records[0].Values
	assert.Equal(t, "101", j101[FExternalID])
	assert.Equal(t, "Nightly janitorial - Riverside Tower", j101[FTitle])
	assert.Equal(t, "2024-03-04", j101[FScheduledStart])
	assert.Equal(t, "2024-03-31", j101[FCompletedAt])
	assert.Equal(t, JobCompleted, j101[FStatus], "archived means done")
	assert.Equal(t, "maintenance", j101[FJobType], "Jobber's recurring/one-off is not a job type")

	assert.Equal(t, JobScheduled, p.Records[1].Values[FStatus], "upcoming visit")
	assert.Equal(t, JobCompleted, p.Records[2].Values[FStatus], "requires invoicing means done")

	require.Contains(t, errs, 5)
	assert.Equal(t, FScheduledStart, errs[5][0].Field)
}

func TestJobberInvoices(t *testing.T) {
	p, errs := load(t, "jobber_invoices.csv", SourceJobber, EntityInvoices)
	assert.Empty(t, p.Missing)
	require.Len(t, p.Records, 5)

	inv := p.Records[0].Values
	assert.Equal(t, "1001", inv[FInvoiceNumber])
	assert.Equal(t, "4200.00", inv[FBalance])
	assert.Equal(t, "4200.00", inv[FTotal])
	assert.Equal(t, "2024-04-01", inv[FIssueDate])
	assert.Equal(t, "2024-05-01", inv[FDueDate])
	assert.Empty(t, inv[SkipKey])

	assert.Equal(t, "nothing left to pay", p.Records[1].Values[SkipKey])
	partial := p.Records[2].Values
	assert.Equal(t, "850.00", partial[FBalance])
	assert.Equal(t, "1850.00", partial[FTotal])
	assert.Equal(t, "2024-05-08", partial[FIssueDate], "US slash dates")
	assert.Equal(t, "2024-06-07", partial[FDueDate])

	require.Contains(t, errs, 5, "unreadable balance")
	assert.Equal(t, FBalance, errs[5][0].Field)
	assert.Equal(t, "abc", p.Records[3].Values[FBalance], "an unreadable balance is not replaced by the total")
	assert.Equal(t, "status is Draft", p.Records[4].Values[SkipKey])
}

func TestCustomerRowsMatchDatabaseChecks(t *testing.T) {
	base := map[string]string{
		FFirstName: "Ann", FLastName: "Lee", FEmail: "ann@lee.example", FPhone: "512 555 0100 x7",
		FPhoneSecondary: "+44 20 7946 0958", FBillingStreet: "1 A St", FBillingCity: "Austin",
		FBillingState: "TX", FBillingZip: "78701",
	}
	rec := func(over map[string]string) *Record {
		v := map[string]string{}
		for k, x := range base {
			v[k] = x
		}
		for k, x := range over {
			v[k] = x
		}
		return &Record{Row: 2, Values: v}
	}
	r := rec(nil)
	assert.Empty(t, Validate(SourceGeneric, EntityCustomers, r, testNow))
	assert.Equal(t, "512-555-0100", r.Values[FPhone])
	assert.Empty(t, r.Values[FPhoneSecondary], "a foreign second number isn't stored as a phone")
	assert.Equal(t, "Other phone: +44 20 7946 0958\nPhone extension: 7", r.Values[FNotes])

	for name, tc := range map[string]struct {
		over  map[string]string
		field string
	}{
		"no phone":          {map[string]string{FPhone: "", FPhoneSecondary: ""}, FPhone},
		"foreign phone":     {map[string]string{FPhone: "+44 20 7946 0958"}, FPhone},
		"province":          {map[string]string{FBillingState: "Ontario"}, FBillingState},
		"postal code":       {map[string]string{FBillingZip: "M5V 3L9"}, FBillingZip},
		"single name":       {map[string]string{FLastName: ""}, FLastName},
		"undotted domain":   {map[string]string{FEmail: "ann@localhost"}, FEmail},
		"no address":        {map[string]string{FBillingStreet: ""}, FBillingStreet},
		"bad customer type": {map[string]string{FCustomerType: "alien"}, FCustomerType},
	} {
		errs := Validate(SourceGeneric, EntityCustomers, rec(tc.over), testNow)
		require.NotEmpty(t, errs, name)
		assert.Equal(t, tc.field, errs[0].Field, name)
	}
}

func TestHousecallProCustomers(t *testing.T) {
	p, errs := load(t, "hcp_customers.csv", SourceHousecallPro, EntityCustomers)
	assert.Empty(t, p.Missing)
	require.Len(t, p.Records, 3)

	maria := p.Records[0].Values
	assert.Equal(t, "cus_8f2a1c", maria[FExternalID])
	assert.Equal(t, "Maria", maria[FFirstName])
	assert.Equal(t, "Lopez Facilities LLC", maria[FCompanyName])
	assert.Equal(t, "214-555-0100", maria[FPhone])
	assert.Equal(t, "214-555-0101", maria[FPhoneSecondary])
	assert.Equal(t, "5 Elm St", maria[FStreet], "first address block")
	assert.Equal(t, "5 Elm St", maria[FBillingStreet], "is also the billing address")
	assert.Equal(t, "Prefers email\nTags: VIP", maria[FNotes], "the customer notes, not the address notes")

	tom := p.Records[1].Values
	assert.Equal(t, "TX", tom[FBillingState])
	assert.Equal(t, "residential", tom[FCustomerType])

	require.Contains(t, errs, 4)
	assert.Equal(t, FEmail, errs[4][0].Field)
	assert.Len(t, errs[4], 1, "one message per problem")
}

func TestHousecallProJobsAndInvoices(t *testing.T) {
	jobs, errs := load(t, "hcp_jobs.csv", SourceHousecallPro, EntityJobs)
	assert.Empty(t, jobs.Missing)
	assert.Empty(t, errs)
	j := jobs.Records[0].Values
	assert.Equal(t, "2041", j[FExternalID])
	assert.Equal(t, "Maria Lopez", j[FCustomerName])
	assert.Equal(t, "Deep clean of floor 3 offices", j[FTitle], "title from the description")
	assert.Equal(t, "2024-07-08T09:00", j[FScheduledStart])
	assert.Equal(t, "2024-07-08T11:30", j[FScheduledEnd], "end time on the start's day")
	assert.Equal(t, "600 Commerce St, Floor 3", j[FStreet])
	assert.Equal(t, JobCompleted, j[FStatus])
	assert.Equal(t, JobScheduled, jobs.Records[2].Values[FStatus])
	assert.Equal(t, JobCancelled, jobs.Records[3].Values[FStatus])

	inv, errs := load(t, "hcp_jobs.csv", SourceHousecallPro, EntityInvoices)
	assert.Empty(t, inv.Missing)
	assert.Empty(t, errs)
	open := inv.Records[0].Values
	assert.Equal(t, "2041", open[FInvoiceNumber])
	assert.Equal(t, "640.00", open[FBalance])
	assert.Equal(t, "2024-07-09", open[FIssueDate], "invoice sent date")
	assert.Empty(t, open[SkipKey])
	assert.Equal(t, "nothing left to pay", inv.Records[1].Values[SkipKey])
	assert.Equal(t, "invoice not sent yet", inv.Records[2].Values[SkipKey])
	assert.Equal(t, "nothing left to pay", inv.Records[3].Values[SkipKey])
}

func TestGenericNeedsMapping(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "generic_customers.csv"))
	require.NoError(t, err)
	tbl, err := ReadTable(data, "generic_customers.csv")
	require.NoError(t, err)

	auto, err := Prepare(tbl, SourceGeneric, EntityCustomers, nil)
	require.NoError(t, err)
	assert.Contains(t, auto.Missing, "Email", "unusual headers aren't guessed")

	mapped, err := Prepare(tbl, SourceGeneric, EntityCustomers, map[string]string{
		FFullName: "Customer", FEmail: "Contact email", FPhone: "Tel", FBillingStreet: "Addr",
		FBillingCity: "Town", FBillingState: "ST", FBillingZip: "Postcode", FCustomerType: "Kind",
		FNotes: "",
	})
	require.NoError(t, err)
	assert.Empty(t, mapped.Missing)
	r := mapped.Records[1]
	assert.Empty(t, Validate(SourceGeneric, EntityCustomers, &r, testNow))
	assert.Equal(t, "Bob", r.Values[FFirstName])
	assert.Equal(t, "Jones", r.Values[FLastName])
	assert.Equal(t, "OR", r.Values[FBillingState])

	_, err = Prepare(tbl, SourceGeneric, EntityCustomers, map[string]string{FEmail: "Nope"})
	assert.ErrorContains(t, err, "not in the file")
	_, err = Prepare(tbl, SourceGeneric, EntityCustomers, map[string]string{FEmail: "Tel", FPhone: "Tel"})
	assert.ErrorContains(t, err, "mapped to both")
	_, err = Prepare(tbl, SourceGeneric, EntityCustomers, map[string]string{"password": "Tel"})
	assert.ErrorContains(t, err, "unknown field")
}

func TestServiceProTemplateAutoMaps(t *testing.T) {
	csv := "first_name,last_name,company_name,email,phone_primary,phone_secondary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,service_address_street,service_address_city,service_address_state,service_address_zip,customer_type,status,notes\n" +
		"Ann,Lee,,ann@lee.example,555-111-2222,,1 A St,Austin,TX,78701,,,,,residential,active,\n"
	tbl, err := ReadTable([]byte(csv), "t.csv")
	require.NoError(t, err)
	p, err := Prepare(tbl, SourceGeneric, EntityCustomers, nil)
	require.NoError(t, err)
	assert.Empty(t, p.Missing)
	assert.Len(t, p.Columns, 17, "every template column maps")
}

func TestReadTableCaps(t *testing.T) {
	_, err := ReadTable(bytes.Repeat([]byte("a"), int(MaxFileBytes)+1), "big.csv")
	assert.ErrorIs(t, err, ErrTooLarge)

	var b strings.Builder
	b.WriteString("email\n")
	for i := 0; i <= MaxRows; i++ {
		fmt.Fprintf(&b, "u%d@x.io\n", i)
	}
	_, err = ReadTable([]byte(b.String()), "rows.csv")
	assert.ErrorIs(t, err, ErrTooManyRows)

	_, err = ReadTable([]byte("email\n\n\n"), "empty.csv")
	assert.ErrorIs(t, err, ErrNoRows)
	_, err = ReadTable([]byte(""), "none.csv")
	assert.ErrorIs(t, err, ErrNoHeader)
	_, err = ReadTable([]byte("PK\x03\x04garbage"), "x.xlsx")
	assert.ErrorIs(t, err, ErrUnreadable)
}

func TestReadTableFormats(t *testing.T) {
	// BOM, semicolons, duplicate and blank headers.
	tbl, err := ReadTable([]byte("\xef\xbb\xbfName;City;City;\nA;X;Y;\n"), "s.csv")
	require.NoError(t, err)
	assert.Equal(t, []string{"Name", "City", "City (2)"}, tbl.Headers)
	assert.Equal(t, []string{"A", "X", "Y"}, tbl.Rows[0])

	// Windows-1252 bytes are read as Latin-1, not rejected.
	tbl, err = ReadTable([]byte("Name\nJos\xe9\n"), "w.csv")
	require.NoError(t, err)
	assert.Equal(t, "José", tbl.Rows[0][0])

	// XLSX (first sheet).
	f := excelize.NewFile()
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"Email", "First Name"}))
	require.NoError(t, f.SetSheetRow("Sheet1", "A2", &[]any{"a@b.io", "Al"}))
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	tbl, err = ReadTable(buf.Bytes(), "customers.xlsx")
	require.NoError(t, err)
	assert.Equal(t, []string{"Email", "First Name"}, tbl.Headers)
	assert.Equal(t, [][]string{{"a@b.io", "Al"}}, tbl.Rows)
}

func TestNormalizers(t *testing.T) {
	for in, want := range map[string]string{
		"2024-01-05": "2024-01-05", "1/5/2024": "2024-01-05", "Jan 5, 2024": "2024-01-05",
		"01/05/2024 2:30 pm": "2024-01-05T14:30", "2024-01-05 14:30": "2024-01-05T14:30",
		"45296": "2024-01-05", "Jan 05, 2024 at 9:00 AM": "2024-01-05T09:00",
	} {
		got, hasTime, err := parseDateTime(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, formatDate(got, hasTime), in)
	}
	_, _, err := parseDateTime("soon")
	assert.Error(t, err)

	for in, want := range map[string]string{
		"$1,234.50": "1234.50", "(12.00)": "-12.00", "0": "0.00", "1234.567": "1234.57", "USD 5": "5.00",
	} {
		d, err := parseMoney(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, d.StringFixed(2), in)
	}
	_, err = parseMoney("12 dollars")
	assert.Error(t, err)

	for in, want := range map[string]string{"Texas": "TX", "tx": "TX", "N.Y.": "NY", "Ontario": "ON", "": ""} {
		got, ok := normalizeState(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	_, ok := normalizeState("Narnia")
	assert.False(t, ok)

	zip, _ := normalizeZip("2134")
	assert.Equal(t, "02134", zip, "Excel drops leading zeros")

	p, ok := normalizePhone("512-555-0100 ext. 12, 512-555-0101")
	assert.True(t, ok)
	assert.Equal(t, "512-555-0100 x 12", p)
	_, ok = normalizePhone("call me")
	assert.False(t, ok)

	for in, want := range map[string]string{
		"(512) 555-0142": "512-555-0142", "+1 512.555.0142": "512-555-0142", "5125550142": "512-555-0142",
	} {
		got, _, ok := usPhone(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	got, ext, ok := usPhone("512-555-0100 ext. 12")
	assert.True(t, ok)
	assert.Equal(t, "512-555-0100", got)
	assert.Equal(t, "12", ext)
	_, _, ok = usPhone("+44 20 7946 0958")
	assert.False(t, ok, "not a US number")

	z, ok := usZip("787011234")
	assert.True(t, ok)
	assert.Equal(t, "78701-1234", z)
	_, ok = usZip("M5V 3L9")
	assert.False(t, ok)
	assert.True(t, IsUSState("TX"))
	assert.True(t, IsUSState("PR"))
	assert.False(t, IsUSState("ON"))
}
