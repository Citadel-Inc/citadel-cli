package completion

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemove_ClearsDiskAndMemory(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })

	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	now = func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) }

	const resolved = "https://api.example.com"
	const key = "orgs"
	writeCache(resolved, key, []string{"only"})
	path := filepath.Join(cacheHome, "citadel-cli", "completion", "api.example.com", "orgs.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected disk file: %v", err)
	}
	if _, ok := readCache(resolved, key); !ok {
		t.Fatal("expected cache hit before Remove")
	}
	Remove(resolved, key)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected disk file removed, stat err=%v", err)
	}
	if _, ok := readCache(resolved, key); ok {
		t.Fatal("expected miss after Remove")
	}
}

func TestRemoveAsync_ClearsCache(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	now = func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) }

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
