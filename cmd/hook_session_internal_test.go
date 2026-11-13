package cmd

import (
	"testing"
)

func TestSessionDir_RejectsUnusableIDs(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	for _, id := range []string{"", ".", "..", "a/b", "CON", "con", "CON.txt", "NUL", "COM1", "LPT9"} {
		if got := sessionDir(id); got != "" {
			t.Errorf("sessionDir(%q) = %q, want empty", id, got)
		}
	}
	if got := sessionDir("sess-1"); got == "" {
		t.Fatal("valid id rejected")
	}
}
