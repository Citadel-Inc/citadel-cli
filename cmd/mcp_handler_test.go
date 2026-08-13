package cmd_test

import (
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func prepareMcpLocalGuardEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func requireMcpError(t *testing.T, want string, args ...string) {
	t.Helper()
	prepareMcpLocalGuardEnv(t)
	err := rootFor(cmd.McpCmd, args...).Execute()
	if err == nil {
		t.Fatalf("expected %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

func TestMcpCall_BadArg_Hermetic(t *testing.T) {
	requireMcpError(t, `bad --arg "noeq" (expected key=value)`, "call", "lookup", "--arg", "noeq")
}

func TestMcpPromptsGet_BadArg_Hermetic(t *testing.T) {
	requireMcpError(t, `bad --arg "noeq" (expected key=value)`, "prompts", "get", "summarize", "--arg", "noeq")
}

func TestMcpCall_EmptyTool_Hermetic(t *testing.T) {
	requireMcpError(t, "tool name cannot be empty", "call", " \t")
}

func TestMcpResourcesRead_EmptyURI_Hermetic(t *testing.T) {
	requireMcpError(t, "resource URI cannot be empty", "resources", "read", " \t")
}

func TestMcpPromptsGet_EmptyName_Hermetic(t *testing.T) {
	requireMcpError(t, "prompt name cannot be empty", "prompts", "get", " \t")
}
