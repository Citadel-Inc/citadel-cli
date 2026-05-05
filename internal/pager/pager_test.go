package pager

import (
	"os"
	"testing"
)

// TestResolveCitadelOverride asserts CITADEL_PAGER wins over the others.
func TestResolveCitadelOverride(t *testing.T) {
	t.Setenv("PAGER", "less")
	t.Setenv("GIT_PAGER", "delta")
	t.Setenv("CITADEL_PAGER", "moar -mousable")
	if got := Resolve(); got != "moar -mousable" {
		t.Fatalf("CITADEL_PAGER must win, got %q", got)
	}
}

// TestResolveGitOverPager asserts GIT_PAGER wins over PAGER.
func TestResolveGitOverPager(t *testing.T) {
	t.Setenv("PAGER", "less")
	t.Setenv("GIT_PAGER", "delta")
	if got := Resolve(); got != "delta" {
		t.Fatalf("GIT_PAGER must win over PAGER, got %q", got)
	}
}

// TestResolveExplicitEmpty asserts an explicit empty CITADEL_PAGER
// short-circuits the chain (matches git's `PAGER=""` convention).
func TestResolveExplicitEmpty(t *testing.T) {
	t.Setenv("CITADEL_PAGER", "")
	t.Setenv("GIT_PAGER", "delta")
	if got := Resolve(); got != "" {
		t.Fatalf("explicit-empty CITADEL_PAGER must short-circuit, got %q", got)
	}
}

// When no pager tier is set, match git: fall back to less with sane flags.
func TestResolve_DefaultLessWhenEnvUnset(t *testing.T) {
	keys := []string{"CITADEL_PAGER", "GIT_PAGER", "PAGER"}
	saved := make([]struct {
		key string
		val string
		ok  bool
	}, 0, len(keys))
	for _, k := range keys {
		v, ok := os.LookupEnv(k)
		saved = append(saved, struct {
			key string
			val string
			ok  bool
		}{k, v, ok})
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, e := range saved {
			if e.ok {
				_ = os.Setenv(e.key, e.val)
			} else {
				_ = os.Unsetenv(e.key)
			}
		}
	})

	if got := Resolve(); got != "less -FRX" {
		t.Fatalf("default pager, got %q", got)
	}
}
