package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func authProviderServerEnv(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	return srv
}

func withServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	return srv
}

func setOutRecursive(c *cobra.Command, out, err io.Writer) {
	c.SetOut(out)
	c.SetErr(err)
	for _, child := range c.Commands() {
		setOutRecursive(child, out, err)
	}
}

func resetFlagsRecursive(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
			f.Changed = false
			return
		}
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, child := range c.Commands() {
		resetFlagsRecursive(child)
	}
}

func rootFor(args ...string) *cobra.Command {
	resetFlagsRecursive(AuthCmd)
	setOutRecursive(AuthCmd, io.Discard, io.Discard)
	root := &cobra.Command{Use: "test"}
	root.AddCommand(AuthCmd)
	root.SetArgs(append([]string{"auth"}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

func rootForOut(stdout io.Writer, args ...string) *cobra.Command {
	resetFlagsRecursive(AuthCmd)
	setOutRecursive(AuthCmd, stdout, io.Discard)
	root := &cobra.Command{Use: "test"}
	root.AddCommand(AuthCmd)
	root.SetArgs(append([]string{"auth"}, args...))
	root.SetOut(stdout)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func assertTestBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("authorization header = %q", got)
	}
}

func assertTestJSONBody(t *testing.T, r *http.Request, want map[string]any) {
	t.Helper()
	defer func() { _ = r.Body.Close() }()
	var got map[string]any
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("body = %#v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("body[%q] = %#v, want %#v", k, got[k], v)
		}
	}
}

func TestAuthProviderList_JSON_Unauthenticated(t *testing.T) {
	authProviderServerEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/auth/providers" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"providers": []map[string]any{
				{"id": "github", "label": "GitHub"},
				{"id": "google", "label": "Google"},
			},
		})
	})

	var out strings.Builder
	if err := rootForOut(&out, "provider", "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "github"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestAuthProviderLink_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/link-provider" {
			http.NotFound(w, r)
			return
		}
		assertTestBearer(t, r)
		assertTestJSONBody(t, r, map[string]any{"provider": "github"})
		writeJSON(t, w, http.StatusOK, map[string]any{
			"provider":     "github",
			"redirect_url": "https://supabase.example/auth/v1/authorize?provider=github",
		})
	})

	var out strings.Builder
	if err := rootForOut(&out, "provider", "link", "github", "--json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"redirect_url": "https://supabase.example/auth/v1/authorize?provider=github"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestAuthProviderLink_DefaultLaunchesBrowser(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/link-provider" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"provider":     "github",
			"redirect_url": "https://supabase.example/auth/v1/authorize?provider=github",
		})
	})

	orig := launchBrowser
	t.Cleanup(func() { launchBrowser = orig })
	var opened string
	launchBrowser = func(target string) { opened = target }

	if err := rootFor("provider", "link", "github").Execute(); err != nil {
		t.Fatal(err)
	}
	if opened != "https://supabase.example/auth/v1/authorize?provider=github" {
		t.Fatalf("launchBrowser target = %q", opened)
	}
}

func TestAuthProviderUnlink_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/unlink-provider" {
			http.NotFound(w, r)
			return
		}
		assertTestBearer(t, r)
		assertTestJSONBody(t, r, map[string]any{"provider": "github"})
		writeJSON(t, w, http.StatusOK, map[string]any{"status": "ok"})
	})

	var out strings.Builder
	if err := rootForOut(&out, "provider", "unlink", "github", "--yes", "--json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"provider": "github"`) || !strings.Contains(out.String(), `"status": "ok"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
