package cmd_test

import (
	"bytes"
	"net/http"
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

func TestOrgInvRevoke_EmptyID_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "revoke", "acme", " ").Execute()
	if err == nil || err.Error() != "invitation ID is required" {
		t.Fatalf("want invitation ID required error, got %v", err)
	}
}

func TestOrgInvRevoke_EmptyOrgSlug_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, "invitation", "revoke", " \t", "inv-123").Execute()
	if err == nil || err.Error() != "organization slug is required" {
		t.Fatalf("want organization slug required error, got %v", err)
	}
}

func TestOrgInvRevoke_HappyOutput(t *testing.T) {
	var stdout bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /orgs/acme/invitations/inv-123": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))

	if err := rootForOut(cmd.OrgCmd, &stdout, "invitation", "revoke", " acme ", " inv-123 ").Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Revoked invitation inv-123.\n" {
		t.Fatalf("revoke output = %q", got)
	}
}

func assertOrgInvitationBadOutput(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	err := rootFor(cmd.OrgCmd, append([]string{"invitation"}, append(args, "--output", "toml")...)...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
