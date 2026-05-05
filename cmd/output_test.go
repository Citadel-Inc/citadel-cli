package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
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

func TestEmitNDJSONLines_rows(t *testing.T) {
	var buf bytes.Buffer
	if err := emitNDJSONLinesTo(&buf, []map[string]any{{"a": 1.0}, {"b": "two"}}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %q", buf.String())
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatal(err)
	}
	if row["a"].(float64) != 1 {
		t.Fatalf("first row: %v", row)
	}
}

func TestEmitNDJSONLines_empty(t *testing.T) {
	var buf bytes.Buffer
	if err := emitNDJSONLinesTo(&buf, []string{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("empty slice should write nothing, got %q", buf.String())
	}
}

func TestVerboseFlag_QuietSuppressesVerbose(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().BoolP("quiet", "q", false, "")
	c.Flags().BoolP("verbose", "v", false, "")
	if err := c.Flags().Set("quiet", "true"); err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("verbose", "true"); err != nil {
		t.Fatal(err)
	}
	if verboseFlag(c) {
		t.Fatal("--quiet must force verboseFlag false even when --verbose is set")
	}
}

func TestVerboseFlag_trueWhenVerbose(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().BoolP("quiet", "q", false, "")
	c.Flags().BoolP("verbose", "v", false, "")
	if err := c.Flags().Set("verbose", "true"); err != nil {
		t.Fatal(err)
	}
	if !verboseFlag(c) {
		t.Fatal("verbose without quiet should be true")
	}
}

func TestVerboseFlag_falseByDefault(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().BoolP("quiet", "q", false, "")
	c.Flags().BoolP("verbose", "v", false, "")
	if verboseFlag(c) {
		t.Fatal("default verbose is false")
	}
}
