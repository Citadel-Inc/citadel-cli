package cmd_test

import (
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
