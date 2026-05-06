package clicfg

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the CLI configuration state: server URL, auth tokens, and user info.
type Config struct {
	ServerURL    string    `toml:"server_url"`
	AccessToken  string    `toml:"access_token"`
	RefreshToken string    `toml:"refresh_token"`
	ExpiresAt    time.Time `toml:"expires_at"`
	UserUUID     string    `toml:"user_uuid"`
	AgentID      string    `toml:"agent_id,omitempty"`
	AgentName    string    `toml:"agent_name,omitempty"`
}

// configPath returns the full path to the config file.
// Respects XDG_CONFIG_HOME if set, else defaults to ~/.config/citadel/config.toml.
func configPath() (string, error) {
	var configDir string
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		configDir = filepath.Join(xdgHome, "citadel")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config", "citadel")
	}

	return filepath.Join(configDir, "config.toml"), nil
}

// Load reads the config from disk and applies env-var overrides for
// non-interactive automation. If the file does not exist, returns a
// zero-value Config (fresh install state).
//
// Env overrides (apply on top of the file):
//
//   - CITADEL_ACCESS_TOKEN — replaces stored access_token; pinned to a
//     1-hour expiry from now. Useful for CI / scripts that mint a JWT
//     externally and do not want to write to ~/.config/citadel.
//   - CITADEL_SERVER — already honored by ResolveServerURL at request
//     time; not duplicated here.
//
// The refresh-token flow is NOT yet wired (see specs/HUMAN_BLOCKERS.md
// CLI auth gaps): refresh_token is stored on `auth login` but never
// exchanged for a new access_token, so a CLI session past the 1-hour
// JWT expiry currently requires `auth login` again or a fresh env-var
// JWT.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	_, statErr := os.Stat(path)
	switch {
	case errors.Is(statErr, os.ErrNotExist):
		// Fresh install — leave cfg zero-valued; env overrides may still apply.
	case statErr != nil:
		return Config{}, statErr
	default:
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	if env := os.Getenv("CITADEL_ACCESS_TOKEN"); env != "" {
		cfg.AccessToken = env
		// Trust the env value for an hour; the JWT itself is what gets
		// validated server-side, so a wrong expiry only affects the
		// client's own EXPIRED check.
		cfg.ExpiresAt = time.Now().Add(time.Hour)
	}

	return cfg, nil
}

// Save writes the config to disk with mode 0600.
// Creates the config directory if it doesn't exist.
func (c Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Write to a temporary file first, then atomic rename
	tmpFile := path + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	enc := toml.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}

	// Ensure final file has correct permissions (paranoia check)
	return os.Chmod(path, 0600)
}

// ResolveServerURL picks the effective server URL with this precedence:
//
//  1. --server flag (passed as flagOverride; empty string = unset)
//  2. CITADEL_SERVER env var
//  3. stored config (c.ServerURL)
//  4. default https://api.src.land
//
// Used by every subcommand that issues HTTP requests so the precedence
// is consistent.
func (c Config) ResolveServerURL(flagOverride string) string {
	return cmp.Or(flagOverride, os.Getenv("CITADEL_SERVER"), c.ServerURL, "https://api.src.land")
}
