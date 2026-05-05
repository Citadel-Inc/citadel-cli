package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/sseclient"
)

func addWatchFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().BoolP("watch", "w", false, "Stream list changes via Server-Sent Events (until interrupted)")
	}
}

func watchFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("watch")
	return v
}

func validateWatchOutput(cmd *cobra.Command) error {
	if !watchFlag(cmd) {
		return nil
	}
	o := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	switch o {
	case "", "table", "ndjson":
		return nil
	case "json":
		return fmt.Errorf("--output json cannot be used with --watch; use --output ndjson for streaming JSON lines, or omit --output for the live table")
	case "yaml":
		return fmt.Errorf("--output yaml cannot be used with --watch; use --output ndjson for streaming JSON lines, or omit --output for the live table")
	case "csv":
		return fmt.Errorf("--output csv cannot be used with --watch; use --output ndjson for streaming JSON lines, or omit --output for the live table")
	default:
		return fmt.Errorf("--watch requires --output ndjson or default table mode")
	}
}

func stdoutIsTTY(cmd *cobra.Command) bool {
	f, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func notifyWatchContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

func flushOut(w io.Writer) {
	if f, ok := w.(*os.File); ok {
		_ = f.Sync()
	}
}

func consumeSSEWatch(cmd *cobra.Command, c *apiclient.Client, path string, h sseEventHandler) error {
	ctx, stop := notifyWatchContext(cmd.Context())
	defer stop()

	stream := sseclient.Open(ctx, c, path)
	defer func() { _ = stream.Close() }()

	for {
		ev, err := stream.Next()
		if err != nil {
			if ctx.Err() != nil {
				flushOut(cmd.OutOrStdout())
				return nil
			}
			if errors.Is(err, context.Canceled) {
				flushOut(cmd.OutOrStdout())
				return nil
			}
			return err
		}
		if err := h.Handle(ev); err != nil {
			return err
		}
	}
}

// sseEventHandler receives decoded SSE events for one list watch session.
type sseEventHandler interface {
	Handle(ev sseclient.Event) error
}

type ndjsonWatchEmitter struct {
	enc *json.Encoder
}

func newNdjsonWatchEmitter(w io.Writer) *ndjsonWatchEmitter {
	return &ndjsonWatchEmitter{enc: json.NewEncoder(w)}
}

func (e *ndjsonWatchEmitter) Handle(ev sseclient.Event) error {
	var payload json.RawMessage
	if len(ev.Data) > 0 {
		payload = json.RawMessage(ev.Data)
	}
	line := struct {
		Type    string          `json:"type"`
		TS      string          `json:"ts"`
		Payload json.RawMessage `json:"payload"`
	}{
		Type:    ev.Type,
		TS:      time.Now().UTC().Format(time.RFC3339Nano),
		Payload: payload,
	}
	return e.enc.Encode(line)
}

func newWatchSSEHandler(cmd *cobra.Command, kind watchListKind, ctx watchTableCtx) (sseEventHandler, error) {
	out := cmd.OutOrStdout()
	switch strings.TrimSpace(strings.ToLower(outputFlag(cmd))) {
	case "ndjson":
		return newNdjsonWatchEmitter(out), nil
	default:
		return newTableWatchEmitter(cmd, kind, ctx), nil
	}
}

func sseWatchQuery(limit int, cursor string, all bool, extra url.Values) string {
	q := url.Values{}
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if all {
		q.Set("all", "true")
	}
	return q.Encode()
}

func runRepoListWatch(cmd *cobra.Command, c *apiclient.Client, ns string, limit int, cursor string, all bool) error {
	path := "/namespaces/" + url.PathEscape(ns) + "/repos?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchRepos, watchTableCtx{repoParentNS: ns})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runAgentListWatch(cmd *cobra.Command, c *apiclient.Client, limit int, cursor string, all bool) error {
	path := "/agents?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchAgents, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runOAuthClientsListWatch(cmd *cobra.Command, c *apiclient.Client, orgSlug string, limit int, cursor string, all bool) error {
	ex := url.Values{}
	if strings.TrimSpace(orgSlug) != "" {
		ex.Set("namespace", orgSlug)
	}
	path := "/oauth/clients?" + sseWatchQuery(limit, cursor, all, ex)
	h, err := newWatchSSEHandler(cmd, watchOAuthClients, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runNsListWatch(cmd *cobra.Command, c *apiclient.Client, limit int, cursor string, all bool) error {
	path := "/orgs?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchOrgs, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runNsMembersWatch(cmd *cobra.Command, c *apiclient.Client, orgSlug string, limit int, cursor string, all bool) error {
	path := "/orgs/" + url.PathEscape(orgSlug) + "/members?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchOrgMembers, watchTableCtx{orgSlug: orgSlug})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runNsTransferListPendingWatch(cmd *cobra.Command, c *apiclient.Client, limit int, cursor string, all bool) error {
	path := "/transfers/pending?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchTransfersPending, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func runTokenListWatch(cmd *cobra.Command, c *apiclient.Client, agentID string, limit int, cursor string, all bool) error {
	ex := url.Values{}
	ex.Set("agent_id", agentID)
	path := "/agent-tokens?" + sseWatchQuery(limit, cursor, all, ex)
	h, err := newWatchSSEHandler(cmd, watchAgentTokens, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}
