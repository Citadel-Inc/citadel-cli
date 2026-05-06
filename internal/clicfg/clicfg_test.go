package clicfg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	cfg := Config{
		ServerURL:    "https://api.src.land",
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresAt:    time.Unix(1234567890, 0).UTC(),
		UserUUID:     "test-user-uuid-1234",
		AgentID:      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		AgentName:    "citadel-cli@test",
	}

	// Save the config
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file permissions are 0600
	path := filepath.Join(tmpDir, "citadel", "config.toml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file has wrong permissions: got %o, want 0600", info.Mode().Perm())
	}

	// Load the config back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify all fields were preserved
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL mismatch: got %q, want %q", loaded.ServerURL, cfg.ServerURL)
	}
	if loaded.AccessToken != cfg.AccessToken {
		t.Errorf("AccessToken mismatch: got %q, want %q", loaded.AccessToken, cfg.AccessToken)
	}
	if loaded.RefreshToken != cfg.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %q, want %q", loaded.RefreshToken, cfg.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(cfg.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", loaded.ExpiresAt, cfg.ExpiresAt)
	}
	if loaded.UserUUID != cfg.UserUUID {
		t.Errorf("UserUUID mismatch: got %q, want %q", loaded.UserUUID, cfg.UserUUID)
	}
	if loaded.AgentID != cfg.AgentID {
		t.Errorf("AgentID mismatch: got %q, want %q", loaded.AgentID, cfg.AgentID)
	}
	if loaded.AgentName != cfg.AgentName {
		t.Errorf("AgentName mismatch: got %q, want %q", loaded.AgentName, cfg.AgentName)
	}
}

func TestLoadMissingFile(t *testing.T) {
	// Create a temporary config directory that doesn't have the file
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Load from non-existent file should return zero value + nil error
	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() should not error on missing file, got: %v", err)
	}

	// Verify it's a zero-value config
	if cfg.ServerURL != "" || cfg.AccessToken != "" || cfg.UserUUID != "" {
		t.Error("Expected zero-value config for missing file")
	}
}

func TestSavePermissions(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	cfg := Config{
		ServerURL:   "https://example.com",
		AccessToken: "token123",
	}

	// Save and verify permissions
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	path := filepath.Join(tmpDir, "citadel", "config.toml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file has wrong permissions: got %o, want 0600", info.Mode().Perm())
	}
}

func TestResolveServerURL(t *testing.T) {
	// Precedence: flag → env → cfg → default
	t.Setenv("CITADEL_SERVER", "")
	c := Config{ServerURL: "https://stored"}
	if got := c.ResolveServerURL(""); got != "https://stored" {
		t.Errorf("stored cfg, got %q", got)
	}
	t.Setenv("CITADEL_SERVER", "https://env")
	if got := c.ResolveServerURL(""); got != "https://env" {
		t.Errorf("env override, got %q", got)
	}
	if got := c.ResolveServerURL("https://flag"); got != "https://flag" {
		t.Errorf("flag override, got %q", got)
	}
	t.Setenv("CITADEL_SERVER", "")
	if got := (Config{}).ResolveServerURL(""); got != "https://api.src.land" {
		t.Errorf("default, got %q", got)
	}
}

func TestLoad_AccessTokenEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("CITADEL_ACCESS_TOKEN", "env-token")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccessToken != "env-token" {
		t.Fatalf("got %q", cfg.AccessToken)
	}
	if time.Until(cfg.ExpiresAt) < 30*time.Minute || time.Until(cfg.ExpiresAt) > 90*time.Minute {
		t.Fatalf("env token expiry should pin ~1h, got %v", time.Until(cfg.ExpiresAt))
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if oldXDG != "" {
			t.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	cfg := Config{
		ServerURL: "https://example.com",
	}

	// Save to a path whose directory doesn't exist yet
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify the directory was created
	path := filepath.Join(tmpDir, "citadel", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Config file was not created: %v", err)
	}
}

func TestSave_OverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	first := Config{ServerURL: "https://first.example", AccessToken: "a"}
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	second := Config{ServerURL: "https://second.example", AccessToken: "b"}
	if err := second.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ServerURL != "https://second.example" || got.AccessToken != "b" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	path := filepath.Join(tmpDir, "citadel", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("this is not valid TOML [[[\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("Load: expected error for invalid TOML")
	}
}

// TestLoad_UsesHomeWhenXDGUnset exercises configPath's branch that joins
// ~/.config/citadel when XDG_CONFIG_HOME is empty.
func TestLoad_UsesHomeWhenXDGUnset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	dir := filepath.Join(tmpDir, ".config", "citadel")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	content := `
server_url = "https://home-path.example"
access_token = "from-file"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ServerURL != "https://home-path.example" || cfg.AccessToken != "from-file" {
		t.Fatalf("got %+v", cfg)
	}
}
