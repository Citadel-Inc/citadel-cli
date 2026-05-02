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
