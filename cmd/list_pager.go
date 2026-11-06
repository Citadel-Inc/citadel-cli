package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listPager renders the successive pages of one cursor-paginated list verb in
// the selected --output format, so each verb supplies only its table layout.
type listPager[T CSVRow] struct {
	cmd    *cobra.Command
	output string
	all    bool
	// empty prints the human message for a first page with no rows.
	empty func(w io.Writer)
	table func(w *tabwriter.Writer, rows []T)

	first     bool
	csvHdr    bool
	yamlAccum []T
}

func newListPager[T CSVRow](cmd *cobra.Command, output string, all bool, empty func(w io.Writer), table func(w *tabwriter.Writer, rows []T)) *listPager[T] {
	return &listPager[T]{cmd: cmd, output: output, all: all, empty: empty, table: table, first: true}
}

// page emits one page. When stop is true the caller returns err; otherwise it
// continues with cursor = next.
func (p *listPager[T]) page(rows []T, cursor, next string) (stop bool, err error) {
	if len(rows) == 0 && cursor != "" && next == "" {
		return true, p.finish()
	}
	if p.first && len(rows) == 0 && cursor == "" {
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
		return emitCSVHeaderOnly[T](p.cmd)
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
		return false, emitCSVRows(p.cmd, &p.csvHdr, rows)
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
