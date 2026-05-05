package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
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

type token struct {
	ID        uuid.UUID  `json:"id"`
	AgentID   uuid.UUID  `json:"agent_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Scopes    any        `json:"scopes"`
}

type tokenWithCleartext struct {
	token
	CleartextToken string `json:"cleartext_token"`
}

type agent struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// listAgentsViaClient is the apiclient-based replacement for the legacy
// listAgents helper. agentRow (from cmd/agent.go) is the canonical shape; we
// re-decode here as []agent for the uuid.UUID-typed fields used by token verbs.
func listAgentsViaClient(ctx context.Context, c *apiclient.Client) ([]agent, error) {
	var rows []agent
	if err := c.Get(ctx, "/agents", &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// findOrCreateAgent returns the agent's UUID for name, creating a new agent
// if no match exists.
func findOrCreateAgent(ctx context.Context, c *apiclient.Client, name string) (uuid.UUID, error) {
	rows, err := listAgentsViaClient(ctx, c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, a := range rows {
		if a.Name == name {
			return a.ID, nil
		}
	}
	var created agent
	if err := c.Post(ctx, "/agents", map[string]string{"name": name}, &created); err != nil {
		return uuid.Nil, fmt.Errorf("create agent: %w", err)
	}
	return created.ID, nil
}

// findAgentID returns the agent's UUID for name, or an error if not found.
func findAgentID(ctx context.Context, c *apiclient.Client, name string) (uuid.UUID, error) {
	rows, err := listAgentsViaClient(ctx, c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, a := range rows {
		if a.Name == name {
			return a.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("agent not found: %s", name)
}

func runTokenList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")
	if agentName == "" {
		return fmt.Errorf("--agent flag required")
	}

	agentID, err := findAgentID(cmd.Context(), c, agentName)
	if err != nil {
		return err
	}

	var tokens []token
	if err := c.Get(cmd.Context(), "/agent-tokens?agent_id="+url.QueryEscape(agentID.String()), &tokens); err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return emitJSON(tokens)
	}
	if len(tokens) == 0 {
		fmt.Printf("No tokens for agent '%s'\n", agentName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprint(w, "ID\tCREATED\tEXPIRES\tREVOKED\n"); err != nil {
		return err
	}
	for _, t := range tokens {
		expires := ""
		if t.ExpiresAt != nil {
			expires = t.ExpiresAt.Format(time.RFC3339)
		}
		revoked := ""
		if t.RevokedAt != nil {
			revoked = t.RevokedAt.Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			t.ID.String(),
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			expires,
			revoked); err != nil {
			return err
		}
	}
	return w.Flush()
}

func runTokenIssue(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	agentName, _ := cmd.Flags().GetString("agent")
	if agentName == "" {
		return fmt.Errorf("--agent flag required")
	}
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

	if err := c.Delete(cmd.Context(), "/agent-tokens/"+url.PathEscape(tokenID)); err != nil {
		return err
	}
	fmt.Printf("Token revoked: %s\n", tokenID)
	return nil
}

func init() {
	TokenCmd.AddCommand(listCmd)
	TokenCmd.AddCommand(issueCmd)
	TokenCmd.AddCommand(revokeCmd)

	listCmd.Flags().String("agent", "", "Agent name (required)")
	listCmd.Flags().String("output", "", "Output format: json")
	issueCmd.Flags().String("agent", "", "Agent name (required)")
	issueCmd.Flags().StringSlice("scopes", []string{}, "Token scopes (optional)")
	issueCmd.Flags().String("expires", "", "Expiration duration (optional, e.g. '24h')")
}
