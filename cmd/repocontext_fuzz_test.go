package cmd

import (
	"net/url"
	"strings"
	"testing"
)

// An accepted remote must name the same host that git and url.Parse connect to.
func FuzzParseOriginIntoRepoHost(f *testing.F) {
	f.Add("https://src.land/acme/widget.git")
	f.Add("ssh://git@src.land:2222/acme/widget.git")
	f.Add("https://evil.example/x@src.land/acme/widget.git")
	hosts := map[string]struct{}{"src.land": {}}
	f.Fuzz(func(t *testing.T, remote string) {
		if _, _, err := parseOriginIntoRepo(remote, hosts); err != nil || !strings.Contains(remote, "://") {
			return
		}
		u, err := url.Parse(strings.TrimSpace(remote))
		if err != nil {
			return
		}
		// url.Parse keeps a malformed port in Hostname ("ssh://h:0:/x"), and an
		// empty host connects nowhere; only a different name is a confusion.
		h, _, _ := strings.Cut(strings.ToLower(u.Hostname()), ":")
		if h != "" && h != "src.land" {
			t.Fatalf("%q accepted for src.land but connects to %q", remote, h)
		}
	})
}
