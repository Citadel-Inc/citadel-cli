package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestKgFetchPagesAndNDJSON(t *testing.T) {
	var cursors []string
	pages, err := kgFetchPages(func(cursor string) (any, error) {
		cursors = append(cursors, cursor)
		switch cursor {
		case "":
			return map[string]any{
				"results":     []any{map[string]any{"path": "one.go"}},
				"next_cursor": "page-2",
			}, nil
		case "page-2":
			return map[string]any{
				"results": []any{map[string]any{"path": "two.go"}},
			}, nil
		default:
			t.Fatalf("unexpected cursor %q", cursor)
			return nil, nil
		}
	}, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(cursors, want) {
		t.Fatalf("cursors = %#v, want %#v", cursors, want)
	}

	cmd := kgTestOutputCommand(t, "ndjson")
	if err := kgWritePages(cmd, pages, true, "results"); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(cmd.OutOrStdout().(*bytes.Buffer).String()), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if want := []map[string]any{{"path": "one.go"}, {"path": "two.go"}}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestKgWritePagesTable(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		keys    []string
		header  []string
		values  []string
	}{
		{
			name: "search",
			payload: map[string]any{
				"results": []any{map[string]any{
					"namespace_name":  "acme",
					"repository_slug": "cli",
					"file_path":       "cmd/main.go",
					"line_number":     12,
					"symbol_name":     "Run",
					"match":           "serve",
				}},
			},
			keys:   []string{"results"},
			header: []string{"NAMESPACE", "REPOSITORY", "PATH", "LINE", "SYMBOL", "SNIPPET"},
			values: []string{"acme", "cli", "cmd/main.go", "12", "Run", "serve"},
		},
		{
			name: "symbols",
			payload: map[string]any{
				"symbols": []any{map[string]any{
					"symbol_name":    "Run",
					"symbol_kind":    "function",
					"file_path":      "cmd/main.go",
					"line_start":     12,
					"signature_text": "func Run()",
				}},
			},
			keys:   []string{"symbols"},
			header: []string{"NAME", "KIND", "PATH", "LINE", "SIGNATURE"},
			values: []string{"Run", "function", "cmd/main.go", "12", "func Run()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := kgTestOutputCommand(t, "table")
			if err := kgWritePages(cmd, []any{tt.payload}, false, tt.keys...); err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(cmd.OutOrStdout().(*bytes.Buffer).String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("table lines = %#v, want header plus one row", lines)
			}
			if got := strings.Fields(lines[0]); !reflect.DeepEqual(got, tt.header) {
				t.Fatalf("header = %#v, want %#v", got, tt.header)
			}
			for _, value := range tt.values {
				if !strings.Contains(lines[1], value) {
					t.Fatalf("row %q does not contain %q", lines[1], value)
				}
			}
		})
	}
}

func TestKgWritePagesAllRejectsJSON(t *testing.T) {
	cmd := kgTestOutputCommand(t, "json")
	err := kgWritePages(cmd, []any{map[string]any{"results": []any{}}}, true, "results")
	if err == nil || !strings.Contains(err.Error(), "--output json") {
		t.Fatalf("error = %v, want --output json conflict", err)
	}
}

func TestKgSearchBadOutputHermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var output strings.Builder
	err := kgRootForOut(&output, "search", "needle", "--output", "bogus").Execute()
	if err == nil || !strings.Contains(err.Error(), `--output: unknown format "bogus"`) {
		t.Fatalf("error = %v, want local output validation", err)
	}
}

func TestKgSearchAllJSONHermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var output strings.Builder
	err := kgRootForOut(&output, "search", "needle", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used with --output json") {
		t.Fatalf("error = %v, want local --all/output validation", err)
	}
}

