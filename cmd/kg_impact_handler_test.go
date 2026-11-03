package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/cmd"
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
	if lines[0] != "impact foo (function) x.go: 1 direct, 0 transitive, 1 files" {
		t.Fatalf("impact header = %q", lines[0])
	}
	if !strings.Contains(stdout.String(), "  caller.go: bar\n") {
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

func TestKgImpact_DepthRange_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_REPO", "")

	for _, depth := range []string{"0", "4", "-1"} {
		t.Run(depth, func(t *testing.T) {
			err := rootFor(cmd.KgCmd, "impact", "myorg", "01234567-89ab-cdef-0123-456789abcdef", "--depth", depth).Execute()
			if err == nil || err.Error() != "--depth must be between 1 and 3" {
				t.Fatalf("want depth validation error, got %v", err)
			}
		})
	}
}

func TestKgImpact_GroupsByFileAndLabelsNonCallEdges(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/kg/impact": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("tests") != "exclude" {
				t.Errorf("--no-tests should exclude tests: %s", r.URL.RawQuery)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"symbol": map[string]any{"id": id, "kind": "function", "name": "Load", "qualifier": "Store", "path": "s.go"},
				"direct_callers": []map[string]any{
					{"id": "1", "name": "Show", "qualifier": "H", "path": "a.go", "via": "call"},
					{"id": "2", "name": "run", "path": "a.go", "via": "dispatch"},
				},
				"transitive_callers": []map[string]any{{"id": "3", "name": "fetchIt", "path": "w.ts", "via": "http"}},
				"affected_files":     []string{"a.go", "w.ts"},
				"unresolved":         []map[string]any{{"name": "Tick", "qualifier": "W", "path": "b.go"}},
				"unresolved_total":   7,
			})
		},
	}))
	var stdout strings.Builder
	if err := rootForOut(cmd.KgCmd, &stdout, "impact", "myorg", id, "--no-tests").Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"impact Store.Load (function) s.go: 2 direct, 1 transitive, 2 files\n",
		"  a.go: H.Show, run [dispatch]\n",
		"  w.ts: fetchIt [http]\n",
		"not counted (7):\n  b.go: W.Tick\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestKgImpact_QualifiedNamesPickTheRightSymbol(t *testing.T) {
	target := "00000000-0000-0000-0000-000000000002"
	cases := map[string]string{
		"Service.Skip":                "receiver",
		"queue.Skip":                  "package",
		"queue.Service.Skip":          "package and receiver",
		"internal/queue/q.go:Skip":    "file",
		"internal/queue:Service.Skip": "directory and receiver",
	}
	for name, why := range cases {
		t.Run(why, func(t *testing.T) {
			withServer(t, route(t, map[string]http.HandlerFunc{
				"GET /api/namespaces/myorg/kg/symbols": func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("q") != "Skip" {
						t.Errorf("search should use the bare name: %s", r.URL.RawQuery)
					}
					writeJSON(t, w, 200, map[string]any{"matches": []map[string]any{
						{"id": "00000000-0000-0000-0000-000000000001", "name": "Skip", "kind": "function", "path": "internal/other/o.go", "qualifier": "Other"},
						{"id": target, "name": "Skip", "kind": "function", "path": "internal/queue/q.go", "qualifier": "Service"},
					}})
				},
				"GET /api/namespaces/myorg/kg/impact": func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("symbol") != target {
						t.Errorf("%s resolved to %s", name, r.URL.Query().Get("symbol"))
					}
					writeJSON(t, w, http.StatusOK, map[string]any{"symbol": map[string]any{"id": target}})
				},
			}))
			if err := rootFor(cmd.KgCmd, "impact", "myorg", name).Execute(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestKgWalk_FromFile(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/kg/walk": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("from") != "a.go" || r.URL.Query().Get("seed_id") != "" {
				t.Errorf("query = %s", r.URL.RawQuery)
			}
			writeJSON(t, w, 200, map[string]any{"nodes": []any{}})
		},
	}))
	if err := rootFor(cmd.KgCmd, "walk", "acme/r1", "--from", "a.go").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestKgUnreferenced_StatesCertaintyAndNote(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/kg/unreferenced": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"note": "Candidates, not proof", "total_count": 1, "items": []map[string]any{
				{"name": "Dead", "path": "a.go", "certainty": "uncertain", "test_callers": 2, "unresolved_name_matches": 1},
			}})
		},
	}))
	var stdout strings.Builder
	if err := rootForOut(cmd.KgCmd, &stdout, "unreferenced", "acme/r1").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Candidates, not proof") || !strings.Contains(stdout.String(), "a.go Dead  uncertain test_callers=2 unresolved=1") {
		t.Errorf("output = %s", stdout.String())
	}
}
