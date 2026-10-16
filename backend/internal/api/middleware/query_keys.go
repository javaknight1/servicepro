package middleware

import (
	"net/url"
	"sort"
	"strings"
)

// Request logs never carry query-string values (T239): password-reset and
// verification tokens, invite codes and emails can arrive as query params and
// would otherwise sit in Cloud Logging. Log sites record the path and, at
// most, the sorted parameter names from QueryKeys.

// maxLoggedQueryKeys caps how many names one log line carries.
const maxLoggedQueryKeys = 20

// QueryKeys returns the distinct parameter names of rawQuery, sorted and
// comma-joined ("" when there are none). Values are never included.
func QueryKeys(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	seen := map[string]struct{}{}
	for _, part := range strings.FieldsFunc(rawQuery, func(r rune) bool { return r == '&' || r == ';' }) {
		key, _, _ := strings.Cut(part, "=")
		if k, err := url.QueryUnescape(key); err == nil {
			key = k
		}
		key = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f || r == ',' {
				return -1
			}
			return r
		}, key)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > maxLoggedQueryKeys {
		keys = append(keys[:maxLoggedQueryKeys], "…")
	}
	return strings.Join(keys, ",")
}

// splitQuery splits a request target such as gin's LogFormatterParams.Path
// ("/path?raw") into the path and its raw query.
func splitQuery(target string) (path, rawQuery string) {
	path, rawQuery, _ = strings.Cut(target, "?")
	return path, rawQuery
}
