package cmd

// Unit tests for unexported helper functions that benefit from being in the
// same package (no http server required).

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/sseclient"
)

// ── decodePasskeyRows ─────────────────────────────────────────────────────────

func TestDecodePasskeyRows_wrapped(t *testing.T) {
	raw := []byte(`{"passkeys":[{"id":"pk1","name":"YubiKey","created_at":"2026-01-01T00:00:00Z"}]}`)
	rows, ok := decodePasskeyRows(raw)
	if !ok {
		t.Fatal("expected ok=true for wrapped passkeys JSON")
	}
	if len(rows) != 1 || rows[0].ID != "pk1" {
		t.Fatalf("got %+v", rows)
	}
}

func TestDecodePasskeyRows_directArray(t *testing.T) {
	raw := []byte(`[{"id":"pk1","name":"Key","created_at":"2026-01-01T00:00:00Z"},{"id":"pk2","name":"Key2","created_at":"2026-01-02T00:00:00Z"}]`)
	rows, ok := decodePasskeyRows(raw)
	if !ok {
		t.Fatal("expected ok=true for direct array")
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestDecodePasskeyRows_emptyWrapped(t *testing.T) {
	raw := []byte(`{"passkeys":[]}`)
	rows, ok := decodePasskeyRows(raw)
	if !ok {
		t.Fatal("expected ok=true for empty wrapped passkeys")
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestDecodePasskeyRows_emptyArray(t *testing.T) {
	raw := []byte(`[]`)
	rows, ok := decodePasskeyRows(raw)
	if !ok {
		t.Fatal("expected ok=true for empty array")
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestDecodePasskeyRows_invalid(t *testing.T) {
	_, ok := decodePasskeyRows([]byte(`not json`))
	if ok {
		t.Fatal("expected ok=false for invalid JSON")
	}
}

// ── deltaLabel ────────────────────────────────────────────────────────────────

// deltaLabel is called by printDelta when lastDeltaLabel is empty/whitespace.
// We trigger this by sending a "remove" event with nil data; shortRowLabel(nil)
// returns "" because string(nil) == "", so printDelta falls through to deltaLabel.

func TestTableWatchEmitter_deltaLabel_empty(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	c.SetOut(buf)

	em := newTableWatchEmitter(c, watchRepos, watchTableCtx{repoParentNS: "ns"})
	// Empty rows: send remove with nil data → deltaLabel returns "(empty)".
	if err := em.Handle(sseclient.Event{Type: "remove", Data: nil}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "(empty)") {
		t.Fatalf("expected '(empty)' in output, got %q", out)
	}
}

func TestTableWatchEmitter_deltaLabel_withRows(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	c.SetOut(buf)

	em := newTableWatchEmitter(c, watchRepos, watchTableCtx{repoParentNS: "ns"})
	// Add a row via "init".
	if err := em.Handle(sseclient.Event{
		Type: "init",
		Data: []byte(`{"path":"ns/a","visibility":"private","default_branch":"main","created_at":"2026-01-01"}`),
	}); err != nil {
		t.Fatal(err)
	}
	buf.Reset()

	// Remove with nil data: rowKey(nil)==""  so row "ns/a" stays in the map;
	// deltaLabel() iterates sortedKeys() → ["ns/a"] and returns its label.
	if err := em.Handle(sseclient.Event{Type: "remove", Data: nil}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ns/a") {
		t.Fatalf("expected 'ns/a' in delta label output, got %q", out)
	}
}

// ── resolveSSHPublicKeyMaterial ───────────────────────────────────────────────

func TestResolveSSHPublicKeyMaterial_bothFlags(t *testing.T) {
	_, _, err := resolveSSHPublicKeyMaterial("key-material", "/path/to/file")
	if err == nil || !strings.Contains(err.Error(), "either") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestResolveSSHPublicKeyMaterial_inlineFlag(t *testing.T) {
	mat, src, err := resolveSSHPublicKeyMaterial("ssh-ed25519 AAAA...", "")
	if err != nil {
		t.Fatal(err)
	}
	if mat != "ssh-ed25519 AAAA..." || src != "--public-key" {
		t.Fatalf("mat=%q src=%q", mat, src)
	}
}

func TestResolveSSHPublicKeyMaterial_fileFlag(t *testing.T) {
	content := "ssh-ed25519 AAAA keycomment"
	keyPath := t.TempDir() + "/id_ed25519.pub"
	if err := os.WriteFile(keyPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	mat, src, err := resolveSSHPublicKeyMaterial("", keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if mat != content {
		t.Fatalf("mat=%q want %q", mat, content)
	}
	if src != keyPath {
		t.Fatalf("src=%q want %q", src, keyPath)
	}
}

func TestResolveSSHPublicKeyMaterial_fileMissing(t *testing.T) {
	_, _, err := resolveSSHPublicKeyMaterial("", "/nonexistent/path/key.pub")
	if err == nil || !strings.Contains(err.Error(), "--key-file") {
		t.Fatalf("want read error, got %v", err)
	}
}
