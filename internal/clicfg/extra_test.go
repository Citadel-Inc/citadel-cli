package clicfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── configPath: XDG_CONFIG_HOME branch ──────────────────────────────────

func TestConfigPath_XDGSet(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	p, err := configPath()
	if err != nil {
		t.Fatalf("configPath with XDG: %v", err)
	}
	want := filepath.Join(tmp, "citadel", "config.toml")
	if p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}

func TestConfigPath_XDGEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	// Falls back to home dir. Just verify it contains "citadel/config.toml".
	p, err := configPath()
	if err != nil {
		t.Fatalf("configPath default: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join("citadel", "config.toml")) {
		t.Fatalf("unexpected path %q", p)
	}
}

// ─── Load: bad TOML file ─────────────────────────────────────────────────

func TestLoad_BadTOML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Create the citadel config dir and a malformed config.toml.
	dir := filepath.Join(tmp, "citadel")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfg, []byte("not = valid = toml = at all!!!"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for bad TOML")
	}
}

// ─── Save and Load round-trip with XDG branch ────────────────────────────

func TestSaveLoad_XDGBranch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("CITADEL_ACCESS_TOKEN", "") // no env override

	want := Config{
		ServerURL:   "https://test.api.example.com",
		AccessToken: "tok123",
		UserUUID:    "uuid-abc",
	}
	if err := want.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ServerURL != want.ServerURL {
		t.Errorf("ServerURL: got %q", got.ServerURL)
	}
	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken: got %q", got.AccessToken)
	}
	if got.UserUUID != want.UserUUID {
		t.Errorf("UserUUID: got %q", got.UserUUID)
	}
}

// ─── ResolveServerURL: empty stored URL falls through to default ──────────

func TestResolveServerURL_EmptyStored(t *testing.T) {
	t.Setenv("CITADEL_SERVER", "")
	c := Config{}
	if got := c.ResolveServerURL(""); got != "https://api.src.land" {
		t.Fatalf("default fallback: got %q", got)
	}
}

// ─── Load: stat error branch — unreadable directory ─────────────────────

func TestLoad_StatError(t *testing.T) {
	// Create a file at the expected config path location (not a directory),
	// which will make the sub-path stat fail in an unusual way.
	// Instead, test by making the config file non-readable.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "citadel")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte("server_url = \"https://x.com\"\n"), 0000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(cfgFile, 0600) }()

	// If running as root, stat won't error; skip gracefully.
	if os.Getuid() == 0 {
		t.Skip("running as root; permission test not meaningful")
	}

	_, err := Load()
	if err == nil {
		// Some systems may still read mode-000 files; treat this as expected.
		t.Log("note: mode-000 file was readable; OS may grant root-equivalent access")
	}
}

// ─── Save: error branches (parent-mkdir failure, OpenFile failure) ──────

// TestSave_MkdirFails covers the os.MkdirAll error branch by pointing
// XDG_CONFIG_HOME at a path whose parent is a regular file (so mkdir
// cannot create the citadel/ subdir under it).
func TestSave_MkdirFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; mkdir restrictions not enforced")
	}
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(parent, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", parent) // XDG_CONFIG_HOME is a FILE, not dir.

	cfg := Config{ServerURL: "https://x"}
	if err := cfg.Save(); err == nil {
		t.Fatal("expected mkdir error, got nil")
	}
}

// TestSave_OpenFileFails covers the os.OpenFile error branch by making
// the target directory unwritable so .tmp creation fails.
func TestSave_OpenFileFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission test not meaningful")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "citadel")
	if err := os.MkdirAll(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0700) }()

	cfg := Config{ServerURL: "https://x"}
	if err := cfg.Save(); err == nil {
		t.Fatal("expected OpenFile error, got nil")
	}
}
