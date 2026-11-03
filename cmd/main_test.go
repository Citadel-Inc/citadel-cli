package cmd_test

import (
	"os"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

// Cobra commands are package globals wired lazily by NewRootCmd; tests that
// drive a verb directly need the flags and subcommands registered first.
func TestMain(m *testing.M) {
	cmd.NewRootCmd()
	os.Exit(m.Run())
}
