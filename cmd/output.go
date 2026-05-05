package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// addOutputFlag registers the standard `--output` flag on c.
func addOutputFlag(c *cobra.Command) {
	c.Flags().String("output", "", "Output format: json")
}

// addYesFlag registers the standard `--yes` flag on c.
func addYesFlag(c *cobra.Command) {
	c.Flags().Bool("yes", false, "Skip confirmation prompt")
}

// addJSONFlag registers the standard `--json` flag on c.
func addJSONFlag(c *cobra.Command) {
	c.Flags().Bool("json", false, "Output raw JSON")
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

// emitJSON writes v as indented JSON to stdout.
func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTabWriter returns a tabwriter configured for the table-output style
// shared by every list/get verb (2-space padding, no minwidth, no padchar
// other than space).
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// emitList centralises the list-handler "json or table" branch and the
// empty-result message printed in human mode. It returns nil if rows is
// empty in human mode (after printing emptyMsg); otherwise it calls table
// with a configured tabwriter and flushes.
//
// rows must be a slice; emptyMsg is printed (with a trailing newline) when
// in human mode and len(rows) == 0.
func emitList[T any](output string, rows []T, emptyMsg string, table func(w *tabwriter.Writer, rows []T)) error {
	if output == "json" {
		return emitJSON(rows)
	}
	if len(rows) == 0 {
		if emptyMsg != "" {
			fmt.Println(emptyMsg)
		}
		return nil
	}
	w := newTabWriter()
	table(w, rows)
	return w.Flush()
}

// emitOne centralises the single-object "json or human" dispatch used by
// get / show / create / accept / etc. verbs. In json mode it emits v;
// otherwise it calls human with a configured tabwriter and flushes.
func emitOne[T any](output string, v T, human func(w *tabwriter.Writer, v T)) error {
	if output == "json" {
		return emitJSON(v)
	}
	w := newTabWriter()
	human(w, v)
	return w.Flush()
}
