package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

var authGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Open a short-lived SSH session for git as yourself",
	Long: `Registers a fresh SSH key with the server as a human session (8 hours, your
own grants) and points git at it, for organizations that refuse long-lived SSH
keys for pushes. The key is offered only to Citadel hosts; other hosts keep
using your usual keys. Run it again when the session expires.`,
	Args: cobra.NoArgs,
	RunE: runAuthGit,
}

func authGitKeyDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		base = cache
	}
	return filepath.Join(base, "citadel", "git"), nil
}

// humanAccessToken trades the refresh token kept by `auth login` for a JWT that
// acts as the developer, for routes a stored agent token is refused on.
func humanAccessToken(ctx context.Context, cfg clicfg.Config, serverURL string) (string, error) {
	refresh := strings.TrimSpace(cfg.RefreshToken)
	if refresh == "" {
		return "", errors.New("no stored login to act as you; run `citadel-cli auth login` first")
	}
	tok, err := exchangeRefreshToken(ctx, serverURL, refresh)
	if err == nil && strings.TrimSpace(tok.AccessToken) == "" {
		err = errors.New("empty access token")
	}
	if err != nil {
		return "", fmt.Errorf("refresh login (run `citadel-cli auth login` again): %w", err)
	}
	if next := strings.TrimSpace(tok.RefreshToken); next != "" {
		if err := clicfg.Update(func(c *clicfg.Config) error { c.RefreshToken = next; return nil }); err != nil {
			return "", fmt.Errorf("save refreshed login: %w", err)
		}
	}
	return tok.AccessToken, nil
}

func runAuthGit(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(contextForCommand(cmd), 30*time.Second)
	defer cancel()
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	serverURL := cfg.ResolveServerURL(serverFlag(cmd))
	accessToken, err := humanAccessToken(ctx, cfg, serverURL)
	if err != nil {
		return err
	}

	dir, err := authGitKeyDir()
	if err != nil {
		return err
	}
	pub, err := writeSessionKey(dir)
	if err != nil {
		return err
	}
	work := cfg
	work.ServerURL = serverURL
	work.AccessToken = accessToken
	c, err := apiclient.New(work, apiclient.Options{Server: serverFlag(cmd), Verbose: verboseFlag(cmd), DebugHTTP: debugHTTPFlag(cmd), UserAgent: "citadel-cli/" + Version})
	if err != nil {
		return err
	}
	var sess struct {
		SessionID string    `json:"session_id"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := c.Post(ctx, "/git-sessions", map[string]any{"client": "citadel-cli", "ssh_public_key": pub}, &sess); err != nil {
		return fmt.Errorf("open git session: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "citadel-cli"
	}
	sshCmd := shellQuote(exe) + " hook git-ssh " + shellQuote(filepath.Join(dir, "id_ed25519"))
	out := cmd.OutOrStdout()
	cur, _ := exec.CommandContext(ctx, "git", "config", "--global", "--get", "core.sshCommand").Output()
	if have := strings.TrimSpace(string(cur)); have != "" && !strings.Contains(have, "hook git-ssh") {
		_, _ = fmt.Fprintf(out, "Git session %s open until %s, but core.sshCommand is already set to %q; not replacing it. To use the session set:\n  git config --global core.sshCommand %s\n", sess.SessionID, sess.ExpiresAt.Local().Format(time.RFC3339), have, shellQuote(sshCmd))
		return nil
	}
	if err := exec.CommandContext(ctx, "git", "config", "--global", "core.sshCommand", sshCmd).Run(); err != nil { //nolint:gosec // fixed git invocation, sshCmd is one argv element
		return fmt.Errorf("set git core.sshCommand: %w", err)
	}
	_, _ = fmt.Fprintf(out, "Git session %s open until %s; git now pushes over SSH with a short-lived key.\n", sess.SessionID, sess.ExpiresAt.Local().Format(time.RFC3339))
	return nil
}
