package term

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"plain name":                   "plain name",
		"line\n\tindent":               "line\n\tindent",
		"café 日本":                      "café 日本",
		"\x1b[31mred\x1b[0m":           "red",
		"\x1b]8;;https://evil\x07link": "link",
		"\x1b]0;title\x1b\\after":      "after",
		"\x1bcreset":                   "reset",
		"bell\x07 cr\r bs\b del\x7f":   "bell cr bs del",
		"c1\u009b31mcsi":               "c131mcsi",
		"\xc2\x1b[m\x9b":               "��",
		"trailing esc \x1b":            "trailing esc ",
		"unterminated \x1b]0;title":    "unterminated ",
		"unterminated csi \x1b[12;":    "unterminated csi ",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizerWriterReportsInputLength(t *testing.T) {
	var b strings.Builder
	n, err := NewSanitizer(&b).Write([]byte("a\x1b[2Jb\n"))
	if err != nil || n != 7 || b.String() != "ab\n" {
		t.Fatalf("n=%d err=%v out=%q", n, err, b.String())
	}
}

func FuzzSanitize(f *testing.F) {
	for _, s := range []string{"ok", "\x1b[31mx", "\x1b]0;t\x07", "\u009b1m", "\xc2\x1b[m\x9b", "a\r\nb"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := Sanitize(in)
		if !utf8.ValidString(out) {
			t.Fatalf("invalid UTF-8 from %q: %q", in, out)
		}
		for _, r := range out {
			if isControl(r) {
				t.Fatalf("control %U survived %q -> %q", r, in, out)
			}
		}
		if utf8.ValidString(in) && !strings.ContainsFunc(in, isControl) && out != in {
			t.Fatalf("clean input changed: %q -> %q", in, out)
		}
	})
}
