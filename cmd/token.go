package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
	"github.com/Citadel-Inc/citadel-cli/internal/completion"
)

var TokenCmd = &cobra.Command{
	Use:     "token",
	GroupID: "auth",
	Short:   "Manage agent tokens (list, issue, revoke)",
	Long:    `Commands for managing agent authentication tokens.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent tokens owned by the authenticated user",
	Long: `Retrieves and displays all agent tokens, including token ID, agent name,
scopes, creation time, expiry, and revocation status. Run with --agent <name>
to filter tokens for a specific agent.`,
	RunE: runTokenList,
}

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue a new agent token",
	Long: `Creates or finds an agent with the given name and issues a new token.
Prints the clear-text token once to stdout (it is never stored or cached).
Subsequent 'token list' calls will show only metadata, not the secret.`,
	RunE: runTokenIssue,
}

var revokeCmd = &cobra.Command{
	Use:   "revoke <token-id>",
	Short: "Revoke an agent token",
	Long:  `Sets the revoked_at timestamp on the token; idempotent if already revoked.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenRevoke,
}

// token mirrors the server agent_tokens row (canonical shape used by every
// token verb plus agent rotate-token).
type token struct {
	ID        uuid.UUID  `json:"id"`
	AgentID   uuid.UUID  `json:"agent_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	// Namespaces is null with no limit; empty once every listed repo is deleted.
	Namespaces []string `json:"namespaces"`
}

// tokenWithCleartext is the issue / rotate response shape.
type tokenWithCleartext struct {
	token
	CleartextToken string `json:"cleartext_token"`
}

// findOrCreateAgent returns the agent's UUID for name, creating a new agent
// if no match exists.
func findOrCreateAgent(ctx context.Context, c *apiclient.Client, name string) (uuid.UUID, error) {
	rows, err := listAgentRows(ctx, c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, a := range rows {
		if a.Name == name {
			return a.ID, nil
		}
	}
	var created agentRow
	if err := c.Post(ctx, "/agents", map[string]string{"name": name}, &created); err != nil {
		return uuid.Nil, fmt.Errorf("create agent: %w", err)
	}
	return created.ID, nil
}

func runTokenList(cmd *cobra.Command, _ []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all && output == "json" {
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	if err := validateWatchOutput(cmd); err != nil {
		return err
	}

	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")

	a, err := findAgentByName(cmd.Context(), c, agentName)
	if err != nil {
		return err
	}

	if watchFlag(cmd) {
		return runTokenListWatch(cmd, c, a.ID.String(), limit, cursor, all)
	}

	pg := newListPager(cmd, output, all,
		func(w io.Writer) { _, _ = fmt.Fprintf(w, "No tokens for agent '%s'\n", agentName) },
		func(w *tabwriter.Writer, rows []token) {
			_, _ = fmt.Fprint(w, "ID\tCREATED\tEXPIRES\tREVOKED\n")
			for _, t := range rows {
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
			}
		})
	for {
		q := url.Values{}
		q.Set("agent_id", a.ID.String())
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Tokens []token `json:"tokens"`
			Next   string  `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/agent-tokens?"+q.Encode(), &payload); err != nil {
			return err
		}
		rows := payload.Tokens
		next := strings.TrimSpace(payload.Next)

		if stop, err := pg.page(rows, cursor, next); stop {
			return err
		}
		cursor = next
	}
}

func runTokenIssue(cmd *cobra.Command, _ []string) error {
	expiresIn, hasExpires, err := parseExpiresFlag(cmd)
	if err != nil {
		return err
	}
	var expiresInSeconds *int64
	if hasExpires {
		expiresInSeconds = &expiresIn
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")
	scopes := scopeFlag(cmd)

	agentID, err := findOrCreateAgent(cmd.Context(), c, agentName)
	if err != nil {
		return err
	}

	body := struct {
		AgentID          uuid.UUID `json:"agent_id"`
		ExpiresInSeconds *int64    `json:"expires_in_seconds,omitempty"`
		Scopes           []string  `json:"scopes,omitempty"`
		Namespaces       []string  `json:"namespaces,omitempty"`
	}{
		AgentID:          agentID,
		ExpiresInSeconds: expiresInSeconds,
		Scopes:           scopes,
		Namespaces:       namespaceFlag(cmd),
	}

	var tok tokenWithCleartext
	if err := c.Post(cmd.Context(), "/agent-tokens", body, &tok); err != nil {
		return err
	}

	// Print the token (once, no debug noise)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), tok.CleartextToken)
	return nil
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	tokenID := args[0]
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE /agent-tokens/%s (skipped; --dry-run)\n", tokenID)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), "/agent-tokens/"+url.PathEscape(tokenID)); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token revoked: %s\n", tokenID)
	return nil
}

func completeTokenIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeyAgentTokens, completion.FetchAgentTokenIDs)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func registerTokenCommands() {
	TokenCmd.AddCommand(listCmd)
	TokenCmd.AddCommand(issueCmd)
	TokenCmd.AddCommand(revokeCmd)

	listCmd.Flags().String("agent", "", "Agent name (required)")
	addOutputFlag(listCmd)
	addPaginationFlags(listCmd)
	addWatchFlag(listCmd)
	_ = listCmd.MarkFlagRequired("agent")
	issueCmd.Flags().String("agent", "", "Agent name (required)")
	addScopeFlag(issueCmd)
	issueCmd.Flags().String("expires", "", "Expiration duration (optional, e.g. '24h')")
	_ = issueCmd.MarkFlagRequired("agent")

	revokeCmd.ValidArgsFunction = completeTokenIDs
	addDryRunFlag(revokeCmd)

	issueCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyAgentTokens)
	}
	revokeCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyAgentTokens)
	}
}

// addScopeFlag registers the repeatable --scope flag. Omitting it lets the
// server apply its read-only default (metadata:read, code:read).
func addScopeFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringArray("scope", nil, "Token scope, repeatable (metadata:read, code:read, code:write, issues:read, issues:write, pulls:read, pulls:write, admin); default metadata:read and code:read")
		c.Flags().StringArray("namespace", nil, "Limit the token to this repo, project or org, repeatable. Omit for everything you can reach; rotate-token keeps the old limit")
	}
}

func namespaceFlag(cmd *cobra.Command) []string {
	namespaces, _ := cmd.Flags().GetStringArray("namespace")
	return namespaces
}

func scopeFlag(cmd *cobra.Command) []string {
	scopes, _ := cmd.Flags().GetStringArray("scope")
	return scopes
}
