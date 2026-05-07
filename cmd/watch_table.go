package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/sseclient"
)

type watchListKind int

const (
	watchRepos watchListKind = iota
	watchAgents
	watchOAuthClients
	watchOrgs
	watchOrgMembers
	watchTransfersPending
	watchAgentTokens
	watchDeployTokens
)

// watchTableCtx carries slug fields needed to interpret partial SSE payloads.
type watchTableCtx struct {
	repoParentNS string // repos list namespace
	orgSlug      string // members list org
}

type tableWatchEmitter struct {
	cmd    *cobra.Command
	kind   watchListKind
	ctx    watchTableCtx
	out    io.Writer
	redraw bool

	rows           map[string]json.RawMessage
	paintedLines   int
	lastDeltaLabel string
}

func newTableWatchEmitter(cmd *cobra.Command, kind watchListKind, ctx watchTableCtx) *tableWatchEmitter {
	useRedraw := colorEnabled(cmd) && stdoutIsTTY(cmd)
	return &tableWatchEmitter{
		cmd:    cmd,
		kind:   kind,
		ctx:    ctx,
		out:    cmd.OutOrStdout(),
		redraw: useRedraw,
		rows:   make(map[string]json.RawMessage),
	}
}

func (e *tableWatchEmitter) Handle(ev sseclient.Event) error {
	raw := json.RawMessage(ev.Data)
	if len(ev.Data) == 0 || strings.TrimSpace(string(ev.Data)) == "" {
		raw = nil
	}

	switch ev.Type {
	case "init", "add", "update":
		if raw == nil {
			return nil
		}
		k := e.rowKey(raw)
		if k == "" {
			k = string(raw)
		}
		e.rows[k] = raw
		e.lastDeltaLabel = e.shortRowLabel(raw)
		return e.afterChange(ev.Type)
	case "remove":
		e.lastDeltaLabel = e.shortRowLabel(raw)
		k := e.rowKey(raw)
		if k != "" {
			delete(e.rows, k)
		}
		return e.afterChange(ev.Type)
	default:
		return nil
	}
}

func (e *tableWatchEmitter) afterChange(kind string) error {
	live := kind == "add" || kind == "update" || kind == "remove"
	if e.redraw {
		return e.paintRedraw()
	}
	if kind == "init" {
		return e.paintSnapshot()
	}
	if live {
		return e.printDelta(kind)
	}
	return nil
}

func (e *tableWatchEmitter) paintRedraw() error {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	e.writeHeader(w)
	for _, k := range e.sortedKeys() {
		e.writeRow(w, k, e.rows[k])
	}
	if err := w.Flush(); err != nil {
		return err
	}
	block := buf.String()
	lines := strings.Count(block, "\n")
	if e.paintedLines > 0 {
		_, _ = fmt.Fprintf(e.out, "\033[%dA", e.paintedLines)
	}
	_, _ = e.out.Write([]byte(block))
	e.paintedLines = lines
	flushOut(e.out)
	return nil
}

func (e *tableWatchEmitter) paintSnapshot() error {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	e.writeHeader(w)
	for _, k := range e.sortedKeys() {
		e.writeRow(w, k, e.rows[k])
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, _ = e.out.Write([]byte(buf.String()))
	_, _ = e.out.Write([]byte("\n"))
	flushOut(e.out)
	return nil
}

func (e *tableWatchEmitter) printDelta(kind string) error {
	var prefix rune
	switch kind {
	case "add":
		prefix = '+'
	case "remove":
		prefix = '-'
	default:
		prefix = '~'
	}
	label := e.lastDeltaLabel
	if strings.TrimSpace(label) == "" {
		label = e.deltaLabel()
	}
	_, err := fmt.Fprintf(e.out, "%c %s\n", prefix, label)
	if err != nil {
		return err
	}
	flushOut(e.out)
	return nil
}

func (e *tableWatchEmitter) deltaLabel() string {
	keys := e.sortedKeys()
	if len(keys) == 0 {
		return "(empty)"
	}
	k := keys[len(keys)-1]
	raw := e.rows[k]
	if raw == nil {
		return k
	}
	return e.shortRowLabel(raw)
}

func (e *tableWatchEmitter) sortedKeys() []string {
	out := make([]string, 0, len(e.rows))
	for k := range e.rows {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (e *tableWatchEmitter) writeHeader(w io.Writer) {
	switch e.kind {
	case watchRepos:
		_, _ = fmt.Fprintln(w, "PATH\tVISIBILITY\tBRANCH\tCREATED")
	case watchAgents:
		_, _ = fmt.Fprintln(w, "NAME\tID\tMODEL HINT")
	case watchOAuthClients:
		_, _ = fmt.Fprintln(w, "CLIENT ID\tNAME\tSCOPES")
	case watchOrgs:
		_, _ = fmt.Fprintln(w, "SLUG\tDISPLAY NAME\tCREATED")
	case watchOrgMembers:
		_, _ = fmt.Fprintln(w, "SLUG\tDISPLAY NAME\tROLE\tJOINED")
	case watchTransfersPending:
		_, _ = fmt.Fprintln(w, "ID\tORG\tFROM\tEXPIRES")
	case watchAgentTokens:
		_, _ = fmt.Fprintln(w, "ID\tCREATED\tEXPIRES\tREVOKED")
	case watchDeployTokens:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tCREATED\tEXPIRES\tREVOKED")
	default:
		_, _ = fmt.Fprintln(w, "KEY\tJSON")
	}
}

func (e *tableWatchEmitter) writeRow(w io.Writer, key string, raw json.RawMessage) {
	switch e.kind {
	case watchRepos:
		var r repoRow
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Path, r.Visibility, r.DefaultBranch, r.CreatedAt)
	case watchAgents:
		var r agentRow
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		hint := ""
		if r.ModelHint != nil {
			hint = *r.ModelHint
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.ID, hint)
	case watchOAuthClients:
		var r oauthClient
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		scopes := strings.Join(r.AllowedScopes, ",")
		if scopes == "" {
			scopes = "—"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.ClientID, r.Name, scopes)
	case watchOrgs:
		var r nsOrgRow
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.Slug, r.DisplayName, r.CreatedAt.Format("2006-01-02"))
	case watchOrgMembers:
		var r nsMemberRow
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		role := "member"
		if r.IsOwner {
			role = "owner"
		}
		name := r.DisplayName
		if name == "" {
			name = r.Slug
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Slug, name, role, r.JoinedAt.Format("2006-01-02"))
	case watchTransfersPending:
		var r nsTransferRow
		if json.Unmarshal(raw, &r) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		shortID := r.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			shortID, r.OrgSlug, r.FromUserSlug, r.ExpiresAt.Format("2006-01-02"))
	case watchAgentTokens:
		var t token
		if json.Unmarshal(raw, &t) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		expires := ""
		if t.ExpiresAt != nil {
			expires = t.ExpiresAt.Format(time.RFC3339)
		}
		revoked := ""
		if t.RevokedAt != nil {
			revoked = t.RevokedAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			t.ID.String(),
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			expires,
			revoked)
	case watchDeployTokens:
		var t deployTokenRow
		if json.Unmarshal(raw, &t) != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
			return
		}
		name := t.Name
		if name == "" {
			name = "—"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t.ID,
			name,
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			formatTimePtr(t.ExpiresAt),
			formatTimePtr(t.RevokedAt),
		)
	default:
		_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
	}
}

