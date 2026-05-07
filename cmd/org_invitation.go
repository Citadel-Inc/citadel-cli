package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// OrgCmd is the top-level `citadel-cli org` command.
var OrgCmd = &cobra.Command{
	Use:   "org",
	Short: "Organization operations (invitations, …)",
	Long: `Organization-scoped commands such as membership invitations.

These routes require appropriate org grants (for example members:read /
members:write) where noted by the server.`,
}

var orgInvitationCmd = &cobra.Command{
	Use:   "invitation",
	Short: "Manage organization invitations",
	Long: `List, create, revoke, and accept invitations for org namespaces.

Create requires either --email or --slug (the invitee's public user namespace
slug). Permissions use server permission atom strings; repeat --permission or
pass a comma-separated list.`,
}

var orgInvPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List invitations pending for your account",
	Long: `Shows invitations addressed to email addresses associated with your
signed-in user.`,
	RunE: runOrgInvPending,
}

var orgInvListCmd = &cobra.Command{
	Use:   "list <org-slug>",
	Short: "List invitations for an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgInvList,
}

var orgInvCreateCmd = &cobra.Command{
	Use:   "create <org-slug>",
	Short: "Create an invitation to join an organization",
	Long: `Creates a pending invitation. Provide --email and/or --slug (invitee's
public user slug). When neither flag is set and stdin is a TTY, you are
prompted for an email interactively.

Repeat --permission for each grant, or pass comma-separated atoms.`,
	Args: cobra.ExactArgs(1),
	RunE: runOrgInvCreate,
}

var orgInvRevokeCmd = &cobra.Command{
	Use:   "revoke <org-slug> <invite-id>",
	Short: "Revoke a pending invitation",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgInvRevoke,
}

var orgInvAcceptCmd = &cobra.Command{
	Use:   "accept [token]",
	Short: "Accept an invitation using its token",
	Long: `Accepts a pending invitation. Pass the token as an argument, or use
--token-file to read it from a file (recommended: invitation URLs contain
secrets that would otherwise be saved in your shell history).

Never share invitation tokens; treat them like passwords.`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runOrgInvAccept,
}

