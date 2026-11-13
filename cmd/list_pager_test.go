package cmd

import (
	"io"
	"testing"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func TestListPagerStopsOnRepeatedCursor(t *testing.T) {
	c := &cobra.Command{}
	c.SetOut(io.Discard)
	p := newListPagerCSV(c, "ndjson", true, func(io.Writer) {}, func(*tabwriter.Writer, []int) {}, nil, nil)
	stop, err := p.page([]int{1}, "same", "same")
	if !stop || err == nil {
		t.Fatalf("page = %v, %v; want stop with an error", stop, err)
	}
}
