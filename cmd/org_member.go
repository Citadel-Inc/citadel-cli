package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

var orgMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage org members",
	Long: `List members and manage their permissions within an org namespace.

Subcommands:
  list             List all members with their permission sets
  set-permissions  Replace a member's permission set
  remove           Remove a member from the org

Requires members:read (list) or members:write (set-permissions, remove) on the
org namespace, or org ownership.`,
}

var orgMemberListCmd = &cobra.Command{
	Use:   "list <org-slug>",
	Short: "List members of an organization",
	Long: `Lists all members of an org namespace, including their permission sets and
join date. The org owner is listed first and marked in the ROLE column.

Accepts --limit / --cursor / --all for pagination, and --output for format.

Examples:
  citadel-cli org member list myorg
  citadel-cli org member list myorg --output json
  citadel-cli org member list myorg --all --output ndjson`,
	Args: cobra.ExactArgs(1),
	RunE: runOrgMemberList,
}

var orgMemberSetPermissionsCmd = &cobra.Command{
	Use:   "set-permissions <org-slug> <member>",
	Short: "Replace the permission set for an org member",
	Long: `Replaces the complete permission set for a member. Existing permissions are
cleared and replaced with the specified atoms.

Pass each permission atom with --permission (repeatable) or as a comma-separated
list. Omit --permission entirely to clear all grants.

<member> is a user UUID or a user slug (slug is resolved via the member list).
The org owner's permissions cannot be modified.

Examples:
  citadel-cli org member set-permissions myorg alice --permission members:read,code:read
  citadel-cli org member set-permissions myorg alice --permission code:read --permission code:write
  citadel-cli org member set-permissions myorg alice   # clears all grants`,
	Args: cobra.ExactArgs(2),
	RunE: runOrgMemberSetPermissions,
}

var orgMemberRemoveCmd = &cobra.Command{
	Use:   "remove <org-slug> <member>",
	Short: "Remove a member from an organization",
	Long: `Removes a member from the org namespace and clears all their grants.

The org owner cannot be removed. Removing yourself is blocked if you are the
only members:write holder (to prevent lockout).

Prompts for confirmation on a TTY unless --yes is passed.
<member> is a user UUID or a user slug.

Examples:
  citadel-cli org member remove myorg alice
  citadel-cli org member remove myorg alice --yes`,
	Args: cobra.ExactArgs(2),
	RunE: runOrgMemberRemove,
}

// isUUIDShaped returns true if s matches the 8-4-4-4-12 hex UUID layout.
func isUUIDShaped(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	lens := [5]int{8, 4, 4, 4, 12}
	for i, p := range parts {
		if len(p) != lens[i] {
			return false
		}
		for _, c := range p {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				return false
			}
		}
	}
	return true
}

