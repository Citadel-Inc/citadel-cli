package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

func TestReleaseAssetDownloadOutputFileMode(t *testing.T) {
	oldMask := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(oldMask) })

	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var downloadURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/download/") {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("asset-bytes"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "asset-1",
			"download_url": downloadURL,
		})
	}))
	t.Cleanup(srv.Close)
	downloadURL = srv.URL + "/download/asset.bin"

	if err := clicfg.Update(func(cfg *clicfg.Config) error {
		cfg.ServerURL = srv.URL
		cfg.AccessToken = "test-token"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "asset.bin")
	cmd := &cobra.Command{Use: "download"}
	cmd.Flags().StringP("output-file", "o", "", "")
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.PersistentFlags().String("server", srv.URL, "")
	cmd.PersistentFlags().Bool("verbose", false, "")
	cmd.PersistentFlags().Bool("debug-http", false, "")
	cmd.PersistentFlags().Bool("quiet", false, "")
	if err := cmd.Flags().Set("output-file", out); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("repo", "org/repo"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(t.Context())

	if err := runReleaseAssetDownload(cmd, []string{"v1.0.0", "asset-1"}); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("downloaded file mode = %04o, want 0600", perm)
	}
}
