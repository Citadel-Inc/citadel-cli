package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func releaseTagBody() map[string]any {
	return map[string]any{
		"id":            "11111111-1111-1111-1111-111111111111",
		"namespace_id":  "ns1",
		"repo_id":       "repo1",
		"tag_name":      "v1.0.0",
		"name":          "v1.0.0",
		"body_markdown": "Initial GA",
		"draft":         false,
		"prerelease":    false,
		"author_id":     "user1",
		"published_at":  "2026-05-10T00:00:00Z",
		"created_at":    "2026-05-09T00:00:00Z",
		"updated_at":    "2026-05-10T00:00:00Z",
	}
}

func TestReleaseList_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases", "/namespaces/acme/demo/releases") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"releases": []map[string]any{releaseTagBody()},
			"cursor":   "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "list", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"tag_name": "v1.0.0"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestReleaseList_TableEmpty(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{"releases": []map[string]any{}})
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "list", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No releases") {
		t.Fatalf("expected 'No releases' message, got: %s", out.String())
	}
}

func TestReleaseList_IncludeDrafts(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("include_drafts") != "true" {
			t.Errorf("missing include_drafts=true query")
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"releases": []map[string]any{}})
	})
	if err := rootFor(cmd.ReleaseCmd, "list", "-R", "acme/demo", "--include-drafts").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseLatest_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases/latest", "/namespaces/acme/demo/releases/latest") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "latest", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "v1.0.0") {
		t.Fatalf("expected tag in output: %s", out.String())
	}
}

func TestReleaseLatest_None(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "latest", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "no published releases") {
		t.Fatalf("want no-published error, got %v", err)
	}
}

func TestReleaseView_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases/v1.0.0", "/namespaces/acme/demo/releases/v1.0.0") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "view", "v1.0.0", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseView_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "view", "v9.9.9", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestReleaseView_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"view", "v1.0.0", "-R", "acme/demo",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseCreate_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		if body["tag_name"] != "v1.0.0" {
			t.Errorf("tag_name = %v", body["tag_name"])
		}
		if body["name"] != "release-1" {
			t.Errorf("name = %v", body["name"])
		}
		writeJSON(t, w, http.StatusCreated, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v1.0.0", "--name", "release-1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseCreate_DuplicateTag(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})
	err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v1.0.0").Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestReleaseCreate_TagNotPushed(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	})
	err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v9.9.9").Execute()
	if err == nil || !strings.Contains(err.Error(), "does not exist on the remote") {
		t.Fatalf("want 422 error, got %v", err)
	}
}

func TestReleaseCreate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"create", "-R", "acme/demo",
		"--tag", "v1.0.0",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseCreate_MissingTag_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"create", "-R", "acme/demo",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "tag") {
		t.Fatalf("want missing tag validation error, got %v", err)
	}
}

func TestReleaseCreate_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out,
		"create", "-R", "acme/demo",
		"--tag", "v1.0.0",
		"--dry-run",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Would POST /namespaces/acme%2Fdemo/releases tag=v1.0.0 (skipped; --dry-run)\n"
	if out.String() != want {
		t.Fatalf("dry-run output = %q, want %q", out.String(), want)
	}
}

func TestReleaseEdit_NoChange(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should not call server when nothing to update")
		w.WriteHeader(http.StatusOK)
	})
	err := rootFor(cmd.ReleaseCmd, "edit", "v1.0.0", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("want nothing-to-update error, got %v", err)
	}
}

func TestReleaseEdit_NoChange_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"edit", "v1.0.0", "-R", "acme/demo",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("want nothing-to-update error, got %v", err)
	}
}

func TestReleaseEdit_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"edit", "v1.0.0", "-R", "acme/demo",
		"--name", "renamed",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseEdit_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out,
		"edit", "v1.0.0", "-R", "acme/demo",
		"--name", "renamed",
		"--dry-run",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Would PATCH /namespaces/acme%2Fdemo/releases/v1.0.0 (skipped; --dry-run)\n"
	if out.String() != want {
		t.Fatalf("dry-run output = %q, want %q", out.String(), want)
	}
}

