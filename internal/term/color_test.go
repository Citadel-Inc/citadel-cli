package term

import (
	"testing"
)

func TestParseColorMode(t *testing.T) {
	tests := []struct {
		in   string
		want ColorMode
	}{
		{"", ColorAuto},
		{"auto", ColorAuto},
		{"always", ColorAlways},
		{"force", ColorAlways},
		{"yes", ColorAlways},
		{"never", ColorNever},
		{"off", ColorNever},
		{"no", ColorNever},
		{"garbage", ColorAuto},
	}
	for _, tc := range tests {
		if got := ParseColorMode(tc.in); got != tc.want {
			t.Fatalf("ParseColorMode(%q) = %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestColorEnabled_AlwaysIgnoresNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !ColorEnabled(ColorAlways) {
		t.Fatal("ColorAlways must win over NO_COLOR")
	}
}

func TestColorEnabled_Never(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if ColorEnabled(ColorNever) {
		t.Fatal("ColorNever must disable")
	}
}

func TestColorEnabled_AutoRespectsNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled(ColorAuto) {
		t.Fatal("auto + NO_COLOR must disable")
	}
}

func TestColorEnabled_AutoWithoutNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	// Whether color is on depends on stdout TTY; only assert no panic.
	_ = ColorEnabled(ColorAuto)
}
