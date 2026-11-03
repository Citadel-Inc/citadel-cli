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
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

// hookGitCredentialCmd is a git credential helper (git runs it as
// `<helper> get`). It answers from the session token in the environment, so the
// token is never written to git config or argv.
var hookGitCredentialCmd = &cobra.Command{
	Use:    "git-credential <operation>",
	Short:  "Git credential helper backed by the agent session token",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runHookGitCredential,
}

var hookSessionEndCmd = &cobra.Command{
	Use:   "session-end",
	Short: "Harness hook: end the agent session and delete its key",
	Args:  cobra.NoArgs,
	RunE:  runHookSessionEnd,
}

// hookGitSSHCmd is GIT_SSH_COMMAND for hook sessions. git appends its ssh
// arguments after the key path, so flag parsing is off.
var hookGitSSHCmd = &cobra.Command{
	Use:                "git-ssh <key> [ssh args...]",
	Short:              "SSH wrapper that offers the session key only to Citadel hosts",
	Hidden:             true,
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runHookGitSSH,
}

type hookInput struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// hookState is also how a long-running session finds its current token, since
// a hook cannot change the environment of the running harness.
type hookState struct {
	SessionID string    `json:"session_id"`
	AgentName string    `json:"agent_name"`
	ExpiresAt time.Time `json:"expires_at"`
	Token     string    `json:"token,omitempty"`
	Scopes    []string  `json:"scopes,omitempty"`
	// Revoked sessions stay revoked for the rest of the harness session.
	Revoked bool `json:"revoked,omitempty"`
}

// useSessionToken lets every CLI call in a hook session use the token from
// session state: the env var exported at session start goes stale after a
// reissue.
func useSessionToken() {
	dir := os.Getenv("CITADEL_HOOK_STATE")
	if os.Getenv("CITADEL_AGENT_SESSION") == "" || dir == "" {
		return
	}
	if st, ok := loadHookState(dir); ok && st.Token != "" {
		_ = os.Setenv("CITADEL_ACCESS_TOKEN", st.Token)
	}
}

// hookAPIClient authenticates as the developer's stored login. Inside a hook
// session CITADEL_ACCESS_TOKEN is the session token, which cannot manage
// sessions, so it is dropped here.
func hookAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	if os.Getenv("CITADEL_AGENT_SESSION") != "" {
		_ = os.Unsetenv("CITADEL_ACCESS_TOKEN")
	}
	return newAPIClient(cmd)
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

// citadelHostSet is the defaults, CITADEL_GIT_HOSTS, and the configured
// server's host.
func citadelHostSet(cmd *cobra.Command) map[string]struct{} {
	hosts := mergeCitadelHosts()
	if cfg, err := clicfg.Load(); err == nil {
		if u, err := url.Parse(cfg.ResolveServerURL(serverFlag(cmd))); err == nil && u.Hostname() != "" {
			hosts[strings.ToLower(u.Hostname())] = struct{}{}
		}
	}
	return hosts
}

// citadelOrigin returns dir's origin remote when it is on a Citadel git host.
func citadelOrigin(ctx context.Context, cmd *cobra.Command, dir string) (string, bool) {
	remote, err := gitOriginURL(ctx, dir)
	if err != nil {
		return "", false
	}
	_, _, err = parseOriginIntoRepo(remote, citadelHostSet(cmd))
	return remote, err == nil
}

// httpsCredentialEnv registers the credential helper for an https origin's
// host through GIT_CONFIG_COUNT, appending after any entries already set. It
// returns nil for a non-https origin.
func httpsCredentialEnv(remote string) [][2]string {
	u, err := url.Parse(remote)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil
	}
	n, _ := strconv.Atoi(os.Getenv("GIT_CONFIG_COUNT"))
	n = max(n, 0)
	exe, err := os.Executable()
	if err != nil {
		exe = "citadel-cli"
	}
	key := "credential.https://" + u.Host + ".helper"
	// The empty first value clears helpers inherited from the user's config.
	return [][2]string{
		{"GIT_CONFIG_COUNT", strconv.Itoa(n + 2)},
		{"GIT_CONFIG_KEY_" + strconv.Itoa(n), key},
		{"GIT_CONFIG_VALUE_" + strconv.Itoa(n), ""},
		{"GIT_CONFIG_KEY_" + strconv.Itoa(n+1), key},
		{"GIT_CONFIG_VALUE_" + strconv.Itoa(n+1), "!" + shellQuote(exe) + " hook git-credential"},
	}
}