func (e *tableWatchEmitter) shortRowLabel(raw json.RawMessage) string {
	switch e.kind {
	case watchRepos:
		var r repoRow
		if json.Unmarshal(raw, &r) == nil && r.Path != "" {
			return r.Path
		}
	case watchAgents:
		var r agentRow
		if json.Unmarshal(raw, &r) == nil && r.Name != "" {
			return r.Name
		}
	case watchOAuthClients:
		var r oauthClient
		if json.Unmarshal(raw, &r) == nil && r.ClientID != "" {
			return r.ClientID
		}
	case watchOrgs:
		var r nsOrgRow
		if json.Unmarshal(raw, &r) == nil && r.Slug != "" {
			return r.Slug
		}
	case watchOrgMembers:
		var r nsMemberRow
		if json.Unmarshal(raw, &r) == nil {
			if r.Slug != "" {
				return r.Slug
			}
			if r.UserID != "" {
				return r.UserID
			}
		}
	case watchTransfersPending:
		var r nsTransferRow
		if json.Unmarshal(raw, &r) == nil && r.ID != "" {
			return r.ID
		}
	case watchAgentTokens:
		var t token
		if json.Unmarshal(raw, &t) == nil {
			return t.ID.String()
		}
	case watchDeployTokens:
		var t deployTokenRow
		if json.Unmarshal(raw, &t) == nil && t.ID != "" {
			if t.Name != "" {
				return t.Name
			}
			return t.ID
		}
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) == nil {
		for _, field := range []string{"path", "slug", "name", "client_id", "id"} {
			if v, ok := m[field]; ok {
				return fmt.Sprint(v)
			}
		}
	}
	return string(raw)
}

func (e *tableWatchEmitter) rowKey(raw json.RawMessage) string {
	switch e.kind {
	case watchRepos:
		var x struct {
			Path string `json:"path"`
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(raw, &x)
		if x.Path != "" {
			return x.Path
		}
		if x.Slug != "" && e.ctx.repoParentNS != "" {
			return e.ctx.repoParentNS + "/" + x.Slug
		}
		return x.Slug
	case watchAgents:
		var x struct {
			Name string    `json:"name"`
			ID   uuid.UUID `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		if x.Name != "" {
			return x.Name
		}
		if x.ID != uuid.Nil {
			return x.ID.String()
		}
	case watchOAuthClients:
		var x struct {
			ClientID string `json:"client_id"`
			ID       string `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		if x.ClientID != "" {
			return x.ClientID
		}
		return strings.TrimSpace(x.ID)
	case watchOrgs:
		var x struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(raw, &x)
		return x.Slug
	case watchOrgMembers:
		var x struct {
			UserID string `json:"user_id"`
			Slug   string `json:"slug"`
		}
		_ = json.Unmarshal(raw, &x)
		if x.UserID != "" {
			return x.UserID
		}
		if x.Slug != "" && e.ctx.orgSlug != "" {
			return e.ctx.orgSlug + "/" + x.Slug
		}
		return x.Slug
	case watchTransfersPending:
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		return x.ID
	case watchAgentTokens:
		var x struct {
			ID uuid.UUID `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		if x.ID != uuid.Nil {
			return x.ID.String()
		}
	case watchDeployTokens:
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		return strings.TrimSpace(x.ID)
	}
	return ""
}
