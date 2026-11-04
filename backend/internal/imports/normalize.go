package imports

import (
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// DateLayout and DateTimeLayout are the normalised forms Validate stores
// date fields in (tenant-local wall time, no zone: the runner applies the
// tenant's time zone).
const (
	DateLayout     = "2006-01-02"
	DateTimeLayout = "2006-01-02T15:04"
)

// Date layouts accepted in files. Slash dates are read US-style
// (month/day/year), as both Jobber and Housecall Pro export them for US
// accounts.
var dateTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02 3:04 PM",
	"2006-01-02 3:04PM",
	"1/2/2006 15:04:05",
	"1/2/2006 15:04",
	"1/2/2006 3:04:05 PM",
	"1/2/2006 3:04 PM",
	"1/2/2006 3:04PM",
	"1/2/06 15:04",
	"1/2/06 3:04 PM",
	"Jan 2, 2006 3:04 PM",
	"Jan 2, 2006 15:04",
	"Jan 2, 2006, 3:04 PM",
	"January 2, 2006 3:04 PM",
	"Mon, Jan 2, 2006 3:04 PM",
}

var dateLayouts = []string{
	"2006-01-02",
	"2006/01/02",
	"1/2/2006",
	"1/2/06",
	"1-2-2006",
	"Jan 2, 2006",
	"Jan 2 2006",
	"January 2, 2006",
	"January 2 2006",
	"Mon, Jan 2, 2006",
	"Monday, January 2, 2006",
	"2 Jan 2006",
	"2-Jan-2006",
	"2-Jan-06",
}

var timeLayouts = []string{"15:04", "15:04:05", "3:04 PM", "3:04PM", "3:04:05 PM", "3PM", "3 PM"}

var errBadDate = errors.New("is not a date we can read (use YYYY-MM-DD or MM/DD/YYYY)")

// parseDateTime reads a date or date-time cell. hasTime is false for a date
// alone. Excel serial day numbers (unformatted date cells) are accepted.
func parseDateTime(s string) (t time.Time, hasTime bool, err error) {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Replace(s, " at ", " ", 1)
	upper := strings.ToUpper(s)
	if strings.HasSuffix(upper, "AM") || strings.HasSuffix(upper, "PM") {
		s = s[:len(s)-2] + upper[len(upper)-2:]
	}
	for _, l := range dateTimeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true, nil
		}
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, false, nil
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 && f < 80000 {
		// Excel serial date: days since 1899-12-30.
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		days := int(f)
		t := base.AddDate(0, 0, days).Add(time.Duration((f - float64(days)) * 24 * float64(time.Hour)).Round(time.Minute))
		return t, f != float64(days), nil
	}
	return time.Time{}, false, errBadDate
}

// parseTimeOfDay reads a time alone ("2:30 PM", "14:30").
func parseTimeOfDay(s string) (time.Time, bool) {
	s = strings.ToUpper(strings.Join(strings.Fields(s), " "))
	for _, l := range timeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// formatDate stores a parsed date in the normalised form.
func formatDate(t time.Time, hasTime bool) string {
	if hasTime {
		return t.Format(DateTimeLayout)
	}
	return t.Format(DateLayout)
}

// ParseLocal reads a normalised date or date-time in loc. Date-only values
// return hasTime false.
func ParseLocal(v string, loc *time.Location) (t time.Time, hasTime bool, err error) {
	if t, err := time.ParseInLocation(DateTimeLayout, v, loc); err == nil {
		return t, true, nil
	}
	t, err = time.ParseInLocation(DateLayout, v, loc)
	return t, false, err
}

var moneyStrip = strings.NewReplacer("$", "", "€", "", "£", "", ",", "", " ", "", "USD", "", "CAD", "", "usd", "", "cad", "")

// parseMoney reads an amount like "$1,234.50", "1234.5" or "(12.00)"
// (negative) into a decimal rounded to cents. Never a float.
func parseMoney(s string) (decimal.Decimal, error) {
	neg := false
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = s[1 : len(s)-1]
	}
	s = moneyStrip.Replace(s)
	if strings.HasSuffix(s, "-") {
		neg = !neg
		s = strings.TrimSuffix(s, "-")
	}
	if s == "" || s == "-" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, errors.New("is not an amount")
	}
	if neg {
		d = d.Neg()
	}
	return d.Round(2), nil
}

var listSplit = regexp.MustCompile(`\s*(?:[,;/|\n]|\s+or\s+)\s*`)

// firstOf returns the first non-empty entry of a cell that may list several
// values ("a@x.com, b@x.com", "512-555-0100 / 512-555-0101").
func firstOf(s string) string {
	for _, p := range listSplit.Split(s, -1) {
		if p = strings.TrimSpace(p); p != "" {
			return p
		}
	}
	return ""
}

// normalizeEmail picks the first address in the cell and lower-cases it.
func normalizeEmail(s string) (string, bool) {
	for _, p := range listSplit.Split(s, -1) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		addr, err := mail.ParseAddress(p)
		if err != nil || !strings.Contains(addr.Address, ".") {
			return p, false
		}
		return strings.ToLower(addr.Address), true
	}
	return "", true
}

var phoneJunk = regexp.MustCompile(`[^0-9+()\-. x]`)