func runHookGitCredential(cmd *cobra.Command, args []string) error {
	if args[0] != "get" {
		return nil
	}
	token := os.Getenv("CITADEL_ACCESS_TOKEN")
	if token == "" {
		return nil
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "username=x\npassword=%s\n", token)
	return err
}

// sshArgsWithValue are the ssh flags whose value is the next argument.
const sshArgsWithValue = "BbcDEeFIiJLlmOopQRSWw"

// sshTargetHost returns the host of an ssh invocation as git builds it
// (options, then [user@]host, then the remote command).
func sshTargetHost(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			if i+1 < len(args) {
				return hostOfTarget(args[i+1])
			}
			return ""
		case len(a) > 1 && a[0] == '-':
			if len(a) == 2 && strings.ContainsRune(sshArgsWithValue, rune(a[1])) {
				i++
			}
		default:
			return hostOfTarget(a)
		}
	}
	return ""
}

func hostOfTarget(t string) string {
	if i := strings.LastIndex(t, "@"); i >= 0 {
		t = t[i+1:]
	}
	return strings.ToLower(strings.Trim(t, "[]"))
}

func runHookGitSSH(cmd *cobra.Command, args []string) error {
	key, rest := args[0], args[1:]
	if _, ok := citadelHostSet(cmd)[sshTargetHost(rest)]; ok {
		rest = append([]string{"-i", key, "-o", "IdentitiesOnly=yes"}, rest...)
	}
	c := exec.CommandContext(cmd.Context(), "ssh", rest...) //nolint:gosec // ssh is the program git itself would have run
	c.Stdin, c.Stdout, c.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
	err := c.Run()
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		os.Exit(ee.ExitCode())
	}
	return err
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
	if st, ok := loadHookState(dir); ok && st.Revoked {
		return nil
	}
	remote, ok := citadelOrigin(ctx, cmd, in.CWD)
	if !ok {
		return nil
	}
	scopes, _ := cmd.Flags().GetStringSlice("scopes")
	if len(scopes) == 0 {
		scopes = defaultHookScopes
	}
	sess, err := openAgentSession(ctx, cmd, dir, scopes)
	if err != nil {
		_ = os.RemoveAll(dir)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: no agent session: %v\n", err)
		emitSessionContext(cmd, "Citadel agent identity is unavailable for this session ("+err.Error()+"). Git will use the developer's own SSH key.")
		return nil
	}
	if envFile := os.Getenv("CLAUDE_ENV_FILE"); envFile != "" {
		key := filepath.Join(dir, "id_ed25519")
		exe, err := os.Executable()
		if err != nil {
			exe = "citadel-cli"
		}
		vars := [][2]string{
			{"GIT_SSH_COMMAND", shellQuote(exe) + " hook git-ssh " + shellQuote(key)},
			{"CITADEL_AGENT_SESSION", sess.SessionID},
			{"CITADEL_HOOK_STATE", dir},
			{"CITADEL_ACCESS_TOKEN", sess.Token},
		}
		vars = append(vars, httpsCredentialEnv(remote)...)
		if err := appendEnvFile(envFile, vars); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: write env file: %v\n", err)
		}
	}
	emitSessionContext(cmd, fmt.Sprintf(
		"This session acts as Citadel agent %q on src.land: git over SSH or HTTPS and citadel-cli calls use its short-lived credentials. Code questions can use `citadel-cli kg` commands.",
		sess.AgentName))
	return nil
}

// openAgentSession opens a session on a fresh key written over the one at the
// same path, so the running session's GIT_SSH_COMMAND stays valid while the
// server never sees a fingerprint twice.
func openAgentSession(ctx context.Context, cmd *cobra.Command, dir string, scopes []string) (agentSessionResp, error) {
	var sess agentSessionResp
	c, err := hookAPIClient(cmd)
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
		"scopes":         scopes,
	}
	if err := c.Post(ctx, "/agents/"+url.PathEscape(agentSessionName())+"/sessions", body, &sess); err != nil {
		if he, ok := errors.AsType[*apiclient.HTTPError](err); ok && he.StatusCode == 403 && strings.Contains(he.Body, `"forbidden"`) {
			return sess, errors.New("this login predates hook sessions (token lacks sessions:write); run `citadel-cli auth login` again")
		}
		return sess, err
	}
	if sess.SessionID == "" || sess.Token == "" {
		return sess, errors.New("server returned an incomplete session")
	}
	return sess, saveHookState(dir, hookState{SessionID: sess.SessionID, AgentName: sess.AgentName, ExpiresAt: sess.ExpiresAt, Token: sess.Token, Scopes: scopes})
}

