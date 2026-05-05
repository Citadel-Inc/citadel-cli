package cmd

import (
	"context"
	"fmt"
	"slices"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
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

// agentRow is the canonical CLI-side representation of an agent. Shared
// across cmd/agent.go and cmd/token.go.
type agentRow struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   string    `json:"owner_user_id,omitempty"`
	Name      string    `json:"name"`
	ModelHint *string   `json:"model_hint,omitempty"`
}

// listAgentRows fetches all agents owned by the authenticated user.
func listAgentRows(ctx context.Context, c *apiclient.Client) ([]agentRow, error) {
	var rows []agentRow
	if err := c.Get(ctx, "/agents", &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// findAgentByName returns the agent row matching name, or a not-found error.
func findAgentByName(ctx context.Context, c *apiclient.Client, name string) (agentRow, error) {
	rows, err := listAgentRows(ctx, c)
	if err != nil {
		return agentRow{}, err
	}
	if i := slices.IndexFunc(rows, func(a agentRow) bool { return a.Name == name }); i >= 0 {
		return rows[i], nil
	}
	return agentRow{}, fmt.Errorf("agent '%s' not found", name)
}

func runAgentList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)

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
	a, err := findAgentByName(cmd.Context(), c, args[0])
	if err != nil {
		return err
	}
	return emitOne(outputFlag(cmd), a, func(w *tabwriter.Writer, a agentRow) {
		_, _ = fmt.Fprintf(w, "Name:\t%s\n", a.Name)
		_, _ = fmt.Fprintf(w, "ID:\t%s\n", a.ID)
		if a.ModelHint != nil {
			_, _ = fmt.Fprintf(w, "Model hint:\t%s\n", *a.ModelHint)
		}
	})
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name := args[0]

	a, err := findAgentByName(cmd.Context(), c, name)
	if err != nil {
		return err
	}

	if err := confirmSlug(yesFlag(cmd), "delete agent", name); err != nil {
		return err
	}

	if err := c.Delete(cmd.Context(), "/agents/"+a.ID.String()); err != nil {
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

	if err := confirmSlug(yesFlag(cmd), "rotate token for agent", name); err != nil {
		return err
	}

	a, err := findAgentByName(cmd.Context(), c, name)
	if err != nil {
		return err
	}

	// Server-side atomic rotate: issues new token + revokes every other
	// live token for this agent in a single transaction.
	var newTok tokenWithCleartext
	if err := c.Post(cmd.Context(), "/agents/"+a.ID.String()+"/rotate-token", nil, &newTok); err != nil {
		return fmt.Errorf("rotate token: %w", err)
	}
	fmt.Println(newTok.CleartextToken)
	return nil
}

func init() {
	AgentCmd.AddCommand(agentListCmd)
	AgentCmd.AddCommand(agentGetCmd)
	AgentCmd.AddCommand(agentDeleteCmd)
	AgentCmd.AddCommand(agentRotateTokenCmd)

	addOutputFlag(agentListCmd)
	addOutputFlag(agentGetCmd)
	addOutputFlag(agentDeleteCmd)
	addOutputFlag(agentRotateTokenCmd)
	addYesFlag(agentDeleteCmd)
	addYesFlag(agentRotateTokenCmd)
}
