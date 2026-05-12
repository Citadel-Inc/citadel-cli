package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestMaskSecret_Short(t *testing.T) {
	if got := maskSecret("short"); got != "***" {
		t.Fatalf("short: got %q want ***", got)
	}
}

func TestMaskSecret_Exact8(t *testing.T) {
	if got := maskSecret("12345678"); got != "***" {
		t.Fatalf("8-char: got %q want ***", got)
	}
}

func TestMaskSecret_Long(t *testing.T) {
	got := maskSecret("abcdefghijkl")
	if got == "abcdefghijkl" {
		t.Fatal("long: must not return raw secret")
	}
	if got == "***" {
		t.Fatal("long: must not be fully masked")
	}
}

// TestSelfHostDebugHelpers exercises selfHostDebugFlag, selfHostLogger, and
// selfHostDebugf in both the disabled (default) and enabled (--debug=true) paths.
// A cobra.Command with a local "debug" bool flag simulates the persistent flag
// that SelfHostCmd declares without requiring the full command tree.
func TestSelfHostDebugHelpers(t *testing.T) {
	newCmd := func(debug bool) *cobra.Command {
		c := &cobra.Command{Use: "t"}
		c.Flags().Bool("debug", debug, "")
		stderr := &bytes.Buffer{}
		c.SetErr(stderr)
		return c
	}

	noDebug := newCmd(false)
	if selfHostDebugFlag(noDebug) {
		t.Fatal("debug flag off: selfHostDebugFlag should return false")
	}
	loggerOff := selfHostLogger(noDebug)
	if loggerOff == nil {
		t.Fatal("debug off: selfHostLogger must return non-nil logger")
	}
	selfHostDebugf(noDebug, "should not appear") // exercises the if-false path

	withDebug := newCmd(true)
	if !selfHostDebugFlag(withDebug) {
		t.Fatal("debug flag on: selfHostDebugFlag should return true")
	}
	loggerOn := selfHostLogger(withDebug)
	if loggerOn == nil {
		t.Fatal("debug on: selfHostLogger must return non-nil logger")
	}
	// selfHostDebugf should write to stderr when --debug is set.
	var errBuf bytes.Buffer
	withDebug.SetErr(&errBuf)
	selfHostDebugf(withDebug, "ping %s", "pong")
	if !bytes.Contains(errBuf.Bytes(), []byte("ping pong")) {
		t.Fatalf("debug on: expected 'ping pong' in stderr, got %q", errBuf.String())
	}
}
