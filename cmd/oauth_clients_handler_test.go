package cmd_test

import (
	"net/http"
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

func TestOAuthClientsCreate_HumanSecretOutput(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": "01234567-89ab-cdef-0123-456789abcdef", "client_id": "c1", "name": "App",
				"is_public": false, "redirect_uris": []string{"https://x"}, "client_secret": "shh",
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.OauthCmd, &stdout, "clients", "create", "--name", "App", "--redirect-uri", "https://x").Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "shh\n" {
		t.Fatalf("create secret output = %q", stdout.String())
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
	if err == nil || err.Error() != `--output for rotate supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOAuthClientsRotateSecret_HumanSecretOutput(t *testing.T) {
	const clientID = "550e8400-e29b-41d4-a716-446655440000"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients/" + clientID + "/rotate-secret": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"id": clientID, "client_secret": "new-secret"})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.OauthCmd, &stdout, "clients", "rotate-secret", clientID).Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "new-secret\n" {
		t.Fatalf("rotate secret output = %q", stdout.String())
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
	if err == nil || err.Error() != `--output for revoke supports json or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestOAuthClientsRevoke_DryRun_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	const clientID = "550e8400-e29b-41d4-a716-446655440000"
	var stdout strings.Builder
	err := rootForOut(cmd.OauthCmd, &stdout, "clients", "revoke", clientID, "--dry-run").Execute()
	if err != nil {
		t.Fatalf("oauth clients revoke dry-run: %v", err)
	}
	if stdout.String() != "Would DELETE /oauth/clients/"+clientID+" (skipped; --dry-run)\n" {
		t.Fatalf("oauth clients revoke dry-run output = %q", stdout.String())
	}
}
