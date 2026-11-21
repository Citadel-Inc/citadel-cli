package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// The server issues cursors in several shapes (issues use a number cursor);
// the CLI must forward any of them untouched.
func TestReadPagination_cursorIsOpaque(t *testing.T) {
	c := &cobra.Command{}
	addPaginationFlags(c)
	if err := c.Flags().Set("cursor", " bnVtYmVyLWN1cnNvcg "); err != nil {
		t.Fatal(err)
	}
	_, cur, _, err := readPagination(c)
	if err != nil || cur != "bnVtYmVyLWN1cnNvcg" {
		t.Fatalf("cursor = %q, err = %v", cur, err)
	}
}
