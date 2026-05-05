package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/mcpclient"
)

// DoctorCmd verifies the local environment is healthy enough for the
// rest of the CLI to operate. Each check renders a green/yellow/red
// glyph + one-line summary; failures do not abort the run so operators
// get the full picture.
var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify CLI environment health (server, auth, MCP, config)",
	Long: `Runs a sequence of read-only checks against the configured server,
authentication state, MCP endpoint, and on-disk config file. Each line
prints PASS / WARN / FAIL with one-line context. Exits non-zero if any
FAIL fires.

Checks performed:
  - Server reachable (HTTP HEAD / GET against the configured base URL)
  - Auth token present + not expired (claim-only inspection; no round trip)
  - MCP endpoint reachable (initialize handshake)
  - ~/.config/citadel/config.toml mode 0600 (UNIX only)`,
	RunE: runDoctor,
}

type checkStatus int

const (
	statusPass checkStatus = iota
	statusWarn
	statusFail
)

type checkResult struct {
	name   string
	status checkStatus
	detail string
}

func (c checkResult) String() string {
	var glyph string
	switch c.status {
	case statusPass:
		glyph = "PASS"
	case statusWarn:
		glyph = "WARN"
	case statusFail:
		glyph = "FAIL"
	}
	return fmt.Sprintf("[%s] %s — %s", glyph, c.name, c.detail)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	cfg, _ := clicfg.Load()
	server := cfg.ResolveServerURL(serverFlag(cmd))

	results := []checkResult{
		checkServer(cmd.Context(), server),
		checkAuthToken(cfg),
		checkMCP(cmd.Context(), cfg, server, mcpclient.Options{Verbose: verboseFlag(cmd), DebugHTTP: debugHTTPFlag(cmd)}),
		checkConfigPerms(),
	}
	for _, r := range results {
		fmt.Println(r.String())
	}
	if anyFailed(results) {
		return errors.New("one or more checks failed")
	}
	return nil
}

func anyFailed(results []checkResult) bool {
	for _, r := range results {
		if r.status == statusFail {
			return true
		}
	}
	return false
}

func checkServer(ctx context.Context, base string) checkResult {
	const name = "server-reachable"
	if base == "" {
		return checkResult{name, statusFail, "no server URL configured (set CITADEL_SERVER or pass --server)"}
	}
	c := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/healthz", nil)
	if err != nil {
		return checkResult{name, statusFail, fmt.Sprintf("build request: %v", err)}
	}
	resp, err := c.Do(req)
	if err != nil {
		return checkResult{name, statusFail, fmt.Sprintf("%s unreachable: %v", base, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	// Any 2xx/3xx/4xx tells us the server is alive (auth-gated paths return
	// 401, which still proves reachability). 5xx is a real problem.
	if resp.StatusCode >= 500 {
		return checkResult{name, statusFail, fmt.Sprintf("%s returned HTTP %d", base, resp.StatusCode)}
	}
	return checkResult{name, statusPass, fmt.Sprintf("%s reachable (HTTP %d)", base, resp.StatusCode)}
}

func checkAuthToken(cfg clicfg.Config) checkResult {
	const name = "auth-token"
	if cfg.AccessToken == "" {
		return checkResult{name, statusFail, "no access token; run `citadel-cli auth login` or `auth set-token`"}
	}
	claims, err := claimsFromJWT(cfg.AccessToken)
	if err != nil {
		// Could be an opaque agent token (post-bootstrap shape). Treat as
		// PASS since the server-reachable + MCP checks will exercise it.
		return checkResult{name, statusPass, "non-JWT token present (likely opaque agent token)"}
	}
	exp := expiryFromClaims(claims, time.Time{})
	if exp.IsZero() {
		return checkResult{name, statusWarn, "JWT has no exp claim"}
	}
	remaining := time.Until(exp)
	if remaining < 0 {
		return checkResult{name, statusFail, fmt.Sprintf("token expired %s ago — run `citadel-cli auth login`", (-remaining).Round(time.Second))}
	}
	if remaining < 5*time.Minute {
		return checkResult{name, statusWarn, fmt.Sprintf("token expires in %s — run `citadel-cli auth login` soon", remaining.Round(time.Second))}
	}
	return checkResult{name, statusPass, fmt.Sprintf("authenticated (expires in %s)", remaining.Round(time.Second))}
}

func checkMCP(ctx context.Context, cfg clicfg.Config, server string, opts mcpclient.Options) checkResult {
	const name = "mcp-endpoint"
	token := pickToken("", cfg.AccessToken)
	if token == "" {
		return checkResult{name, statusWarn, "skipped: no auth token (cannot test MCP without credentials)"}
	}
	mcpURL := resolveMCPURL(server)
	c := mcpclient.New(mcpURL, token, 5*time.Second, opts)
	if err := c.Initialize(ctx); err != nil {
		if mcpclient.IsUnauthorized(err) {
			return checkResult{name, statusFail, "MCP rejected our token (401) — run `citadel-cli auth login`"}
		}
		return checkResult{name, statusFail, fmt.Sprintf("%s unreachable: %v", mcpURL, err)}
	}
	return checkResult{name, statusPass, fmt.Sprintf("%s initialized", mcpURL)}
}

func checkConfigPerms() checkResult {
	const name = "config-perms"
	home, err := os.UserHomeDir()
	if err != nil {
		return checkResult{name, statusWarn, fmt.Sprintf("cannot resolve home dir: %v", err)}
	}
	configPath := home + "/.config/citadel/config.toml"
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		configPath = v + "/citadel/config.toml"
	}
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return checkResult{name, statusWarn, "no config file (yet) — first run? run `citadel-cli auth login`"}
		}
		return checkResult{name, statusWarn, fmt.Sprintf("stat: %v", err)}
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		return checkResult{name, statusFail, fmt.Sprintf("%s mode is %#o; should be 0600", configPath, mode)}
	}
	return checkResult{name, statusPass, fmt.Sprintf("%s mode %#o", configPath, mode)}
}
