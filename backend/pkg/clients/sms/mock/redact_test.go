package mock

import "testing"

// T256: the mock must not write phone numbers to logs.
func TestRedactPhone(t *testing.T) {
	cases := map[string]string{"+15555550123": "***23", "12": "**", "": "**"}
	for in, want := range cases {
		if got := redactPhone(in); got != want {
			t.Errorf("redactPhone(%q) = %q, want %q", in, got, want)
		}
	}
}
