package selfhost_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/selfhost"
)

func TestConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)

	original := selfhost.Config{
		APIEndpoint: "https://citadel.example.com",
		SupabaseURL: "https://abc.supabase.co",
		AdminKey:    "service_role_key_here",
		JWTSecret:   "super-secret-jwt-value",
		Telemetry:   false,
	}
	if err := original.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File must be mode 0600.
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("file mode = %#o; want 0600", got)
	}

	loaded, err := selfhost.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.APIEndpoint != original.APIEndpoint {
		t.Errorf("APIEndpoint: got %q, want %q", loaded.APIEndpoint, original.APIEndpoint)
	}
	if loaded.SupabaseURL != original.SupabaseURL {
		t.Errorf("SupabaseURL: got %q, want %q", loaded.SupabaseURL, original.SupabaseURL)
	}
	if loaded.AdminKey != original.AdminKey {
		t.Errorf("AdminKey: got %q, want %q", loaded.AdminKey, original.AdminKey)
	}
	if loaded.JWTSecret != original.JWTSecret {
		t.Errorf("JWTSecret: got %q, want %q", loaded.JWTSecret, original.JWTSecret)
	}
	if loaded.Telemetry != original.Telemetry {
		t.Errorf("Telemetry: got %v, want %v", loaded.Telemetry, original.Telemetry)
	}
}

func TestConfigLoadMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CITADEL_SELF_HOST_CONFIG", filepath.Join(dir, "nonexistent.yaml"))

	cfg, err := selfhost.Load()
	if err != nil {
		t.Fatalf("Load of missing file should return zero Config, not error: %v", err)
	}
	if cfg.APIEndpoint != "" || cfg.SupabaseURL != "" {
		t.Errorf("expected zero Config for missing file, got %+v", cfg)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     selfhost.Config
		wantErr bool
	}{
		{
			name: "valid",
			cfg: selfhost.Config{
				APIEndpoint: "https://citadel.example.com",
				SupabaseURL: "https://abc.supabase.co",
				AdminKey:    "key",
			},
			wantErr: false,
		},
		{
			name:    "missing api_endpoint",
			cfg:     selfhost.Config{SupabaseURL: "https://abc.supabase.co", AdminKey: "key"},
			wantErr: true,
		},
		{
			name:    "missing supabase_url",
			cfg:     selfhost.Config{APIEndpoint: "https://citadel.example.com", AdminKey: "key"},
			wantErr: true,
		},
		{
			name:    "missing admin_key",
			cfg:     selfhost.Config{APIEndpoint: "https://citadel.example.com", SupabaseURL: "https://abc.supabase.co"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigTelemetryToggle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CITADEL_SELF_HOST_CONFIG", filepath.Join(dir, "self-host.yaml"))

	cfg := selfhost.Config{
		APIEndpoint: "https://citadel.example.com",
		SupabaseURL: "https://abc.supabase.co",
		AdminKey:    "key",
		Telemetry:   false,
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if cfg.IsTelemetryEnabled() {
		t.Error("expected telemetry disabled by default")
	}

	cfg.Telemetry = true
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save after enable: %v", err)
	}
	reloaded, err := selfhost.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reloaded.IsTelemetryEnabled() {
		t.Error("expected telemetry enabled after toggle")
	}
}

func TestConfigEnvOverride(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom-self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", customPath)

	path, err := selfhost.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	if path != customPath {
		t.Errorf("ConfigPath with env = %q; want %q", path, customPath)
	}
}
