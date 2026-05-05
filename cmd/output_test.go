package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestColorEnabledNever asserts the resolved --color=never decision wins
// regardless of TTY / NO_COLOR.
func TestColorEnabledNever(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	c := &cobra.Command{}
	c.Flags().String("color", "never", "")
	if colorEnabled(c) {
		t.Fatal("--color=never must disable color")
	}
}

// TestColorEnabledAlways asserts --color=always overrides NO_COLOR per the
// no-color.org spec (NO_COLOR is a default suppressor, not a hard kill).
func TestColorEnabledAlways(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	c := &cobra.Command{}
	c.Flags().String("color", "always", "")
	if !colorEnabled(c) {
		t.Fatal("--color=always must enable color even with NO_COLOR set")
	}
}

// TestColorEnabledAutoNoColor: auto + NO_COLOR=1 → off.
func TestColorEnabledAutoNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	c := &cobra.Command{}
	c.Flags().String("color", "auto", "")
	if colorEnabled(c) {
		t.Fatal("auto + NO_COLOR must disable color")
	}
}
