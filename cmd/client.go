package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
	"github.com/Citadel-Inc/citadel-cli/internal/httpx"
)

// errSessionExpired is returned when the stored credential cannot be refreshed
// (missing agent binding or rotate-token rejected with 401).
var errSessionExpired = errors.New("session expired — run `citadel-cli auth login` again")

// newAPIClient loads the on-disk config (with env overrides) and returns an
// apiclient.Client honoring the persistent --server flag override. The stored
// login token reads and opens sessions; a write acts as the developer through
// the login's JWT. A token given explicitly (a hook session, or CI through
// CITADEL_ACCESS_TOKEN or set-token) is used as given for both.
func newAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	return newClient(cmd, true)
}

// newLoginAPIClient always uses the stored login token, for the session calls
// its sessions:write scope exists for.
func newLoginAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	return newClient(cmd, false)
}

func newClient(cmd *cobra.Command, writesAsHuman bool) (*apiclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	opts := apiclient.Options{
		Server:     serverFlag(cmd),
		Verbose:    verboseFlag(cmd),
		DebugHTTP:  debugHTTPFlag(cmd),
		UserAgent:  "citadel-cli/" + Version,
		RetryOn401: rotateAccessTokenOn401Hook(cmd),
	}
	if writesAsHuman && os.Getenv("CITADEL_ACCESS_TOKEN") == "" && cfg.AgentID != "" && cfg.RefreshToken != "" {
		opts.WriteToken = humanWriteToken(cmd, cfg)
	}
	return apiclient.New(cfg, opts)
}

// humanWriteToken exchanges the refresh token once per command, since each
// exchange rotates it.
func humanWriteToken(cmd *cobra.Command, cfg clicfg.Config) func(context.Context) (string, error) {
	var once sync.Once
	var token string
	var err error
	return func(ctx context.Context) (string, error) {
		once.Do(func() {
			token, err = humanAccessToken(ctx, cfg, cfg.ResolveServerURL(serverFlag(cmd)))
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli: writing with the stored agent token: %v\n", err)
			}
		})
		return token, err
	}
}

// newHumanAPIClient is for routes closed to agent tokens, such as the audit log:
// the stored login's agent token is swapped for a JWT acting as the developer.
// An explicit CITADEL_ACCESS_TOKEN, as in a hook session, is used as given so an
// agent never gains its owner's access.
func newHumanAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if os.Getenv("CITADEL_ACCESS_TOKEN") != "" || cfg.RefreshToken == "" {
		return newAPIClient(cmd)
	}
	serverURL := cfg.ResolveServerURL(serverFlag(cmd))
	token, err := humanAccessToken(contextForCommand(cmd), cfg, serverURL)
	if err != nil {
		return nil, err
	}
	cfg.ServerURL = serverURL
	cfg.AccessToken = token
	return apiclient.New(cfg, apiclient.Options{
		Server:    serverFlag(cmd),
		Verbose:   verboseFlag(cmd),
		DebugHTTP: debugHTTPFlag(cmd),
		UserAgent: "citadel-cli/" + Version,
	})
}

func publicAPIBaseURL(cmd *cobra.Command) (string, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	return apiclient.ResolveRESTServerURL(cfg.ResolveRawServerURL(serverFlag(cmd))), nil
}