func TestReleaseEdit_DraftFalse(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q", r.Method)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		if v, ok := body["draft"].(bool); !ok || v != false {
			t.Errorf("draft = %v (%T)", body["draft"], body["draft"])
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "edit", "v1.0.0", "-R", "acme/demo", "--draft=false").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDelete_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.ReleaseCmd, "delete", "v1.0.0", "-R", "acme/demo", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDelete_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"delete", "v1.0.0", "-R", "acme/demo",
		"--output", "toml", "--yes",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), `--output for delete supports json or default human summary only; got "toml"`) {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseDelete_DryRun(t *testing.T) {
	withServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("dry-run must not call the server")
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "delete", "v1.0.0", "-R", "acme/demo", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would DELETE") {
		t.Fatalf("expected dry-run preview, got: %s", out.String())
	}
}

func TestReleaseDelete_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "delete", "v9.9.9", "-R", "acme/demo", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestReleaseAssetCRUD_RoundTrip(t *testing.T) {
	const assetID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	payload := []byte{0x00, 0x01, 0x7f, 0x80, 0xff}
	var downloadURL string
	srv := withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && issuePathMatches(r,
			"/namespaces/acme%2Fdemo/releases/v1.0.0/assets",
			"/namespaces/acme/demo/releases/v1.0.0/assets",
		):
			writeJSON(t, w, http.StatusOK, map[string]any{"assets": []map[string]any{{
				"id": assetID, "filename": "artifact.bin", "content_type": "application/octet-stream",
				"size_bytes": len(payload), "created_at": "2026-05-10T00:00:00Z",
			}}})
		case r.Method == http.MethodPost && issuePathMatches(r,
			"/namespaces/acme%2Fdemo/releases/v1.0.0/assets",
			"/namespaces/acme/demo/releases/v1.0.0/assets",
		):
			part, err := r.MultipartReader()
			if err != nil {
				t.Fatal(err)
			}
			filePart, err := part.NextPart()
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(filePart)
			if err != nil {
				t.Fatal(err)
			}
			if filePart.FormName() != "file" || filePart.FileName() != "artifact.bin" || string(got) != string(payload) {
				t.Fatalf("upload part = %q/%q %v", filePart.FormName(), filePart.FileName(), got)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"id": assetID, "filename": "artifact.bin", "content_type": "application/octet-stream",
				"size_bytes": len(payload), "created_at": "2026-05-10T00:00:00Z",
			})
		case r.Method == http.MethodGet && issuePathMatches(r,
			"/namespaces/acme%2Fdemo/releases/v1.0.0/assets/"+assetID,
			"/namespaces/acme/demo/releases/v1.0.0/assets/"+assetID,
		):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id": assetID, "filename": "artifact.bin", "content_type": "application/octet-stream",
				"size_bytes": len(payload), "download_url": downloadURL,
				"created_at": "2026-05-10T00:00:00Z",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/signed/artifact.bin":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(payload)
		case r.Method == http.MethodDelete && issuePathMatches(r,
			"/namespaces/acme%2Fdemo/releases/v1.0.0/assets/"+assetID,
			"/namespaces/acme/demo/releases/v1.0.0/assets/"+assetID,
		):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected asset request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})
	downloadURL = srv.URL + "/signed/artifact.bin"

	var listOut strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &listOut, "asset", "list", "v1.0.0", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(listOut.String(), assetID) {
		t.Fatalf("list output = %s", listOut.String())
	}

	uploadPath := t.TempDir() + "/artifact.bin"
	if err := os.WriteFile(uploadPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	var uploadOut strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &uploadOut, "asset", "upload", "v1.0.0", uploadPath, "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uploadOut.String(), assetID) {
		t.Fatalf("upload output = %s", uploadOut.String())
	}

	downloadPath := t.TempDir() + "/download.bin"
	if err := rootFor(cmd.ReleaseCmd, "asset", "download", "v1.0.0", assetID, "-R", "acme/demo", "-o", downloadPath).Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(downloadPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("download = %v, want %v", got, payload)
	}

	if err := rootFor(cmd.ReleaseCmd, "asset", "delete", "v1.0.0", assetID, "-R", "acme/demo", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseAssetDelete_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"asset", "delete", "v1.0.0", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"-R", "acme/demo", "--output", "toml", "--yes",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), `--output for delete supports json or default human summary only; got "toml"`) {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseAssetUpload_StoreUnavailable(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		writeJSON(t, w, http.StatusServiceUnavailable, map[string]string{"error": "asset_store_unavailable"})
	})
	uploadPath := t.TempDir() + "/artifact.bin"
	if err := os.WriteFile(uploadPath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rootFor(cmd.ReleaseCmd, "asset", "upload", "v1.0.0", uploadPath, "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "object store") {
		t.Fatalf("want clear object-store error, got %v", err)
	}
}

