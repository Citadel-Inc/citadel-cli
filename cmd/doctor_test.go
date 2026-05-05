package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/mcpclient"
)

func TestCheckServer_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	got := checkServer(context.Background(), srv.URL)
	if got.status != statusPass {
		t.Errorf("server reachable: %s", got)
	}
}

func TestCheckServer_5xxFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	got := checkServer(context.Background(), srv.URL)
	if got.status != statusFail {
		t.Errorf("5xx must FAIL: %s", got)
	}
}

func TestCheckServer_401Passes(t *testing.T) {
	// Auth-gated endpoint returns 401 — server is still reachable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	got := checkServer(context.Background(), srv.URL)
	if got.status != statusPass {
		t.Errorf("401 means reachable; want PASS: %s", got)
	}
}

func TestCheckServer_NoURL(t *testing.T) {
	got := checkServer(context.Background(), "")
	if got.status != statusFail {
		t.Errorf("empty URL must FAIL: %s", got)
	}
}

func TestCheckAuthToken_Missing(t *testing.T) {
	if got := checkAuthToken(clicfg.Config{}); got.status != statusFail {
		t.Errorf("no token: %s", got)
	}
}

func TestCheckAuthToken_OpaqueToken(t *testing.T) {
	got := checkAuthToken(clicfg.Config{AccessToken: "ag_at_opaque_random_bytes"})
	if got.status != statusPass {
		t.Errorf("opaque token should PASS: %s", got)
	}
}

func TestCheckAuthToken_ValidJWT(t *testing.T) {
	tok := makeJWT(t, jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	if got := checkAuthToken(clicfg.Config{AccessToken: tok}); got.status != statusPass {
		t.Errorf("valid JWT: %s", got)
	}
}

func TestCheckAuthToken_ExpiredJWT(t *testing.T) {
	tok := makeJWT(t, jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111",
		"exp": float64(time.Now().Add(-time.Hour).Unix()),
	})
	if got := checkAuthToken(clicfg.Config{AccessToken: tok}); got.status != statusFail {
		t.Errorf("expired JWT: %s", got)
	}
}

func TestCheckAuthToken_NearExpiry(t *testing.T) {
	tok := makeJWT(t, jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111",
		"exp": float64(time.Now().Add(2 * time.Minute).Unix()),
	})
	if got := checkAuthToken(clicfg.Config{AccessToken: tok}); got.status != statusWarn {
		t.Errorf("near-expiry JWT must WARN: %s", got)
	}
}

func TestCheckConfigPerms_NoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got := checkConfigPerms()
	if got.status != statusWarn {
		t.Errorf("missing file should WARN (first run): %s", got)
	}
}

func TestCheckResult_StringRendersGlyph(t *testing.T) {
	r := checkResult{name: "x", status: statusPass, detail: "ok"}
	if !strings.Contains(r.String(), "[PASS]") {
		t.Errorf("PASS glyph: %q", r.String())
	}
	r.status = statusFail
	if !strings.Contains(r.String(), "[FAIL]") {
		t.Errorf("FAIL glyph: %q", r.String())
	}
}

// makeJWT mirrors the test helper in auth_internal_test.go but lives here so
// doctor_test.go does not depend on test-file ordering.
func makeJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func doctorTestMux(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/healthz"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/mcp"):
			var req struct {
				ID     int             `json:"id"`
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("mcp decode: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			if req.Method != "initialize" {
				t.Errorf("unexpected MCP method %q", req.Method)
				http.Error(w, "nope", http.StatusBadRequest)
				return
			}
			w.Header().Set("Mcp-Session-Id", "sess-doctor-test")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"protocolVersion": mcpclient.ProtocolVersion,
					"serverInfo":      map[string]any{"name": "test-mcp", "version": "0"},
				},
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("unhandled doctor test request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}
}

func TestRunDoctor_allChecksGreen(t *testing.T) {
	srv := httptest.NewServer(doctorTestMux(t))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "opaque-agent-token-for-doctor")

	root := NewRootCmd()
	t.Cleanup(func() {
		// NewRootCmd re-parents every global *cobra.Command onto a fresh root.
		// Without this, the prior root can retain SetArgs(["doctor"]) while
		// shared subcommands (namespace, agent, …) still point at it, which
		// breaks unrelated tests that Execute those subcommands directly.
		_ = NewRootCmd()
	})
	root.SetArgs([]string{"doctor"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("doctor: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "[PASS]") || !strings.Contains(s, "server-reachable") {
		t.Fatalf("expected PASS server line, got:\n%s", s)
	}
	if !strings.Contains(s, "mcp-endpoint") {
		t.Fatalf("expected MCP line, got:\n%s", s)
	}
}

func TestAnyFailed_trueOnFail(t *testing.T) {
	if !anyFailed([]checkResult{{status: statusPass}, {status: statusFail}}) {
		t.Fatal("anyFailed must be true when a FAIL is present")
	}
}

func TestAnyFailed_falseWithoutFail(t *testing.T) {
	if anyFailed([]checkResult{{status: statusPass}, {status: statusWarn}}) {
		t.Fatal("WARN must not count as failure")
	}
}
