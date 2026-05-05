package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// AuditCmd is the top-level `citadel-cli audit` command.
var AuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Query Citadel audit events",
	Long: `List and inspect audit log events and sessions for namespaces you can access.

Requires a token with audit visibility for the target namespace (audit:read grant
or namespace ownership). Operators with operator:audit:read see cross-tenant
events.`,
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent audit events",
	Long: `Lists audit events with optional filters. When --since is omitted, the server
defaults to the last 24 hours.

Time filters accept Go durations (e.g. 1h, 30m) or RFC3339 timestamps.`,
	RunE: runAuditList,
}

var auditShowCmd = &cobra.Command{
	Use:   "show <event-id>",
	Short: "Show one audit event with payload and cascade children",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuditShow,
}

func init() {
	AuditCmd.AddCommand(auditListCmd)
	AuditCmd.AddCommand(auditShowCmd)
	addPaginationFlags(auditListCmd)
	addOutputFlag(auditListCmd)
	addOutputFlag(auditShowCmd)
	auditListCmd.Flags().String("since", "", "Start of window: duration (e.g. 1h) or RFC3339 (server default: 24h when omitted)")
	auditListCmd.Flags().String("until", "", "End of window: duration or RFC3339 (server default: now when omitted)")
	auditListCmd.Flags().String("kind", "", "Glob filter on event kind (e.g. repo.*, oauth.**)")
	auditListCmd.Flags().StringP("namespace", "n", "", "Filter to events for this namespace slug")
	auditListCmd.Flags().String("actor", "", "Filter by actor UUID or user slug")
}

type auditEventPayload struct {
	ID            string          `json:"id"`
	TS            string          `json:"ts"`
	Kind          string          `json:"kind"`
	ActorID       string          `json:"actor_id,omitempty"`
	ActorSlug     string          `json:"actor_slug,omitempty"`
	ActorType     string          `json:"actor_type"`
	NamespaceID   string          `json:"namespace_id,omitempty"`
	NamespaceSlug string          `json:"namespace_slug,omitempty"`
	SubjectID     string          `json:"subject_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	RequestID     string          `json:"request_id,omitempty"`
	ClientIP      string          `json:"client_ip,omitempty"`
	SessionID     string          `json:"session_id,omitempty"`
}

func (r auditEventPayload) CSVHeader() []string {
	return []string{"id", "ts", "kind", "actor_slug", "actor_id", "namespace_slug", "namespace_id", "subject_id", "actor_type"}
}

func (r auditEventPayload) CSVRecord() []string {
	return []string{
		r.ID, r.TS, r.Kind, r.ActorSlug, r.ActorID, r.NamespaceSlug, r.NamespaceID, r.SubjectID, r.ActorType,
	}
}

type auditShowPayload struct {
	auditEventPayload
	CascadeChildren []auditEventPayload `json:"cascade_children,omitempty"`
}

func runAuditList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all && output == "json" {
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson or omit --all")
	}
	if err := validateAuditCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	kind, _ := cmd.Flags().GetString("kind")
	ns, _ := cmd.Flags().GetString("namespace")
	actor, _ := cmd.Flags().GetString("actor")

	var yamlAccum []auditEventPayload
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if strings.TrimSpace(since) != "" {
			q.Set("since", strings.TrimSpace(since))
		}
		if strings.TrimSpace(until) != "" {
			q.Set("until", strings.TrimSpace(until))
		}
		if strings.TrimSpace(kind) != "" {
			q.Set("kind", strings.TrimSpace(kind))
		}
		if strings.TrimSpace(ns) != "" {
			q.Set("namespace", strings.TrimSpace(ns))
		}
		if strings.TrimSpace(actor) != "" {
			q.Set("actor", strings.TrimSpace(actor))
		}

		var payload struct {
			Events     []auditEventPayload `json:"events"`
			NextCursor string              `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/audit/events?"+q.Encode(), &payload); err != nil {
			return err
		}
		rows := payload.Events
		next := strings.TrimSpace(payload.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []auditEventPayload{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[auditEventPayload](cmd)
			case "yaml":
				return emitYAML(cmd, []auditEventPayload{})
			default:
				fmt.Println("No audit events found.")
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(cmd, rows)
		case "ndjson":
			if err := emitNDJSONLines(cmd, rows); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, rows); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, rows...)
			} else {
				return emitYAML(cmd, rows)
			}
		default:
			w := newTabWriter(cmd)
			_, _ = fmt.Fprintln(w, "TS\tKIND\tACTOR\tNAMESPACE\tSUBJECT\tID")
			for _, e := range rows {
				actorCol := e.ActorSlug
				if actorCol == "" {
					actorCol = e.ActorID
				}
				nsCol := e.NamespaceSlug
				if nsCol == "" {
					nsCol = "-"
				}
				subj := e.SubjectID
				if subj == "" {
					subj = "-"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", e.TS, e.Kind, actorCol, nsCol, subj, e.ID)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !all || next == "" {
			break
		}
		cursor = next
	}

	if all && output == "yaml" {
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runAuditShow(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}

	id := strings.TrimSpace(args[0])
	var detail auditShowPayload
	if err := c.Get(cmd.Context(), "/audit/events/"+url.PathEscape(id), &detail); err != nil {
		return err
	}

	switch output {
	case "json":
		return emitJSON(cmd, detail)
	case "yaml":
		return emitYAML(cmd, detail)
	default:
		w := newTabWriter(cmd)
		actor := detail.ActorSlug
		if actor == "" {
			actor = detail.ActorID
		}
		ns := detail.NamespaceSlug
		if ns == "" && detail.NamespaceID != "" {
			ns = detail.NamespaceID
		}
		if ns == "" {
			ns = "-"
		}
		_, _ = fmt.Fprintf(w, "ID:\t%s\n", detail.ID)
		_, _ = fmt.Fprintf(w, "Time:\t%s\n", detail.TS)
		_, _ = fmt.Fprintf(w, "Kind:\t%s\n", detail.Kind)
		_, _ = fmt.Fprintf(w, "Actor:\t%s\n", actor)
		_, _ = fmt.Fprintf(w, "Actor type:\t%s\n", detail.ActorType)
		_, _ = fmt.Fprintf(w, "Namespace:\t%s\n", ns)
		if detail.SubjectID != "" {
			_, _ = fmt.Fprintf(w, "Subject:\t%s\n", detail.SubjectID)
		}
		if detail.SessionID != "" {
			_, _ = fmt.Fprintf(w, "Session:\t%s\n", detail.SessionID)
		}
		if detail.RequestID != "" {
			_, _ = fmt.Fprintf(w, "Request:\t%s\n", detail.RequestID)
		}
		if detail.ClientIP != "" {
			_, _ = fmt.Fprintf(w, "Client IP:\t%s\n", detail.ClientIP)
		}
		_, _ = fmt.Fprintln(w, "\nPayload:")
		pretty := detail.Payload
		if len(pretty) > 0 {
			buf := &bytes.Buffer{}
			if err := json.Indent(buf, pretty, "", "  "); err == nil {
				for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
					_, _ = fmt.Fprintf(w, "  %s\n", line)
				}
			} else {
				_, _ = fmt.Fprintf(w, "  %s\n", string(pretty))
			}
		}
		if len(detail.CascadeChildren) > 0 {
			_, _ = fmt.Fprintf(w, "\nCascade children (%d):\n", len(detail.CascadeChildren))
			for _, ch := range detail.CascadeChildren {
				_, _ = fmt.Fprintf(w, "  - [%s] %s %s\n", ch.ID, ch.TS, ch.Kind)
			}
		}
		return w.Flush()
	}
}
