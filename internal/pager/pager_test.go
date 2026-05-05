package pager

import "testing"

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
