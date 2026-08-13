package cmd_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestAPI_GetHappy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/acme/demo/issues": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"issues": []any{},
			})
		},
	}))
	if err := rootFor(cmd.APICmd, "/namespaces/acme/demo/issues").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_PostWithFields(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/acme/demo/issues/1/comments": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"id":            "00000000-0000-0000-0000-000000000099",
				"body_markdown": "LGTM",
			})
		},
	}))
	if err := rootFor(cmd.APICmd, "-X", "POST", "/namespaces/acme/demo/issues/1/comments", "-f", "body_markdown=LGTM").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_DeleteHappy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/demo/issues/42": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.APICmd, "-X", "DELETE", "/namespaces/acme/demo/issues/42").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_InvalidField(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.APICmd, "-X", "POST", "/foo", "-f", "noequalssign").Execute()
	if err == nil || err.Error() != `invalid field "noequalssign": must be key=value` {
		t.Fatalf(`want invalid field error, got %v`, err)
	}
}

func TestAPI_UnsupportedMethod(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.APICmd, "-X", "HEAD", "/foo").Execute()
	if err == nil || err.Error() != `unsupported method "HEAD"; use GET, POST, PUT, PATCH, or DELETE` {
		t.Fatalf(`want unsupported-method error, got %v`, err)
	}
}

func TestAPI_SlashPrepended(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /ping": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"ok": true})
		},
	}))
	// Path without leading slash should be prepended automatically.
	if err := rootFor(cmd.APICmd, "ping").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_PostWithInputStdin(t *testing.T) {
	const input = `{"title":"hello","labels":["bug"],"metadata":{"priority":3}}`

	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/acme/demo/issues": func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				return
			}
			if string(body) != input {
				t.Errorf("request body = %q, want %q", body, input)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{"ok": true})
		},
	}))

	root := rootFor(cmd.APICmd, "-X", "POST", "/namespaces/acme/demo/issues", "--input", "-")
	root.SetIn(strings.NewReader(input))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_PatchWithInputFile(t *testing.T) {
	const input = `{"state":"closed","details":{"reason":"done"}}`
	inputPath := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(inputPath, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}

	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /namespaces/acme/demo/issues/42": func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				return
			}
			if string(body) != input {
				t.Errorf("request body = %q, want %q", body, input)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"ok": true})
		},
	}))

	if err := rootFor(cmd.APICmd, "-X", "PATCH", "/namespaces/acme/demo/issues/42", "--input", inputPath).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_PutWithInputStdin(t *testing.T) {
	const input = `{"title":"updated","enabled":true}`

	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT /namespaces/acme/demo/issues/42": func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				return
			}
			if string(body) != input {
				t.Errorf("request body = %q, want %q", body, input)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"ok": true})
		},
	}))

	root := rootFor(cmd.APICmd, "-X", "PUT", "/namespaces/acme/demo/issues/42", "--input", "-")
	root.SetIn(strings.NewReader(input))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_EmptyInput(t *testing.T) {
	var requests int
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /foo": func(w http.ResponseWriter, _ *http.Request) {
			requests++
			writeJSON(t, w, http.StatusCreated, map[string]any{"ok": true})
		},
	}))

	root := rootFor(cmd.APICmd, "-X", "POST", "/foo", "--input", "-")
	root.SetIn(strings.NewReader(""))
	err := root.Execute()
	if err == nil {
		t.Fatal("want error for empty JSON input")
	}
	if !strings.Contains(err.Error(), "--input") {
		t.Fatalf("error = %q, want --input", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestAPI_InvalidJSONInput(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(inputPath, []byte(`{"title":`), 0600); err != nil {
		t.Fatal(err)
	}

	var requests int
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /foo": func(w http.ResponseWriter, _ *http.Request) {
			requests++
			writeJSON(t, w, http.StatusCreated, map[string]any{"ok": true})
		},
	}))

	err := rootFor(cmd.APICmd, "-X", "POST", "/foo", "--input", inputPath).Execute()
	if err == nil {
		t.Fatal("want error for invalid JSON input")
	}
	if !strings.Contains(err.Error(), "--input") {
		t.Fatalf("error = %q, want --input", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

func TestAPI_InputAndFieldsConflict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.APICmd, "-X", "POST", "/foo", "--input", "-", "-f", "name=value").Execute()
	if err == nil || err.Error() != `--input and --field are mutually exclusive` {
		t.Fatalf(`want input/field conflict error, got %v`, err)
	}
}

func TestAPI_InputRejectsGet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.APICmd, "--input", "-", "/foo").Execute()
	if err == nil || err.Error() != `--input is only supported with POST, PUT, or PATCH` {
		t.Fatalf(`want GET input error, got %v`, err)
	}
}

func TestAPI_InputRejectsDelete(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.APICmd, "-X", "DELETE", "/foo", "--input", "-").Execute()
	if err == nil || err.Error() != `--input is only supported with POST, PUT, or PATCH` {
		t.Fatalf(`want DELETE input error, got %v`, err)
	}
}
