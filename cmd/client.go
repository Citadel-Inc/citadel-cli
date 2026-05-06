package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
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
		RetryOn401: rotateAccessTokenOn401Hook(cmd),
	})
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
