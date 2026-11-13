package cmd_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/cmd"
	"github.com/Citadel-Inc/citadel-cli/internal/selfhost"
)

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

// runSelfHostInitWith drives `self-host init --batch` with the given stdin and
// returns stderr, so secret-source behaviour can be asserted.
func runSelfHostInitWith(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "self-host.yaml")
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	var stderr strings.Builder
	base := []string{
		"init", "--batch",
		"--api-endpoint", "https://citadel.example.com",
		"--supabase-url", "https://abc.supabase.co",
	}
	root := rootForOut(cmd.SelfHostCmd, io.Discard, append(base, args...)...)
	for _, c := range cmd.SelfHostCmd.Commands() {
		c.SetErr(&stderr)
		c.SetIn(strings.NewReader(stdin))
	}
	err := root.Execute()
	if err == nil {
		cfg, lerr := selfhost.Load()
		if lerr != nil {
			t.Fatal(lerr)
		}
		if cfg.AdminKey != "service-role-key" || cfg.JWTSecret != "jwt-signing-secret" {
			t.Fatalf("secrets not saved: %+v", cfg)
		}
	}
	return stderr.String(), err
}

func TestSelfHostInit_BatchSecretsFromEnv(t *testing.T) {
	t.Setenv("CITADEL_SELF_HOST_ADMIN_KEY", "service-role-key")
	t.Setenv("CITADEL_SELF_HOST_JWT_SECRET", "jwt-signing-secret")
	stderr, err := runSelfHostInitWith(t, "")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Fatalf("env secrets must not warn: %q", stderr)
	}
}

func TestSelfHostInit_BatchSecretFromStdin(t *testing.T) {
	t.Setenv("CITADEL_SELF_HOST_JWT_SECRET", "jwt-signing-secret")
	if _, err := runSelfHostInitWith(t, "service-role-key\n", "--admin-key-stdin"); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func TestSelfHostInit_ArgvSecretWarns(t *testing.T) {
	stderr, err := runSelfHostInitWith(t, "", "--admin-key", "service-role-key", "--jwt-secret", "jwt-signing-secret")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, flag := range []string{"--admin-key", "--jwt-secret"} {
		if !strings.Contains(stderr, "warning: "+flag) {
			t.Fatalf("want argv warning for %s, got %q", flag, stderr)
		}
	}
}

func TestSelfHostInit_BothSecretsFromStdinRejected(t *testing.T) {
	if _, err := runSelfHostInitWith(t, "x\n", "--admin-key-stdin", "--jwt-secret-stdin"); err == nil {
		t.Fatal("want error: stdin can carry only one secret")
	}
}