func TestReleaseAssetUpload_SizeLimit(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		writeJSON(t, w, http.StatusRequestEntityTooLarge, map[string]string{"error": "payload_too_large"})
	})
	uploadPath := t.TempDir() + "/artifact.bin"
	if err := os.WriteFile(uploadPath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rootFor(cmd.ReleaseCmd, "asset", "upload", "v1.0.0", uploadPath, "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("want size-limit error, got %v", err)
	}
}

func TestReleaseAssetUpload_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"asset", "upload", "v1.0.0", "artifact.bin", "-R", "acme/demo",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseAssetUpload_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	filePath := t.TempDir() + "/artifact.bin"
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out,
		"asset", "upload", "v1.0.0", filePath,
		"-R", "acme/demo",
		"--dry-run",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Would POST /namespaces/acme%2Fdemo/releases/v1.0.0/assets file=" + filePath + " (skipped; --dry-run)\n"
	if out.String() != want {
		t.Fatalf("dry-run output = %q, want %q", out.String(), want)
	}
}

func TestReleaseAssetDownload_MissingURL(t *testing.T) {
	const assetID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && issuePathMatches(r,
			"/namespaces/acme%2Fdemo/releases/v1.0.0/assets/"+assetID,
			"/namespaces/acme/demo/releases/v1.0.0/assets/"+assetID,
		) {
			writeJSON(t, w, http.StatusOK, map[string]any{"id": assetID})
			return
		}
		http.NotFound(w, r)
	})
	err := rootFor(cmd.ReleaseCmd, "asset", "download", "v1.0.0", assetID, "-R", "acme/demo", "-o", t.TempDir()+"/asset.bin").Execute()
	if err == nil || !strings.Contains(err.Error(), "no download URL") {
		t.Fatalf("want missing-download-url error, got %v", err)
	}
}

func TestReleaseList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"list", "-R", "acme/demo",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleaseAssetList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.ReleaseCmd,
		"asset", "list", "v1.0.0", "-R", "acme/demo",
		"--output", "toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestReleasePathGuards_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	tests := []struct {
		name string
		args []string
	}{
		{name: "list", args: []string{"list", "-R", "/"}},
		{name: "latest", args: []string{"latest", "-R", "/"}},
		{name: "view", args: []string{"view", "v1.0.0", "-R", "/"}},
		{name: "asset list", args: []string{"asset", "list", "v1.0.0", "-R", "/"}},
		{name: "asset download", args: []string{"asset", "download", "v1.0.0", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "-R", "/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rootFor(cmd.ReleaseCmd, tt.args...).Execute()
			if err == nil || err.Error() != "namespace path required" {
				t.Fatalf("want namespace-path validation error, got %v", err)
			}
		})
	}
}
