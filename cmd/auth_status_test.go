package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunStatus_opaqueTokenExpiryUnknown(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "opaque-not-a-jwt")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()

	runErr := runStatus(&cobra.Command{}, nil)
	if closeErr := w.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if runErr != nil {
		t.Fatalf("runStatus: %v", runErr)
	}
	got := string(out)
	if !strings.Contains(got, "Session: opaque token (expiry unknown)") {
		t.Fatalf("status = %q, want opaque-token unknown-expiry line", got)
	}
	if strings.Contains(got, "Session: EXPIRED") {
		t.Fatalf("status = %q, opaque token must not print EXPIRED", got)
	}
}
