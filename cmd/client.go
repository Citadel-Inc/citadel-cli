package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/httpx"
)

// errSessionExpired is returned when the stored credential cannot be refreshed
// (missing agent binding or rotate-token rejected with 401).
var errSessionExpired = errors.New("session expired — run `citadel-cli auth login` again")

// newAPIClient loads the on-disk config (with env overrides) and returns an
// apiclient.Client honoring the persistent --server flag override.
func newAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return apiclient.New(cfg, apiclient.Options{
		Server:     serverFlag(cmd),
		Verbose:    verboseFlag(cmd),
		DebugHTTP:  debugHTTPFlag(cmd),
		UserAgent:  "citadel-cli/" + Version,
		RetryOn401: rotateAccessTokenOn401Hook(cmd),
	})
}

func publicAPIBaseURL(cmd *cobra.Command) (string, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	return apiclient.ResolveRESTServerURL(cfg.ResolveServerURL(serverFlag(cmd))), nil
}

func doPublicJSON(cmd *cobra.Command, method, path string, body, out any) error {
	base, err := publicAPIBaseURL(cmd)
	if err != nil {
		return err
	}
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(cmd.Context(), method, strings.TrimRight(base, "/")+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "citadel-cli/"+Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: httpx.Stack(nil, httpx.Options{Verbose: verboseFlag(cmd), DebugHTTP: debugHTTPFlag(cmd)}),
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return &apiclient.HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(raw)),
			RetryAfter: apiclient.ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func rotateAccessTokenOn401Hook(cmd *cobra.Command) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		cfg, err := clicfg.Load()
		if err != nil {
			return "", fmt.Errorf("reload config for token rotation: %w", err)
		}
		agentIDStr := strings.TrimSpace(cfg.AgentID)
		if agentIDStr == "" {
			// JWT-only / legacy config — cannot rotate; let the client surface the original 401.
			return "", nil
		}
		uid, err := uuid.Parse(agentIDStr)
		if err != nil {
			return "", fmt.Errorf("stored agent_id: %w", err)
		}
		c, err := apiclient.New(cfg, apiclient.Options{
			Server:    serverFlag(cmd),
			Verbose:   verboseFlag(cmd),
			DebugHTTP: debugHTTPFlag(cmd),
			UserAgent: "citadel-cli/" + Version,
		})
		if err != nil {
			return "", err
		}
		var newTok tokenWithCleartext
		if err := c.Post(ctx, "/agents/"+uid.String()+"/rotate-token", nil, &newTok); err != nil {
			if apiclient.IsStatus(err, http.StatusUnauthorized) {
				return "", errSessionExpired
			}
			return "", fmt.Errorf("rotate agent token: %w", err)
		}
		if newTok.CleartextToken == "" {
			return "", errors.New("rotate token: empty cleartext_token in response")
		}
		cfg.AccessToken = newTok.CleartextToken
		if newTok.ExpiresAt != nil {
			cfg.ExpiresAt = *newTok.ExpiresAt
		}
		if err := cfg.Save(); err != nil {
			return "", fmt.Errorf("save config after token rotation: %w", err)
		}
		return newTok.CleartextToken, nil
	}
}
