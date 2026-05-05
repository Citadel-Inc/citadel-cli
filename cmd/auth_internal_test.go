package cmd

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestResolveSupabaseURL(t *testing.T) {
	t.Setenv("SUPABASE_URL", "")
	if got := resolveSupabaseURL(); got != "https://ucnlqqhgqhenzthzkdpi.supabase.co" {
		t.Errorf("default: got %q", got)
	}
	t.Setenv("SUPABASE_URL", "https://override.example")
	if got := resolveSupabaseURL(); got != "https://override.example" {
		t.Errorf("env override: got %q", got)
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	got := buildAuthorizeURL("https://x.supabase.co", "http://127.0.0.1:1/callback", "CHAL")
	for _, want := range []string{"provider=github", "code_challenge=CHAL", "code_challenge_method=S256", "response_type=code", "redirect_uri=http%3A%2F%2F127.0.0.1%3A1%2Fcallback"} {
		if !strings.Contains(got, want) {
			t.Errorf("authorize URL missing %q: %s", want, got)
		}
	}
}

func TestExchangePKCECode_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("grant_type") != "pkce" || r.PostForm.Get("code") != "abc" || r.PostForm.Get("code_verifier") != "ver" {
			t.Errorf("form mismatch: %v", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt"}`))
	}))
	t.Cleanup(srv.Close)

	got, err := exchangePKCECode(srv.URL, "abc", "ver")
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

	_, err := exchangePKCECode(srv.URL, "x", "y")
	if err == nil || !strings.Contains(err.Error(), "bad code") {
		t.Errorf("got %v", err)
	}
}

func TestExchangePKCECode_Unreachable(t *testing.T) {
	// Use an obviously-bad host so http.PostForm fails immediately.
	if _, err := exchangePKCECode("http://127.0.0.1:1", "x", "y"); err == nil {
		t.Error("expected dial error")
	}
}

func TestExchangePKCECode_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-json"))
	}))
	t.Cleanup(srv.Close)
	if _, err := exchangePKCECode(srv.URL, "x", "y"); err == nil || !strings.Contains(err.Error(), "decode") {
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
