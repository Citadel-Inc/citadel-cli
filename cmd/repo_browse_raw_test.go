package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunRepoBrowseRaw_RefusesBinaryTTY(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/namespaces/acme/repos/demo/raw" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x00, 0xff, 0x01})
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	originalTTYCheck := repoBrowseOutputIsTTY
	repoBrowseOutputIsTTY = func(io.Writer) bool { return true }
	defer func() { repoBrowseOutputIsTTY = originalTTYCheck }()

	var out bytes.Buffer
	command := &cobra.Command{}
	command.SetContext(context.Background())
	command.SetOut(&out)
	command.SetErr(io.Discard)
	command.Flags().String("repo", "", "")
	command.Flags().String("ref", "", "")
	command.Flags().String("output-file", "", "")
	if err := command.Flags().Set("repo", "acme/demo"); err != nil {
		t.Fatal(err)
	}

	err := runRepoBrowseRaw(command, []string{"artifact.bin"})
	if err == nil || !strings.Contains(err.Error(), "redirect stdout") {
		t.Fatalf("want TTY binary refusal, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("TTY output received %d bytes", out.Len())
	}
}
