// Package term carries terminal-aware helpers shared by the cmd handlers.
// Today: color-enable resolution honoring NO_COLOR, CLICOLOR, and --color.
package term

import (
	"os"

	"golang.org/x/term"
)

// ColorMode is the resolved value of the --color flag.
type ColorMode int

const (
	// ColorAuto enables color when stdout is a TTY and NO_COLOR is unset.
	ColorAuto ColorMode = iota
	// ColorAlways forces color on regardless of environment.
	ColorAlways
	// ColorNever forces color off regardless of environment.
	ColorNever
)

// ParseColorMode parses the --color flag value. Empty string == auto.
// Unknown values fall back to auto so the CLI never errors on a stray flag.
func ParseColorMode(s string) ColorMode {
	switch s {
	case "always", "force", "yes":
		return ColorAlways
	case "never", "off", "no":
		return ColorNever
	default:
		return ColorAuto
	}
}

// ColorEnabled reports whether color output should be emitted on stdout under
// the given mode. Honors NO_COLOR (https://no-color.org): any non-empty value
// disables color in auto mode.
func ColorEnabled(mode ColorMode) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}
