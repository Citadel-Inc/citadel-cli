package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type pagerTableRow struct {
	path string
	vis  string
}

func TestListPagerAllTablePrintsHeaderOnce(t *testing.T) {
	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)
	p := newListPagerCSV(c, "table", true,
		func(io.Writer) {},
		func(w *tabwriter.Writer, rows []pagerTableRow) {
			fmt.Fprintln(w, "PATH\tVISIBILITY")
			for _, r := range rows {
				fmt.Fprintf(w, "%s\t%s\n", r.path, r.vis)
			}
		},
		func(*cobra.Command, *bool, []pagerTableRow) error { return nil },
		func(*cobra.Command) error { return nil },
	)

	stop, err := p.page([]pagerTableRow{{path: "ns/r1", vis: "private"}}, "", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if stop {
		t.Fatal("expected --all to continue after first page")
	}
	stop, err = p.page([]pagerTableRow{{path: "ns/r2", vis: "public"}}, "c2", "")
	if err != nil {
		t.Fatal(err)
	}
	if !stop {
		t.Fatal("expected --all to stop after last page")
	}

	out := buf.String()
	if n := strings.Count(out, "PATH"); n != 1 {
		t.Fatalf("header appeared %d times, want 1\n%s", n, out)
	}
	if !strings.Contains(out, "ns/r1") || !strings.Contains(out, "ns/r2") {
		t.Fatalf("missing rows:\n%s", out)
	}
}
