package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// makeUnsignedJWT crafts an HS256-signed JWT with the given claims. The
// CLI parses unverified, so any signature is accepted.
func makeUnsignedJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestClaimsFromJWT(t *testing.T) {
	tok := makeUnsignedJWT(t, jwt.MapClaims{"sub": "u1", "exp": float64(1700000000)})
	claims, err := claimsFromJWT(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims["sub"] != "u1" {
		t.Errorf("sub = %v", claims["sub"])
	}

	if _, err := claimsFromJWT("not-a-jwt"); err == nil {
		t.Error("expected parse error for non-JWT input")
	}
}

func TestUserUUIDFromClaims(t *testing.T) {
	if got := userUUIDFromClaims(jwt.MapClaims{"sub": "abc"}); got != "abc" {
		t.Errorf("got %q", got)
	}
	if got := userUUIDFromClaims(jwt.MapClaims{"sub": 42}); got != "" {
		t.Errorf("non-string sub must yield empty: %q", got)
	}
	if got := userUUIDFromClaims(jwt.MapClaims{}); got != "" {
		t.Errorf("missing sub must yield empty: %q", got)
	}
}

func TestRandomOAuthState(t *testing.T) {
	a, err := randomOAuthState()
	if err != nil {
		t.Fatal(err)
	}
	b, err := randomOAuthState()
	if err != nil {
		t.Fatal(err)
	}
	if a == "" || b == "" || a == b {
		t.Fatalf("unexpected states %q %q", a, b)
	}
}

func TestBootstrapAgentToken_Happy(t *testing.T) {
	ctx := context.Background()
	agentID := "10000000-0000-4000-8000-000000000001"
	exp := time.Now().Add(48 * time.Hour).UTC().Round(time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents":
			_, _ = w.Write([]byte(`{"agents":[],"next_cursor":""}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agents":
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode POST /agents: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": agentID, "name": body.Name})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/"+agentID+"/rotate-token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "88888888-8888-8888-8888-888888888888",
				"agent_id":        agentID,
				"cleartext_token": "opaque-agent-token",
				"expires_at":      exp.Format(time.RFC3339Nano),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	jwt := makeUnsignedJWT(t, jwt.MapClaims{
		"sub": "22222222-2222-2222-2222-222222222222",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	var cfg clicfg.Config
	cmd := &cobra.Command{}
	if err := bootstrapAgentToken(ctx, cmd, &cfg, srv.URL, "", jwt); err != nil {
		t.Fatalf("bootstrapAgentToken: %v", err)
	}
	if cfg.AccessToken != "opaque-agent-token" {
		t.Errorf("AccessToken = %q", cfg.AccessToken)
	}
	if cfg.AgentID != agentID {
		t.Errorf("AgentID = %q", cfg.AgentID)
	}
	wantName, err := defaultCLIAgentName()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AgentName != wantName {
		t.Errorf("AgentName = %q want %q", cfg.AgentName, wantName)
	}
	if cfg.RefreshToken != "" {
		t.Errorf("RefreshToken should be cleared, got %q", cfg.RefreshToken)
	}
	if !cfg.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt = %v want %v", cfg.ExpiresAt, exp)
	}
}

func TestMaybeEagerMigrateLegacyJWT_RootExecute(t *testing.T) {
	agentID := "20000000-0000-4000-8000-000000000002"
	exp := time.Now().Add(90 * 24 * time.Hour).UTC()
	var rotateHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agents":
			_, _ = w.Write([]byte(`{"agents":[],"next_cursor":""}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agents":
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": agentID, "name": body.Name})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/"+agentID+"/rotate-token":
			rotateHits++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "99999999-9999-9999-9999-999999999999",
				"agent_id":        agentID,
				"cleartext_token": "post-migrate-token",
				"expires_at":      exp.Format(time.RFC3339Nano),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	jwt := makeUnsignedJWT(t, jwt.MapClaims{
		"sub": "33333333-3333-3333-3333-333333333333",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	cfgDir := filepath.Join(xdg, "citadel")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	toml := "server_url = \"" + srv.URL + "\"\naccess_token = \"" + jwt + "\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0600); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"agent", "list"})
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	loaded, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessToken != "post-migrate-token" {
		t.Fatalf("token not upgraded: %q", loaded.AccessToken)
	}
	if loaded.AgentID != agentID {
		t.Fatalf("agent id %q", loaded.AgentID)
	}
	if rotateHits != 1 {
		t.Fatalf("rotate hits = %d", rotateHits)
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	got := buildAuthorizeURL("https://x.example/", "http://127.0.0.1:1/callback", "CHAL", "STATEVAL")
	for _, want := range []string{
		"client_id=citadel-cli",
		"code_challenge=CHAL",
		"code_challenge_method=S256",
		"response_type=code",
		"state=STATEVAL",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("authorize URL missing %q: %s", want, got)
		}
	}
	if !strings.Contains(got, "127.0.0.1") {
		t.Errorf("expected redirect host in URL: %s", got)
	}
	if !strings.HasPrefix(got, "https://x.example/api/oauth/authorize?") {
		t.Errorf("unexpected prefix: %s", got)
	}
}

func TestExchangePKCECode_Happy(t *testing.T) {
	redirect := "http://127.0.0.1:9/callback"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oauth/token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("code") != "abc" || r.PostForm.Get("code_verifier") != "ver" {
			t.Errorf("form mismatch: %v", r.PostForm)
		}
		if r.PostForm.Get("client_id") != oauthClientID {
			t.Errorf("client_id = %q", r.PostForm.Get("client_id"))
		}
		if r.PostForm.Get("redirect_uri") != redirect {
			t.Errorf("redirect_uri = %q", r.PostForm.Get("redirect_uri"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt"}`))
	}))
	t.Cleanup(srv.Close)

	got, err := exchangePKCECode(srv.URL, redirect, "abc", "ver")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" {
		t.Errorf("decoded %+v", got)
	}
}

func TestExchangePKCECode_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad code"))
	}))
	t.Cleanup(srv.Close)

	_, err := exchangePKCECode(srv.URL, "http://127.0.0.1:1/callback", "x", "y")
	if err == nil || !strings.Contains(err.Error(), "bad code") {
		t.Errorf("got %v", err)
	}
}

func TestExchangePKCECode_Unreachable(t *testing.T) {
	// Use an obviously-bad host so http.PostForm fails immediately.
	if _, err := exchangePKCECode("http://127.0.0.1:1", "http://127.0.0.1:2/callback", "x", "y"); err == nil {
		t.Error("expected dial error")
	}
}

func TestExchangePKCECode_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-json"))
	}))
	t.Cleanup(srv.Close)
	if _, err := exchangePKCECode(srv.URL, "http://127.0.0.1:1/callback", "x", "y"); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("got %v", err)
	}
}

