package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

// NamespaceCmd is the top-level `citadel namespace` command.
var NamespaceCmd = &cobra.Command{
	Use:     "namespace",
	Aliases: []string{"ns"},
	Short:   "Manage namespaces (list, get, members, transfer)",
	Long: `Operations against Citadel org namespaces.

Note: namespace list/get/members/transfer currently operates on org namespaces.
Personal namespace transfer support requires a future server-side extension.`,
}

// ── list ─────────────────────────────────────────────────────────────────────

var nsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List org namespaces you own or belong to",
	Long: `Returns all org namespaces the authenticated user owns or is a member of.

Examples:
  citadel-cli namespace list
  citadel-cli namespace list --output json`,
	RunE: runNsList,
}

// ── get ──────────────────────────────────────────────────────────────────────

var nsGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get details of a namespace",
	Long: `Fetches metadata for a single org namespace by slug.

Examples:
  citadel-cli namespace get myorg
  citadel-cli namespace get myorg --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runNsGet,
}

// ── members ──────────────────────────────────────────────────────────────────

var nsMembersCmd = &cobra.Command{
	Use:   "members <slug>",
	Short: "List members of an org namespace",
	Long: `Lists members of an org namespace. Requires members:read permission.
Only works for org namespaces (kind=org).

Examples:
  citadel-cli namespace members myorg
  citadel-cli namespace members myorg --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runNsMembers,
}

// ── transfer ─────────────────────────────────────────────────────────────────

var nsTransferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer namespace ownership (initiate, accept, decline, revoke)",
	Long: `Two-party namespace ownership transfer flow.

Initiate a transfer with 'initiate'; the recipient confirms via 'accept'.
Use 'list-pending' to see incoming transfers awaiting your acceptance.`,
}

var nsTransferInitiateCmd = &cobra.Command{
	Use:   "initiate <org-slug>",
	Short: "Initiate a namespace ownership transfer",
	Long: `Proposes transferring ownership of an org namespace to another user.
The recipient must accept via 'citadel-cli namespace transfer accept <id>'.
Requires typed-slug confirmation unless --yes is set.

Note: only org namespaces are currently supported by the server.

Examples:
  citadel-cli namespace transfer initiate myorg --to newowner
  citadel-cli namespace transfer initiate myorg --to newowner --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runNsTransferInitiate,
}

var nsTransferListPendingCmd = &cobra.Command{
	Use:   "list-pending",
	Short: "List pending incoming namespace transfers",
	Long: `Lists namespace transfer requests addressed to the authenticated user
that have not yet been accepted, declined, or expired.

Examples:
  citadel-cli namespace transfer list-pending
  citadel-cli namespace transfer list-pending --output json`,
	RunE: runNsTransferListPending,
}

var nsTransferAcceptCmd = &cobra.Command{
	Use:   "accept <transfer-id>",
	Short: "Accept a pending namespace transfer",
	Long: `Accepts an incoming namespace ownership transfer request.

Examples:
  citadel-cli namespace transfer accept 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: runNsTransferAccept,
}

var nsTransferDeclineCmd = &cobra.Command{
	Use:   "decline <transfer-id>",
	Short: "Decline a pending namespace transfer",
	Long: `Declines an incoming namespace ownership transfer request.

Examples:
  citadel-cli namespace transfer decline 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: runNsTransferDecline,
}

var nsTransferRevokeCmd = &cobra.Command{
	Use:   "revoke <transfer-id>",
	Short: "Revoke an outgoing namespace transfer you initiated",
	Long: `Cancels a namespace transfer that you initiated but the recipient has
not yet accepted. Requires typed-slug confirmation unless --yes is set.

Examples:
  citadel-cli namespace transfer revoke 550e8400-e29b-41d4-a716-446655440000
  citadel-cli namespace transfer revoke 550e8400-e29b-41d4-a716-446655440000 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runNsTransferRevoke,
}

// ── shapes ───────────────────────────────────────────────────────────────────

type nsOrgRow struct {
	NamespaceID     string    `json:"namespace_id"`
	Slug            string    `json:"slug"`
	DisplayName     string    `json:"display_name,omitempty"`
	LegalEntityName string    `json:"legal_entity_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type nsMemberRow struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email,omitempty"`
	Slug        string    `json:"slug,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	IsOwner     bool      `json:"is_owner"`
	Permissions []string  `json:"permissions"`
	JoinedAt    time.Time `json:"joined_at"`
}

type nsTransferRow struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_namespace_id"`
	OrgSlug      string    `json:"org_slug"`
	OrgName      string    `json:"org_name,omitempty"`
	FromUserID   string    `json:"from_user_id"`
	FromUserSlug string    `json:"from_user_slug,omitempty"`
	ToUserID     string    `json:"to_user_id"`
	ToUserSlug   string    `json:"to_user_slug,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── handlers ─────────────────────────────────────────────────────────────────

func listOrgNamespaces(serverURL, accessToken string) ([]nsOrgRow, error) {
	req, _ := http.NewRequest(http.MethodGet, serverURL+"/api/orgs", nil)
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

	var payload struct {
		Orgs []nsOrgRow `json:"orgs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return payload.Orgs, nil
}

func runNsList(cmd *cobra.Command, args []string) error {
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

	orgs, err := listOrgNamespaces(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(orgs)
	}

	if len(orgs) == 0 {
		fmt.Println("No org namespaces found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SLUG\tDISPLAY NAME\tCREATED")
	for _, o := range orgs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", o.Slug, o.DisplayName, o.CreatedAt.Format("2006-01-02"))
	}
	return w.Flush()
}

