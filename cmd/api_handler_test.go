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
	err := rootFor(cmd.APICmd, "-X", "POST", "/foo", "-f", "noequalssign").Execute()
	if err == nil || !strings.Contains(err.Error(), "key=value") {
		t.Fatalf("want key=value error, got %v", err)
	}
}

func TestAPI_UnsupportedMethod(t *testing.T) {
	err := rootFor(cmd.APICmd, "-X", "HEAD", "/foo").Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported method") {
		t.Fatalf("want unsupported-method error, got %v", err)
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

func TestAPI_InputAndFieldsConflict(t *testing.T) {
	err := rootFor(cmd.APICmd, "-X", "POST", "/foo", "--input", "-", "-f", "name=value").Execute()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("want input/field conflict error, got %v", err)
	}
}

func TestAPI_InputRejectsGet(t *testing.T) {
	err := rootFor(cmd.APICmd, "--input", "-", "/foo").Execute()
	if err == nil || !strings.Contains(err.Error(), "only supported with POST, PUT, or PATCH") {
		t.Fatalf("want GET input error, got %v", err)
	}
}

func TestAPI_InputRejectsDelete(t *testing.T) {
	err := rootFor(cmd.APICmd, "-X", "DELETE", "/foo", "--input", "-").Execute()
	if err == nil || !strings.Contains(err.Error(), "only supported with POST, PUT, or PATCH") {
		t.Fatalf("want DELETE input error, got %v", err)
	}
}
