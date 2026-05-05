package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateWatchOutput_rejectsJSONFamily(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("watch", false, "")
	c.Flags().String("output", "", "")
	if err := c.Flags().Set("watch", "true"); err != nil {
		t.Fatal(err)
	}
	for _, o := range []string{"json", "yaml", "csv"} {
		if err := c.Flags().Set("output", o); err != nil {
			t.Fatal(err)
		}
		if err := validateWatchOutput(c); err == nil {
			t.Fatalf("expected error for output=%s", o)
		}
	}
}

func TestValidateWatchOutput_acceptsNDJSONAndEmpty(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("watch", false, "")
	c.Flags().String("output", "", "")
	if err := c.Flags().Set("watch", "true"); err != nil {
		t.Fatal(err)
	}
	for _, o := range []string{"", "ndjson", "table"} {
		if err := c.Flags().Set("output", o); err != nil {
			t.Fatal(err)
		}
		if err := validateWatchOutput(c); err != nil {
			t.Fatalf("output=%q: %v", o, err)
		}
	}
}

func TestValidateWatchOutput_noWatchAlwaysOK(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("watch", false, "")
	c.Flags().String("output", "", "")
	if err := c.Flags().Set("watch", "false"); err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("output", "json"); err != nil {
		t.Fatal(err)
	}
	if err := validateWatchOutput(c); err != nil {
		t.Fatal(err)
	}
}
