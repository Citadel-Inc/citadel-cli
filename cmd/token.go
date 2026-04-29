package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
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
	ID        uuid.UUID       `json:"id"`
	AgentID   uuid.UUID       `json:"agent_id"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	RevokedAt *time.Time      `json:"revoked_at,omitempty"`
	Scopes    interface{}     `json:"scopes"`
}

type tokenWithCleartext struct {
	token
	CleartextToken string `json:"cleartext_token"`
}

type agent struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func runTokenList(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel auth login' first")
	}

	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = "https://api.src.land"
	}

	agentName, _ := cmd.Flags().GetString("agent")
	if agentName == "" {
		return fmt.Errorf("--agent flag required")
	}

	// Look up agent ID by name
	agents, err := listAgents(cfg)
	if err != nil {
		return err
	}

	var agentID uuid.UUID
	for _, a := range agents {
		if a.Name == agentName {
			agentID = a.ID
			break
		}
	}

	if agentID == uuid.Nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}

	// List tokens for this agent
	listURL := fmt.Sprintf("%s/api/agent-tokens?agent_id=%s", serverURL, agentID.String())

	req, _ := http.NewRequest("GET", listURL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %s", string(body))
	}

	var tokens []token
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if len(tokens) == 0 {
		fmt.Printf("No tokens for agent '%s'\n", agentName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(w, "ID\tCREATED\tEXPIRES\tREVOKED\n")
	for _, t := range tokens {
		expires := ""
		if t.ExpiresAt != nil {
			expires = t.ExpiresAt.Format(time.RFC3339)
		}
		revoked := ""
		if t.RevokedAt != nil {
			revoked = t.RevokedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			t.ID.String()[:8],
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			expires,
			revoked)
	}
	w.Flush()

	return nil
}

func runTokenIssue(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel auth login' first")
	}

	agentName, _ := cmd.Flags().GetString("agent")
	if agentName == "" {
		return fmt.Errorf("--agent flag required")
	}

	scopes, _ := cmd.Flags().GetStringSlice("scopes")
	expiresStr, _ := cmd.Flags().GetString("expires")

	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = "https://api.src.land"
	}

	// Create or find agent
	agents, err := listAgents(cfg)
	if err != nil {
		return err
	}

	var agentID uuid.UUID
	for _, a := range agents {
		if a.Name == agentName {
			agentID = a.ID
			break
		}
	}

	if agentID == uuid.Nil {
		// Create agent
		createURL := fmt.Sprintf("%s/api/agents", serverURL)
		createReq := struct {
			Name string `json:"name"`
		}{Name: agentName}

		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", createURL, bytes.NewReader(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("create agent request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("create agent failed: %s", string(body))
		}

		var ag agent
		if err := json.NewDecoder(resp.Body).Decode(&ag); err != nil {
			return fmt.Errorf("decode agent response: %w", err)
		}
		agentID = ag.ID
	}

	// Issue token
	var expiresIn *int64
	if expiresStr != "" {
		d, err := time.ParseDuration(expiresStr)
		if err == nil {
			sec := int64(d.Seconds())
			expiresIn = &sec
		}
	}

	issueURL := fmt.Sprintf("%s/api/agent-tokens", serverURL)
	issueReq := struct {
		AgentID         uuid.UUID `json:"agent_id"`
		ExpiresInSeconds *int64   `json:"expires_in_seconds,omitempty"`
		Scopes          []string `json:"scopes,omitempty"`
	}{
		AgentID:         agentID,
		ExpiresInSeconds: expiresIn,
		Scopes:          scopes,
	}

	body, _ := json.Marshal(issueReq)
	req, _ := http.NewRequest("POST", issueURL, bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("issue token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("issue token failed: %s", string(body))
	}

	var tok tokenWithCleartext
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	// Print the token (once, no debug noise)
	fmt.Println(tok.CleartextToken)

	return nil
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel auth login' first")
	}

	tokenID := args[0]

	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = "https://api.src.land"
	}

	revokeURL := fmt.Sprintf("%s/api/agent-tokens/%s", serverURL, tokenID)

	req, _ := http.NewRequest("DELETE", revokeURL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revoke failed: %s", string(body))
	}

	fmt.Printf("Token revoked: %s\n", tokenID)
	return nil
}

// Helper to list agents for the authenticated user
func listAgents(cfg clicfg.Config) ([]agent, error) {
	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = "https://api.src.land"
	}

	listURL := fmt.Sprintf("%s/api/agents", serverURL)

	req, _ := http.NewRequest("GET", listURL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s", string(body))
	}

	var agents []agent
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return agents, nil
}

func init() {
	TokenCmd.AddCommand(listCmd)
	TokenCmd.AddCommand(issueCmd)
	TokenCmd.AddCommand(revokeCmd)

	// Add flags
	listCmd.Flags().String("agent", "", "Agent name (required)")
	issueCmd.Flags().String("agent", "", "Agent name (required)")
	issueCmd.Flags().StringSlice("scopes", []string{}, "Token scopes (optional)")
	issueCmd.Flags().String("expires", "", "Expiration duration (optional, e.g. '24h')")
}