// normalizePhone picks the first number in the cell; it must fit the 20
// characters ServicePro stores and hold 7–15 digits.
func normalizePhone(s string) (string, bool) {
	p := firstOf(s)
	if p == "" {
		return "", true
	}
	p = strings.ToLower(p)
	p = strings.Replace(p, "ext.", "x", 1)
	p = strings.Replace(p, "ext", "x", 1)
	p = strings.Join(strings.Fields(phoneJunk.ReplaceAllString(p, "")), " ")
	digits := 0
	for _, r := range p {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits < 7 || len(p) > 20 {
		return p, false
	}
	return p, true
}

var states = map[string]string{
	"alabama": "AL", "alaska": "AK", "arizona": "AZ", "arkansas": "AR", "california": "CA",
	"colorado": "CO", "connecticut": "CT", "delaware": "DE", "district of columbia": "DC",
	"washington dc": "DC", "washington d c": "DC", "florida": "FL", "georgia": "GA", "hawaii": "HI",
	"idaho": "ID", "illinois": "IL", "indiana": "IN", "iowa": "IA", "kansas": "KS",
	"kentucky": "KY", "louisiana": "LA", "maine": "ME", "maryland": "MD", "massachusetts": "MA",
	"michigan": "MI", "minnesota": "MN", "mississippi": "MS", "missouri": "MO", "montana": "MT",
	"nebraska": "NE", "nevada": "NV", "new hampshire": "NH", "new jersey": "NJ", "new mexico": "NM",
	"new york": "NY", "north carolina": "NC", "north dakota": "ND", "ohio": "OH", "oklahoma": "OK",
	"oregon": "OR", "pennsylvania": "PA", "rhode island": "RI", "south carolina": "SC",
	"south dakota": "SD", "tennessee": "TN", "texas": "TX", "utah": "UT", "vermont": "VT",
	"virginia": "VA", "washington": "WA", "west virginia": "WV", "wisconsin": "WI", "wyoming": "WY",
	"puerto rico": "PR", "guam": "GU", "us virgin islands": "VI", "american samoa": "AS",
	"northern mariana islands": "MP",
	// Canadian provinces and territories (Jobber is used in Canada too).
	"alberta": "AB", "british columbia": "BC", "manitoba": "MB", "new brunswick": "NB",
	"newfoundland and labrador": "NL", "newfoundland": "NL", "nova scotia": "NS", "ontario": "ON",
	"prince edward island": "PE", "quebec": "QC", "québec": "QC", "saskatchewan": "SK",
	"northwest territories": "NT", "nunavut": "NU", "yukon": "YT",
}

var stateCodes = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range states {
		m[c] = true
	}
	return m
}()

// normalizeState turns a state or province name or code into its 2-letter
// code.
func normalizeState(s string) (string, bool) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "."))
	if s == "" {
		return "", true
	}
	up := strings.ToUpper(strings.ReplaceAll(s, ".", ""))
	if len(up) == 2 && stateCodes[up] {
		return up, true
	}
	key := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(s, ".", " ")), " "))
	if c, ok := states[key]; ok {
		return c, true
	}
	return s, false
}

// normalizeZip upper-cases and trims a ZIP or postal code; ServicePro stores
// up to 10 characters.
func normalizeZip(s string) (string, bool) {
	s = strings.ToUpper(strings.Join(strings.Fields(s), " "))
	// Excel drops leading zeros: a 4-digit ZIP is a New England one.
	if len(s) == 4 && strings.Trim(s, "0123456789") == "" {
		s = "0" + s
	}
	return s, len(s) <= 10
}

// Customer rows are checked by the database (migration 000001): US phone
// numbers, US state codes and 5 or 5+4 digit ZIP codes, and an email
// address with a dotted domain. These mirror those checks so a row fails in
// the preview rather than at import.
var (
	usZipRe   = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	dbEmailRe = regexp.MustCompile(`(?i)^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
	extRe     = regexp.MustCompile(`(?i)\s*(?:x|ext\.?|extension|#)\s*(\d{1,6})\s*$`)
)

// usTerritories are the codes the customers table accepts besides the 50
// states.
var usTerritories = map[string]bool{"DC": true, "AS": true, "GU": true, "MP": true, "PR": true, "VI": true}

var canadian = map[string]bool{"AB": true, "BC": true, "MB": true, "NB": true, "NL": true, "NS": true, "ON": true, "PE": true, "QC": true, "SK": true, "NT": true, "NU": true, "YT": true}

// IsUSState reports whether a normalised state code is one a customer
// address may use.
func IsUSState(code string) bool {
	return stateCodes[code] && !canadian[code] || usTerritories[code]
}

// IsUSZip reports whether a ZIP is a 5 or 5+4 digit US ZIP code.
func IsUSZip(zip string) bool { return usZipRe.MatchString(zip) }

// usZip normalises a US ZIP ("787011234" → "78701-1234", "2134" → "02134").
func usZip(s string) (string, bool) {
	z, _ := normalizeZip(s)
	if len(z) == 9 && strings.Trim(z, "0123456789") == "" {
		z = z[:5] + "-" + z[5:]
	}
	return z, IsUSZip(z)
}

// usPhone normalises a US phone number to 512-555-0142 (the form the
// customers table accepts), returning any extension separately.
func usPhone(s string) (phone, ext string, ok bool) {
	p := firstOf(s)
	if m := extRe.FindStringSubmatch(p); m != nil {
		ext = m[1]
		p = p[:len(p)-len(m[0])]
	}
	var digits []byte
	for i := 0; i < len(p); i++ {
		if p[i] >= '0' && p[i] <= '9' {
			digits = append(digits, p[i])
		}
	}
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}
	if len(digits) != 10 {
		return p, "", false
	}
	d := string(digits)
	return d[:3] + "-" + d[3:6] + "-" + d[6:], ext, true
}

// splitName splits "Pat Q. Smith" into first "Pat Q." and last "Smith".
func splitName(full string) (first, last string) {
	parts := strings.Fields(full)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
	}
}
