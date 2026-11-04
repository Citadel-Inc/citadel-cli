package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	info := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	for _, c := range []struct {
		name, linked, module, want string
	}{
		{"go install", "dev", "v0.3.0", "v0.3.0"},
		{"local build", "dev", "(devel)", "dev"},
		{"release", "v1.2.3", "v9.9.9", "v1.2.3"},
	} {
		if got := resolveVersion(c.linked, info(c.module)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
