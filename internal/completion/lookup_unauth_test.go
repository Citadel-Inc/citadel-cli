package completion

import (
	"context"
	"testing"
)

func TestLookup_NotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://127.0.0.1:9")

	_, err := Lookup(context.Background(), "", KeyOrgs, FetchOrgNamespaceSlugs)
	if err == nil {
		t.Fatal("expected error without access token")
	}
}
