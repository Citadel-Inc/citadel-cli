package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
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

func listAgentRows(serverURL, accessToken string) ([]agentRow, error) {
	req, _ := http.NewRequest(http.MethodGet, serverURL+"/api/agents", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var rows []agentRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return rows, nil
}

func runAgentList(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)
	output, _ := cmd.Flags().GetString("output")

	rows, err := listAgentRows(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(rows)
	}

	if len(rows) == 0 {
		fmt.Println("No agents found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tID\tMODEL HINT")
	for _, a := range rows {
		hint := ""
		if a.ModelHint != nil {
			hint = *a.ModelHint
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.ID, hint)
	}
	return w.Flush()
}

func runAgentGet(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)
	output, _ := cmd.Flags().GetString("output")

	name := args[0]

	rows, err := listAgentRows(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}

	for _, a := range rows {
		if a.Name == name {
			if output == "json" {
				return emitJSON(a)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "Name:\t%s\n", a.Name)
			_, _ = fmt.Fprintf(w, "ID:\t%s\n", a.ID)
			if a.ModelHint != nil {
				_, _ = fmt.Fprintf(w, "Model hint:\t%s\n", *a.ModelHint)
			}
			return w.Flush()
		}
	}
	return fmt.Errorf("agent '%s' not found", name)
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)

	name := args[0]

	rows, err := listAgentRows(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}

	var agentID string
	for _, a := range rows {
		if a.Name == name {
			agentID = a.ID
			break
		}
	}
	if agentID == "" {
		return fmt.Errorf("agent '%s' not found", name)
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "delete agent", name); err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/api/agents/%s", serverURL, url.PathEscape(agentID))
	req, _ := http.NewRequest(http.MethodDelete, apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Agent '%s' deleted.\n", name)
	return nil
}

func runAgentRotateToken(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)

	name := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "rotate token for agent", name); err != nil {
		return err
	}

	// Resolve agent ID.
	rows, err := listAgentRows(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}
	var agentID string
	for _, a := range rows {
		if a.Name == name {
			agentID = a.ID
			break
		}
	}
	if agentID == "" {
		return fmt.Errorf("agent '%s' not found", name)
	}

	// Issue new token first.
	issueBody := struct {
		AgentID string `json:"agent_id"`
	}{AgentID: agentID}
	bodyBytes, _ := json.Marshal(issueBody)

	issueReq, _ := http.NewRequest(http.MethodPost, serverURL+"/api/agent-tokens", bytes.NewReader(bodyBytes))
	issueReq.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	issueReq.Header.Set("Content-Type", "application/json")

	issueResp, err := http.DefaultClient.Do(issueReq)
	if err != nil {
		return fmt.Errorf("issue token request failed: %w", err)
	}
	defer func() { _ = issueResp.Body.Close() }()

	if issueResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(issueResp.Body)
		return fmt.Errorf("issue token failed %d: %s", issueResp.StatusCode, string(body))
	}

	var newTok agentTokenWithCleartext
	if err := json.NewDecoder(issueResp.Body).Decode(&newTok); err != nil {
		return fmt.Errorf("decode issue response: %w", err)
	}

	// List existing tokens to revoke all but the new one.
	listURL := fmt.Sprintf("%s/api/agent-tokens?agent_id=%s", serverURL, agentID)
	listReq, _ := http.NewRequest(http.MethodGet, listURL, nil)
	listReq.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		return fmt.Errorf("list tokens request failed: %w", err)
	}
	defer func() { _ = listResp.Body.Close() }()

	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		return fmt.Errorf("list tokens failed %d: %s", listResp.StatusCode, string(body))
	}

	var existingTokens []agentToken
	if err := json.NewDecoder(listResp.Body).Decode(&existingTokens); err != nil {
		return fmt.Errorf("decode tokens response: %w", err)
	}

	// Revoke all tokens except the new one.
	var revokeErr error
	for _, t := range existingTokens {
		if t.ID == newTok.ID {
			continue
		}
		if t.RevokedAt != nil {
			continue
		}
		revokeAPIURL := fmt.Sprintf("%s/api/agent-tokens/%s", serverURL, url.PathEscape(t.ID))
		revokeReq, _ := http.NewRequest(http.MethodDelete, revokeAPIURL, nil)
		revokeReq.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
		revokeResp, err := http.DefaultClient.Do(revokeReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to revoke token %s: %v\n", t.ID, err)
			revokeErr = err
			continue
		}
		_ = revokeResp.Body.Close()
		if revokeResp.StatusCode != http.StatusNoContent && revokeResp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "warning: revoke token %s returned %d\n", t.ID, revokeResp.StatusCode)
			revokeErr = fmt.Errorf("revoke returned %d", revokeResp.StatusCode)
		}
	}

	// Print new token once to stdout before surfacing any revoke warnings.
	fmt.Println(newTok.CleartextToken)
	if revokeErr != nil {
		return fmt.Errorf("new token issued but one or more old tokens could not be revoked; check 'citadel-cli token list --agent %s'", name)
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
