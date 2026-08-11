package cmd_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
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
	setNamespaceHermeticEnv(t)

	err := rootFor(cmd.NamespaceCmd, "transfer", "initiate", "acme", "--to", "newowner", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for transfer supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceTransferAccept_BadOutput_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	err := rootFor(cmd.NamespaceCmd, "transfer", "accept", "550e8400-e29b-41d4-a716-446655440000", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for transfer supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceTransferDecline_BadOutput_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	err := rootFor(cmd.NamespaceCmd, "transfer", "decline", "550e8400-e29b-41d4-a716-446655440000", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for transfer supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceRename_BadOutput_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	err := rootFor(cmd.NamespaceCmd, "rename", "acme", "--new-slug", "new-acme", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for rename supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestNamespaceList_EmptyHuman(t *testing.T) {
	output, err := executeNamespaceHTTPCommand(t, `{"orgs":[],"next_cursor":""}`, "list")
	if err != nil {
		t.Fatalf("namespace list empty: %v", err)
	}
	if output != "No org namespaces found.\n" {
		t.Fatalf("namespace list empty output = %q", output)
	}
}

func TestNamespaceMembers_EmptyHuman(t *testing.T) {
	output, err := executeNamespaceHTTPCommand(t, `{"members":[],"next_cursor":""}`, "members", "acme")
	if err != nil {
		t.Fatalf("namespace members empty: %v", err)
	}
	if output != "No members in namespace 'acme'\n" {
		t.Fatalf("namespace members empty output = %q", output)
	}
}

func TestNamespaceTransferListPending_EmptyHuman(t *testing.T) {
	output, err := executeNamespaceHTTPCommand(t, `{"transfers":[],"next_cursor":""}`, "transfer", "list-pending")
	if err != nil {
		t.Fatalf("namespace transfer list-pending empty: %v", err)
	}
	if output != "No pending transfers.\n" {
		t.Fatalf("namespace transfer list-pending empty output = %q", output)
	}
}

func TestNamespaceList_PaginationHint(t *testing.T) {
	output, err := executeNamespaceHTTPCommand(t, `{"orgs":[{"slug":"acme","display_name":"Acme"}],"next_cursor":"next-123"}`, "list")
	if err != nil {
		t.Fatalf("namespace list pagination: %v", err)
	}
	want := "SLUG  DISPLAY NAME  CREATED\nacme  Acme          0001-01-01\n(use --cursor next-123 for more, or --all to fetch everything)\n"
	if output != want {
		t.Fatalf("namespace list pagination output = %q, want %q", output, want)
	}
}

func TestNamespaceTransferRevoke_DryRun_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	transferID := "550e8400-e29b-41d4-a716-446655440000"
	output, err := executeNamespaceCommand(t, "transfer", "revoke", transferID, "--dry-run")
	if err != nil {
		t.Fatalf("namespace transfer revoke dry-run: %v", err)
	}
	want := "Would DELETE /transfers/" + transferID + " (skipped; --dry-run)\n"
	if output != want {
		t.Fatalf("namespace transfer revoke dry-run output = %q, want %q", output, want)
	}
}

func TestNamespaceDelete_DryRun_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	output, err := executeNamespaceCommand(t, "delete", "acme", "--dry-run")
	if err != nil {
		t.Fatalf("namespace delete dry-run: %v", err)
	}
	want := "Would DELETE /namespaces/acme (skipped; --dry-run)\n"
	if output != want {
		t.Fatalf("namespace delete dry-run output = %q, want %q", output, want)
	}
}

func TestNamespaceRename_DryRun_Hermetic(t *testing.T) {
	setNamespaceHermeticEnv(t)

	output, err := executeNamespaceCommand(t, "rename", "acme", "--new-slug", "new-acme", "--dry-run")
	if err != nil {
		t.Fatalf("namespace rename dry-run: %v", err)
	}
	want := "Would rename acme → new-acme (skipped; --dry-run)\n"
	if output != want {
		t.Fatalf("namespace rename dry-run output = %q, want %q", output, want)
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
	setNamespaceHermeticEnv(t)

	err := rootFor(cmd.NamespaceCmd, append(args, "--output", "toml")...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func assertNamespaceWatchJSON(t *testing.T, args ...string) {
	t.Helper()
	setNamespaceHermeticEnv(t)

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

func setNamespaceHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
}

func executeNamespaceHTTPCommand(t *testing.T, response string, args ...string) (string, error) {
	t.Helper()
	setNamespaceHermeticEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("CITADEL_SERVER", server.URL)

	var output bytes.Buffer
	root := rootForOut(cmd.NamespaceCmd, &output, args...)
	err := root.Execute()
	return output.String(), err
}
