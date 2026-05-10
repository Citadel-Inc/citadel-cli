// Package selfhost manages the local configuration for self-hosted Citadel
// deployments.  The config is stored as YAML at ~/.citadel/self-host.yaml
// (or the path specified by CITADEL_SELF_HOST_CONFIG).
package selfhost

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config holds the operator-supplied parameters for a self-hosted Citadel
// instance.
type Config struct {
	// APIEndpoint is the base URL of the Citadel API server
	// (e.g. "https://citadel.example.com").
	APIEndpoint string `yaml:"api_endpoint"`

	// SupabaseURL is the base URL of the Supabase project
	// (e.g. "https://abc.supabase.co" or a self-hosted Supabase URL).
	SupabaseURL string `yaml:"supabase_url"`

	// AdminKey is the Supabase service-role key (secret — never log).
	AdminKey string `yaml:"admin_key"`

	// JWTSecret is the Supabase JWT signing secret used to mint admin tokens.
	// Required for the bootstrap-token verb.
	JWTSecret string `yaml:"jwt_secret"`

	// Telemetry, when true, allows anonymous usage data to be sent to
	// Rethunk-Tech telemetry endpoints.  Disabled by default; enable
	// explicitly via `citadel self-host telemetry enable`.
	Telemetry bool `yaml:"telemetry"`
}

// ConfigPath returns the effective path of the self-host config file.
//
// Precedence:
//  1. CITADEL_SELF_HOST_CONFIG env var
//  2. ~/.citadel/self-host.yaml
func ConfigPath() (string, error) {
	if env := os.Getenv("CITADEL_SELF_HOST_CONFIG"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("self-host config: resolve home dir: %w", err)
	}
	return filepath.Join(home, ".citadel", "self-host.yaml"), nil
}

// Load reads the self-host config from disk.  Returns a zero-value Config
// (no error) when the file does not yet exist — callers treat that as a
// fresh install.
func Load() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read self-host config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse self-host config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config to disk with mode 0600, creating the parent
// directory if necessary.  Writes atomically via a temp-file rename.
func (c Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir for self-host config: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal self-host config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write self-host config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install self-host config: %w", err)
	}
	return os.Chmod(path, 0600)
}

// Validate returns an error if required fields are missing.
func (c Config) Validate() error {
	if c.APIEndpoint == "" {
		return errors.New("api_endpoint is required")
	}
	if c.SupabaseURL == "" {
		return errors.New("supabase_url is required")
	}
	if c.AdminKey == "" {
		return errors.New("admin_key is required")
	}
	return nil
}

// IsTelemetryEnabled returns true only when the operator has explicitly opted in.
func (c Config) IsTelemetryEnabled() bool {
	return c.Telemetry
}

// redactedAdminKey returns a safely redacted representation of the admin key
// for use in debug output (never the raw secret).
func (c Config) redactedAdminKey() string {
	if len(c.AdminKey) <= 8 {
		return "***"
	}
	return c.AdminKey[:4] + "..." + c.AdminKey[len(c.AdminKey)-4:]
}

// redactedJWTSecret returns a safely redacted JWT secret for debug output.
func (c Config) redactedJWTSecret() string {
	if c.JWTSecret == "" {
		return "(not set)"
	}
	return "***"
}

// DebugSummary returns a human-readable summary with secrets redacted.
func (c Config) DebugSummary() string {
	return fmt.Sprintf(
		"api_endpoint=%s supabase_url=%s admin_key=%s jwt_secret=%s telemetry=%v",
		c.APIEndpoint, c.SupabaseURL,
		c.redactedAdminKey(), c.redactedJWTSecret(), c.Telemetry,
	)
}
