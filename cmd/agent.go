package cmd

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

// AgentCmd is the top-level `citadel agent` command.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents (list, get, delete, rotate-token, create)",
	Long:  `CRUD operations against the Citadel agent API.`,
}

var agentCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new agent and print its initial token",
	Long: `Registers a new agent in the authenticated user's personal namespace,
then issues an initial token. The token is printed to stdout once and is
not stored; save it securely before closing your terminal.

Examples:
  citadel-cli agent create mybot
  citadel-cli agent create mybot --output json
  citadel-cli agent create ci-runner --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentCreate,
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

// agentCreateResult is the machine-readable output for `agent create`.
type agentCreateResult struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	CreatedAt string    `json:"created_at"`
}

// listAgentRows fetches every page of agents owned by the authenticated user.
func listAgentRows(ctx context.Context, c *apiclient.Client) ([]agentRow, error) {
	var all []agentRow
	cur := ""
	for {
		var payload struct {
			Agents []agentRow `json:"agents"`
			Next   string     `json:"next_cursor"`
		}
		q := url.Values{}
		q.Set("limit", "200")
		if cur != "" {
			q.Set("cursor", cur)
		}
		if err := c.Get(ctx, "/agents?"+q.Encode(), &payload); err != nil {
			return nil, err
		}
		all = append(all, payload.Agents...)
		if strings.TrimSpace(payload.Next) == "" {
			break
		}
		cur = payload.Next
	}
	return all, nil
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
		return runAgentListWatch(cmd, c, limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []agentRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Agents []agentRow `json:"agents"`
			Next   string     `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/agents?"+q.Encode(), &payload); err != nil {
			return err
		}
		rows := payload.Agents
		next := strings.TrimSpace(payload.Next)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []agentRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[agentRow](cmd)
			case "yaml":
				return emitYAML(cmd, []agentRow{})
			default:
				fmt.Println("No agents found.")
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
			_, _ = fmt.Fprintln(w, "NAME\tID\tMODEL HINT")
			for _, a := range rows {
				hint := ""
				if a.ModelHint != nil {
					hint = *a.ModelHint
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.ID, hint)
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
			yamlAccum = []agentRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runAgentGet(cmd *cobra.Command, args []string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	out := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	a, err := findAgentByName(cmd.Context(), c, args[0])
	if err != nil {
		return err
	}
	return emitOne(cmd, out, a, func(w *tabwriter.Writer, a agentRow) {
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

	if dryRunFlag(cmd) {
		fmt.Printf("Would DELETE /agents/%s (skipped; --dry-run)\n", a.ID)
		return nil
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

func runAgentCreate(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("agent name must not be empty")
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateMutationOutput(output, "create"); err != nil {
		return err
	}

	// Create the agent.
	var created agentRow
	if err := c.Post(cmd.Context(), "/agents", map[string]string{"name": name}, &created); err != nil {
		return decorateAgentCreateError(err, name)
	}

	// Issue an initial token for the new agent.
	var tok tokenWithCleartext
	if err := c.Post(cmd.Context(), "/agent-tokens", map[string]any{"agent_id": created.ID}, &tok); err != nil {
		return fmt.Errorf("created agent %s but failed to issue initial token: %w", created.ID, err)
	}

	if output == "json" {
		return emitJSON(cmd, agentCreateResult{
			ID:        created.ID,
			Name:      created.Name,
			Token:     tok.CleartextToken,
			CreatedAt: tok.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Agent created\n")
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  id:   %s\n", created.ID)
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  name: %s\n", created.Name)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), tok.CleartextToken)
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "(agent token printed once above — store it securely)")
	return nil
}

// decorateAgentCreateError maps well-known server errors to friendly messages.
func decorateAgentCreateError(err error, name string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "409") || strings.Contains(msg, "already exists"):
		return fmt.Errorf("agent name already taken: %q", name)
	case strings.Contains(msg, "403"):
		return fmt.Errorf("insufficient permission to create agent %q", name)
	case strings.Contains(msg, "422"):
		return fmt.Errorf("validation error creating agent: %w", err)
	default:
		return fmt.Errorf("create agent: %w", err)
	}
}

func completeAgentNames(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeyAgents, completion.FetchAgentNames)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	AgentCmd.AddCommand(agentCreateCmd)
	AgentCmd.AddCommand(agentListCmd)
	AgentCmd.AddCommand(agentGetCmd)
	AgentCmd.AddCommand(agentDeleteCmd)
	AgentCmd.AddCommand(agentRotateTokenCmd)

	addOutputFlag(agentCreateCmd, agentListCmd, agentGetCmd, agentDeleteCmd, agentRotateTokenCmd)
	addPaginationFlags(agentListCmd)
	addWatchFlag(agentListCmd)
	addYesFlag(agentDeleteCmd, agentRotateTokenCmd)
	addDryRunFlag(agentDeleteCmd)

	agentGetCmd.ValidArgsFunction = completeAgentNames
	agentDeleteCmd.ValidArgsFunction = completeAgentNames
	agentRotateTokenCmd.ValidArgsFunction = completeAgentNames

	agentDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyAgents, completion.KeyAgentTokens)
	}
	agentRotateTokenCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyAgents, completion.KeyAgentTokens)
	}
	agentCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyAgents)
	}
}
