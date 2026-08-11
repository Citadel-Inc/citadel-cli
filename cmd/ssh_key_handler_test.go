package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestSSHKeyList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.SSHKeyCmd, "list", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestSSHKeyAdd_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(
		cmd.SSHKeyCmd,
		"add",
		"--public-key",
		"ssh-ed25519 AAAA test",
		"--output",
		"toml",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), `--output for add supports json or default human summary only; got "toml"`) {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestSSHKeyAdd_EmptyKey_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	keyFile := t.TempDir() + "/empty.pub"
	if err := os.WriteFile(keyFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := rootFor(cmd.SSHKeyCmd, "add", "--key-file", keyFile, "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "empty public key from "+keyFile) {
		t.Fatalf("want empty key validation error, got %v", err)
	}
}