// orgInvitationRow mirrors the org invitation list payload from orgsmembersapi.
type orgInvitationRow struct {
	ID          string     `json:"id"`
	OrgSlug     string     `json:"org_slug,omitempty"`
	Email       string     `json:"email,omitempty"`
	UserSlug    string     `json:"user_slug,omitempty"`
	Status      string     `json:"status,omitempty"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func runOrgInvPending(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}

	var payload struct {
		Invitations []orgInvitationRow `json:"invitations"`
	}
	if err := c.Get(cmd.Context(), "/invitations/pending", &payload); err != nil {
		return err
	}
	rows := payload.Invitations
	return emitOrgInvitationRows(cmd, output, rows, "No pending invitations for your account.")
}

func runOrgInvList(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	orgSlug := strings.TrimSpace(args[0])

	var payload struct {
		Invitations []orgInvitationRow `json:"invitations"`
	}
	path := "/orgs/" + url.PathEscape(orgSlug) + "/invitations"
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return err
	}
	rows := payload.Invitations
	return emitOrgInvitationRows(cmd, output, rows, fmt.Sprintf("No invitations for org '%s'.", orgSlug))
}

func emitOrgInvitationRows(cmd *cobra.Command, output string, rows []orgInvitationRow, emptyHuman string) error {
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		if len(rows) == 0 {
			return emitCSVHeaderOnly[orgInvitationRow](cmd)
		}
		var csvHdr bool
		return emitCSVRows(cmd, &csvHdr, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			fmt.Println(emptyHuman)
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tORG\tEMAIL\tUSER\tSTATUS\tPERMISSIONS\tCREATED")
		for _, r := range rows {
			short := r.ID
			if len(short) > 8 {
				short = short[:8]
			}
			perms := strings.Join(r.Permissions, ",")
			user := r.UserSlug
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				short, r.OrgSlug, r.Email, user, r.Status, perms, r.CreatedAt.Format(time.RFC3339))
		}
		return w.Flush()
	}
}

func runOrgInvCreate(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	orgSlug := strings.TrimSpace(args[0])
	email, _ := cmd.Flags().GetString("email")
	email = strings.TrimSpace(email)
	inviteeSlug, _ := cmd.Flags().GetString("slug")
	inviteeSlug = strings.TrimSpace(inviteeSlug)
	perms, _ := cmd.Flags().GetStringSlice("permission")
	perms = normalizePermissionSlice(perms)

	if email == "" && inviteeSlug == "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Email address: ")
			sc := bufio.NewScanner(os.Stdin)
			if !sc.Scan() {
				if err := sc.Err(); err != nil {
					return fmt.Errorf("read email: %w", err)
				}
				return fmt.Errorf("email required: pass --email or --slug, or enter an email when prompted")
			}
			email = strings.TrimSpace(sc.Text())
		}
		if email == "" && inviteeSlug == "" {
			return fmt.Errorf("invitee required: set --email or --slug (public user slug), or run interactively on a TTY")
		}
	}

	body := map[string]any{
		"permissions": perms,
	}
	if email != "" {
		body["email"] = email
	}
	if inviteeSlug != "" {
		body["slug"] = inviteeSlug
	}

	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	path := "/orgs/" + url.PathEscape(orgSlug) + "/invitations"
	var created orgInvitationRow
	if err := c.Post(cmd.Context(), path, body, &created); err != nil {
		return err
	}
	if output == "json" {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Invitation %s created for org %s.\n", created.ID, orgSlug)
	return nil
}

func normalizePermissionSlice(raw []string) []string {
	var out []string
	for _, chunk := range raw {
		for _, p := range strings.Split(chunk, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func runOrgInvRevoke(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	orgSlug := strings.TrimSpace(args[0])
	id := strings.TrimSpace(args[1])
	path := "/orgs/" + url.PathEscape(orgSlug) + "/invitations/" + url.PathEscape(id)
	if err := c.Delete(cmd.Context(), path); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Revoked invitation %s.\n", id)
	return nil
}

func runOrgInvAccept(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	tokFile, _ := cmd.Flags().GetString("token-file")
	tokFile = strings.TrimSpace(tokFile)
	var token string
	switch {
	case tokFile != "":
		b, err := os.ReadFile(tokFile)
		if err != nil {
			return fmt.Errorf("read --token-file: %w", err)
		}
		token = strings.TrimSpace(string(b))
	case len(args) == 1:
		token = strings.TrimSpace(args[0])
	default:
		return fmt.Errorf("invitation token required: pass as an argument or use --token-file")
	}
	if token == "" {
		return fmt.Errorf("invitation token is empty")
	}

	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	path := "/invitations/" + url.PathEscape(token) + "/accept"
	var result map[string]any
	if err := c.Post(cmd.Context(), path, nil, &result); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && he.StatusCode == http.StatusNotFound {
			return fmt.Errorf("invitation not found or expired (HTTP 404)")
		}
		return err
	}
	if output == "json" {
		return emitJSON(cmd, result)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Invitation accepted.")
	return nil
}

func init() {
	OrgCmd.AddCommand(orgInvitationCmd)
	orgInvitationCmd.AddCommand(orgInvPendingCmd)
	orgInvitationCmd.AddCommand(orgInvListCmd)
	orgInvitationCmd.AddCommand(orgInvCreateCmd)
	orgInvitationCmd.AddCommand(orgInvRevokeCmd)
	orgInvitationCmd.AddCommand(orgInvAcceptCmd)

	addOutputFlag(orgInvPendingCmd, orgInvListCmd, orgInvCreateCmd, orgInvAcceptCmd)
	orgInvCreateCmd.Flags().String("email", "", "Invitee email address")
	orgInvCreateCmd.Flags().String("slug", "", "Invitee public user namespace slug (alternative to --email)")
	orgInvCreateCmd.Flags().StringSlice("permission", nil, "Permission atom (repeat flag or comma-separated list)")
	orgInvAcceptCmd.Flags().String("token-file", "", "Read invitation token from file instead of argv (safer for secrets)")

	orgInvListCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	orgInvCreateCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	orgInvRevokeCmd.ValidArgsFunction = completeOrgNamespaceSlugs
}
