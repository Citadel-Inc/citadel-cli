package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunReleaseAssetDownload_RefusesBinaryTTY(t *testing.T) {
	const assetID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/namespaces/acme/demo/releases/v1.0.0/assets/" + assetID:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": assetID, "download_url": srv.URL + "/signed/artifact.bin",
			})
		case "/signed/artifact.bin":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte{0x00, 0xff})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	originalTTYCheck := downloadOutputIsTTY
	downloadOutputIsTTY = func(io.Writer) bool { return true }
	defer func() { downloadOutputIsTTY = originalTTYCheck }()

	var out bytes.Buffer
	command := &cobra.Command{}
	command.SetContext(context.Background())
	command.SetOut(&out)
	command.SetErr(io.Discard)
	command.Flags().String("repo", "", "")
	command.Flags().String("output-file", "", "")
	if err := command.Flags().Set("repo", "acme/demo"); err != nil {
		t.Fatal(err)
	}

	err := runReleaseAssetDownload(command, []string{"v1.0.0", assetID})
	if err == nil || !strings.Contains(err.Error(), "redirect stdout") {
		t.Fatalf("want TTY binary refusal, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("TTY output received %d bytes", out.Len())
	}
}
