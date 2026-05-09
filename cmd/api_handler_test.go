package cmd_test

import (
	"net/http"
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