func runNsGet(cmd *cobra.Command, args []string) error {
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

	slug := args[0]

	orgs, err := listOrgNamespaces(serverURL, cfg.AccessToken)
	if err != nil {
		return err
	}

	for _, o := range orgs {
		if strings.EqualFold(o.Slug, slug) {
			if output == "json" {
				return emitJSON(o)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "Slug:\t%s\n", o.Slug)
			_, _ = fmt.Fprintf(w, "Display name:\t%s\n", o.DisplayName)
			if o.LegalEntityName != "" {
				_, _ = fmt.Fprintf(w, "Legal entity:\t%s\n", o.LegalEntityName)
			}
			_, _ = fmt.Fprintf(w, "Namespace ID:\t%s\n", o.NamespaceID)
			_, _ = fmt.Fprintf(w, "Created:\t%s\n", o.CreatedAt.Format(time.RFC3339))
			return w.Flush()
		}
	}
	return fmt.Errorf("namespace '%s' not found", slug)
}

func runNsMembers(cmd *cobra.Command, args []string) error {
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

	slug := args[0]

	apiURL := fmt.Sprintf("%s/api/orgs/%s/members", serverURL, url.PathEscape(slug))
	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Members []nsMemberRow `json:"members"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	members := payload.Members

	if output == "json" {
		return emitJSON(members)
	}

	if len(members) == 0 {
		fmt.Printf("No members in namespace '%s'\n", slug)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SLUG\tDISPLAY NAME\tROLE\tJOINED")
	for _, m := range members {
		role := "member"
		if m.IsOwner {
			role = "owner"
		}
		name := m.DisplayName
		if name == "" {
			name = m.Slug
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Slug, name, role, m.JoinedAt.Format("2006-01-02"))
	}
	return w.Flush()
}

func runNsTransferInitiate(cmd *cobra.Command, args []string) error {
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

	orgSlug := args[0]
	to, _ := cmd.Flags().GetString("to")
	if to == "" {
		return fmt.Errorf("--to is required")
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "transfer", orgSlug); err != nil {
		return err
	}

	reqBody := struct {
		ToUsername string `json:"to_username,omitempty"`
	}{
		ToUsername: to,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	apiURL := fmt.Sprintf("%s/api/orgs/%s/transfer", serverURL, url.PathEscape(orgSlug))
	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if output == "json" {
		return emitJSON(result)
	}

	id, _ := result["id"].(string)
	fmt.Printf("Transfer initiated. ID: %s\n", id)
	fmt.Printf("The recipient '%s' must accept via:\n", to)
	fmt.Printf("  citadel-cli namespace transfer accept %s\n", id)
	return nil
}

func runNsTransferListPending(cmd *cobra.Command, args []string) error {
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

	req, _ := http.NewRequest(http.MethodGet, serverURL+"/api/transfers/pending", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Transfers []nsTransferRow `json:"transfers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if output == "json" {
		return emitJSON(payload.Transfers)
	}

	if len(payload.Transfers) == 0 {
		fmt.Println("No pending transfers.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tORG\tFROM\tEXPIRES")
	for _, t := range payload.Transfers {
		shortID := t.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			shortID, t.OrgSlug, t.FromUserSlug, t.ExpiresAt.Format("2006-01-02"))
	}
	return w.Flush()
}

func runNsTransferAccept(cmd *cobra.Command, args []string) error {
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

	transferID := args[0]

	apiURL := fmt.Sprintf("%s/api/transfers/%s/accept", serverURL, url.PathEscape(transferID))
	req, _ := http.NewRequest(http.MethodPost, apiURL, http.NoBody)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if output == "json" {
		return emitJSON(result)
	}
	fmt.Printf("Transfer %s accepted.\n", transferID)
	return nil
}

func runNsTransferDecline(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)

	transferID := args[0]

	apiURL := fmt.Sprintf("%s/api/transfers/%s/decline", serverURL, url.PathEscape(transferID))
	req, _ := http.NewRequest(http.MethodPost, apiURL, http.NoBody)
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
	fmt.Printf("Transfer %s declined.\n", transferID)
	return nil
}

func runNsTransferRevoke(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)

	transferID := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "revoke transfer", transferID); err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/api/transfers/%s", serverURL, url.PathEscape(transferID))
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
	fmt.Printf("Transfer %s revoked.\n", transferID)
	return nil
}

func init() {
	NamespaceCmd.AddCommand(nsListCmd)
	NamespaceCmd.AddCommand(nsGetCmd)
	NamespaceCmd.AddCommand(nsMembersCmd)
	NamespaceCmd.AddCommand(nsTransferCmd)

	nsTransferCmd.AddCommand(nsTransferInitiateCmd)
	nsTransferCmd.AddCommand(nsTransferListPendingCmd)
	nsTransferCmd.AddCommand(nsTransferAcceptCmd)
	nsTransferCmd.AddCommand(nsTransferDeclineCmd)
	nsTransferCmd.AddCommand(nsTransferRevokeCmd)

	nsListCmd.Flags().String("output", "", "Output format: json")
	nsGetCmd.Flags().String("output", "", "Output format: json")
	nsMembersCmd.Flags().String("output", "", "Output format: json")

	nsTransferInitiateCmd.Flags().String("to", "", "Recipient username (required)")
	nsTransferInitiateCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	nsTransferInitiateCmd.Flags().String("output", "", "Output format: json")

	nsTransferListPendingCmd.Flags().String("output", "", "Output format: json")
	nsTransferAcceptCmd.Flags().String("output", "", "Output format: json")
	nsTransferDeclineCmd.Flags().String("output", "", "Output format: json")
	nsTransferRevokeCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	nsTransferRevokeCmd.Flags().String("output", "", "Output format: json")
}
