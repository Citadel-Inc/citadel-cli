package completion

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskCacheTTL(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return start }

	const resolved = "https://api.example.com"
	const key = "orgs"

	writeCache(resolved, key, []string{"z", "a"})
	got, ok := readCache(resolved, key)
	if !ok || len(got) != 2 {
		t.Fatalf("cache hit: ok=%v got=%v", ok, got)
	}

	now = func() time.Time { return start.Add(30 * time.Second) }
	got, ok = readCache(resolved, key)
	if !ok {
		t.Fatal("expected hit within TTL")
	}
	if got[0] != "z" || got[1] != "a" {
		t.Fatalf("unexpected order preserved: %v", got)
	}

	now = func() time.Time { return start.Add(61 * time.Second) }
	_, ok = readCache(resolved, key)
	if ok {
		t.Fatal("expected miss after 60s TTL")
	}
}

func TestNoDiskCacheUsesMemory(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })

	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "1")
	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return start }

	const resolved = "https://api.example.com"
	const key = "agents"
	writeCache(resolved, key, []string{"x"})
	got, ok := readCache(resolved, key)
	if !ok || len(got) != 1 || got[0] != "x" {
		t.Fatalf("memory hit: ok=%v got=%v", ok, got)
	}
}

func TestReadCache_CorruptDiskJSON(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })

	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return start }

	const resolved = "https://api.example.com"
	const key = "corrupt-disk-json"
	path := filepath.Join(cacheHome, "citadel-cli", "completion", "api.example.com", "corrupt-disk-json.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, ok := readCache(resolved, key)
	if ok {
		t.Fatal("expected miss for corrupt JSON on disk")
	}
}

func TestReadCache_WrongServerInEnvelope(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })

	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return start }

	const resolved = "https://api.example.com"
	const key = "wrong-server-envelope"
	path := filepath.Join(cacheHome, "citadel-cli", "completion", "api.example.com", "wrong-server-envelope.json")
	payload := `{"fetched_at":"2026-05-05T12:00:00Z","server":"https://other.example","resource":"wrong-server-envelope","values":["x"]}`
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	_, ok := readCache(resolved, key)
	if ok {
		t.Fatal("expected miss when envelope server does not match")
	}
}

func TestRemoveAsync_NoKeysIsNoop(t *testing.T) {
	RemoveAsync("https://api.example.com")
}

func TestRemove_NoDiskCache(t *testing.T) {
	origNow := now
	t.Cleanup(func() { now = origNow })
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "1")
	now = func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) }

	const resolved = "https://api.example.com"
	const key = "remove-no-disk"
	writeCache(resolved, key, []string{"x"})
	// Verify it is still in memory despite disk being disabled.
	if _, ok := readCache(resolved, key); !ok {
		t.Fatal("expected memory cache hit")
	}
	// Remove should clear memory and skip disk (early return branch).
	Remove(resolved, key)
	if _, ok := readCache(resolved, key); ok {
		t.Fatal("expected cache miss after Remove with no-disk-cache")
	}
}
