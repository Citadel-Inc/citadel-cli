package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestSelfHostTelemetry_UnknownAction(t *testing.T) {
	err := rootFor(cmd.SelfHostCmd, "telemetry", "garbage").Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown telemetry action") {
		t.Fatalf("want unknown telemetry action error, got %v", err)
	}
}

// TestSelfHostInit_BatchAllFlags runs `self-host init --batch` with all four
// required flags supplied, covering runSelfHostInit without stdin interaction.
func TestSelfHostInit_BatchAllFlags(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)

	var out strings.Builder
	err := rootForOut(cmd.SelfHostCmd, &out,
		"init", "--batch",
		"--api-endpoint", "https://citadel.example.com",
		"--supabase-url", "https://abc.supabase.co",
		"--admin-key", "service-role-key",
		"--jwt-secret", "jwt-signing-secret",
	).Execute()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out.String(), "Self-host config written to") {
		t.Fatalf("init: unexpected output %q", out.String())
	}
}

func TestSelfHostBootstrapToken_NoJWTSecret(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	// No config file → zero Config → JWTSecret empty.
	err := rootFor(cmd.SelfHostCmd, "bootstrap-token").Execute()
	if err == nil || !strings.Contains(err.Error(), "jwt_secret") {
		t.Fatalf("want jwt_secret error, got %v", err)
	}
}

func TestSelfHostBootstrapToken_Success(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	yaml := "api_endpoint: https://citadel.example.com\nsupabase_url: https://abc.supabase.co\nadmin_key: adminkey\njwt_secret: super-secret-jwt\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := rootForOut(cmd.SelfHostCmd, &out, "bootstrap-token").Execute(); err != nil {
		t.Fatalf("bootstrap-token: %v", err)
	}
	tok := strings.TrimSpace(out.String())
	if !strings.Contains(tok, ".") {
		t.Fatalf("expected JWT (contains .), got %q", tok)
	}
}

func TestSelfHostBootstrapToken_BadDuration(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	yaml := "jwt_secret: super-secret-jwt\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rootFor(cmd.SelfHostCmd, "bootstrap-token", "--duration", "notvalid").Execute()
	if err == nil {
		t.Fatal("expected parse error for bad --duration")
	}
}

func TestSelfHostBootstrapToken_NegativeDuration(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	yaml := "jwt_secret: super-secret-jwt\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rootFor(cmd.SelfHostCmd, "bootstrap-token", "--duration", "-1h").Execute()
	if err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("want positive-duration error, got %v", err)
	}
}

func TestSelfHostMigrate_NoConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	err := rootFor(cmd.SelfHostCmd, "migrate").Execute()
	if err == nil || !strings.Contains(err.Error(), "self-host config incomplete") {
		t.Fatalf("want config-incomplete error, got %v", err)
	}
}

func TestSelfHostHealth_NoConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	err := rootFor(cmd.SelfHostCmd, "health").Execute()
	if err == nil || !strings.Contains(err.Error(), "self-host config incomplete") {
		t.Fatalf("want config-incomplete error, got %v", err)
	}
}

func TestSelfHostTelemetry_EnableDisable(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)

	var out strings.Builder

	if err := rootForOut(cmd.SelfHostCmd, &out, "telemetry", "enable").Execute(); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !strings.Contains(out.String(), "Telemetry enabled") {
		t.Fatalf("enable: unexpected output %q", out.String())
	}

	out.Reset()
	if err := rootForOut(cmd.SelfHostCmd, &out, "telemetry", "disable").Execute(); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if !strings.Contains(out.String(), "Telemetry disabled") {
		t.Fatalf("disable: unexpected output %q", out.String())
	}
}
