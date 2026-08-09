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
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

func TestRequestDeviceAuthorization_Print(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/oauth/device" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.PostForm.Get("client_id"); got != oauthClientID {
			t.Errorf("client_id = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deviceAuthorizationResponse{
			DeviceCode:              "device-secret",
			UserCode:                "ABCD-2345",
			VerificationURI:         "https://src.land/oauth/device",
			VerificationURIComplete: "https://src.land/oauth/device?user_code=ABCD-2345",
			ExpiresIn:               1800,
			Interval:                5,
		})
	}))
	t.Cleanup(srv.Close)

	got, err := requestDeviceAuthorization(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("requestDeviceAuthorization: %v", err)
	}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	printDeviceAuthorization(cmd, got)
	printed := out.String()
	for _, want := range []string{
		"User code: ABCD-2345",
		"Visit: https://src.land/oauth/device",
		"Complete URL: https://src.land/oauth/device?user_code=ABCD-2345",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("output missing %q: %s", want, printed)
		}
	}
}

func TestRunDeviceLogin_SuccessStoresAgentToken(t *testing.T) {
	jwtToken := makeUnsignedJWT(t, mapClaimsForDeviceTest())
	agentID := "70000000-0000-4000-8000-000000000007"
	exp := time.Now().Add(72 * time.Hour).UTC().Round(time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/oauth/device":
			_ = json.NewEncoder(w).Encode(deviceAuthorizationResponse{
				DeviceCode:      "device-secret",
				UserCode:        "ABCD-2345",
				VerificationURI: "https://src.land/oauth/device",
				Interval:        1,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
			}
			if got := r.PostForm.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Errorf("grant_type = %q", got)
			}
			if got := r.PostForm.Get("device_code"); got != "device-secret" {
				t.Errorf("device_code = %q", got)
			}
			if got := r.PostForm.Get("client_id"); got != oauthClientID {
				t.Errorf("client_id = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": jwtToken,
				"expires_in":   3600,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/agents":
			_, _ = w.Write([]byte(`{"agents":[],"next_cursor":""}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agents":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":   agentID,
				"name": "citadel-cli@test-host",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agents/"+agentID+"/rotate-token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "77777777-7777-4777-8777-777777777777",
				"agent_id":        agentID,
				"cleartext_token": "opaque-device-agent-token",
				"expires_at":      exp.Format(time.RFC3339Nano),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	var cfg clicfg.Config
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runDeviceLogin(cmd, &cfg, srv.URL, ""); err != nil {
		t.Fatalf("runDeviceLogin: %v", err)
	}

	loaded, err := clicfg.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.AccessToken != "opaque-device-agent-token" {
		t.Errorf("AccessToken = %q", loaded.AccessToken)
	}
	if loaded.AgentID != agentID {
		t.Errorf("AgentID = %q", loaded.AgentID)
	}
	if loaded.AgentName == "" {
		t.Error("AgentName is empty")
	}
	if loaded.RefreshToken != "" {
		t.Errorf("RefreshToken = %q, want empty", loaded.RefreshToken)
	}
	if loaded.ServerURL != srv.URL {
		t.Errorf("ServerURL = %q", loaded.ServerURL)
	}
	if loaded.UserUUID != "88888888-8888-4888-8888-888888888888" {
		t.Errorf("UserUUID = %q", loaded.UserUUID)
	}
	if !loaded.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt = %v want %v", loaded.ExpiresAt, exp)
	}
	if !strings.Contains(out.String(), "Authentication successful!") {
		t.Errorf("success output = %s", out.String())
	}
}

func TestPollDeviceToken_DeniedAndExpired(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		description string
	}{
		{name: "denied", code: "access_denied", description: "user denied"},
		{name: "expired", code: "expired_token", description: "expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/oauth/token" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":             tt.code,
					"error_description": tt.description,
				})
			}))
			t.Cleanup(srv.Close)

			_, err := pollDeviceTokenWithInterval(context.Background(), srv.URL, deviceAuthorizationResponse{
				DeviceCode: "device-secret",
			}, time.Nanosecond)
			if err == nil || !strings.Contains(err.Error(), tt.code) || !strings.Contains(err.Error(), tt.description) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func mapClaimsForDeviceTest() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": "88888888-8888-4888-8888-888888888888",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
}
