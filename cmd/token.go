package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var TokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage agent tokens (list, issue, revoke)",
	Long:  `Commands for managing agent authentication tokens.`,
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
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")

	a, err := findAgentByName(cmd.Context(), c, agentName)
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
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	if err := validateWatchOutput(cmd); err != nil {
		return err
	}
	if watchFlag(cmd) {
		if err := validateDescCursor(cursor); err != nil {
			return fmt.Errorf("invalid --cursor: %w", err)
		}
		return runTokenListWatch(cmd, c, a.ID.String(), limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []token
	csvHdr := false
	first := true
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

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			empty := fmt.Sprintf("No tokens for agent '%s'", agentName)
			switch output {
			case "json":
				return emitJSON(cmd, []token{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[token](cmd)
			case "yaml":
				return emitYAML(cmd, []token{})
			default:
				fmt.Println(empty)
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
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !all {
			if isHumanListOutput(output) && next != "" {
				fmt.Println("(use --cursor " + next + " for more, or --all to fetch everything)")
			}
			return nil
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if all && output == "yaml" {
		if yamlAccum == nil {
			yamlAccum = []token{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runTokenIssue(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")
	scopes, _ := cmd.Flags().GetStringSlice("scopes")
	expiresStr, _ := cmd.Flags().GetString("expires")

	agentID, err := findOrCreateAgent(cmd.Context(), c, agentName)
	if err != nil {
		return err
	}

	var expiresIn *int64
	if expiresStr != "" {
		d, err := time.ParseDuration(expiresStr)
		if err == nil {
			sec := int64(d.Seconds())
			expiresIn = &sec
		}
	}

	body := struct {
		AgentID          uuid.UUID `json:"agent_id"`
		ExpiresInSeconds *int64    `json:"expires_in_seconds,omitempty"`
		Scopes           []string  `json:"scopes,omitempty"`
	}{
		AgentID:          agentID,
		ExpiresInSeconds: expiresIn,
		Scopes:           scopes,
	}

	var tok tokenWithCleartext
	if err := c.Post(cmd.Context(), "/agent-tokens", body, &tok); err != nil {
		return err
	}

	// Print the token (once, no debug noise)
	fmt.Println(tok.CleartextToken)
	return nil
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	tokenID := args[0]
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE /agent-tokens/%s (skipped; --dry-run)\n", tokenID)
		return nil
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

func init() {
	TokenCmd.AddCommand(listCmd)
	TokenCmd.AddCommand(issueCmd)
	TokenCmd.AddCommand(revokeCmd)

	listCmd.Flags().String("agent", "", "Agent name (required)")
	addOutputFlag(listCmd)
	addPaginationFlags(listCmd)
	addWatchFlag(listCmd)
	_ = listCmd.MarkFlagRequired("agent")
	issueCmd.Flags().String("agent", "", "Agent name (required)")
	issueCmd.Flags().StringSlice("scopes", []string{}, "Token scopes (optional)")
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
