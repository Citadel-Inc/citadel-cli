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

	"github.com/Citadel-Inc/citadel-cli/internal/sseclient"
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
	watchAudit
	watchNotifications
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
	case watchAudit:
		_, _ = fmt.Fprintln(w, "TS\tKIND\tACTOR\tNAMESPACE\tSUBJECT\tID")
	case watchNotifications:
		_, _ = fmt.Fprintln(w, "ID\tKIND\tSUMMARY\tREAD\tCREATED")
	default:
		_, _ = fmt.Fprintln(w, "KEY\tJSON")
	}
}

func (e *tableWatchEmitter) writeRow(w io.Writer, key string, raw json.RawMessage) {
	if f, ok := watchRowFormatters[e.kind]; ok {
		_, _ = fmt.Fprintln(w, f(key, raw))
		return
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\n", key, string(raw))
}

// watchRowFormatters renders one tab-separated table line per list kind; a
// payload that does not decode shows as key and raw JSON.
var watchRowFormatters = map[watchListKind]func(key string, raw json.RawMessage) string{
	watchRepos: watchRowOf(func(r repoRow) string {
		return fmt.Sprintf("%s\t%s\t%s\t%s", r.Path, r.Visibility, r.DefaultBranch, r.CreatedAt)
	}),
	watchAgents: watchRowOf(func(r agentRow) string {
		hint := ""
		if r.ModelHint != nil {
			hint = *r.ModelHint
		}
		return fmt.Sprintf("%s\t%s\t%s", r.Name, r.ID, hint)
	}),
	watchOAuthClients: watchRowOf(func(r oauthClient) string {
		scopes := strings.Join(r.AllowedScopes, ",")
		if scopes == "" {
			scopes = "—"
		}
		return fmt.Sprintf("%s\t%s\t%s", r.ClientID, r.Name, scopes)
	}),
	watchOrgs: watchRowOf(func(r nsOrgRow) string {
		return fmt.Sprintf("%s\t%s\t%s", r.Slug, r.DisplayName, r.CreatedAt.Format("2006-01-02"))
	}),
	watchOrgMembers: watchRowOf(func(r nsMemberRow) string {
		role := "member"
		if r.IsOwner {
			role = "owner"
		}
		name := r.DisplayName
		if name == "" {
			name = r.Slug
		}
		return fmt.Sprintf("%s\t%s\t%s\t%s", r.Slug, name, role, r.JoinedAt.Format("2006-01-02"))
	}),
	watchTransfersPending: watchRowOf(func(r nsTransferRow) string {
		shortID := r.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		return fmt.Sprintf("%s\t%s\t%s\t%s", shortID, r.OrgSlug, r.FromUserSlug, r.ExpiresAt.Format("2006-01-02"))
	}),
	watchAgentTokens: watchRowOf(func(t token) string {
		expires := ""
		if t.ExpiresAt != nil {
			expires = t.ExpiresAt.Format(time.RFC3339)
		}
		revoked := ""
		if t.RevokedAt != nil {
			revoked = t.RevokedAt.Format(time.RFC3339)
		}
		return fmt.Sprintf("%s\t%s\t%s\t%s", t.ID.String(), t.CreatedAt.Format("2006-01-02 15:04:05"), expires, revoked)
	}),
	watchDeployTokens: watchRowOf(func(t deployTokenRow) string {
		name := t.Name
		if name == "" {
			name = "—"
		}
		return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", t.ID, name, t.CreatedAt.Format("2006-01-02 15:04:05"), formatTimePtr(t.ExpiresAt), formatTimePtr(t.RevokedAt))
	}),
	watchAudit: watchRowOf(func(r auditEventPayload) string {
		return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
			r.TS, r.Kind, dashIfEmpty(firstNonEmpty(r.ActorSlug, r.ActorID)), dashIfEmpty(r.NamespaceSlug), dashIfEmpty(r.SubjectID), r.ID)
	}),
	watchNotifications: watchRowOf(func(r notifItem) string {
		read := "no"
		if r.ReadAt != nil {
			read = "yes"
		}
		return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", r.ID, r.Kind, r.Summary, read, r.CreatedAt.Format("2006-01-02 15:04:05"))
	}),
}

func watchRowOf[T any](format func(T) string) func(string, json.RawMessage) string {
	return func(key string, raw json.RawMessage) string {
		var r T
		if json.Unmarshal(raw, &r) != nil {
			return key + "\t" + string(raw)
		}
		return format(r)
	}
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (e *tableWatchEmitter) shortRowLabel(raw json.RawMessage) string {
	if f, ok := watchLabelers[e.kind]; ok {
		if l := f(raw); l != "" {
			return l
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

// watchLabelers return a short human label for a delta line, or "" to fall
// back to generic field probing.
var watchLabelers = map[watchListKind]func(json.RawMessage) string{
	watchRepos:        watchLabelOf(func(r repoRow) string { return r.Path }),
	watchAgents:       watchLabelOf(func(r agentRow) string { return r.Name }),
	watchOAuthClients: watchLabelOf(func(r oauthClient) string { return r.ClientID }),
	watchOrgs:         watchLabelOf(func(r nsOrgRow) string { return r.Slug }),
	watchOrgMembers: watchLabelOf(func(r nsMemberRow) string {
		return firstNonEmpty(r.Slug, r.UserID)
	}),
	watchTransfersPending: watchLabelOf(func(r nsTransferRow) string { return r.ID }),
	watchAgentTokens:      watchLabelOf(func(t token) string { return t.ID.String() }),
	watchDeployTokens: watchLabelOf(func(t deployTokenRow) string {
		if t.ID == "" {
			return ""
		}
		return firstNonEmpty(t.Name, t.ID)
	}),
	watchAudit: watchLabelOf(func(r auditEventPayload) string {
		if r.Kind == "" {
			return ""
		}
		return r.Kind + " " + r.ID
	}),
	watchNotifications: watchLabelOf(func(r notifItem) string { return r.Summary }),
}

func watchLabelOf[T any](label func(T) string) func(json.RawMessage) string {
	return func(raw json.RawMessage) string {
		var r T
		if json.Unmarshal(raw, &r) != nil {
			return ""
		}
		return label(r)
	}
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
	case watchAudit, watchNotifications:
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &x)
		return strings.TrimSpace(x.ID)
	}
	return ""
}
