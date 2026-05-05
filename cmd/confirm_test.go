package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// withStdin temporarily replaces os.Stdin with a pipe seeded with input.
func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, input); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
	fn()
}

func TestConfirmTypedValue_YesShortCircuits(t *testing.T) {
	if err := confirmTypedValue(true, "delete", "x"); err != nil {
		t.Fatalf("--yes path must skip prompt: %v", err)
	}
}

func TestConfirmTypedValue_MatchAccepts(t *testing.T) {
	withStdin(t, "myrepo\n", func() {
		if err := confirmTypedValue(false, "delete", "myrepo"); err != nil {
			t.Errorf("matching input must accept: %v", err)
		}
	})
}

func TestConfirmTypedValue_MismatchRejects(t *testing.T) {
	withStdin(t, "wrong\n", func() {
		err := confirmTypedValue(false, "delete", "myrepo")
		if err == nil || !strings.Contains(err.Error(), "confirmation mismatch") {
			t.Errorf("mismatch must reject: %v", err)
		}
	})
}

func TestConfirmSlug_DelegatesToTypedValue(t *testing.T) {
	if err := confirmSlug(true, "delete", "x"); err != nil {
		t.Fatalf("confirmSlug --yes: %v", err)
	}
}