func runHookRenew(cmd *cobra.Command, _ []string) error {
	in := readHookInput(cmd)
	live := false
	if dir := sessionDir(in.SessionID); dir != "" {
		live = renewSession(cmd, dir)
	}
	if require, _ := cmd.Flags().GetBool("require"); require && !live {
		denyPushWithoutSession(cmd, in)
	}
	return nil
}

// renewSession renews or reissues the session when it is near expiry and
// reports whether a live session remains.
func renewSession(cmd *cobra.Command, dir string) bool {
	st, ok := loadHookState(dir)
	if !ok || st.Revoked {
		return false
	}
	if time.Until(st.ExpiresAt) > renewWindow {
		return true
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 8*time.Second)
	defer cancel()
	var out struct {
		ExpiresAt time.Time `json:"expires_at"`
	}
	c, err := hookAPIClient(cmd)
	if err == nil {
		err = c.Post(ctx, "/agent-sessions/"+url.PathEscape(st.SessionID)+"/renew", nil, &out)
	}
	switch {
	case apiclient.IsStatus(err, 410):
		revokeLocalSession(cmd, dir, st)
		return false
	case apiclient.IsStatus(err, 409):
		return reissueSession(ctx, cmd, dir, st)
	case err != nil:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: renew failed: %v\n", err)
		return time.Until(st.ExpiresAt) > 0
	}
	st.ExpiresAt = out.ExpiresAt
	_ = saveHookState(dir, st)
	return true
}

// revokeLocalSession drops the key and token of a revoked session and keeps
// the state as a tombstone so no later hook revives it.
func revokeLocalSession(cmd *cobra.Command, dir string, st hookState) {
	_ = os.Remove(filepath.Join(dir, "id_ed25519"))
	_ = saveHookState(dir, hookState{SessionID: st.SessionID, AgentName: st.AgentName, Revoked: true})
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "citadel-cli hook: this agent session was revoked; git and citadel-cli calls in this Claude Code session are no longer authenticated as the agent. Start a new Claude Code session to get a fresh one.")
}

// reissueSession replaces an ended session (idle past its 15-minute window)
// with a new one on a fresh key and the same scopes. The new token reaches later CLI calls
// through the state file; the harness environment cannot be updated from a hook.
func reissueSession(ctx context.Context, cmd *cobra.Command, dir string, old hookState) bool {
	scopes := old.Scopes
	if len(scopes) == 0 {
		scopes = defaultHookScopes
	}
	sess, err := openAgentSession(ctx, cmd, dir, scopes)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: agent session ended or expired and could not be reissued (%v); git and citadel-cli calls in this session are no longer authenticated as the agent\n", err)
		return false
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "citadel-cli hook: agent session expired; reissued as %s\n", sess.SessionID)
	return true
}

var gitPushCmd = regexp.MustCompile(`(^|[;&|(\s])git(\s+-\S+(\s+\S+)?)*\s+push(\s|$)`)

// denyPushWithoutSession emits the PreToolUse deny decision for a Bash
// `git push` run in a Citadel repository.
func denyPushWithoutSession(cmd *cobra.Command, in hookInput) {
	if in.ToolName != "Bash" || !gitPushCmd.MatchString(in.ToolInput.Command) {
		return
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
	defer cancel()
	if _, ok := citadelOrigin(ctx, cmd, in.CWD); !ok {
		return
	}
	_ = json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": "No live Citadel agent session: git push to src.land is blocked (hook installed with --require). Restart the session after `citadel-cli auth login`.",
	}})
}

func runHookSessionEnd(cmd *cobra.Command, _ []string) error {
	dir := sessionDir(readHookInput(cmd).SessionID)
	if dir == "" {
		return nil
	}
	if st, ok := loadHookState(dir); ok {
		ctx, cancel := context.WithTimeout(cmd.Context(), 8*time.Second)
		defer cancel()
		c, err := hookAPIClient(cmd)
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
