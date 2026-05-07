package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// AccountCmd is the top-level `citadel-cli account` command.
var AccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account security (passkeys and signed-in devices)",
	Long: `Commands for passkeys and device sessions registered to your account.

WebAuthn passkey **enrolment** (begin/finish) is not available in the CLI; use
the web app. List, rename, and delete already-registered passkeys here.

Revoking a **device** may require recent MFA (HTTP 412 from the server). Complete
MFA or step-up verification in a logged-in **browser** session for this account,
then retry the CLI command.`,
}

var accountPasskeyCmd = &cobra.Command{
	Use:   "passkey",
	Short: "List, rename, or remove WebAuthn passkeys",
}

var accountPasskeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List passkeys on your account",
	RunE:  runAccountPasskeyList,
}

var accountPasskeyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a passkey by id",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountPasskeyDelete,
}

var accountPasskeyRenameCmd = &cobra.Command{
	Use:   "rename <id>",
	Short: "Rename a passkey (display name)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountPasskeyRename,
}

var accountDeviceCmd = &cobra.Command{
	Use:   "device",
	Short: "List or revoke signed-in devices",
}

var accountDeviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List devices / sessions for your account",
	RunE:  runAccountDeviceList,
}

var accountDeviceRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a device session by id",
	Long: `Calls DELETE on the device record. The server may require recent MFA
step-up; if you see an MFA error, complete verification in the web app (same
account), then run this command again.`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountDeviceRevoke,
}

type passkeyRow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type deviceRow struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	UserAgent  string    `json:"user_agent,omitempty"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

func loadPasskeyList(ctx context.Context, c *apiclient.Client) ([]passkeyRow, error) {
	var raw json.RawMessage
	if err := c.Get(ctx, "/account/passkeys", &raw); err != nil {
		return nil, err
	}
	rows, ok := decodePasskeyRows([]byte(raw))
	if !ok {
		return nil, fmt.Errorf("unrecognized passkeys list JSON shape from server (expected passkeys[] or a top-level array)")
	}
	return rows, nil
}

func decodePasskeyRows(raw []byte) ([]passkeyRow, bool) {
	var wrap struct {
		Passkeys []passkeyRow `json:"passkeys"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && jsonHasTopKey(raw, "passkeys") {
		return wrap.Passkeys, true
	}
	var direct []passkeyRow
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, true
	}
	return nil, false
}

func jsonHasTopKey(raw []byte, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func loadDeviceList(ctx context.Context, c *apiclient.Client) ([]deviceRow, error) {
	var raw json.RawMessage
	if err := c.Get(ctx, "/auth/devices", &raw); err != nil {
		return nil, err
	}
	rbytes := []byte(raw)
	var wrap struct {
		Devices []deviceRow `json:"devices"`
	}
	if err := json.Unmarshal(rbytes, &wrap); err == nil && jsonHasTopKey(rbytes, "devices") {
		return wrap.Devices, nil
	}
	var direct []deviceRow
	if err := json.Unmarshal(rbytes, &direct); err == nil {
		return direct, nil
	}
	return nil, fmt.Errorf("unrecognized devices list JSON shape from server (expected devices[] or a top-level array)")
}

func runAccountPasskeyList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	rows, err := loadPasskeyList(cmd.Context(), c)
	if err != nil {
		return err
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		if len(rows) == 0 {
			return emitCSVHeaderOnly[passkeyRow](cmd)
		}
		var hdr bool
		return emitCSVRows(cmd, &hdr, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No passkeys registered.")
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tCREATED")
		for _, r := range rows {
			short := r.ID
			if len(short) > 8 {
				short = short[:8]
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", short, r.Name, r.CreatedAt.Format(time.RFC3339))
		}
		return w.Flush()
	}
}

func runAccountPasskeyDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE /account/passkeys/%s (skipped; --dry-run)\n", id)
		return nil
	}
	path := "/account/passkeys/" + url.PathEscape(id)
	if err := c.Delete(cmd.Context(), path); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && (he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusForbidden) {
			return fmt.Errorf("passkey %s not found for this account", id)
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted passkey %s.\n", id)
	return nil
}

func runAccountPasskeyRename(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	id := strings.TrimSpace(args[0])
	path := "/account/passkeys/" + url.PathEscape(id)
	body := map[string]string{"name": name}

	out := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	switch out {
	case "", "table", "json", "yaml":
	default:
		return fmt.Errorf("--output for rename supports json, yaml, or default table; got %q", out)
	}

	if err := c.Patch(cmd.Context(), path, body, nil); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && (he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusForbidden) {
			return fmt.Errorf("passkey %s not found for this account", id)
		}
		return err
	}
	switch out {
	case "", "table":
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Renamed passkey %s → %q.\n", id, name)
		return nil
	case "json":
		return emitJSON(cmd, map[string]string{"id": id, "name": name})
	case "yaml":
		return emitYAML(cmd, map[string]string{"id": id, "name": name})
	}
	return fmt.Errorf("internal: bad output mode")
}

func runAccountDeviceList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	rows, err := loadDeviceList(cmd.Context(), c)
	if err != nil {
		return err
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		if len(rows) == 0 {
			return emitCSVHeaderOnly[deviceRow](cmd)
		}
		var hdr bool
		return emitCSVRows(cmd, &hdr, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No devices registered.")
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tLAST SEEN\tUSER AGENT")
		for _, r := range rows {
			short := r.ID
			if len(short) > 8 {
				short = short[:8]
			}
			last := ""
			if !r.LastSeenAt.IsZero() {
				last = r.LastSeenAt.Format(time.RFC3339)
			} else if !r.CreatedAt.IsZero() {
				last = r.CreatedAt.Format(time.RFC3339)
			}
			ua := r.UserAgent
			if len(ua) > 48 {
				ua = ua[:45] + "..."
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", short, r.Name, last, ua)
		}
		return w.Flush()
	}
}

func runAccountDeviceRevoke(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE /auth/devices/%s (skipped; --dry-run)\n", id)
		return nil
	}
	path := "/auth/devices/" + url.PathEscape(id)
	if err := c.Delete(cmd.Context(), path); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && (he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusForbidden) {
			return fmt.Errorf("device %s not found for this account", id)
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Revoked device %s.\n", id)
	return nil
}

func init() {
	AccountCmd.AddCommand(accountPasskeyCmd)
	AccountCmd.AddCommand(accountDeviceCmd)

	accountPasskeyCmd.AddCommand(accountPasskeyListCmd)
	accountPasskeyCmd.AddCommand(accountPasskeyDeleteCmd)
	accountPasskeyCmd.AddCommand(accountPasskeyRenameCmd)

	accountDeviceCmd.AddCommand(accountDeviceListCmd)
	accountDeviceCmd.AddCommand(accountDeviceRevokeCmd)

	addOutputFlag(accountPasskeyListCmd, accountDeviceListCmd)
	addOutputFlag(accountPasskeyRenameCmd)
	addDryRunFlag(accountPasskeyDeleteCmd, accountDeviceRevokeCmd)
	accountPasskeyRenameCmd.Flags().String("name", "", "New display name (required)")
	_ = accountPasskeyRenameCmd.MarkFlagRequired("name")
}
