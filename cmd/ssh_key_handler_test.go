package cmd_test

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestSSHKeyList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.SSHKeyCmd, "list", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
