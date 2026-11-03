package clicfg

import (
	"cmp"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/Rethunk-Tech/go-safefs"
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

// ConfigPath returns the effective on-disk CLI configuration path.
func ConfigPath() (string, error) {
	return configPath()
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
// Refresh tokens are persisted by `auth login` and exchanged by the command
// client when an access-token request receives a 401.
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
		//nolint:gosec // The config path is the operator-selected local config root.
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		if err := toml.Unmarshal(data, &cfg); err != nil {
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	return safefs.WriteStreamAtomic(path, 0o600, func(w io.Writer) error {
		//nolint:gosec // Config serialization intentionally persists the token in the protected config file.
		return toml.NewEncoder(w).Encode(c)
	})
}

// Update serializes a load-mutate-save cycle across CLI processes.
func Update(fn func(*Config) error) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	//nolint:gosec // The lock path is derived from the protected config path.
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	unlock, err := lockExclusive(lock)
	if err != nil {
		return err
	}
	defer unlock()

	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := fn(&cfg); err != nil {
		return err
	}
	return cfg.Save()
}

// ResolveRawServerURL picks the configured server URL exactly as written, with
// this precedence:
//
//  1. --server flag (passed as flagOverride; empty string = unset)
//  2. CITADEL_SERVER env var
//  3. stored config (c.ServerURL)
//  4. default https://mcp.src.land
//
// A trailing /api is meaningful here: REST callers read it to learn that the
// server fronts its API under /api on the same origin.
func (c Config) ResolveRawServerURL(flagOverride string) string {
	return cmp.Or(flagOverride, os.Getenv("CITADEL_SERVER"), c.ServerURL, "https://mcp.src.land")
}

// DescribeServerSource names where ResolveRawServerURL got its answer, so a
// self-hosted user who forgot to configure a server sees it is the hosted one.
func (c Config) DescribeServerSource(flagOverride string) string {
	switch {
	case flagOverride != "":
		return "from --server"
	case os.Getenv("CITADEL_SERVER") != "":
		return "from CITADEL_SERVER"
	case c.ServerURL != "":
		return "from the config file"
	}
	return "built-in default, the hosted service; set CITADEL_SERVER or pass --server for a self-hosted one"
}

// ResolveServerURL is the bare server origin used by OAuth, MCP and git
// callers: ResolveRawServerURL without a trailing slash or /api, so one
// CITADEL_SERVER value serves REST and non-REST commands alike.
func (c Config) ResolveServerURL(flagOverride string) string {
	return StripAPISuffix(c.ResolveRawServerURL(flagOverride))
}

// StripAPISuffix removes trailing slashes and one trailing /api segment.
func StripAPISuffix(server string) string {
	s := strings.TrimRight(strings.TrimSpace(server), "/")
	return strings.TrimRight(strings.TrimSuffix(s, "/api"), "/")
}
