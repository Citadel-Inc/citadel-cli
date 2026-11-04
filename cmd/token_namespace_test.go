package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

func TestTokenIssueSendsNamespaces(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"limited": {[]string{"--namespace", "acme/api", "--namespace", "acme/tools"}, []string{"acme/api", "acme/tools"}},
		"unset":   {nil, nil},
	} {
		t.Run(name, func(t *testing.T) {
			var got struct {
				Namespaces []string `json:"namespaces"`
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "GET /agents":
					_, _ = io.WriteString(w, `{"agents":[{"id":"00000000-0000-0000-0000-00000000000a","name":"bot"}]}`)
				case "POST /agent-tokens":
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Error(err)
					}
					w.WriteHeader(http.StatusCreated)
					_, _ = io.WriteString(w, `{"id":"00000000-0000-0000-0000-00000000000b","agent_id":"00000000-0000-0000-0000-00000000000a","created_at":"2026-01-01T00:00:00Z","cleartext_token":"t"}`)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
			}))
			defer srv.Close()
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			t.Setenv("CITADEL_SERVER", srv.URL)
			t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
			if err := rootFor(cmd.TokenCmd, append([]string{"issue", "--agent", "bot"}, tc.args...)...).Execute(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got.Namespaces, tc.want) {
				t.Errorf("namespaces = %v, want %v", got.Namespaces, tc.want)
			}
		})
	}
}
