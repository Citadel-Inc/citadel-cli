package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
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

func runNsList(cmd *cobra.Command, _ []string) error {
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
		return runNsListWatch(cmd, c, limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []nsOrgRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Orgs       []nsOrgRow `json:"orgs"`
			NextCursor string     `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/orgs?"+q.Encode(), &payload); err != nil {
			return err
		}
		orgs := payload.Orgs
		next := strings.TrimSpace(payload.NextCursor)

		if len(orgs) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(orgs) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []nsOrgRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[nsOrgRow](cmd)
			case "yaml":
				return emitYAML(cmd, []nsOrgRow{})
			default:
				fmt.Println("No org namespaces found.")
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(cmd, orgs)
		case "ndjson":
			if err := emitNDJSONLines(cmd, orgs); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, orgs); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, orgs...)
			} else {
				return emitYAML(cmd, orgs)
			}
		default:
			w := newTabWriter(cmd)
			_, _ = fmt.Fprintln(w, "SLUG\tDISPLAY NAME\tCREATED")
			for _, o := range orgs {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", o.Slug, o.DisplayName, o.CreatedAt.Format("2006-01-02"))
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
			yamlAccum = []nsOrgRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runNsGet(cmd *cobra.Command, args []string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	slug := args[0]

	var ns nsRow
	if err := c.Get(cmd.Context(), "/namespaces/"+url.PathEscape(slug), &ns); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("namespace '%s' not found", slug)
		}
		return err
	}

	return emitOne(cmd, output, ns, func(w *tabwriter.Writer, ns nsRow) {
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
	})
}

func runNsMembers(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	slug := args[0]
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
		if err := validateMemberCursor(cursor); err != nil {
			return fmt.Errorf("invalid --cursor: %w", err)
		}
		return runNsMembersWatch(cmd, c, slug, limit, cursor, all)
	}
	if err := validateMemberCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []nsMemberRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Members    []nsMemberRow `json:"members"`
			NextCursor string        `json:"next_cursor"`
		}
		path := "/orgs/" + url.PathEscape(slug) + "/members?" + q.Encode()
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return err
		}
		members := payload.Members
		next := strings.TrimSpace(payload.NextCursor)

		if len(members) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(members) == 0 && cursor == "" {
			empty := fmt.Sprintf("No members in namespace '%s'", slug)
			switch output {
			case "json":
				return emitJSON(cmd, []nsMemberRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[nsMemberRow](cmd)
			case "yaml":
				return emitYAML(cmd, []nsMemberRow{})
			default:
				fmt.Println(empty)
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(cmd, members)
		case "ndjson":
			if err := emitNDJSONLines(cmd, members); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, members); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, members...)
			} else {
				return emitYAML(cmd, members)
			}
		default:
			w := newTabWriter(cmd)
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
			yamlAccum = []nsMemberRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runNsTransferInitiate(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	orgSlug := args[0]
	to, _ := cmd.Flags().GetString("to")

	if err := confirmSlug(yesFlag(cmd), "transfer", orgSlug); err != nil {
		return err
	}

	var result map[string]any
	if err := c.Post(cmd.Context(), "/orgs/"+url.PathEscape(orgSlug)+"/transfer", map[string]string{"to_username": to}, &result); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(cmd, result)
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
		return runNsTransferListPendingWatch(cmd, c, limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []nsTransferRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Transfers  []nsTransferRow `json:"transfers"`
			NextCursor string          `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/transfers/pending?"+q.Encode(), &payload); err != nil {
			return err
		}
		transfers := payload.Transfers
		next := strings.TrimSpace(payload.NextCursor)

		if len(transfers) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(transfers) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []nsTransferRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[nsTransferRow](cmd)
			case "yaml":
				return emitYAML(cmd, []nsTransferRow{})
			default:
				fmt.Println("No pending transfers.")
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(cmd, transfers)
		case "ndjson":
			if err := emitNDJSONLines(cmd, transfers); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, transfers); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, transfers...)
			} else {
				return emitYAML(cmd, transfers)
			}
		default:
			w := newTabWriter(cmd)
			_, _ = fmt.Fprintln(w, "ID\tORG\tFROM\tEXPIRES")
			for _, tr := range transfers {
				shortID := tr.ID[:min(len(tr.ID), 8)]
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					shortID, tr.OrgSlug, tr.FromUserSlug, tr.ExpiresAt.Format("2006-01-02"))
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
			yamlAccum = []nsTransferRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runNsTransferAccept(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	transferID := args[0]

	var result map[string]any
	if err := c.Post(cmd.Context(), "/transfers/"+url.PathEscape(transferID)+"/accept", nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(cmd, result)
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

	if dryRunFlag(cmd) {
		fmt.Printf("Would DELETE /transfers/%s (skipped; --dry-run)\n", transferID)
		return nil
	}
	if err := confirmSlug(yesFlag(cmd), "revoke transfer", transferID); err != nil {
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
	if dryRunFlag(cmd) {
		fmt.Printf("Would DELETE /namespaces/%s (skipped; --dry-run)\n", slug)
		return nil
	}
	if err := confirmSlug(yesFlag(cmd), "delete namespace", slug); err != nil {
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
				_ = he.DecodeBody(&body)
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

func completeOrgNamespaceSlugs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeyOrgs, completion.FetchOrgNamespaceSlugs)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
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

	addOutputFlag(nsListCmd, nsGetCmd, nsMembersCmd, nsDeleteCmd,
		nsTransferInitiateCmd, nsTransferListPendingCmd,
		nsTransferAcceptCmd, nsTransferDeclineCmd, nsTransferRevokeCmd)
	addPaginationFlags(nsListCmd, nsMembersCmd, nsTransferListPendingCmd)
	addWatchFlag(nsListCmd, nsMembersCmd, nsTransferListPendingCmd)
	addYesFlag(nsDeleteCmd, nsTransferInitiateCmd, nsTransferRevokeCmd)
	addDryRunFlag(nsDeleteCmd, nsTransferRevokeCmd)
	nsTransferInitiateCmd.Flags().String("to", "", "Recipient username (required)")
	_ = nsTransferInitiateCmd.MarkFlagRequired("to")

	nsGetCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	nsMembersCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	nsDeleteCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	nsTransferInitiateCmd.ValidArgsFunction = completeOrgNamespaceSlugs

	nsDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOrgs)
	}
	nsTransferInitiateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOrgs)
	}
	nsTransferAcceptCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOrgs)
	}
	nsTransferDeclineCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOrgs)
	}
	nsTransferRevokeCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOrgs)
	}
}
