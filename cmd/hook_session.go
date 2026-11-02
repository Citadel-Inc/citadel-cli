package cmd

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	safefs "github.com/Rethunk-Tech/go-safefs"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

// renewWindow is how close to expiry a session must be before the PreToolUse
// hook spends a network call on renewal.
const renewWindow = 5 * time.Minute

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Harness hook: open an agent session for this coding session",
	Args:  cobra.NoArgs,
	RunE:  runHookSessionStart,
}

var hookRenewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Harness hook: renew the agent session when it nears expiry",
	Args:  cobra.NoArgs,
	RunE:  runHookRenew,
}

var hookSessionEndCmd = &cobra.Command{
	Use:   "session-end",
	Short: "Harness hook: end the agent session and delete its key",
	Args:  cobra.NoArgs,
	RunE:  runHookSessionEnd,
}

type hookInput struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
}

type hookState struct {
	SessionID string    `json:"session_id"`
	AgentName string    `json:"agent_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

type agentSessionResp struct {
	SessionID string    `json:"session_id"`
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// readHookInput never fails the hook: a missing or malformed payload yields a
// zero input, which every verb treats as nothing to do.
func readHookInput(cmd *cobra.Command) hookInput {
	var in hookInput
	_ = json.NewDecoder(io.LimitReader(cmd.InOrStdin(), 1<<20)).Decode(&in)
	return in
}

// sessionDir returns the per-session directory, or "" when the id is unusable
// as a path element.
func sessionDir(sessionID string) string {
	if sessionID == "" || sessionID != filepath.Base(sessionID) || sessionID == "." || sessionID == ".." {
		return ""
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		base = cache
	}
	return filepath.Join(base, "citadel", "sessions", sessionID)
}

func loadHookState(dir string) (hookState, bool) {
	var st hookState
	raw, err := os.ReadFile(filepath.Join(dir, "state.json")) //nolint:gosec // dir is derived from the session id
	if err != nil || json.Unmarshal(raw, &st) != nil || st.SessionID == "" {
		return hookState{}, false
	}
	return st, true
}

func saveHookState(dir string, st hookState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return safefs.WriteFileAtomic(filepath.Join(dir, "state.json"), raw, 0o600)
}

var nonHostChars = regexp.MustCompile(`[^a-z0-9-]+`)

func agentSessionName() string {
	host, _ := os.Hostname()
	host, _, _ = strings.Cut(strings.ToLower(host), ".")
	host = strings.Trim(nonHostChars.ReplaceAllString(host, "-"), "-")
	if host == "" {
		host = "host"
	}
	return "claude-code-" + host
}

// isCitadelRepo reports whether dir's origin remote is on a Citadel git host
// (the defaults, CITADEL_GIT_HOSTS, or the configured server's host).
func isCitadelRepo(ctx context.Context, cmd *cobra.Command, dir string) bool {
	remote, err := gitOriginURL(ctx, dir)
	if err != nil {
		return false
	}
	hosts := mergeCitadelHosts()
	if cfg, err := clicfg.Load(); err == nil {
		if u, err := url.Parse(cfg.ResolveServerURL(serverFlag(cmd))); err == nil && u.Hostname() != "" {
			hosts[strings.ToLower(u.Hostname())] = struct{}{}
		}
	}
	_, _, err = parseOriginIntoRepo(remote, hosts)
	return err == nil
}

// writeSessionKey creates the ephemeral keypair and returns the OpenSSH public key line.
func writeSessionKey(dir string) (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	block, err := ssh.MarshalPrivateKey(priv, "citadel-agent-session")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := safefs.WriteFileAtomic(filepath.Join(dir, "id_ed25519"), pem.EncodeToMemory(block), 0o600); err != nil {
		return "", err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))), nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func appendEnvFile(path string, vars [][2]string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600) //nolint:gosec // path is the harness-provided CLAUDE_ENV_FILE
	if err != nil {
		return err
	}
	for _, kv := range vars {
		if _, err := fmt.Fprintf(f, "export %s=%s\n", kv[0], shellQuote(kv[1])); err != nil {
			_ = f.Close()
			return err
		}
	}
	return f.Close()
}

func emitSessionContext(cmd *cobra.Command, text string) {
	_ = json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "SessionStart",
		"additionalContext": text,
	}})
}

func runHookSessionStart(cmd *cobra.Command, _ []string) error {
	in := readHookInput(cmd)
	dir := sessionDir(in.SessionID)
	if dir == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()
	if !isCitadelRepo(ctx, cmd, in.CWD) {
		return nil
	}
	sess, err := openAgentSession(ctx, cmd, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: no agent session: %v\n", err)
		emitSessionContext(cmd, "Citadel agent identity is unavailable for this session ("+err.Error()+"). Git will use the developer's own SSH key.")
		return nil
	}
	if envFile := os.Getenv("CLAUDE_ENV_FILE"); envFile != "" {
		key := filepath.Join(dir, "id_ed25519")
		if err := appendEnvFile(envFile, [][2]string{
			{"GIT_SSH_COMMAND", "ssh -i " + shellQuote(key) + " -o IdentitiesOnly=yes"},
			{"CITADEL_AGENT_SESSION", sess.SessionID},
			{"CITADEL_ACCESS_TOKEN", sess.Token},
		}); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: write env file: %v\n", err)
		}
	}
	emitSessionContext(cmd, fmt.Sprintf(
		"This session acts as Citadel agent %q on src.land: git over SSH and citadel-cli calls use its short-lived credentials. Code questions can use `citadel-cli kg` commands.",
		sess.AgentName))
	return nil
}

func openAgentSession(ctx context.Context, cmd *cobra.Command, dir string) (agentSessionResp, error) {
	var sess agentSessionResp
	c, err := newAPIClient(cmd)
	if err != nil {
		return sess, err
	}
	pub, err := writeSessionKey(dir)
	if err != nil {
		return sess, err
	}
	body := map[string]any{
		"client":         "claude-code",
		"ssh_public_key": pub,
		"scopes":         []string{"code:read", "code:write", "metadata:read"},
	}
	if err := c.Post(ctx, "/agents/"+url.PathEscape(agentSessionName())+"/sessions", body, &sess); err != nil {
		return sess, err
	}
	if sess.SessionID == "" || sess.Token == "" {
		return sess, errors.New("server returned an incomplete session")
	}
	return sess, saveHookState(dir, hookState{SessionID: sess.SessionID, AgentName: sess.AgentName, ExpiresAt: sess.ExpiresAt})
}

func runHookRenew(cmd *cobra.Command, _ []string) error {
	dir := sessionDir(readHookInput(cmd).SessionID)
	if dir == "" {
		return nil
	}
	st, ok := loadHookState(dir)
	if !ok || time.Until(st.ExpiresAt) > renewWindow {
		return nil
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 8*time.Second)
	defer cancel()
	var out struct {
		ExpiresAt time.Time `json:"expires_at"`
	}
	c, err := newAPIClient(cmd)
	if err == nil {
		err = c.Post(ctx, "/agent-sessions/"+url.PathEscape(st.SessionID)+"/renew", nil, &out)
	}
	switch {
	case apiclient.IsStatus(err, 409):
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "citadel-cli hook: agent session ended or expired; git and citadel-cli calls in this session are no longer authenticated as the agent")
	case err != nil:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: renew failed: %v\n", err)
	default:
		st.ExpiresAt = out.ExpiresAt
		_ = saveHookState(dir, st)
	}
	return nil
}

func runHookSessionEnd(cmd *cobra.Command, _ []string) error {
	dir := sessionDir(readHookInput(cmd).SessionID)
	if dir == "" {
		return nil
	}
	if st, ok := loadHookState(dir); ok {
		ctx, cancel := context.WithTimeout(cmd.Context(), 8*time.Second)
		defer cancel()
		c, err := newAPIClient(cmd)
		if err == nil {
			err = c.Delete(ctx, "/agent-sessions/"+url.PathEscape(st.SessionID)+"?reason=session_end")
		}
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: end session: %v\n", err)
		}
	}
	_ = os.RemoveAll(dir)
	return nil
}