// resolveMemberToUUID returns the user UUID for <member>.
// UUID-shaped args are returned as-is; slugs are resolved by paging through
// the org member list and matching on the slug field.
func resolveMemberToUUID(cmd *cobra.Command, c *apiclient.Client, orgSlug, member string) (string, error) {
	if isUUIDShaped(member) {
		return strings.ToLower(strings.TrimSpace(member)), nil
	}
	slugLower := strings.ToLower(strings.TrimSpace(member))
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", "200")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Members    []nsMemberRow `json:"members"`
			NextCursor string        `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/orgs/"+url.PathEscape(orgSlug)+"/members?"+q.Encode(), &page); err != nil {
			return "", err
		}
		for _, m := range page.Members {
			if strings.ToLower(m.Slug) == slugLower {
				return m.UserID, nil
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return "", fmt.Errorf("member %q not found in org %q", member, orgSlug)
}

// orgMemberFriendlyError translates org-member specific 403/400 error codes
// into actionable messages before falling through to the generic error chain.
func orgMemberFriendlyError(err error) error {
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return err
	}
	if he.StatusCode != http.StatusForbidden && he.StatusCode != http.StatusBadRequest {
		return err
	}
	var body struct {
		Error string `json:"error"`
	}
	if decErr := json.Unmarshal([]byte(he.Body), &body); decErr != nil || body.Error == "" {
		return err
	}
	switch body.Error {
	case "cannot_modify_owner":
		return fmt.Errorf("cannot change permissions for the org owner")
	case "cannot_remove_owner":
		return fmt.Errorf("cannot remove the org owner")
	case "self_removal_lockout":
		return fmt.Errorf("cannot remove yourself: no other members:write holder remains in the org")
	case "invalid_permission":
		return fmt.Errorf("unknown permission atom — valid atoms include members:read, members:write, code:read, code:write, issues:read, issues:write, audit:read, and others")
	}
	return err
}

func runOrgMemberList(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson or omit --all")
	}
	if err := validateMemberCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}
	orgSlug := strings.TrimSpace(args[0])

	var yamlAccum []nsMemberRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Members    []nsMemberRow `json:"members"`
			NextCursor string        `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/orgs/"+url.PathEscape(orgSlug)+"/members?"+q.Encode(), &page); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("org '%s' not found", orgSlug)
			}
			return err
		}
		rows := page.Members
		next := strings.TrimSpace(page.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No members found for org '%s'.\n", orgSlug)
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
			_, _ = fmt.Fprintln(w, "USER_ID\tSLUG\tDISPLAY_NAME\tROLE\tPERMISSIONS\tJOINED")
			for _, m := range rows {
				role := "member"
				if m.IsOwner {
					role = "owner"
				}
				short := m.UserID
				if len(short) > 8 {
					short = short[:8]
				}
				name := m.DisplayName
				if name == "" {
					name = "-"
				}
				_, _ = fmt.Fprintf(w, "%s…\t%s\t%s\t%s\t%s\t%s\n",
					short, m.Slug, name, role,
					strings.Join(m.Permissions, ","),
					m.JoinedAt.Format("2006-01-02"))
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if !all && next != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(use --cursor %s for more, or --all to fetch everything)\n", next)
			}
		}

		if !all || next == "" {
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

func runOrgMemberSetPermissions(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	orgSlug := strings.TrimSpace(args[0])
	member := strings.TrimSpace(args[1])
	rawPerms, _ := cmd.Flags().GetStringSlice("permission")
	perms := normalizePermissionSlice(rawPerms)
	if perms == nil {
		perms = []string{}
	}

	userID, err := resolveMemberToUUID(cmd, c, orgSlug, member)
	if err != nil {
		return err
	}

	path := "/orgs/" + url.PathEscape(orgSlug) + "/members/" + url.PathEscape(userID)
	if err := c.Patch(cmd.Context(), path, map[string]any{"permissions": perms}, nil); err != nil {
		return orgMemberFriendlyError(err)
	}

	if len(perms) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "All permissions cleared for member %s in org %s.\n", member, orgSlug)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Permissions updated for member %s in org %s: %s\n",
			member, orgSlug, strings.Join(perms, ", "))
	}
	return nil
}

func runOrgMemberRemove(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	orgSlug := strings.TrimSpace(args[0])
	member := strings.TrimSpace(args[1])

	if err := confirmSlug(yesFlag(cmd), "member removal", member); err != nil {
		return err
	}

	userID, err := resolveMemberToUUID(cmd, c, orgSlug, member)
	if err != nil {
		return err
	}

	path := "/orgs/" + url.PathEscape(orgSlug) + "/members/" + url.PathEscape(userID)
	if err := c.Delete(cmd.Context(), path); err != nil {
		return orgMemberFriendlyError(err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Member %s removed from org %s.\n", member, orgSlug)
	return nil
}

func init() {
	OrgCmd.AddCommand(orgMemberCmd)
	orgMemberCmd.AddCommand(orgMemberListCmd)
	orgMemberCmd.AddCommand(orgMemberSetPermissionsCmd)
	orgMemberCmd.AddCommand(orgMemberRemoveCmd)

	addPaginationFlags(orgMemberListCmd)
	addOutputFlag(orgMemberListCmd)
	addYesFlag(orgMemberRemoveCmd)
	orgMemberSetPermissionsCmd.Flags().StringSlice("permission", nil, "Permission atom (repeat or comma-separated; omit to clear all grants)")

	orgMemberListCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	orgMemberSetPermissionsCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	orgMemberRemoveCmd.ValidArgsFunction = completeOrgNamespaceSlugs
}
