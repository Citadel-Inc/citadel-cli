package term

import (
	"io"
	"strings"
	"unicode/utf8"
)

// Sanitize makes server-supplied text safe to print to a terminal: it drops
// ESC sequences (CSI, OSC and other string sequences, two-byte escapes), every
// other C0/C1 control except newline and tab, and DEL. Invalid UTF-8 becomes
// U+FFFD so dropping a control can never splice bytes into a C1 rune.
func Sanitize(s string) string {
	if !needsSanitize(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i = skipEscape(s, i)
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteRune(utf8.RuneError)
		case !isControl(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func needsSanitize(s string) bool {
	for i := range len(s) {
		if c := s[i]; c >= 0x7f || (c < 0x20 && c != '\n' && c != '\t') {
			return true
		}
	}
	return false
}

func isControl(r rune) bool {
	return (r < 0x20 && r != '\n' && r != '\t') || (r >= 0x7f && r <= 0x9f)
}

// skipEscape returns the index just past the escape sequence starting at s[i].
func skipEscape(s string, i int) int {
	i++
	if i >= len(s) {
		return i
	}
	switch s[i] {
	case '[': // CSI: parameter and intermediate bytes up to a final byte.
		for i++; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return i
	case ']', 'P', 'X', '^', '_': // String sequences end at BEL or ESC \.
		for i++; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return i
	}
	if s[i] >= 0x20 && s[i] <= 0x7e {
		return i + 1
	}
	return i
}

type sanitizer struct{ w io.Writer }

// NewSanitizer wraps w so every write passes through Sanitize. It reports the
// input length as written, which tabwriter and fmt require.
func NewSanitizer(w io.Writer) io.Writer { return sanitizer{w} }

func (s sanitizer) Write(p []byte) (int, error) {
	if _, err := io.WriteString(s.w, Sanitize(string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}
