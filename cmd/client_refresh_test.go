package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

func TestRefreshSessionTokenKeepsCredentialsOn502(t *testing.T) {
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	const access, refresh = "stored-access", "stored-refresh"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	if err := clicfg.Update(func(cfg *clicfg.Config) error {
		cfg.ServerURL = srv.URL
		cfg.AccessToken = access
		cfg.RefreshToken = refresh
		cfg.AgentID = ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cmd := testServerCmd(srv.URL)
	cfg, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = refreshSessionToken(t.Context(), cmd, cfg)

	got, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != access || got.RefreshToken != refresh {
		t.Fatalf("transient 502 cleared stored login: access=%q refresh=%q", got.AccessToken, got.RefreshToken)
	}
}

func TestRotateWithLoginKeepsNewRefreshTokenOnRotate500(t *testing.T) {
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	agentID := uuid.New()
	const oldRefresh = "old-refresh-token"
	const newRefresh = "rotated-refresh-token"
	rotateCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "rotate-token") {
			rotateCalls++
			if rotateCalls == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "refresh_token") || strings.Contains(r.URL.Path, "token") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "new-jwt",
				"refresh_token": newRefresh,
				"expires_in":    3600,
				"token_type":    "Bearer",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	if err := clicfg.Update(func(cfg *clicfg.Config) error {
		cfg.ServerURL = srv.URL
		cfg.AccessToken = "expired-agent-token"
		cfg.RefreshToken = oldRefresh
		cfg.AgentID = agentID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cmd := testServerCmd(srv.URL)
	cfg, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = rotateAgentToken(t.Context(), cmd, cfg, agentID)

	got, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.RefreshToken != newRefresh {
		t.Fatalf("rotate-token 500 discarded rotated refresh token: got %q want %q", got.RefreshToken, newRefresh)
	}
}

func TestRefreshSessionTokenClearsCredentialsOnInvalidGrant(t *testing.T) {
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	t.Cleanup(srv.Close)

	if err := clicfg.Update(func(cfg *clicfg.Config) error {
		cfg.ServerURL = srv.URL
		cfg.AccessToken = "stored-access"
		cfg.RefreshToken = "stored-refresh"
		cfg.AgentID = ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cmd := testServerCmd(srv.URL)
	cfg, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = refreshSessionToken(t.Context(), cmd, cfg)

	got, err := clicfg.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "" || got.RefreshToken != "" {
		t.Fatalf("invalid_grant left stored login: access=%q refresh=%q", got.AccessToken, got.RefreshToken)
	}
}

func TestDoPublicJSON_RejectsOversizedBody(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"`)
		_, _ = io.WriteString(w, strings.Repeat("x", 32<<20))
		_, _ = io.WriteString(w, `"}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CITADEL_SERVER", srv.URL)

	var out struct{ Name string }
	if err := doPublicJSON(t.Context(), testServerCmd(srv.URL), http.MethodGet, "/auth/providers", nil, &out); err == nil || !strings.Contains(err.Error(), "MiB") {
		t.Fatalf("want size-cap error, got %v", err)
	}
}

func testServerCmd(serverURL string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("server", serverURL, "")
	cmd.PersistentFlags().Bool("verbose", false, "")
	cmd.PersistentFlags().Bool("debug-http", false, "")
	cmd.SetContext(context.Background())
	return cmd
}
