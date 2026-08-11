package cmd_test

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestOAuthClientsList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOAuthClientsShow_BadUUID_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "show", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "id must be a UUID") {
		t.Fatalf("want UUID validation error, got %v", err)
	}
}

func TestOAuthClientsRotateSecret_BadUUID_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "rotate-secret", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "id must be a UUID") {
		t.Fatalf("want UUID validation error, got %v", err)
	}
}

func TestOAuthClientsRevoke_BadUUID_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "revoke", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "id must be a UUID") {
		t.Fatalf("want UUID validation error, got %v", err)
	}
}
