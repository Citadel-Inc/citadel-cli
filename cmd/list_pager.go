package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listPager renders the successive pages of one cursor-paginated list verb in
// the selected --output format, so each verb supplies only its table layout.
type listPager[T any] struct {
	cmd    *cobra.Command
	output string
	all    bool
	// empty prints the human message for a first page with no rows.
	empty func(w io.Writer)
	table func(w *tabwriter.Writer, rows []T)
	// csv and csvEmpty differ from the row type's own CSVRow only when a verb
	// exports a flatter shape than it emits as JSON.
	csv      func(cmd *cobra.Command, headerWritten *bool, rows []T) error
	csvEmpty func(cmd *cobra.Command) error

	first     bool
	csvHdr    bool
	yamlAccum []T
}

func newListPager[T CSVRow](cmd *cobra.Command, output string, all bool, empty func(w io.Writer), table func(w *tabwriter.Writer, rows []T)) *listPager[T] {
	return newListPagerCSV(cmd, output, all, empty, table, emitCSVRows[T], emitCSVHeaderOnly[T])
}

func newListPagerCSV[T any](cmd *cobra.Command, output string, all bool, empty func(w io.Writer), table func(w *tabwriter.Writer, rows []T), csv func(*cobra.Command, *bool, []T) error, csvEmpty func(*cobra.Command) error) *listPager[T] {
	return &listPager[T]{cmd: cmd, output: output, all: all, empty: empty, table: table, csv: csv, csvEmpty: csvEmpty, first: true}
}

// page emits one page. When stop is true the caller returns err; otherwise it
// continues with cursor = next.
func (p *listPager[T]) page(rows []T, cursor, next string) (stop bool, err error) {
	return p.pageRaw(len(rows) == 0, rows, cursor, next)
}

// pageRaw is page for callers that filter rows client-side: emptiness is
// judged on the server's page (serverEmpty), not on the filtered rows.
func (p *listPager[T]) pageRaw(serverEmpty bool, rows []T, cursor, next string) (stop bool, err error) {
	if serverEmpty && cursor != "" && next == "" {
		return true, p.finish()
	}
	if p.first && serverEmpty && cursor == "" {
		return true, p.emitEmpty()
	}
	p.first = false

	if stop, err := p.emitRows(rows); stop || err != nil {
		return true, err
	}

	if !p.all {
		if isHumanListOutput(p.output) && next != "" {
			_, _ = fmt.Fprintln(p.cmd.OutOrStdout(), "(use --cursor "+next+" for more, or --all to fetch everything)")
		}
		return true, nil
	}
	if next == "" {
		return true, p.finish()
	}
	return false, nil
}

func (p *listPager[T]) emitEmpty() error {
	switch p.output {
	case "json":
		return emitJSON(p.cmd, []T{})
	case "ndjson":
		return nil
	case "csv":
		return p.csvEmpty(p.cmd)
	case "yaml":
		return emitYAML(p.cmd, []T{})
	default:
		p.empty(p.cmd.OutOrStdout())
		return nil
	}
}

// emitRows reports stop when the format emits the whole result in one document.
func (p *listPager[T]) emitRows(rows []T) (stop bool, err error) {
	switch p.output {
	case "json":
		return true, emitJSON(p.cmd, rows)
	case "ndjson":
		return false, emitNDJSONLines(p.cmd, rows)
	case "csv":
		return false, p.csv(p.cmd, &p.csvHdr, rows)
	case "yaml":
		if !p.all {
			return true, emitYAML(p.cmd, rows)
		}
		p.yamlAccum = append(p.yamlAccum, rows...)
		return false, nil
	default:
		w := newTabWriter(p.cmd)
		p.table(w, rows)
		return false, w.Flush()
	}
}

// finish emits the accumulated YAML document for --all.
func (p *listPager[T]) finish() error {
	if !p.all || p.output != "yaml" {
		return nil
	}
	if p.yamlAccum == nil {
		p.yamlAccum = []T{}
	}
	return emitYAML(p.cmd, p.yamlAccum)
}
