package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestProjectAdminRecoveryScan_HappyPath(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repos_enqueued":4}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdin := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = previousStdin })
	if _, err := writer.WriteString("recovery-scan\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()

	var stdout bytes.Buffer
	cmd := newProjectAdminRecoveryScanTestCommand(&stdout)
	if err := runProjectAdminRecoveryScan(cmd, nil); err != nil {
		t.Fatalf("runProjectAdminRecoveryScan: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/projectgraph/admin/recovery-scan" {
		t.Fatalf("request = %s %s, want POST /api/projectgraph/admin/recovery-scan", gotMethod, gotPath)
	}
	if got, want := stdout.String(), "repos_enqueued:\t4\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestProjectAdminRecoveryScan_ConfirmMismatch(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdin := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = previousStdin })
	if _, err := writer.WriteString("wrong\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()

	var stdout bytes.Buffer
	cmd := newProjectAdminRecoveryScanTestCommand(&stdout)
	if err := runProjectAdminRecoveryScan(cmd, nil); err == nil {
		t.Fatal("expected confirmation mismatch error")
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestProjectAdminRecoveryScan_Yes(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/api/projectgraph/admin/recovery-scan" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repos_enqueued":9}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	cmd := newProjectAdminRecoveryScanTestCommand(&stdout)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("output", "json"); err != nil {
		t.Fatal(err)
	}
	if err := runProjectAdminRecoveryScan(cmd, nil); err != nil {
		t.Fatalf("runProjectAdminRecoveryScan --yes: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	var got struct {
		ReposEnqueued int `json:"repos_enqueued"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if got.ReposEnqueued != 9 {
		t.Fatalf("repos_enqueued = %d, want 9", got.ReposEnqueued)
	}
}

func TestProjectAdminRecoveryScan_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	cmd := newProjectAdminRecoveryScanTestCommand(&stdout)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("output", "toml"); err != nil {
		t.Fatal(err)
	}
	err := runProjectAdminRecoveryScan(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestProjectAdminRecoveryScan_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/projectgraph/admin/recovery-scan" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var stdout bytes.Buffer
	cmd := newProjectAdminRecoveryScanTestCommand(&stdout)
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	err := runProjectAdminRecoveryScan(cmd, nil)
	if err == nil {
		t.Fatal("expected forbidden recovery scan error")
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "forbidden") && !strings.Contains(message, "403") {
		t.Fatalf("error = %q, want forbidden or 403", err)
	}
}

func newProjectAdminRecoveryScanTestCommand(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	addOutputFlag(cmd)
	addJSONFlag(cmd)
	addYesFlag(cmd)
	cmd.SetContext(context.Background())
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd
}
