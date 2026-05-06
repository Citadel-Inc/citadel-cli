package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v3"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/term"
)

// outputFormatCompletions matches cli-output-formats acceptance (static list).
var outputFormatCompletions = []string{"json", "yaml", "ndjson", "csv", "table"}

func completeOutputFormats(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	out := make([]string, len(outputFormatCompletions))
	copy(out, outputFormatCompletions)
	return out, cobra.ShellCompDirectiveNoFileComp
}

// addOutputFlag registers the standard `--output` flag on each command.
func addOutputFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().String("output", "", "Output format: json, yaml, ndjson, csv, or table (default human table)")
		_ = c.RegisterFlagCompletionFunc("output", completeOutputFormats)
	}
}

// addYesFlag registers the standard `--yes` flag on each command.
func addYesFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Bool("yes", false, "Skip confirmation prompt")
	}
}

// addJSONFlag registers the standard `--json` flag on each command.
func addJSONFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Bool("json", false, "Output raw JSON")
	}
}

// addDryRunFlag registers the standard `--dry-run` flag on each command.
// Destructive verbs honor it by printing the action that would have been
// taken and skipping the API round trip.
func addDryRunFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Bool("dry-run", false, "Print the action without executing it")
	}
}

// addRepoFlag registers `-R` / `--repo` and `--no-cwd-repo` for commands that
// target a single repository (CWD inference via git origin when omitted).
func addRepoFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringP("repo", "R", "", "Repository as <namespace>/<slug> (overrides CWD inference)")
		c.Flags().Bool("no-cwd-repo", false, "Disable CWD git-remote inference (require -R or "+citadelRepoEnv+")")
	}
}

// outputFlag returns the resolved `--output` flag value (empty if unset).
func outputFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("output")
	return v
}

// yesFlag returns the resolved `--yes` flag value.
func yesFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("yes")
	return v
}

// jsonFlag returns the resolved `--json` flag value.
func jsonFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// dryRunFlag returns the resolved `--dry-run` flag value.
func dryRunFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("dry-run")
	return v
}

// serverFlag returns the resolved persistent `--server` flag value.
func serverFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("server")
	return v
}

// verboseFlag returns the resolved persistent `--verbose` (-v) flag value.
// Suppressed by `--quiet`.
func verboseFlag(cmd *cobra.Command) bool {
	if quietFlag(cmd) {
		return false
	}
	v, _ := cmd.Flags().GetBool("verbose")
	return v
}

// quietFlag returns the resolved persistent `--quiet` (-q) flag value.
func quietFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("quiet")
	return v
}

// debugHTTPFlag returns the resolved persistent `--debug-http` flag value.
// Implies verbose; emits full request/response dumps to stderr.
func debugHTTPFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("debug-http")
	return v
}

// colorEnabled resolves whether color output is permitted given the resolved
// `--color` flag and the NO_COLOR / TTY environment. Helpers that emit ANSI
// styles MUST consult this before writing escape codes.
func colorEnabled(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetString("color")
	return term.ColorEnabled(term.ParseColorMode(v))
}

// CSVRow is implemented by each stable list-row shape for `--output csv`.
type CSVRow interface {
	CSVHeader() []string
	CSVRecord() []string
}

// validateListOutput rejects unknown machine formats for list verbs.
func validateListOutput(output string) error {
	o := strings.TrimSpace(strings.ToLower(output))
	switch o {
	case "", "json", "yaml", "ndjson", "csv", "table":
		return nil
	default:
		return fmt.Errorf("--output: unknown format %q (use json|yaml|ndjson|csv|table)", output)
	}
}

// validateGetOutput rejects list-only formats on single-resource verbs.
func validateGetOutput(output string) error {
	o := strings.TrimSpace(strings.ToLower(output))
	switch o {
	case "", "json", "yaml", "table":
		return nil
	default:
		return fmt.Errorf("--output: unknown format %q (use json|yaml|table)", output)
	}
}

func isHumanListOutput(output string) bool {
	switch strings.TrimSpace(strings.ToLower(output)) {
	case "", "table":
		return true
	default:
		return false
	}
}

func commandOut(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	return cmd.OutOrStdout()
}

// emitJSON writes v as indented JSON to the command's stdout writer.
func emitJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(commandOut(cmd))
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// emitYAML writes one YAML document using JSON field names (via a JSON
// round-trip) so keys stay aligned with `--output json`.
func emitYAML(cmd *cobra.Command, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var tmp any
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	enc := yaml.NewEncoder(commandOut(cmd))
	defer func() { _ = enc.Close() }()
	return enc.Encode(tmp)
}

// emitNDJSONLines writes one compact JSON object per line (newline-delimited).
func emitNDJSONLines[T any](cmd *cobra.Command, rows []T) error {
	return emitNDJSONLinesTo(commandOut(cmd), rows)
}

// emitNDJSONLinesTo is emitNDJSONLines with an explicit writer (tests use this
// to avoid swapping os.Stdout).
func emitNDJSONLinesTo[T any](w io.Writer, rows []T) error {
	enc := json.NewEncoder(w)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

// emitCSVRows writes CSV rows; the header is emitted on the first non-empty
// batch only (streaming-friendly under `--all`).
func emitCSVRows[T CSVRow](cmd *cobra.Command, headerWritten *bool, rows []T) error {
	return emitCSVRowsTo(commandOut(cmd), headerWritten, rows)
}

func emitCSVRowsTo[T CSVRow](dst io.Writer, headerWritten *bool, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	w := csv.NewWriter(dst)
	if !*headerWritten {
		var z T
		if err := w.Write(z.CSVHeader()); err != nil {
			return err
		}
		*headerWritten = true
	}
	for i := range rows {
		if err := w.Write(rows[i].CSVRecord()); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// emitCSVHeaderOnly writes the CSV header row for an empty list result.
func emitCSVHeaderOnly[T CSVRow](cmd *cobra.Command) error {
	return emitCSVHeaderOnlyTo[T](commandOut(cmd))
}

func emitCSVHeaderOnlyTo[T CSVRow](dst io.Writer) error {
	var z T
	out := csv.NewWriter(dst)
	if err := out.Write(z.CSVHeader()); err != nil {
		return err
	}
	out.Flush()
	return out.Error()
}

// newTabWriter returns a tabwriter configured for the table-output style
// shared by every list/get verb (2-space padding, no minwidth, no padchar
// other than space).
func newTabWriter(cmd *cobra.Command) *tabwriter.Writer {
	return tabwriter.NewWriter(commandOut(cmd), 0, 0, 2, ' ', 0)
}

// emitOne centralizes the single-object "json or human" dispatch used by
// get / show / create / accept / etc. verbs. In json mode it emits v;
// otherwise it calls human with a configured tabwriter and flushes.
func emitOne[T any](cmd *cobra.Command, output string, v T, human func(w *tabwriter.Writer, v T)) error {
	o := strings.TrimSpace(strings.ToLower(output))
	switch o {
	case "json":
		return emitJSON(cmd, v)
	case "yaml":
		return emitYAML(cmd, v)
	case "csv", "ndjson":
		return fmt.Errorf("--output %q is only supported on list commands (use json or yaml here)", o)
	case "", "table":
		w := newTabWriter(cmd)
		human(w, v)
		return w.Flush()
	default:
		return fmt.Errorf("--output: unknown format %q (use json|yaml|table)", output)
	}
}