func doPublicJSON(ctx context.Context, cmd *cobra.Command, method, path string, body, out any) error {
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
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, reqBody)
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
		raw := apiclient.ReadErrorBody(resp.Body)
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
		if os.Getenv("CITADEL_ACCESS_TOKEN") != "" {
			return "", nil
		}
		cfg, err := clicfg.Load()
		if err != nil {
			return "", fmt.Errorf("reload config for token rotation: %w", err)
		}
		agentIDStr := strings.TrimSpace(cfg.AgentID)
		if agentIDStr == "" {
			if strings.TrimSpace(cfg.RefreshToken) == "" {
				// JWT-only / legacy config — cannot rotate; let the client surface the original 401.
				return "", nil
			}
			var refreshedToken string
			var refreshErr error
			if err := clicfg.Update(func(cfg *clicfg.Config) error {
				if strings.TrimSpace(cfg.AgentID) != "" {
					return nil
				}
				refreshToken := strings.TrimSpace(cfg.RefreshToken)
				if refreshToken == "" {
					return nil
				}
				tokenResp, err := exchangeRefreshToken(ctx, cfg.ResolveServerURL(serverFlag(cmd)), refreshToken)
				if err != nil || strings.TrimSpace(tokenResp.AccessToken) == "" {
					if err == nil {
						err = errors.New("refresh token response missing access_token")
					}
					cfg.AccessToken = ""
					cfg.RefreshToken = ""
					cfg.ExpiresAt = time.Time{}
					refreshErr = err
					return nil
				}
				cfg.AccessToken = tokenResp.AccessToken
				if strings.TrimSpace(tokenResp.RefreshToken) != "" {
					cfg.RefreshToken = tokenResp.RefreshToken
				}
				if claims, claimsErr := claimsFromJWT(tokenResp.AccessToken); claimsErr == nil {
					cfg.ExpiresAt = expiryFromClaims(claims, cfg.ExpiresAt)
					if cfg.UserUUID == "" {
						cfg.UserUUID = userUUIDFromClaims(claims)
					}
				} else if tokenResp.ExpiresIn > 0 {
					cfg.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
				}
				refreshedToken = tokenResp.AccessToken
				return nil
			}); err != nil {
				return "", fmt.Errorf("update config after token refresh: %w", err)
			}
			if refreshErr != nil {
				return "", fmt.Errorf("%w: %w", errSessionExpired, refreshErr)
			}
			return refreshedToken, nil
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
		rotatedRefresh := ""
		if err := c.Post(ctx, "/agents/"+uid.String()+"/rotate-token", loginTokenRequest(), &newTok); err != nil {
			// An expired or refused agent token cannot rotate itself, but the stored
			// OAuth login can mint its replacement.
			if he, ok := errors.AsType[*apiclient.HTTPError](err); !ok || (he.StatusCode != http.StatusUnauthorized && he.StatusCode != http.StatusForbidden) {
				return "", fmt.Errorf("rotate agent token: %w", err)
			}
			var humanErr error
			newTok, rotatedRefresh, humanErr = rotateWithLogin(ctx, cmd, cfg, uid)
			if humanErr != nil {
				if clearErr := clicfg.Update(func(cfg *clicfg.Config) error {
					cfg.AccessToken = ""
					cfg.RefreshToken = ""
					cfg.ExpiresAt = time.Time{}
					return nil
				}); clearErr != nil {
					return "", fmt.Errorf("%w: clear credentials: %w", errSessionExpired, clearErr)
				}
				return "", fmt.Errorf("%w: %w", errSessionExpired, humanErr)
			}
		}
		if newTok.CleartextToken == "" {
			if err := clicfg.Update(func(cfg *clicfg.Config) error {
				cfg.AccessToken = ""
				cfg.RefreshToken = ""
				cfg.ExpiresAt = time.Time{}
				return nil
			}); err != nil {
				return "", fmt.Errorf("%w: clear credentials: %w", errSessionExpired, err)
			}
			return "", fmt.Errorf("%w: rotate token: empty cleartext_token in response", errSessionExpired)
		}
		if err := clicfg.Update(func(cfg *clicfg.Config) error {
			cfg.AccessToken = newTok.CleartextToken
			if rotatedRefresh != "" {
				cfg.RefreshToken = rotatedRefresh
			}
			if newTok.ExpiresAt != nil {
				cfg.ExpiresAt = *newTok.ExpiresAt
			}
			return nil
		}); err != nil {
			return "", fmt.Errorf("save config after token rotation: %w", err)
		}
		return newTok.CleartextToken, nil
	}
}

// rotateWithLogin mints a fresh agent token for agentID with a JWT from the stored
// refresh token, and returns the refresh token that exchange rotated to.
func rotateWithLogin(ctx context.Context, cmd *cobra.Command, cfg clicfg.Config, agentID uuid.UUID) (tokenWithCleartext, string, error) {
	var newTok tokenWithCleartext
	refresh := strings.TrimSpace(cfg.RefreshToken)
	if refresh == "" {
		return newTok, "", errors.New("no stored login to renew the agent token")
	}
	serverURL := cfg.ResolveServerURL(serverFlag(cmd))
	jwtResp, err := exchangeRefreshToken(ctx, serverURL, refresh)
	if err != nil {
		return newTok, "", err
	}
	if strings.TrimSpace(jwtResp.AccessToken) == "" {
		return newTok, "", errors.New("refresh token response missing access_token")
	}
	next := strings.TrimSpace(jwtResp.RefreshToken)
	if next == "" {
		next = refresh
	}
	work := cfg
	work.ServerURL = serverURL
	work.AccessToken = jwtResp.AccessToken
	c, err := apiclient.New(work, apiclient.Options{
		Server: serverFlag(cmd), Verbose: verboseFlag(cmd), DebugHTTP: debugHTTPFlag(cmd), UserAgent: "citadel-cli/" + Version,
	})
	if err != nil {
		return newTok, next, err
	}
	err = c.Post(ctx, "/agents/"+agentID.String()+"/rotate-token", loginTokenRequest(), &newTok)
	return newTok, next, err
}
