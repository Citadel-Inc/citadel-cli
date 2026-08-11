package cmd_test

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestOrgInvitationPending_BadOutput_Hermetic(t *testing.T) {
	assertOrgInvitationBadOutput(t, "pending")
}

func TestOrgInvitationList_BadOutput_Hermetic(t *testing.T) {
	assertOrgInvitationBadOutput(t, "list", "acme")
}

func assertOrgInvitationBadOutput(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OrgCmd, append([]string{"invitation"}, append(args, "--output", "toml")...)...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
