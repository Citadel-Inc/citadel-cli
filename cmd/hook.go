package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	safefs "github.com/Rethunk-Tech/go-safefs"
	"github.com/spf13/cobra"
)

// HookCmd is the top-level `citadel-cli hook` command.
var HookCmd = &cobra.Command{
	Use:     "hook",
	GroupID: "ops",
	Short:   "Give each coding-agent session its own agent identity",
	Long: `Installs and serves coding-agent hooks. Once installed, every Claude Code
session in a src.land repository gets its own short-lived agent identity,
ephemeral SSH key and token, with no token handling. Sessions in other
repositories are untouched.

The session-start, renew and session-end verbs are invoked by the agent
harness, not by hand.`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Register the session hooks in the agent's settings file",
	Long: `Merges the SessionStart, PreToolUse and SessionEnd hooks into the Claude Code
settings file, leaving every other key and hook as is. Re-running is a no-op.

Examples:
  citadel-cli hook install
  citadel-cli hook install --scope project
  citadel-cli hook install --scopes code:read,code:write,metadata:read
  citadel-cli hook install --require`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error { return runHookSettings(cmd, true) },
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the session hooks from the agent's settings file",
	Args:  cobra.NoArgs,
	RunE:  func(cmd *cobra.Command, _ []string) error { return runHookSettings(cmd, false) },
}

// defaultHookScopes cover git plus opening and discussing pull requests.
var defaultHookScopes = []string{
	"code:read", "code:write", "metadata:read",
	"pulls:read", "pulls:write", "issues:read", "issues:write",
}

var scopeToken = regexp.MustCompile(`^[a-z0-9_.:*-]+$`)

// hookEvents maps each Claude Code event to the base hook verb it runs;
// installed command lines append the flags chosen at install time.
var hookEvents = map[string]string{
	"SessionStart": "citadel-cli hook session-start",
	"PreToolUse":   "citadel-cli hook renew",
	"SessionEnd":   "citadel-cli hook session-end",
}

// hookTimeouts are in seconds; SessionEnd needs a raised timeout because the
// harness otherwise gives all its hooks a 1.5 s budget.
var hookTimeouts = map[string]int{"SessionStart": 20, "PreToolUse": 10, "SessionEnd": 10}

func registerHookCommands() {
	useSessionToken()
	for _, c := range []*cobra.Command{hookInstallCmd, hookUninstallCmd} {
		c.Flags().String("client", "claude-code", "Agent client to configure (claude-code)")
		c.Flags().String("scope", "user", "Settings scope: user (~/.claude/settings.json) or project (.claude/settings.json)")
	}
	hookInstallCmd.Flags().StringSlice("scopes", nil, "Agent session scopes to request (default "+strings.Join(defaultHookScopes, ",")+")")
	hookInstallCmd.Flags().Bool("require", false, "Deny `git push` to a Citadel host while no live agent session exists")
	hookSessionStartCmd.Flags().StringSlice("scopes", defaultHookScopes, "Agent session scopes to request")
	hookRenewCmd.Flags().Bool("require", false, "Deny `git push` to a Citadel host while no live agent session exists")
	HookCmd.AddCommand(hookInstallCmd, hookUninstallCmd, hookSessionStartCmd, hookRenewCmd, hookSessionEndCmd, hookGitCredentialCmd, hookGitSSHCmd)
}

func runHookSettings(cmd *cobra.Command, install bool) error {
	if client, _ := cmd.Flags().GetString("client"); client != "claude-code" {
		return fmt.Errorf("--client: unsupported %q (use claude-code)", client)
	}
	scope, _ := cmd.Flags().GetString("scope")
	path, err := claudeSettingsPath(scope)
	if err != nil {
		return err
	}
	commands := maps.Clone(hookEvents)
	if install {
		scopes, _ := cmd.Flags().GetStringSlice("scopes")
		for _, sc := range scopes {
			if !scopeToken.MatchString(sc) {
				return fmt.Errorf("--scopes: invalid scope %q", sc)
			}
		}
		if len(scopes) > 0 && strings.Join(scopes, ",") != strings.Join(defaultHookScopes, ",") {
			commands["SessionStart"] += " --scopes " + strings.Join(scopes, ",")
		}
		if require, _ := cmd.Flags().GetBool("require"); require {
			commands["PreToolUse"] += " --require"
		}
	}
	changed, err := mergeClaudeHooks(path, install, commands)
	if err != nil {
		return err
	}
	verb := map[bool]string{true: "installed in", false: "removed from"}[install]
	if !changed {
		verb = map[bool]string{true: "already installed in", false: "not present in"}[install]
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "citadel-cli hooks %s %s\n", verb, path)
	return nil
}

func claudeSettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "settings.json"), nil
	case "project":
		return filepath.Join(".claude", "settings.json"), nil
	}
	return "", fmt.Errorf("--scope: unknown %q (use user|project)", scope)
}

// mergeClaudeHooks adds (or removes) only this CLI's hook entries. It refuses
// to touch a file it cannot parse, and reports whether the file changed. An
// existing entry for the same verb with different flags is replaced.
func mergeClaudeHooks(path string, install bool, commands map[string]string) (bool, error) {
	settings := map[string]any{}
	mode := fs.FileMode(0o600)
	//nolint:gosec // The settings path is operator-selected.
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if !install {
			return false, nil
		}
	case err != nil:
		return false, err
	default:
		if st, serr := os.Stat(path); serr == nil {
			mode = st.Mode().Perm()
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &settings); err != nil {
				return false, fmt.Errorf("%s is not valid JSON; fix it and rerun: %w", path, err)
			}
		}
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		if _, present := settings["hooks"]; present {
			return false, fmt.Errorf("%s: \"hooks\" is not an object; fix it and rerun", path)
		}
		hooks = map[string]any{}
	}

	changed := false
	for event, base := range hookEvents {
		command := commands[event]
		groups, _ := hooks[event].([]any)
		if _, present := hooks[event]; present && groups == nil {
			return false, fmt.Errorf("%s: hooks.%s is not an array; fix it and rerun", path, event)
		}
		had := groupsHaveCommand(groups, command)
		stale := groupsHaveVerb(groups, base)
		switch {
		case install && !had:
			groups = stripVerb(groups, base)
			groups = append(groups, map[string]any{"hooks": []any{map[string]any{
				"type": "command", "command": command, "timeout": hookTimeouts[event],
			}}})
			changed = true
		case !install && stale:
			groups = stripVerb(groups, base)
			changed = true
		}
		if len(groups) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = groups
		}
	}
	if !changed {
		return false, nil
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // settings dir is user-visible config
		return false, err
	}
	return true, safefs.WriteFileAtomic(path, append(out, '\n'), mode)
}

func isVerb(command any, base string) bool {
	c, _ := command.(string)
	return c == base || strings.HasPrefix(c, base+" ")
}

func groupsHaveCommand(groups []any, command string) bool {
	return anyHandler(groups, func(c any) bool { return c == command })
}

func groupsHaveVerb(groups []any, base string) bool {
	return anyHandler(groups, func(c any) bool { return isVerb(c, base) })
}

func anyHandler(groups []any, match func(command any) bool) bool {
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		hs, _ := gm["hooks"].([]any)
		for _, h := range hs {
			if hm, _ := h.(map[string]any); match(hm["command"]) {
				return true
			}
		}
	}
	return false
}

// stripVerb drops handlers running the verb (with any flags) and any group left
// empty by that.
func stripVerb(groups []any, base string) []any {
	var out []any
	for _, g := range groups {
		gm, ok := g.(map[string]any)
		if !ok {
			out = append(out, g)
			continue
		}
		hs, _ := gm["hooks"].([]any)
		var keep []any
		for _, h := range hs {
			if hm, _ := h.(map[string]any); !isVerb(hm["command"], base) {
				keep = append(keep, h)
			}
		}
		if len(keep) == len(hs) {
			out = append(out, g)
		} else if len(keep) > 0 {
			gm["hooks"] = keep
			out = append(out, gm)
		}
	}
	return out
}
