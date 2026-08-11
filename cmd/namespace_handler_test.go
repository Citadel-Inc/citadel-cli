package cmd_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestNamespaceList_BadOutput_Hermetic(t *testing.T) {
	assertNamespaceBadOutput(t, "list")
}

func TestNamespaceMembers_BadOutput_Hermetic(t *testing.T) {
	assertNamespaceBadOutput(t, "members", "acme")
}

func TestNamespaceTransferListPending_BadOutput_Hermetic(t *testing.T) {
	assertNamespaceBadOutput(t, "transfer", "list-pending")
}

func TestNamespaceTransferInitiate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.NamespaceCmd, "transfer", "initiate", "acme", "--to", "newowner", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for transfer supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceTransferAccept_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.NamespaceCmd, "transfer", "accept", "550e8400-e29b-41d4-a716-446655440000", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for transfer supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceRename_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.NamespaceCmd, "rename", "acme", "--new-slug", "new-acme", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for rename supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceTransferRevoke_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	transferID := "550e8400-e29b-41d4-a716-446655440000"
	output, err := executeNamespaceCommand(t, "transfer", "revoke", transferID, "--dry-run")
	if err != nil {
		t.Fatalf("namespace transfer revoke dry-run: %v", err)
	}
	if !strings.Contains(output, "Would DELETE /transfers/"+transferID+" (skipped; --dry-run)") {
		t.Fatalf("namespace transfer revoke dry-run output = %q", output)
	}
}

func TestNamespaceDelete_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output, err := executeNamespaceCommand(t, "delete", "acme", "--dry-run")
	if err != nil {
		t.Fatalf("namespace delete dry-run: %v", err)
	}
	if !strings.Contains(output, "Would DELETE /namespaces/acme (skipped; --dry-run)") {
		t.Fatalf("namespace delete dry-run output = %q", output)
	}
}

func TestNamespaceRename_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output, err := executeNamespaceCommand(t, "rename", "acme", "--new-slug", "new-acme", "--dry-run")
	if err != nil {
		t.Fatalf("namespace rename dry-run: %v", err)
	}
	if !strings.Contains(output, "Would rename acme → new-acme (skipped; --dry-run)") {
		t.Fatalf("namespace rename dry-run output = %q", output)
	}
}

func TestNamespaceList_WatchJSON_Hermetic(t *testing.T) {
	assertNamespaceWatchJSON(t, "list")
}

func TestNamespaceMembers_WatchJSON_Hermetic(t *testing.T) {
	assertNamespaceWatchJSON(t, "members", "acme")
}

func TestNamespaceTransferListPending_WatchJSON_Hermetic(t *testing.T) {
	assertNamespaceWatchJSON(t, "transfer", "list-pending")
}

func assertNamespaceBadOutput(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.NamespaceCmd, append(args, "--output", "toml")...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func assertNamespaceWatchJSON(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.NamespaceCmd, append(args, "--watch", "--output", "json")...).Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Fatalf("want watch/output validation error, got %v", err)
	}
}

func executeNamespaceCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})
	var commandOutput bytes.Buffer
	root := rootForOut(cmd.NamespaceCmd, &commandOutput, args...)
	os.Stdout = writer
	executeErr := root.Execute()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return commandOutput.String() + string(output), executeErr
}
