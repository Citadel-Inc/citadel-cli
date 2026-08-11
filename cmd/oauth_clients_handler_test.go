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

func TestOAuthClientsCreate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "create", "--name", "example", "--redirect-uri", "https://example.com/callback", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output for create supports json or default human summary only") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOAuthClientsCreate_EmptyName_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "create", "--name", "   ", "--redirect-uri", "https://example.com/callback").Execute()
	if err == nil || !strings.Contains(err.Error(), "--name cannot be empty") {
		t.Fatalf("want empty-name validation error, got %v", err)
	}
}

func TestOAuthClientsList_WatchJSON_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "list", "--watch", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Fatalf("want watch/output validation error, got %v", err)
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

func TestOAuthClientsRotateSecret_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "rotate-secret", "550e8400-e29b-41d4-a716-446655440000", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output for rotate supports json or default human summary only") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOAuthClientsRevoke_BadUUID_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "revoke", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "id must be a UUID") {
		t.Fatalf("want UUID validation error, got %v", err)
	}
}

func TestOAuthClientsRevoke_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.OauthCmd, "clients", "revoke", "550e8400-e29b-41d4-a716-446655440000", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output for revoke supports json or default human summary only") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
