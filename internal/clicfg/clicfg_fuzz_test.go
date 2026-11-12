package clicfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func FuzzStripAPISuffix(f *testing.F) {
	f.Add("https://mcp.src.land")
	f.Add("https://mcp.src.land/api")
	f.Add("https://mcp.src.land/api/")
	f.Add("  https://mcp.src.land/api  ")
	f.Add("")
	f.Add("/")
	f.Add("/api")
	f.Fuzz(func(t *testing.T, server string) {
		got := StripAPISuffix(server)
		want := strings.TrimRight(strings.TrimSpace(server), "/")
		want = strings.TrimRight(strings.TrimSuffix(want, "/api"), "/")
		if got != want {
			t.Fatalf("StripAPISuffix(%q) = %q, want %q", server, got, want)
		}
		if got != "" && strings.HasSuffix(got, "/") {
			t.Fatalf("StripAPISuffix(%q) = %q has a trailing slash", server, got)
		}
	})
}

func FuzzLoad(f *testing.F) {
	f.Add("server_url = \"https://mcp.src.land\"\n")
	f.Add("server_url = \"https://mcp.src.land/api\"\naccess_token = \"tok\"\nrefresh_token = \"ref\"\n")
	f.Add("server_url = \"https://example.com\"\nuser_uuid = \"u1\"\nagent_id = \"a1\"\nagent_name = \"n1\"\n")
	f.Add("")
	f.Add("not toml {")
	f.Fuzz(func(t *testing.T, data string) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		t.Setenv("CITADEL_ACCESS_TOKEN", "")
		t.Setenv("CITADEL_SERVER", "")
		cfgDir := filepath.Join(dir, "citadel")
		if err := os.MkdirAll(cfgDir, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(cfgDir, "config.toml")
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}

		var parsed Config
		uerr := toml.Unmarshal([]byte(data), &parsed)
		if uerr == nil && parsed.TokenStore == "keyring" {
			return
		}

		cfg, err := Load()
		if uerr != nil {
			if err == nil {
				t.Fatal("Load succeeded on TOML that failed to unmarshal into Config")
			}
			return
		}
		if err != nil {
			t.Fatalf("Load failed on valid TOML: %v", err)
		}
		if cfg.ServerURL != parsed.ServerURL ||
			cfg.AccessToken != parsed.AccessToken ||
			cfg.RefreshToken != parsed.RefreshToken ||
			cfg.UserUUID != parsed.UserUUID ||
			cfg.AgentID != parsed.AgentID ||
			cfg.AgentName != parsed.AgentName ||
			cfg.TokenStore != parsed.TokenStore ||
			cfg.TokenStoreReason != parsed.TokenStoreReason {
			t.Fatalf("Load disagreed with Unmarshal: got %+v want %+v", cfg, parsed)
		}
		if !cfg.ExpiresAt.Equal(parsed.ExpiresAt) {
			t.Fatalf("Load ExpiresAt %v != Unmarshal %v", cfg.ExpiresAt, parsed.ExpiresAt)
		}
	})
}
