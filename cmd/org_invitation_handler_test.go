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

func TestOrgInvCreate_MissingInvitee_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "create", " acme ", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "invitee required") {
		t.Fatalf("want invitee required error, got %v", err)
	}
}

func TestOrgInvCreate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "create", "acme", "--email", "invitee@example.com", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output for create supports json or default human summary only") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOrgInvAccept_MissingToken_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "accept", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "invitation token required") {
		t.Fatalf("want invitation token required error, got %v", err)
	}
}

func TestOrgInvAccept_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "accept", "token", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output for accept supports json or default human summary only") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func assertOrgInvitationBadOutput(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OrgCmd, append([]string{"invitation"}, append(args, "--output", "toml")...)...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
