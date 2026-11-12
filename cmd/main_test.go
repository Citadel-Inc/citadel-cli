package cmd_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Citadel-Inc/citadel-cli/cmd"
	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
)

// cliBinPath is built once here: several tests launch the real CLI, and a build
// per test cost more than the behaviour they check.
var (
	cliBinPath string
	cliBinErr  error
)

// Cobra commands are package globals wired lazily by NewRootCmd; tests that
// drive a verb directly need the flags and subcommands registered first.
func TestMain(m *testing.M) {
	apiclient.ProbeLoopback = false
	cmd.NewRootCmd()
	os.Exit(runWithCLI(m))
}

func runWithCLI(m *testing.M) int {
	dir, err := os.MkdirTemp("", "citadel-cli-test-")
	if err != nil {
		cliBinErr = err
		return m.Run()
	}
	defer func() { _ = os.RemoveAll(dir) }()
	cliBinPath = filepath.Join(dir, "citadel-cli")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", cliBinPath, "..") //nolint:gosec // fixed go verb, test-owned path
	if b, err := build.CombinedOutput(); err != nil {
		cliBinErr = fmt.Errorf("go build: %w %s", err, b)
	}
	return m.Run()
}