func TestKgExtendedBadOutputHermetic(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "symbols",
			args: []string{"symbols", "org/r1", "--q", "needle", "--output", "bogus"},
			want: `--output: unknown format "bogus"`,
		},
		{
			name: "files",
			args: []string{"files", "org/r1", "--output", "bogus"},
			want: `--output: unknown format "bogus"`,
		},
		{
			name: "fulltext",
			args: []string{"fulltext", "org/r1", "--q", "needle", "--output", "bogus"},
			want: `--output: unknown format "bogus"`,
		},
		{
			name: "walk",
			args: []string{"walk", "org/r1", "--seed-id", "seed", "--output", "bogus"},
			want: `--output supports json or yaml for kg queries; got "bogus"`,
		},
		{
			name: "diff",
			args: []string{"diff", "org/r1", "--output", "bogus"},
			want: `--output supports json or yaml for kg queries; got "bogus"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			var output strings.Builder
			err := kgRootForOut(&output, tt.args...).Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want local output validation containing %q", err, tt.want)
			}
		})
	}
}

func TestKgExtendedRequiredFlagsHermetic(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "symbols query",
			args: []string{"symbols", "org/r1"},
			want: "q",
		},
		{
			name: "walk seed",
			args: []string{"walk", "org/r1"},
			want: "seed-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			var output strings.Builder
			err := kgRootForOut(&output, tt.args...).Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want required flag validation containing %q", err, tt.want)
			}
		})
	}
}

func TestKgCursorValidationHermetic(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "search",
			args: []string{"search", "needle", "--cursor", "not-base64!!!"},
		},
		{
			name: "symbols",
			args: []string{"symbols", "--q", "needle", "--cursor", "not-base64!!!"},
		},
		{
			name: "files",
			args: []string{"files", "--cursor", "not-base64!!!"},
		},
		{
			name: "fulltext",
			args: []string{"fulltext", "--q", "needle", "--cursor", "not-base64!!!"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setKgHermeticEnv(t)

			var output strings.Builder
			err := kgRootForOut(&output, tt.args...).Execute()
			if err == nil {
				t.Fatal("error = nil, want invalid cursor error")
			}
			if got, want := err.Error(), "invalid --cursor: invalid_cursor: desc"; got != want {
				t.Fatalf("error = %q, want %q", got, want)
			}
		})
	}
}

func TestKgMissingRepoHermetic(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "symbols",
			args: []string{"symbols", "--q", "needle", "--no-cwd-repo"},
		},
		{
			name: "files",
			args: []string{"files", "--no-cwd-repo"},
		},
		{
			name: "fulltext",
			args: []string{"fulltext", "--q", "needle", "--no-cwd-repo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setKgHermeticEnv(t)

			var output strings.Builder
			err := kgRootForOut(&output, tt.args...).Execute()
			if err == nil {
				t.Fatal("error = nil, want missing repository error")
			}
			if got, want := err.Error(), "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git"; got != want {
				t.Fatalf("error = %q, want %q", got, want)
			}
		})
	}
}

func setKgHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestKgSearch_AllPagination(t *testing.T) {
	runKgAllPaginationTest(t,
		"/api/kg/search",
		[]string{"search", "needle", "--all", "--output", "ndjson"},
		"results",
		map[string]any{"path": "search-page-one.go"},
		map[string]any{"path": "search-page-two.go"},
		"search-page-one.go",
		"search-page-two.go",
	)
}

func TestKgSymbols_AllPagination(t *testing.T) {
	runKgAllPaginationTest(t,
		"/api/namespaces/org/kg/symbols",
		[]string{"symbols", "org/r1", "--q", "needle", "--all", "--output", "ndjson"},
		"symbols",
		map[string]any{"symbol_name": "SymbolOne"},
		map[string]any{"symbol_name": "SymbolTwo"},
		"SymbolOne",
		"SymbolTwo",
	)
}

func TestKgFiles_AllPagination(t *testing.T) {
	runKgAllPaginationTest(t,
		"/api/namespaces/org/kg/files",
		[]string{"files", "org/r1", "--all", "--output", "ndjson"},
		"files",
		map[string]any{"path": "files-page-one.go"},
		map[string]any{"path": "files-page-two.go"},
		"files-page-one.go",
		"files-page-two.go",
	)
}

func TestKgFulltext_AllPagination(t *testing.T) {
	runKgAllPaginationTest(t,
		"/api/namespaces/org/kg/fulltext",
		[]string{"fulltext", "org/r1", "--q", "needle", "--all", "--output", "ndjson"},
		"results",
		map[string]any{"file_path": "fulltext-page-one.go"},
		map[string]any{"file_path": "fulltext-page-two.go"},
		"fulltext-page-one.go",
		"fulltext-page-two.go",
	)
}

func runKgAllPaginationTest(t *testing.T, path string, args []string, rowKey string, first, second map[string]any, wantValues ...string) {
	t.Helper()
	var calls int
	var cursors []string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, path)
		}
		calls++
		cursor := r.URL.Query().Get("cursor")
		cursors = append(cursors, cursor)
		switch calls {
		case 1:
			if cursor != "" {
				t.Fatalf("first-page cursor = %q, want empty", cursor)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				rowKey:        []any{first},
				"next_cursor": "cursor-page-two",
			})
		case 2:
			if cursor != "cursor-page-two" {
				t.Fatalf("second-page cursor = %q, want cursor-page-two", cursor)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{rowKey: []any{second}})
		default:
			t.Fatalf("unexpected request %d with cursor %q", calls, cursor)
		}
	})

	var output strings.Builder
	if err := kgRootForOut(&output, args...).Execute(); err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("API calls = %d, want at least 2", calls)
	}
	if want := []string{"", "cursor-page-two"}; !reflect.DeepEqual(cursors, want) {
		t.Fatalf("cursors = %#v, want %#v", cursors, want)
	}
	for _, value := range wantValues {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("merged output missing %q: %s", value, output.String())
		}
	}
}

func kgRootForOut(stdout io.Writer, args ...string) *cobra.Command {
	resetFlagsRecursive(KgCmd)
	setOutRecursive(KgCmd, stdout, io.Discard)
	root := NewRootCmd()
	root.SetArgs(append([]string{"kg"}, args...))
	root.SetOut(stdout)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

func kgTestOutputCommand(t *testing.T, output string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().String("output", output, "output format")
	return cmd
}
