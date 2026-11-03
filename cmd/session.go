package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SessionCmd is the top-level `citadel-cli session` command.
var SessionCmd = &cobra.Command{
	Use:     "session",
	GroupID: "ops",
	Short:   "List and revoke your agent sessions",
	Long: `Lists the hook sessions opened under your login and revokes one by id.
Runs as the developer's stored login, never as a session token.`,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your agent sessions, newest first",
	Args:  cobra.NoArgs,
	RunE:  runSessionList,
}

var sessionRevokeCmd = &cobra.Command{
	Use:   "revoke <session-id>",
	Short: "End an agent session so its token and SSH key stop working",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionRevoke,
}

type sessionRow struct {
	ID        string     `json:"id"`
	AgentName string     `json:"agent_name"`
	Client    string     `json:"client"`
	Kind      string     `json:"kind"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	EndReason *string    `json:"end_reason,omitempty"`
}

func (s sessionRow) state(now time.Time) string {
	switch {
	case s.EndedAt != nil:
		if s.EndReason != nil && *s.EndReason != "" {
			return "ended (" + *s.EndReason + ")"
		}
		return "ended"
	case !s.ExpiresAt.After(now):
		return "expired"
	}
	return "live"
}

func registerSessionCommands() {
	SessionCmd.AddCommand(sessionListCmd, sessionRevokeCmd)
	addOutputFlag(sessionListCmd)
	sessionListCmd.Flags().Int("limit", 0, "Page size (0 uses the server default)")
	sessionListCmd.Flags().Bool("live", false, "Show only sessions that are still live")
}

func runSessionList(cmd *cobra.Command, _ []string) error {
	output := strings.ToLower(strings.TrimSpace(outputFlag(cmd)))
	c, err := hookAPIClient(cmd)
	if err != nil {
		return err
	}
	q := url.Values{}
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		q.Set("limit", fmt.Sprint(limit))
	}
	var payload struct {
		Sessions []sessionRow `json:"sessions"`
	}
	if err := c.Get(cmd.Context(), "/agent-sessions?"+q.Encode(), &payload); err != nil {
		return err
	}
	rows := payload.Sessions
	now := time.Now()
	if liveOnly, _ := cmd.Flags().GetBool("live"); liveOnly {
		live := rows[:0]
		for _, s := range rows {
			if s.state(now) == "live" {
				live = append(live, s)
			}
		}
		rows = live
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	case "", "table":
	default:
		return fmt.Errorf("--output: unknown format %q (use json|yaml|table)", output)
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No agent sessions.")
		return nil
	}
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintln(w, "SESSION\tAGENT\tCLIENT\tSTATE\tEXPIRES")
	for _, s := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.AgentName, s.Client, s.state(now), s.ExpiresAt.Format(time.RFC3339))
	}
	return w.Flush()
}

func runSessionRevoke(cmd *cobra.Command, args []string) error {
	id := strings.TrimSpace(args[0])
	c, err := hookAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), "/agent-sessions/"+url.PathEscape(id)+"?reason=revoked"); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "revoked session %s\n", id)
	return nil
}