func TestRunSetToken_FromFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	tok := makeUnsignedJWT(t, jwt.MapClaims{
		"sub": "11111111-1111-1111-1111-111111111111",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})
	if err := setTokenCmd.Flags().Set("token", tok); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = setTokenCmd.Flags().Set("token", "") })

	if err := runSetToken(setTokenCmd, nil); err != nil {
		t.Fatalf("runSetToken: %v", err)
	}
}

func TestRunSetToken_NoSourceErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	// Replace stdin with an empty pipe so io.ReadAll returns "".
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })

	if err := setTokenCmd.Flags().Set("token", ""); err != nil {
		t.Fatal(err)
	}
	err = runSetToken(setTokenCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "no token") {
		t.Fatalf("expected no-token error, got %v", err)
	}
}

func TestExpiryFromClaims(t *testing.T) {
	want := time.Unix(1700000000, 0)
	got := expiryFromClaims(jwt.MapClaims{"exp": float64(1700000000)}, time.Time{})
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}

	fallback := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := expiryFromClaims(jwt.MapClaims{}, fallback); !got.Equal(fallback) {
		t.Errorf("missing exp must use fallback: got %v", got)
	}
	if got := expiryFromClaims(jwt.MapClaims{"exp": "string"}, fallback); !got.Equal(fallback) {
		t.Errorf("malformed exp must use fallback: got %v", got)
	}
}

func TestGeneratePKCE(t *testing.T) {
	v, c, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE: %v", err)
	}
	if v == "" || c == "" {
		t.Fatal("verifier/challenge must be non-empty")
	}
	if v == c {
		t.Error("verifier and challenge must differ")
	}
	if _, err := base64.RawURLEncoding.DecodeString(v); err != nil {
		t.Errorf("verifier not base64url-rawencoded: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(c); err != nil {
		t.Errorf("challenge not base64url-rawencoded: %v", err)
	}
	// Two consecutive verifiers must differ (rand-driven).
	v2, _, err := generatePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if v == v2 {
		t.Error("PKCE verifier should be unique per call")
	}
	if strings.ContainsAny(v, "+/=") {
		t.Error("verifier must use base64url alphabet without padding")
	}
}
