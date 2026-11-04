package imports

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// Caps for one import file (security audit F-23). The handler also wraps the
// request body in http.MaxBytesReader with MaxFileBytes, so an oversized
// upload is cut off before it is buffered.
//
//   - 10 MiB: Housecall Pro's jobs report has 47 columns (~400 bytes a row),
//     so 10 MiB holds ~25,000 jobs, several years of history for a 1–10
//     tech company; Jobber splits its client export into files of 1,500 rows.
//     Cloud Run's own request cap is 32 MiB.
//   - 50,000 rows: an upper bound on what one tenant imports at once; the
//     parsed records are stored as one JSONB payload (~20 MB at the cap) and
//     imported in time-boxed chunks, so a big file never holds a request.
//   - XLSX: the zip is limited to 64 MiB unpacked (16 MiB per XML part),
//     against zip bombs.
const (
	MaxFileBytes   int64 = 10 << 20
	MaxRows              = 50000
	MaxColumns           = 200
	maxUnzipBytes  int64 = 64 << 20
	maxUnzipXML    int64 = 16 << 20
	maxHeaderBytes       = 200
)

// Errors returned by ReadTable. ErrTooManyRows and ErrTooLarge map to 413.
var (
	ErrTooLarge    = fmt.Errorf("file exceeds the maximum size of %d MiB", MaxFileBytes>>20)
	ErrTooManyRows = fmt.Errorf("file exceeds the maximum of %d rows", MaxRows)
	ErrNoHeader    = errors.New("file is empty: the first row must hold the column headers")
	ErrNoRows      = errors.New("file contains no data rows")
	ErrUnreadable  = errors.New("file could not be read: upload a CSV or an Excel (.xlsx) file")
	ErrTooManyCols = fmt.Errorf("file has more than %d columns", MaxColumns)
)

// Table is a parsed file: unique, trimmed headers and the data rows (every
// row padded or cut to len(Headers)). Row i is spreadsheet row i+2.
type Table struct {
	Headers []string
	Rows    [][]string
}

// ReadTable parses a CSV (comma, semicolon or tab separated; UTF-8, BOM
// tolerated) or XLSX (first sheet) file, enforcing the caps above. Fully
// blank rows are dropped.
func ReadTable(data []byte, filename string) (*Table, error) {
	if int64(len(data)) > MaxFileBytes {
		return nil, ErrTooLarge
	}
	var raw [][]string
	var err error
	if isXLSX(data, filename) {
		raw, err = readXLSX(data)
	} else {
		raw, err = readCSV(data)
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrNoHeader
	}
	header := raw[0]
	if len(header) > MaxColumns {
		return nil, ErrTooManyCols
	}
	t := &Table{Headers: uniqueHeaders(header)}
	if len(t.Headers) == 0 {
		return nil, ErrNoHeader
	}
	for _, rec := range raw[1:] {
		if blank(rec) {
			continue
		}
		row := make([]string, len(t.Headers))
		for i := range row {
			if i < len(rec) {
				row[i] = cleanCell(rec[i])
			}
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		return nil, ErrNoRows
	}
	if len(t.Rows) > MaxRows {
		return nil, ErrTooManyRows
	}
	return t, nil
}

func isXLSX(data []byte, filename string) bool {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return true
	}
	return bytes.HasPrefix(data, []byte("PK\x03\x04"))
}

func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		// Excel's "CSV" on Windows is often Windows-1252; take it as Latin-1
		// rather than rejecting the file (the mapping preview shows the result).
		data = latin1ToUTF8(data)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = sniffDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.ReuseRecord = false
	var out [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrUnreadable
		}
		if out, err = appendCapped(out, rec); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// sniffDelimiter picks the separator of the header line: comma unless the
// line has more semicolons or tabs.
func sniffDelimiter(data []byte) rune {
	line := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		line = data[:i]
	}
	best, bestN := ',', bytes.Count(line, []byte(","))
	for _, d := range []rune{';', '\t'} {
		if n := bytes.Count(line, []byte(string(d))); n > bestN {
			best, bestN = d, n
		}
	}
	return best
}

func readXLSX(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    maxUnzipBytes,
		UnzipXMLSizeLimit: maxUnzipXML,
	})
	if err != nil {
		return nil, ErrUnreadable
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrNoHeader
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, ErrUnreadable
	}
	defer func() { _ = rows.Close() }()
	var out [][]string
	for rows.Next() {
		rec, err := rows.Columns()
		if err != nil {
			return nil, ErrUnreadable
		}
		if out, err = appendCapped(out, rec); err != nil {
			return nil, err
		}
	}
	if rows.Error() != nil {
		return nil, ErrUnreadable
	}
	return out, nil
}

// uniqueHeaders trims headers, names blank ones "Column N" and suffixes
// repeats ("City", "City (2)"): Housecall Pro repeats its address block once
// per address. Trailing blank header cells are dropped.
func uniqueHeaders(in []string) []string {
	end := len(in)
	for end > 0 && strings.TrimSpace(in[end-1]) == "" {
		end--
	}
	out := make([]string, 0, end)
	seen := map[string]int{}
	for i, h := range in[:end] {
		h = cleanCell(h)
		if len(h) > maxHeaderBytes {
			h = truncate(h, maxHeaderBytes)
		}
		if h == "" {
			h = fmt.Sprintf("Column %d", i+1)
		}
		key := strings.ToLower(h)
		seen[key]++
		if n := seen[key]; n > 1 {
			h = fmt.Sprintf("%s (%d)", h, n)
		}
		out = append(out, h)
	}
	return out
}

// cleanCell trims spaces and drops control characters other than newlines.
func cleanCell(s string) string {
	s = strings.TrimSpace(s)
	if strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, s)
}

// appendCapped appends a non-blank record (the header always), failing once
// the data rows pass MaxRows, so an oversized file stops being read early.
func appendCapped(out [][]string, rec []string) ([][]string, error) {
	if len(out) > 0 && blank(rec) {
		return out, nil
	}
	if len(out) > MaxRows { // header + MaxRows data rows already
		return nil, ErrTooManyRows
	}
	return append(out, rec), nil
}

func blank(rec []string) bool {
	for _, v := range rec {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func latin1ToUTF8(b []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(b) + len(b)/8)
	for _, c := range b {
		buf.WriteRune(rune(c))
	}
	return buf.Bytes()
}

// normalizeHeader is the header key aliases are compared on: lower case,
// letters and digits only.
func normalizeHeader(h string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(h) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// truncate cuts s to at most n bytes on a rune boundary.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
