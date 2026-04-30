package clicfg

import (
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

// Load reads the config from disk. If the file does not exist,
// returns a zero-value Config with nil error (fresh install state).
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	// File doesn't exist yet — return zero value and nil error
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	// File exists — parse it
	var cfg Config
	_, err = toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, err
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
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		os.Remove(tmpFile)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Ensure final file has correct permissions (paranoia check)
	return os.Chmod(path, 0600)
}

// ResolveServerURL picks the effective server URL with this precedence:
//
//	1. --server flag (passed as flagOverride; empty string = unset)
//	2. CITADEL_SERVER env var
//	3. stored config (c.ServerURL)
//	4. default https://api.src.land
//
// Used by every subcommand that issues HTTP requests so the precedence
// is consistent.
func (c Config) ResolveServerURL(flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if env := os.Getenv("CITADEL_SERVER"); env != "" {
		return env
	}
	if c.ServerURL != "" {
		return c.ServerURL
	}
	return "https://api.src.land"
}
