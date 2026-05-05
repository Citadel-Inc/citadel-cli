package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
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
