package cmd

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var auditSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List and inspect audit sessions",
	Long: `Session views group audit activity by login/session boundaries.

Use audit sessions list with a namespace slug; use audit sessions show for drill-down.`,
}

var auditSessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit sessions for a namespace",
	Long: `Lists audit sessions for one namespace. Either --ns or --namespace (-n) is required;
that maps to the daemon ns query parameter.

When --since is omitted, the server defaults to the last 24 hours.

Pagination uses --limit / --offset (not opaque cursors).`,
	RunE: runAuditSessionsList,
}

var auditSessionsShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show one audit session drill-down payload",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuditSessionsShow,
}

// auditSessionSummary is the subset of ListSessions rows the CLI renders.
// Extra JSON keys from the server are ignored during decode.
type auditSessionSummary struct {
	SessionID     string `json:"session_id,omitempty"`
	ID            string `json:"id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
	ActorSlug     string `json:"actor_slug,omitempty"`
	ActorType     string `json:"actor_type,omitempty"`
	NamespaceSlug string `json:"namespace_slug,omitempty"`
	NamespaceID   string `json:"namespace_id,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	LastEventAt   string `json:"last_event_at,omitempty"`
	EventCount    int    `json:"event_count,omitempty"`
}

func runAuditSessionsList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}

	ns := auditSessionsResolveNS(cmd)
	if ns == "" {
		return fmt.Errorf("namespace required: set --ns or --namespace / -n")
	}

	since, _ := cmd.Flags().GetString("since")
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return err
	}
	offset, err := cmd.Flags().GetInt("offset")
	if err != nil {
		return err
	}
	if limit < 0 || offset < 0 {
		return fmt.Errorf("--limit and --offset must be non-negative")
	}
	actorType, _ := cmd.Flags().GetString("actor-type")

	q := url.Values{}
	q.Set("ns", ns)
	if strings.TrimSpace(since) != "" {
		q.Set("since", strings.TrimSpace(since))
	}
	if limit != 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset != 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	if strings.TrimSpace(actorType) != "" {
		q.Set("actor_type", strings.TrimSpace(actorType))
	}

	var payload struct {
		Sessions []auditSessionSummary `json:"sessions"`
	}
	if err := c.Get(cmd.Context(), "/audit/sessions?"+q.Encode(), &payload); err != nil {
		return err
	}
	rows := payload.Sessions

	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		if len(rows) == 0 {
			return emitCSVHeaderOnly[auditSessionSummary](cmd)
		}
		var hdr bool
		return emitCSVRows(cmd, &hdr, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No audit sessions in window.")
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "SESSION\tACTOR\tTYPE\tNAMESPACE\tEVENTS\tLAST EVENT")
		for _, s := range rows {
			sid := strings.TrimSpace(cmp.Or(s.SessionID, s.ID))
			if len(sid) > 12 {
				sid = sid[:12]
			}
			actor := strings.TrimSpace(cmp.Or(s.ActorSlug, s.ActorID))
			nsCol := strings.TrimSpace(cmp.Or(s.NamespaceSlug, s.NamespaceID))
			if nsCol == "" {
				nsCol = "-"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", sid, actor, s.ActorType, nsCol, s.EventCount, s.LastEventAt)
		}
		return w.Flush()
	}
}

func runAuditSessionsShow(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	var detail json.RawMessage
	if err := c.Get(cmd.Context(), "/audit/sessions/"+url.PathEscape(id), &detail); err != nil {
		return err
	}

	switch output {
	case "json":
		var buf bytes.Buffer
		if err := json.Indent(&buf, detail, "", "  "); err != nil {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(detail))
		} else {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), buf.String())
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		return nil
	case "yaml":
		var tmp any
		if err := json.Unmarshal(detail, &tmp); err != nil {
			return err
		}
		return emitYAML(cmd, tmp)
	default:
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, detail, "", "  "); err != nil {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(detail))
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
			return nil
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(), pretty.String())
		return nil
	}
}

func auditSessionsResolveNS(cmd *cobra.Command) string {
	ns, _ := cmd.Flags().GetString("ns")
	nb, _ := cmd.Flags().GetString("namespace")
	ns = strings.TrimSpace(ns)
	nb = strings.TrimSpace(nb)
	if ns != "" {
		return ns
	}
	return nb
}

func init() {
	AuditCmd.AddCommand(auditSessionsCmd)
	auditSessionsCmd.AddCommand(auditSessionsListCmd)
	auditSessionsCmd.AddCommand(auditSessionsShowCmd)

	addOutputFlag(auditSessionsListCmd)
	addOutputFlag(auditSessionsShowCmd)

	auditSessionsListCmd.Flags().String("ns", "", "Namespace slug for sessions (required unless --namespace / -n set)")
	auditSessionsListCmd.Flags().StringP("namespace", "n", "", "Alias of --ns")
	auditSessionsListCmd.Flags().String("since", "", "Window start: duration (e.g. 1h, 7d) or RFC3339 (server default: 24h when omitted)")
	auditSessionsListCmd.Flags().Int("limit", 0, "Page size (positive; 0 means omit so server uses default)")
	auditSessionsListCmd.Flags().Int("offset", 0, "Offset into result set (non-negative)")
	auditSessionsListCmd.Flags().String("actor-type", "", "Optional actor_type filter")
}
