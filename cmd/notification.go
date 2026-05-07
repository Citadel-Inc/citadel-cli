package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// NotificationCmd is the top-level `citadel-cli notification` command.
var NotificationCmd = &cobra.Command{
	Use:     "notification",
	Aliases: []string{"notifications", "notif"},
	Short:   "Manage your notification inbox and preferences",
	Long: `Browse and manage your Citadel notification inbox.

List unread notifications, mark them read, and configure email digest
preferences.`,
}

var notificationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notifications in your inbox",
	RunE:  runNotificationList,
}

var notificationReadCmd = &cobra.Command{
	Use:   "read <id>",
	Short: "Mark a notification as read",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotificationRead,
}

var notificationReadAllCmd = &cobra.Command{
	Use:   "read-all",
	Short: "Mark all notifications as read",
	RunE:  runNotificationReadAll,
}

var notificationUnreadCountCmd = &cobra.Command{
	Use:   "unread-count",
	Short: "Print the number of unread notifications",
	RunE:  runNotificationUnreadCount,
}

var notificationPrefsCmd = &cobra.Command{
	Use:   "prefs",
	Short: "Get or update notification preferences",
}

var notificationPrefsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show your notification preferences",
	RunE:  runNotificationPrefsGet,
}

var notificationPrefsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update notification preferences",
	Long: `Update notification preferences for your account.

At least one flag must be supplied. Changes are applied incrementally:
unspecified fields are left unchanged.`,
	RunE: runNotificationPrefsSet,
}

// ── domain types ──────────────────────────────────────────────────────────────

