package imports

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// FieldError is one problem with one row.
type FieldError struct {
	Field   string `json:"field,omitempty" example:"email"`
	Message string `json:"message" example:"email is required"`
}

func (e FieldError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// SkipKey is set in Record.Values when a valid row is deliberately not
// imported (e.g. an invoice already paid in full); its value is the reason.
const SkipKey = "_skip"

// Column limits of the tables the records are written to.
const (
	maxName    = 50
	maxCompany = 100
	maxStreet  = 100
	maxCity    = 50
	maxEmail   = 255
	maxTitle   = 200
	maxSite    = 200
	maxContact = 100
	maxNumber  = 50
	maxPO      = 100
	maxNotes   = 10000
)

// Job statuses Validate produces (models.JobStatus values).
const (
	JobNew       = "new"
	JobScheduled = "scheduled"
	JobProgress  = "in_progress"
	JobCompleted = "completed"
	JobCancelled = "cancelled"
)

// Validate normalises a record's values in place (names split, first email
// and phone picked, states as 2-letter codes, dates as DateLayout /
// DateTimeLayout, amounts as decimal strings) and returns its problems. A
// record with problems is not imported. now decides whether an undated
// job status is past or upcoming.
func Validate(src Source, e Entity, r *Record, now time.Time) []FieldError {
	v := &validator{r: r, src: src}
	switch e {
	case EntityCustomers:
		v.customer()
	case EntitySites:
		v.site()
	case EntityJobs:
		v.job(now)
	case EntityInvoices:
		v.invoice()
	}
	return v.errs
}

type validator struct {
	r    *Record
	src  Source
	errs []FieldError
}

func (v *validator) get(k string) string { return strings.TrimSpace(v.r.Values[k]) }

func (v *validator) set(k, val string) {
	if val == "" {
		delete(v.r.Values, k)
		return
	}
	v.r.Values[k] = val
}

func (v *validator) fail(field, format string, args ...any) {
	v.errs = append(v.errs, FieldError{Field: field, Message: fmt.Sprintf(format, args...)})
}

// maxLen fails a field longer than n characters.
func (v *validator) maxLen(k, label string, n int) {
	if len([]rune(v.get(k))) > n {
		v.fail(k, "%s is longer than %d characters", label, n)
	}
}

// clip shortens a free-text field to n characters (names, titles: cutting
// is better than rejecting the row).
func (v *validator) clip(k string, n int) {
	if s := v.get(k); len([]rune(s)) > n {
		v.set(k, strings.TrimSpace(string([]rune(s)[:n])))
	}
}

// joinStreet folds "street line 2" into the street ("12 Main St, Suite 4").
func (v *validator) joinStreet(street, street2 string) {
	s, s2 := v.get(street), v.get(street2)
	delete(v.r.Values, street2)
	if s2 != "" && s != "" {
		v.set(street, s+", "+s2)
	} else if s == "" {
		v.set(street, s2)
	}
}

// address normalises one address (street, city, state, zip keys) and fails
// unreadable parts. label prefixes the messages ("billing", "site").
func (v *validator) address(street, city, state, zip, label string) {
	v.maxLen(street, label+" street", maxStreet)
	v.maxLen(city, label+" city", maxCity)
	if s := v.get(state); s != "" {
		code, ok := normalizeState(s)
		if !ok {
			v.fail(state, "%s state %q is not a US state or Canadian province (use the 2-letter code)", label, s)
		}
		v.set(state, code)
	}
	if z := v.get(zip); z != "" {
		code, ok := normalizeZip(z)
		if !ok {
			v.fail(zip, "%s ZIP %q is longer than 10 characters", label, z)
		}
		v.set(zip, code)
	}
}

func (v *validator) email(k, label string, required bool) {
	raw := v.get(k)
	if raw == "" {
		if required {
			v.fail(k, "%s is required", label)
		}
		return
	}
	addr, ok := normalizeEmail(raw)
	if !ok {
		v.fail(k, "%s %q is not a valid email address", label, addr)
		return
	}
	v.set(k, addr)
	v.maxLen(k, label, maxEmail)
}

func (v *validator) phone(k, label string) {
	raw := v.get(k)
	if raw == "" {
		return
	}
	p, ok := normalizePhone(raw)
	if !ok {
		v.fail(k, "%s %q is not a phone number we can store (7–20 characters)", label, raw)
		return
	}
	v.set(k, p)
}

func (v *validator) customerRef() {
	if v.get(FCustomerExternalID) == "" && v.get(FCustomerName) == "" && v.get(FCustomerEmail) == "" {
		v.fail("", "the row names no customer (customer ID, email or name)")
	}
	if raw := v.get(FCustomerEmail); raw != "" {
		if addr, ok := normalizeEmail(raw); ok {
			v.set(FCustomerEmail, addr)
		}
	}
}

func (v *validator) customer() {
	// Names: first/last, else split the full name, else the company with a
	// placeholder contact the user can rename.
	first, last := v.get(FFirstName), v.get(FLastName)
	company := v.get(FCompanyName)
	// Jobber's display name is the company name for company clients.
	if full := v.get(FFullName); first == "" && last == "" && !strings.EqualFold(full, company) {
		first, last = splitName(full)
	}
	switch {
	case first == "" && last == "" && company != "":
		first, last = "Main", "Contact"
	case first == "" && last == "":
		v.fail(FFirstName, "a name is required (first and last name, or a company)")
	case first == "" || last == "":
		v.fail(FLastName, "both a first and a last name are required (or only a company name)")
	}
	delete(v.r.Values, FFullName)
	v.set(FFirstName, first)
	v.set(FLastName, last)
	v.clip(FFirstName, maxName)
	v.clip(FLastName, maxName)
	v.clip(FCompanyName, maxCompany)

	v.email(FEmail, "email", true)
	if e := v.get(FEmail); e != "" && !dbEmailRe.MatchString(e) {
		v.fail(FEmail, "email %q is not a valid email address", e)
	}

	// Phones: US numbers only (the customers table checks them). An
	// extension, or a second number we can't store, goes to the notes.
	var extra []string
	raw1, raw2 := v.get(FPhone), v.get(FPhoneSecondary)
	if raw1 == "" {
		raw1, raw2 = raw2, ""
	}
	phone, ext, ok := usPhone(raw1)
	switch {
	case raw1 == "":
		v.fail(FPhone, "a phone number is required")
	case !ok:
		v.fail(FPhone, "phone %q is not a 10-digit US phone number", raw1)
	}
	second, ext2, ok2 := usPhone(raw2)
	if raw2 != "" && !ok2 {
		extra = append(extra, "Other phone: "+raw2)
		second, ext2 = "", ""
	}
	v.set(FPhone, phone)
	v.set(FPhoneSecondary, second)
	if ext != "" {
		extra = append(extra, "Phone extension: "+ext)
	}
	if ext2 != "" {
		extra = append(extra, "Second phone extension: "+ext2)
	}

	v.joinStreet(FBillingStreet, FBillingStreet2)
	v.joinStreet(FStreet, FStreet2)
	// No billing address: the service address is the billing address.
	if v.get(FBillingStreet) == "" && v.get(FStreet) != "" {
		for b, s := range map[string]string{FBillingStreet: FStreet, FBillingCity: FCity, FBillingState: FState, FBillingZip: FZip} {
			v.set(b, v.get(s))
		}
	}
	v.address(FBillingStreet, FBillingCity, FBillingState, FBillingZip, "billing")
	if v.get(FBillingStreet) == "" || v.get(FBillingCity) == "" || v.get(FBillingState) == "" || v.get(FBillingZip) == "" {
		v.fail(FBillingStreet, "a full billing (or service) address is required: street, city, state and ZIP")
	} else {
		if st := v.get(FBillingState); !IsUSState(st) {
			v.fail(FBillingState, "billing state %q must be a US state (customer addresses are US-only)", st)
		}
		if z, ok := usZip(v.get(FBillingZip)); ok {
			v.set(FBillingZip, z)
		} else {
			v.fail(FBillingZip, "billing ZIP %q must be a 5-digit US ZIP code", v.get(FBillingZip))
		}
	}
	if v.get(FStreet) != "" {
		v.address(FStreet, FCity, FState, FZip, "service")
		if z, ok := usZip(v.get(FZip)); ok {
			v.set(FZip, z)
		}
	}
	v.clip(FSiteName, maxSite)

	switch t := strings.ToLower(v.get(FCustomerType)); {
	case t == "":
		if company != "" {
			v.set(FCustomerType, "commercial")
		} else {
			v.set(FCustomerType, "residential")
		}
	case strings.Contains(t, "commercial") || strings.Contains(t, "business") || strings.Contains(t, "company"):
		v.set(FCustomerType, "commercial")
	case strings.Contains(t, "residential") || strings.Contains(t, "home") || strings.Contains(t, "individual") || strings.Contains(t, "person"):
		v.set(FCustomerType, "residential")
	default:
		v.fail(FCustomerType, "customer type %q must be residential or commercial", v.get(FCustomerType))
	}

	// Status: unknown values are kept as active rather than failing the row.
	switch s := strings.ToLower(v.get(FStatus)); {
	case strings.Contains(s, "lead") || strings.Contains(s, "prospect") || strings.Contains(s, "potential"):
		v.set(FStatus, "prospect")
	case strings.Contains(s, "inactive") || strings.Contains(s, "archiv") || strings.Contains(s, "disabled") || strings.Contains(s, "former"):
		v.set(FStatus, "inactive")
	default:
		v.set(FStatus, "active")
	}

	notes := v.get(FNotes)
	if tags := v.get(FTags); tags != "" {
		if notes != "" {
			notes += "\n"
		}
		notes += "Tags: " + tags
	}
	for _, line := range extra {
		if notes != "" {
			notes += "\n"
		}
		notes += line
	}
	delete(v.r.Values, FTags)
	v.set(FNotes, notes)
	v.clip(FNotes, maxNotes)
	v.maxLen(FExternalID, "customer ID", maxEmail)
}

func (v *validator) site() {
	v.customerRef()
	v.joinStreet(FStreet, FStreet2)
	if v.get(FStreet) == "" {
		v.fail(FStreet, "street is required")
	}
	v.address(FStreet, FCity, FState, FZip, "site")
	if v.get(FName) == "" {
		v.set(FName, v.get(FStreet))
	}
	v.clip(FName, maxSite)
	v.clip(FContactName, maxContact)
	v.phone(FContactPhone, "site contact phone")
	v.email(FContactEmail, "site contact email", false)
	v.clip(FAccessNotes, maxNotes)
}

// date normalises a date field; it reports unreadable values.
func (v *validator) date(k, label string) (time.Time, bool, bool) {
	raw := v.get(k)
	if raw == "" {
		return time.Time{}, false, false
	}
	t, hasTime, err := parseDateTime(raw)
	if err != nil {
		v.fail(k, "%s %q %s", label, raw, err.Error())
		return time.Time{}, false, false
	}
	v.set(k, formatDate(t, hasTime))
	return t, hasTime, true
}

func (v *validator) job(now time.Time) {
	v.customerRef()
	if n := strings.TrimPrefix(v.get(FExternalID), "#"); n != "" {
		v.set(FExternalID, strings.TrimSpace(n))
		v.maxLen(FExternalID, "job number", maxNumber)
	}
	desc := v.get(FDescription)
	if v.get(FTitle) == "" {
		title := firstLine(desc)
		if title == "" {
			title = "Imported job"
			if n := v.get(FExternalID); n != "" {
				title += " #" + n
			}
		}
		v.set(FTitle, title)
	}
	v.clip(FTitle, maxTitle)
	v.clip(FDescription, maxNotes)
	v.clip(FNotes, maxNotes)

	switch t := strings.ToLower(v.get(FJobType)); {
	case strings.Contains(t, "install"):
		v.set(FJobType, "installation")
	case strings.Contains(t, "repair") || strings.Contains(t, "fix"):
		v.set(FJobType, "repair")
	case strings.Contains(t, "inspect"):
		v.set(FJobType, "inspection")
	case strings.Contains(t, "emergency") || strings.Contains(t, "urgent"):
		v.set(FJobType, "emergency")
	default:
		// Cleaning and facility maintenance: the default for everything else
		// (including Jobber's "one-off"/"recurring" job kinds).
		v.set(FJobType, "maintenance")
	}

	start, startHasTime, hasStart := v.date(FScheduledStart, "scheduled start")
	// End: a full date, or a time alone on the start's day (Housecall Pro's
	// "End Time").
	if raw := v.get(FScheduledEnd); raw != "" {
		if tod, ok := parseTimeOfDay(raw); ok && hasStart {
			end := time.Date(start.Year(), start.Month(), start.Day(), tod.Hour(), tod.Minute(), 0, 0, time.UTC)
			if end.After(start) && startHasTime {
				v.set(FScheduledEnd, end.Format(DateTimeLayout))
			} else {
				delete(v.r.Values, FScheduledEnd)
			}
		} else if end, endHasTime, ok := v.date(FScheduledEnd, "scheduled end"); ok {
			if hasStart && !end.After(start) {
				delete(v.r.Values, FScheduledEnd)
			} else if !endHasTime {
				v.set(FScheduledEnd, end.Format(DateLayout))
			}
		}
	}
	_, _, hasCompleted := v.date(FCompletedAt, "completed date")
	v.clip(FPONumber, maxPO)
	v.joinStreet(FStreet, FStreet2)
	v.address(FStreet, FCity, FState, FZip, "service")

	today := now.Format(DateLayout)
	past := hasStart && start.Format(DateLayout) < today
	switch s := strings.ToLower(v.get(FStatus)); {
	case strings.Contains(s, "cancel") || strings.Contains(s, "void"):
		v.set(FStatus, JobCancelled)
	case hasCompleted || strings.Contains(s, "complete") || strings.Contains(s, "archiv") ||
		strings.Contains(s, "closed") || strings.Contains(s, "done") || strings.Contains(s, "finished") ||
		strings.Contains(s, "invoic") || strings.Contains(s, "paid"):
		v.set(FStatus, JobCompleted)
	case strings.Contains(s, "progress") || strings.Contains(s, "started"):
		v.set(FStatus, JobProgress)
	case s == "" && past:
		// An old job with no status was done.
		v.set(FStatus, JobCompleted)
	case hasStart:
		v.set(FStatus, JobScheduled)
	default:
		v.set(FStatus, JobNew)
	}
}

func (v *validator) invoice() {
	v.customerRef()
	num := strings.TrimSpace(strings.TrimPrefix(v.get(FInvoiceNumber), "#"))
	if num == "" {
		v.fail(FInvoiceNumber, "invoice number is required")
	}
	v.set(FInvoiceNumber, num)
	v.maxLen(FInvoiceNumber, "invoice number", maxNumber)

	amount := func(k, label string) (decimal.Decimal, bool) {
		raw := v.get(k)
		if raw == "" {
			return decimal.Zero, false
		}
		d, err := parseMoney(raw)
		if err != nil {
			v.fail(k, "%s %q %s", label, raw, err.Error())
			return decimal.Zero, false
		}
		v.set(k, d.StringFixed(2))
		return d, true
	}
	total, hasTotal := amount(FTotal, "total")
	paid, hasPaid := amount(FAmountPaid, "amount paid")
	balance, hasBalance := amount(FBalance, "balance due")
	if !hasBalance {
		switch {
		case hasTotal && hasPaid:
			balance, hasBalance = total.Sub(paid), true
		case hasTotal:
			balance, hasBalance = total, true
		}
		if hasBalance {
			v.set(FBalance, balance.StringFixed(2))
		}
	}
	if !hasBalance && len(v.errs) == 0 {
		v.fail(FBalance, "the row has no balance due or total")
	}
	if hasTotal && total.LessThan(balance) {
		v.set(FTotal, balance.StringFixed(2))
	} else if !hasTotal && hasBalance {
		v.set(FTotal, balance.StringFixed(2))
	}

	issue, _, hasIssue := v.date(FIssueDate, "issue date")
	due, _, hasDue := v.date(FDueDate, "due date")
	if hasIssue {
		v.set(FIssueDate, issue.Format(DateLayout))
	}
	if hasDue {
		v.set(FDueDate, due.Format(DateLayout))
	} else if hasIssue {
		v.set(FDueDate, issue.Format(DateLayout))
	}
	if hasIssue && hasDue && due.Before(issue) {
		v.set(FDueDate, issue.Format(DateLayout))
	}
	v.clip(FSubject, maxTitle)
	v.clip(FPONumber, maxPO)
	v.clip(FNotes, maxNotes)

	if len(v.errs) > 0 {
		return
	}
	status := strings.ToLower(v.get(FStatus))
	switch {
	case strings.Contains(status, "void") || strings.Contains(status, "cancel") || strings.Contains(status, "bad debt") || strings.Contains(status, "draft"):
		v.set(SkipKey, fmt.Sprintf("status is %s", v.get(FStatus)))
	case !balance.IsPositive():
		v.set(SkipKey, "nothing left to pay")
	case v.src == SourceHousecallPro && !hasIssue:
		// Housecall Pro's jobs report lists every job with its invoice
		// number; an invoice that was never sent is not an open invoice.
		v.set(SkipKey, "invoice not sent yet")
	}
}

// firstLine is the first non-empty line of s.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}
