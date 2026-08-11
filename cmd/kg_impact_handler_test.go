package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestKgImpact_HumanOutput(t *testing.T) {
	const id = "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/kg/impact": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"symbol":             map[string]any{"id": id, "kind": "function", "name": "foo", "path": "x.go"},
				"direct_callers":     []map[string]any{{"id": "caller-id", "name": "bar", "path": "caller.go"}},
				"transitive_callers": []any{},
				"affected_files":     []string{"x.go"},
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.KgCmd, &stdout, "impact", "myorg", id).Execute(); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(stdout.String(), "\n")
	if lines[0] != "rename-impact for foo (function) at x.go" {
		t.Fatalf("impact header = %q", lines[0])
	}
	if !strings.Contains(stdout.String(), "    - bar  in caller.go\n") {
		t.Fatalf("impact direct caller missing from output: %q", stdout.String())
	}
}

func TestKgImpact_EmptySymbol_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.KgCmd, "impact", "myorg", "   ").Execute()
	if err == nil || err.Error() != "symbol cannot be empty" {
		t.Fatalf("want empty-symbol validation error, got %v", err)
	}
}