// notifItem mirrors the daemon's notification list-item JSON shape.
type notifItem struct {
	ID            string     `json:"id"`
	Kind          string     `json:"kind"`
	Summary       string     `json:"summary"`
	Href          string     `json:"href,omitempty"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	NamespaceSlug string     `json:"namespace_slug,omitempty"`
}

type notifListPage struct {
	Items      []notifItem `json:"items"`
	NextCursor string      `json:"next_cursor"`
}

type notifUnreadCountResp struct {
	Count int `json:"count"`
}

type notifKindPref struct {
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

type notifPrefsResp struct {
	EmailDigestCadence string          `json:"email_digest_cadence"`
	Kinds              []notifKindPref `json:"kinds"`
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	addPaginationFlags(notificationListCmd)
	addOutputFlag(notificationListCmd)
	notificationListCmd.Flags().Bool("unread", false, "Show only unread notifications")

	addOutputFlag(notificationUnreadCountCmd)

	addOutputFlag(notificationPrefsGetCmd)

	addOutputFlag(notificationPrefsSetCmd)
	notificationPrefsSetCmd.Flags().String("email-digest", "", "Email digest cadence: never, daily, or weekly")
	notificationPrefsSetCmd.Flags().StringSlice("enable", nil, "Enable a notification kind (repeatable)")
	notificationPrefsSetCmd.Flags().StringSlice("disable", nil, "Disable a notification kind (repeatable)")

	notificationPrefsCmd.AddCommand(notificationPrefsGetCmd, notificationPrefsSetCmd)
	NotificationCmd.AddCommand(
		notificationListCmd,
		notificationReadCmd,
		notificationReadAllCmd,
		notificationUnreadCountCmd,
		notificationPrefsCmd,
	)
}

// ── list ──────────────────────────────────────────────────────────────────────

func runNotificationList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateListOutput(output); err != nil {
		return err
	}
	unreadOnly, _ := cmd.Flags().GetBool("unread")

	if all && output == "json" {
		return fmt.Errorf("--all with --output json is not supported; use --output ndjson for streaming JSON")
	}

	var yamlAccum []notifItem
	csvHdr := false
	first := true

	for {
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if unreadOnly {
			q.Set("unread", "1")
		}

		var page notifListPage
		if err := c.Get(cmd.Context(), "/api/me/notifications?"+q.Encode(), &page); err != nil {
			return err
		}
		rows := page.Items
		next := strings.TrimSpace(page.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 {
			switch output {
			case "json":
				return emitJSON(cmd, []notifItem{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[notifItem](cmd)
			case "yaml":
				return emitYAML(cmd, []notifItem{})
			default:
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No notifications.")
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
			_, _ = fmt.Fprintln(w, "ID\tKIND\tSTATUS\tNAMESPACE\tSUMMARY")
			for _, n := range rows {
				status := "unread"
				if n.ReadAt != nil {
					status = "read"
				}
				ns := n.NamespaceSlug
				if ns == "" {
					ns = "-"
				}
				summary := n.Summary
				if len(summary) > 72 {
					summary = summary[:69] + "..."
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.ID, n.Kind, status, ns, summary)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !all {
			if isHumanListOutput(output) && next != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(use --cursor "+next+" for more, or --all to fetch everything)")
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
			yamlAccum = []notifItem{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

// ── read ──────────────────────────────────────────────────────────────────────

func runNotificationRead(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	var resp struct {
		OK bool `json:"ok"`
	}
	if err := c.Post(cmd.Context(), "/api/me/notifications/"+url.PathEscape(id)+"/read", nil, &resp); err != nil {
		if apiclient.IsStatus(err, 404) {
			return fmt.Errorf("notification %q not found", id)
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notification %s marked as read.\n", id)
	return nil
}

// ── read-all ──────────────────────────────────────────────────────────────────

func runNotificationReadAll(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	if err := c.Post(cmd.Context(), "/api/me/notifications/read-all", nil, &resp); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Marked all notifications as read.")
	return nil
}

// ── unread-count ──────────────────────────────────────────────────────────────

func runNotificationUnreadCount(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var result notifUnreadCountResp
	if err := c.Get(cmd.Context(), "/api/me/notifications/unread-count", &result); err != nil {
		return err
	}
	output := outputFlag(cmd)
	if output == "json" {
		return emitJSON(cmd, result)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d\n", result.Count)
	return nil
}

// ── prefs get ─────────────────────────────────────────────────────────────────

func runNotificationPrefsGet(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var prefs notifPrefsResp
	if err := c.Get(cmd.Context(), "/api/me/notification-prefs", &prefs); err != nil {
		return err
	}
	output := outputFlag(cmd)
	return emitOne(cmd, output, prefs, func(w *tabwriter.Writer, p notifPrefsResp) {
		_, _ = fmt.Fprintf(w, "Email digest:\t%s\n", p.EmailDigestCadence)
		if len(p.Kinds) > 0 {
			_, _ = fmt.Fprintln(w, "\nKIND\tLABEL\tENABLED")
			for _, k := range p.Kinds {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", k.Kind, k.Label, k.Enabled)
			}
		}
	})
}

// ── prefs set ─────────────────────────────────────────────────────────────────

func runNotificationPrefsSet(cmd *cobra.Command, _ []string) error {
	cadence, _ := cmd.Flags().GetString("email-digest")
	enableKinds, _ := cmd.Flags().GetStringSlice("enable")
	disableKinds, _ := cmd.Flags().GetStringSlice("disable")

	if cadence == "" && len(enableKinds) == 0 && len(disableKinds) == 0 {
		return fmt.Errorf("at least one of --email-digest, --enable, or --disable is required")
	}
	if cadence != "" {
		switch cadence {
		case "never", "daily", "weekly":
		default:
			return fmt.Errorf("--email-digest must be never, daily, or weekly; got %q", cadence)
		}
	}

	body := struct {
		EmailDigestCadence string          `json:"email_digest_cadence,omitempty"`
		KindOverrides      map[string]bool `json:"kind_overrides,omitempty"`
	}{}
	if cadence != "" {
		body.EmailDigestCadence = cadence
	}
	overrides := make(map[string]bool)
	for _, k := range enableKinds {
		overrides[k] = true
	}
	for _, k := range disableKinds {
		overrides[k] = false
	}
	if len(overrides) > 0 {
		body.KindOverrides = overrides
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var updated notifPrefsResp
	if err := c.Patch(cmd.Context(), "/api/me/notification-prefs", body, &updated); err != nil {
		return err
	}

	output := outputFlag(cmd)
	if output == "json" {
		return emitJSON(cmd, updated)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Notification preferences updated.")
	return nil
}
