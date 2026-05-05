package completion

import (
	"testing"
	"time"
)

func TestRemove_ClearsDiskAndMemory(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	now = time.Now

	const resolved = "https://api.example.com"
	const key = "orgs"
	writeCache(resolved, key, []string{"only"})
	if _, ok := readCache(resolved, key); !ok {
		t.Fatal("expected cache hit before Remove")
	}
	Remove(resolved, key)
	if _, ok := readCache(resolved, key); ok {
		t.Fatal("expected miss after Remove")
	}
}

func TestRemoveAsync_ClearsCache(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	now = time.Now

	const resolved = "https://api.example.com"
	writeCache(resolved, "agents", []string{"x"})
	RemoveAsync(resolved, "agents")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := readCache(resolved, "agents"); !ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("RemoveAsync did not clear cache within deadline")
}

func TestRepoKey(t *testing.T) {
	if got := RepoKey("acme"); got != "repos:acme" {
		t.Fatalf("RepoKey: %q", got)
	}
}
