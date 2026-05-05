package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// AgentCmd is the top-level `citadel agent` command.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents (list, get, delete, rotate-token)",
	Long:  `CRUD operations against the Citadel agent API.`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents owned by the authenticated user",
	Long: `Returns all agents registered to the authenticated user.

Examples:
  citadel-cli agent list
  citadel-cli agent list --output json`,
	RunE: runAgentList,
}

var agentGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get details of a single agent",
	Long: `Fetches metadata for a single agent by name.

Examples:
  citadel-cli agent get myagent
  citadel-cli agent get myagent --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentGet,
}

var agentDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an agent",
	Long: `Deletes an agent and all its tokens. Requires typed-slug confirmation
unless --yes is set.

Examples:
  citadel-cli agent delete myagent
  citadel-cli agent delete myagent --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentDelete,
}

var agentRotateTokenCmd = &cobra.Command{
	Use:   "rotate-token <name>",
	Short: "Issue a new token and revoke all previous tokens for an agent",
	Long: `Issues a new agent token, then revokes all previously active tokens.
The new token is printed to stdout once and not stored.

Examples:
  citadel-cli agent rotate-token myagent`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentRotateToken,
}

// agentRow is the CLI-side representation of an agent.
type agentRow struct {
	ID        string  `json:"id"`
	OwnerID   string  `json:"owner_user_id"`
	Name      string  `json:"name"`
	ModelHint *string `json:"model_hint,omitempty"`
}

// agentToken mirrors the server Token shape for rotate-token operations.
type agentToken struct {
	ID        string     `json:"id"`
	AgentID   string     `json:"agent_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// agentTokenWithCleartext is the issue response shape.
type agentTokenWithCleartext struct {
	agentToken
	CleartextToken string `json:"cleartext_token"`
}

// listAgentRows fetches all agents owned by the authenticated user.
func listAgentRows(ctx context.Context, c *apiclient.Client) ([]agentRow, error) {
	var rows []agentRow
	if err := c.Get(ctx, "/agents", &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// resolveAgentID returns the agent's UUID by name; returns a not-found error if absent.
func resolveAgentID(ctx context.Context, c *apiclient.Client, name string) (string, error) {
	rows, err := listAgentRows(ctx, c)
	if err != nil {
		return "", err
	}
	for _, a := range rows {
		if a.Name == name {
			return a.ID, nil
		}
	}
	return "", fmt.Errorf("agent '%s' not found", name)
}

func newAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	flagServer, _ := cmd.Flags().GetString("server")
	return apiclient.New(cfg, flagServer)
}

func runAgentList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")

	rows, err := listAgentRows(cmd.Context(), c)
	if err != nil {
		return err
	}

	return emitList(output, rows, "No agents found.", func(w *tabwriter.Writer, rows []agentRow) {
		_, _ = fmt.Fprintln(w, "NAME\tID\tMODEL HINT")
		for _, a := range rows {
			hint := ""
			if a.ModelHint != nil {
				hint = *a.ModelHint
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.ID, hint)
		}
	})
}

func runAgentGet(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	name := args[0]

	rows, err := listAgentRows(cmd.Context(), c)
	if err != nil {
		return err
	}
	for _, a := range rows {
		if a.Name == name {
			return emitOne(output, a, func(w *tabwriter.Writer, a agentRow) {
				_, _ = fmt.Fprintf(w, "Name:\t%s\n", a.Name)
				_, _ = fmt.Fprintf(w, "ID:\t%s\n", a.ID)
				if a.ModelHint != nil {
					_, _ = fmt.Fprintf(w, "Model hint:\t%s\n", *a.ModelHint)
				}
			})
		}
	}
	return fmt.Errorf("agent '%s' not found", name)
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name := args[0]

	agentID, err := resolveAgentID(cmd.Context(), c, name)
	if err != nil {
		return err
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "delete agent", name); err != nil {
		return err
	}

	if err := c.Delete(cmd.Context(), "/agents/"+url.PathEscape(agentID)); err != nil {
		return err
	}
	fmt.Printf("Agent '%s' deleted.\n", name)
	return nil
}

func runAgentRotateToken(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "rotate token for agent", name); err != nil {
		return err
	}

	agentID, err := resolveAgentID(cmd.Context(), c, name)
	if err != nil {
		return err
	}

	// Issue new token first.
	var newTok agentTokenWithCleartext
	if err := c.Post(cmd.Context(), "/agent-tokens", map[string]string{"agent_id": agentID}, &newTok); err != nil {
		return fmt.Errorf("issue token: %w", err)
	}

	// List existing tokens to revoke all but the new one.
	var existingTokens []agentToken
	if err := c.Get(cmd.Context(), "/agent-tokens?agent_id="+url.QueryEscape(agentID), &existingTokens); err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	// Revoke all tokens except the new one.
	var revokeErrs []error
	for _, t := range existingTokens {
		if t.ID == newTok.ID || t.RevokedAt != nil {
			continue
		}
		if err := c.Delete(cmd.Context(), "/agent-tokens/"+url.PathEscape(t.ID)); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to revoke token %s: %v\n", t.ID, err)
			revokeErrs = append(revokeErrs, err)
		}
	}

	// Print new token once to stdout before surfacing any revoke warnings.
	fmt.Println(newTok.CleartextToken)
	if len(revokeErrs) > 0 {
		return fmt.Errorf("new token issued but %d old token(s) could not be revoked; check 'citadel-cli token list --agent %s': %w", len(revokeErrs), name, errors.Join(revokeErrs...))
	}
	return nil
}

func init() {
	AgentCmd.AddCommand(agentListCmd)
	AgentCmd.AddCommand(agentGetCmd)
	AgentCmd.AddCommand(agentDeleteCmd)
	AgentCmd.AddCommand(agentRotateTokenCmd)

	agentListCmd.Flags().String("output", "", "Output format: json")
	agentGetCmd.Flags().String("output", "", "Output format: json")
	agentDeleteCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	agentDeleteCmd.Flags().String("output", "", "Output format: json")
	agentRotateTokenCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	agentRotateTokenCmd.Flags().String("output", "", "Output format: json")
}
