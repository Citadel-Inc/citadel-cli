package completion

import (
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
