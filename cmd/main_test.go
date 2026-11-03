package cmd_test

import (
	"os"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/cmd"
	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
)

// Cobra commands are package globals wired lazily by NewRootCmd; tests that
// drive a verb directly need the flags and subcommands registered first.
func TestMain(m *testing.M) {
	apiclient.ProbeLoopback = false
	cmd.NewRootCmd()
	os.Exit(m.Run())
}
