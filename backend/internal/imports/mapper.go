package imports

import (
	"fmt"
	"sort"
	"strings"
)

// ColumnMap maps a target field to the file columns it is read from. Each
// row takes the first non-empty column; only automatic mapping uses more
// than one column (for phone numbers).
type ColumnMap map[string][]int

// multiColumnFields may take several columns when mapped automatically.
var multiColumnFields = map[string]bool{FPhone: true}

// AutoMap maps headers to fields by aliases. Each column is used by at most
// one field. The source's own aliases are matched first for every field,
// then the generic ones: so Housecall Pro's "City" is the service city (its
// customer list has no billing columns), while a generic file's "City" is
// the billing city.
func AutoMap(headers []string, src Source, e Entity) ColumnMap {
	norm := make([]string, len(headers))
	for i, h := range headers {
		norm[i] = normalizeHeader(h)
	}
	used := make([]bool, len(headers))
	cm := ColumnMap{}
	match := func(key string, aliases []string) {
		for _, alias := range aliases {
			for i, n := range norm {
				if used[i] || n != alias {
					continue
				}
				cm[key] = append(cm[key], i)
				used[i] = true
			}
			if len(cm[key]) > 0 && !multiColumnFields[key] {
				return
			}
		}
	}
	for _, f := range fields[e] {
		match(f.Key, sourceAliasesFor(src, e, f))
	}
	for _, f := range fields[e] {
		if len(cm[f.Key]) == 0 {
			match(f.Key, genericAliasesFor(e, f))
		}
	}
	return cm
}

// ApplyMapping builds a ColumnMap from an explicit field → header mapping
// (the generic CSV flow, or a user's corrections). Empty headers leave the
// field unmapped. Unknown fields or headers, and a header used twice, are
// errors.
func ApplyMapping(headers []string, e Entity, m map[string]string) (ColumnMap, error) {
	index := map[string]int{}
	for i, h := range headers {
		index[h] = i
	}
	known := map[string]bool{}
	for _, f := range fields[e] {
		known[f.Key] = true
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	cm := ColumnMap{}
	usedBy := map[int]string{}
	for _, key := range keys {
		header := m[key]
		if !known[key] {
			return nil, fmt.Errorf("unknown field %q for %s", key, e)
		}
		if strings.TrimSpace(header) == "" {
			continue
		}
		i, ok := index[header]
		if !ok {
			return nil, fmt.Errorf("column %q is not in the file", header)
		}
		if other, dup := usedBy[i]; dup {
			return nil, fmt.Errorf("column %q is mapped to both %s and %s", header, other, key)
		}
		usedBy[i] = key
		cm[key] = []int{i}
	}
	return cm, nil
}

// HeaderMapping is the mapping as field → header (the first column of each
// field), as shown and edited in the UI.
func (cm ColumnMap) HeaderMapping(headers []string) map[string]string {
	out := make(map[string]string, len(cm))
	for k, cols := range cm {
		if len(cols) > 0 && cols[0] < len(headers) {
			out[k] = headers[cols[0]]
		}
	}
	return out
}

// Missing lists what the mapping lacks before an import can run: required
// fields and "at least one of" groups with no column.
func Missing(e Entity, cm ColumnMap) []string {
	var out []string
	groups := map[string]bool{}
	var groupOrder []string
	for _, f := range fields[e] {
		if f.Required && len(cm[f.Key]) == 0 {
			out = append(out, f.Label)
		}
		if f.Group != "" {
			if _, seen := groups[f.Group]; !seen {
				groupOrder = append(groupOrder, f.Group)
				groups[f.Group] = false
			}
			if len(cm[f.Key]) > 0 {
				groups[f.Group] = true
			}
		}
	}
	for _, g := range groupOrder {
		if !groups[g] {
			out = append(out, groupLabels[g])
		}
	}
	return out
}

// Record is one data row read through a ColumnMap: Values holds every mapped
// field (normalised by Validate). Row is the spreadsheet row number (the
// header is row 1).
type Record struct {
	Row    int               `json:"row"`
	Values map[string]string `json:"values"`
}

// Get returns a field's value ("" when absent).
func (r *Record) Get(key string) string { return r.Values[key] }

// Prepared is a file read through a mapping: the mapping used, what it
// lacks, and every row as a Record (not yet validated).
type Prepared struct {
	Columns ColumnMap
	Missing []string
	Records []Record
}

// Prepare maps a table: through mapping when one is given (field → header),
// else automatically by the source's aliases.
func Prepare(t *Table, src Source, e Entity, mapping map[string]string) (*Prepared, error) {
	var cm ColumnMap
	if len(mapping) > 0 {
		var err error
		if cm, err = ApplyMapping(t.Headers, e, mapping); err != nil {
			return nil, err
		}
	} else {
		cm = AutoMap(t.Headers, src, e)
	}
	return &Prepared{Columns: cm, Missing: Missing(e, cm), Records: Extract(t, cm)}, nil
}

// Extract reads every data row through the mapping.
func Extract(t *Table, cm ColumnMap) []Record {
	out := make([]Record, len(t.Rows))
	for i, row := range t.Rows {
		vals := make(map[string]string, len(cm))
		for key, cols := range cm {
			for _, c := range cols {
				if c < len(row) && row[c] != "" {
					vals[key] = row[c]
					break
				}
			}
		}
		out[i] = Record{Row: i + 2, Values: vals}
	}
	return out
}
