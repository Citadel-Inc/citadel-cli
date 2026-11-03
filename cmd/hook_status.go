package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var hookStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installed hook clients and the current agent session",
	Args:  cobra.NoArgs,
	RunE:  runHookStatus,
}

type hookInstall struct {
	Client string `json:"client"`
	Scope  string `json:"scope"`
	Path   string `json:"path"`
}

type hookStatus struct {
	Installed []hookInstall `json:"installed"`
	SessionID string        `json:"session_id,omitempty"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
	Revoked   bool          `json:"revoked,omitempty"`
	// SessionEnvLive is true when CITADEL_AGENT_SESSION is set and names a
	// session whose local state is neither revoked nor expired.
	SessionEnvLive bool `json:"session_env_live"`
	SessionEnvSet  bool `json:"session_env_set"`
}

// collectHookStatus reads settings files and the hook session state; it makes
// no network call.
func collectHookStatus(now time.Time) hookStatus {
	var st hookStatus
	for _, client := range []string{"claude-code", "cursor"} {
		for _, scope := range []string{"user", "project"} {
			path, err := hookSettingsPath(client, scope)
			if err != nil {
				continue
			}
			//nolint:gosec // path comes from the fixed client and scope tables
			raw, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(raw), "citadel-cli hook ") {
				st.Installed = append(st.Installed, hookInstall{Client: client, Scope: scope, Path: path})
			}
		}
	}
	st.SessionEnvSet = os.Getenv("CITADEL_AGENT_SESSION") != ""
	if dir := os.Getenv("CITADEL_HOOK_STATE"); dir != "" {
		if hs, ok := loadHookState(dir); ok {
			st.SessionID, st.Revoked = hs.SessionID, hs.Revoked
			if !hs.Revoked {
				exp := hs.ExpiresAt
				st.ExpiresAt = &exp
			}
		}
	}
	st.SessionEnvLive = st.SessionEnvSet && st.SessionID != "" && !st.Revoked && st.ExpiresAt != nil && st.ExpiresAt.After(now)
	return st
}

func runHookStatus(cmd *cobra.Command, _ []string) error {
	now := time.Now()
	st := collectHookStatus(now)
	out := cmd.OutOrStdout()
	if len(st.Installed) == 0 {
		_, _ = fmt.Fprintln(out, "hooks: not installed (run `citadel-cli hook install`)")
	}
	for _, in := range st.Installed {
		_, _ = fmt.Fprintf(out, "hooks: %s (%s scope) %s\n", in.Client, in.Scope, in.Path)
	}
	switch {
	case st.SessionID == "":
		_, _ = fmt.Fprintln(out, "session: none (CITADEL_HOOK_STATE is unset or has no state)")
	case st.Revoked:
		_, _ = fmt.Fprintf(out, "session: %s revoked\n", st.SessionID)
	case st.ExpiresAt != nil && st.ExpiresAt.After(now):
		_, _ = fmt.Fprintf(out, "session: %s expires %s (in %s)\n", st.SessionID, st.ExpiresAt.Format(time.RFC3339), time.Until(*st.ExpiresAt).Round(time.Second))
	default:
		_, _ = fmt.Fprintf(out, "session: %s expired %s\n", st.SessionID, st.ExpiresAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(out, "CITADEL_AGENT_SESSION: set=%t live=%t\n", st.SessionEnvSet, st.SessionEnvLive)
	return nil
}

func checkHooks() checkResult {
	const name = "hooks"
	st := collectHookStatus(time.Now())
	if st.SessionEnvSet && !st.SessionEnvLive {
		return checkResult{name, statusWarn, "CITADEL_AGENT_SESSION is set but the session is revoked, expired or has no local state; start a new agent session"}
	}
	if len(st.Installed) == 0 {
		return checkResult{name, statusPass, "no agent hooks installed (optional: `citadel-cli hook install`)"}
	}
	clients := make([]string, len(st.Installed))
	for i, in := range st.Installed {
		clients[i] = in.Client + "/" + in.Scope
	}
	return checkResult{name, statusPass, "installed for " + strings.Join(clients, ", ")}
}
