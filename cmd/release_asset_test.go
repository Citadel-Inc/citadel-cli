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

func TestReleaseAssetLooksBinary(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		prefix      []byte
		want        bool
	}{
		{name: "octet stream", contentType: "application/octet-stream", prefix: []byte("plain"), want: true},
		{name: "image", contentType: "image/png", prefix: []byte("not enough"), want: true},
		{name: "text", contentType: "text/plain; charset=utf-8", prefix: []byte{0x00, 0x01}, want: false},
		{name: "detected binary", prefix: []byte{0x89, 'P', 'N', 'G'}, want: true},
		{name: "detected text", prefix: []byte("release notes\n"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := downloadLooksBinary(test.contentType, test.prefix); got != test.want {
				t.Fatalf("downloadLooksBinary(%q, %v) = %v, want %v", test.contentType, test.prefix, got, test.want)
			}
		})
	}
}

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
