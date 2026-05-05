package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
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

// ── delete ───────────────────────────────────────────────────────────────────

var nsDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Hard-purge an org namespace you own",
	Long: `Hard-purges an org namespace and writes a slug-hold tombstone in
namespace_aliases (default 30 + 30 days post-purge reservation).

Owner-only; only kind=org is accepted (user / system namespaces cannot be
deleted via this surface). 409 has_repos is returned if any live child
repos exist — delete them first via 'citadel-cli repo delete'.

The DELETE is a real hard purge: the namespaces row + every FK-cascaded
child (org_invitations, org_transfers, org_passkey_policies,
namespace_profiles, namespace_avatar_sync, namespace_pins,
namespace_grants, etc.) is removed in one tx. Search index
(searchable_namespaces) refreshes after commit.

Requires typed-slug confirmation unless --yes is set.

Examples:
  citadel-cli namespace delete my-test-org
  citadel-cli namespace delete my-test-org --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runNsDelete,
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

type nsRow struct {
	NamespaceID   string    `json:"namespace_id"`
	Slug          string    `json:"slug"`
	Kind          string    `json:"kind"`
	Path          string    `json:"path"`
	Visibility    string    `json:"visibility"`
	OwnerUserID   string    `json:"owner_user_id,omitempty"`
	DisplayName   string    `json:"display_name,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	Archived      bool      `json:"archived,omitempty"`
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

func listOrgNamespaces(ctx context.Context, c *apiclient.Client) ([]nsOrgRow, error) {
	var payload struct {
		Orgs []nsOrgRow `json:"orgs"`
	}
	if err := c.Get(ctx, "/orgs", &payload); err != nil {
		return nil, err
	}
	return payload.Orgs, nil
}

func runNsList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")

	orgs, err := listOrgNamespaces(cmd.Context(), c)
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
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	slug := args[0]

	var ns nsRow
	if err := c.Get(cmd.Context(), "/namespaces/"+url.PathEscape(slug), &ns); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && he.StatusCode == http.StatusNotFound {
			return fmt.Errorf("namespace '%s' not found", slug)
		}
		return err
	}

	if output == "json" {
		return emitJSON(ns)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Slug:\t%s\n", ns.Slug)
	_, _ = fmt.Fprintf(w, "Kind:\t%s\n", ns.Kind)
	_, _ = fmt.Fprintf(w, "Visibility:\t%s\n", ns.Visibility)
	if ns.DisplayName != "" {
		_, _ = fmt.Fprintf(w, "Display name:\t%s\n", ns.DisplayName)
	}
	if ns.Description != "" {
		_, _ = fmt.Fprintf(w, "Description:\t%s\n", ns.Description)
	}
	_, _ = fmt.Fprintf(w, "Namespace ID:\t%s\n", ns.NamespaceID)
	_, _ = fmt.Fprintf(w, "Created:\t%s\n", ns.CreatedAt.Format(time.RFC3339))
	return w.Flush()
}

func runNsMembers(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	slug := args[0]

	var payload struct {
		Members []nsMemberRow `json:"members"`
	}
	if err := c.Get(cmd.Context(), "/orgs/"+url.PathEscape(slug)+"/members", &payload); err != nil {
		return err
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
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
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

	var result map[string]any
	if err := c.Post(cmd.Context(), "/orgs/"+url.PathEscape(orgSlug)+"/transfer", map[string]string{"to_username": to}, &result); err != nil {
		return err
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

func runNsTransferListPending(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")

	var payload struct {
		Transfers []nsTransferRow `json:"transfers"`
	}
	if err := c.Get(cmd.Context(), "/transfers/pending", &payload); err != nil {
		return err
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
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	transferID := args[0]

	var result map[string]any
	if err := c.Post(cmd.Context(), "/transfers/"+url.PathEscape(transferID)+"/accept", nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(result)
	}
	fmt.Printf("Transfer %s accepted.\n", transferID)
	return nil
}

func runNsTransferDecline(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	transferID := args[0]

	if err := c.Post(cmd.Context(), "/transfers/"+url.PathEscape(transferID)+"/decline", nil, nil); err != nil {
		return err
	}
	fmt.Printf("Transfer %s declined.\n", transferID)
	return nil
}

func runNsTransferRevoke(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	transferID := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "revoke transfer", transferID); err != nil {
		return err
	}

	if err := c.Delete(cmd.Context(), "/transfers/"+url.PathEscape(transferID)); err != nil {
		return err
	}
	fmt.Printf("Transfer %s revoked.\n", transferID)
	return nil
}

func runNsDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	slug := strings.TrimSpace(args[0])
	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "delete namespace", slug); err != nil {
		return err
	}

	if err := c.Delete(cmd.Context(), "/namespaces/"+url.PathEscape(slug)); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) {
			switch he.StatusCode {
			case http.StatusConflict:
				var body struct {
					Error  string `json:"error"`
					Detail string `json:"detail"`
				}
				_ = json.Unmarshal([]byte(he.Body), &body)
				if body.Error == "has_repos" {
					msg := body.Detail
					if msg == "" {
						msg = "delete repos under " + slug + " first"
					}
					return fmt.Errorf("namespace not empty: %s — run 'citadel-cli repo delete <ns>/<slug>' for each", msg)
				}
				return fmt.Errorf("conflict: %s", body.Error)
			case http.StatusForbidden:
				return fmt.Errorf("forbidden: only the owner can delete namespace %s", slug)
			case http.StatusNotFound:
				return fmt.Errorf("namespace %s not found, not an org, or already deleted", slug)
			}
		}
		return err
	}
	fmt.Printf("Deleted namespace %s\n", slug)
	return nil
}

func init() {
	NamespaceCmd.AddCommand(nsListCmd)
	NamespaceCmd.AddCommand(nsGetCmd)
	NamespaceCmd.AddCommand(nsMembersCmd)
	NamespaceCmd.AddCommand(nsDeleteCmd)
	NamespaceCmd.AddCommand(nsTransferCmd)

	nsTransferCmd.AddCommand(nsTransferInitiateCmd)
	nsTransferCmd.AddCommand(nsTransferListPendingCmd)
	nsTransferCmd.AddCommand(nsTransferAcceptCmd)
	nsTransferCmd.AddCommand(nsTransferDeclineCmd)
	nsTransferCmd.AddCommand(nsTransferRevokeCmd)

	nsListCmd.Flags().String("output", "", "Output format: json")
	nsGetCmd.Flags().String("output", "", "Output format: json")
	nsMembersCmd.Flags().String("output", "", "Output format: json")
	nsDeleteCmd.Flags().Bool("yes", false, "Skip confirmation prompt")

	nsTransferInitiateCmd.Flags().String("to", "", "Recipient username (required)")
	nsTransferInitiateCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	nsTransferInitiateCmd.Flags().String("output", "", "Output format: json")

	nsTransferListPendingCmd.Flags().String("output", "", "Output format: json")
	nsTransferAcceptCmd.Flags().String("output", "", "Output format: json")
	nsTransferDeclineCmd.Flags().String("output", "", "Output format: json")
	nsTransferRevokeCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	nsTransferRevokeCmd.Flags().String("output", "", "Output format: json")
}
