package cmd

import (
	"bytes"
	"encoding/json"
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

func kgTestOutputCommand(t *testing.T, output string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().String("output", output, "output format")
	return cmd
}
