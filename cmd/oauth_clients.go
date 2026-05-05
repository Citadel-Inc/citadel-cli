package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

// OauthCmd is the top-level `citadel-cli oauth` command.
var OauthCmd = &cobra.Command{
	Use:   "oauth",
	Short: "OAuth client registry (JWT + oauth:manage)",
}

var oauthClientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "List, create, inspect, rotate, and revoke OAuth clients",
	Long: `Wraps /api/oauth/clients. Requires a logged-in session with oauth:manage
on the target namespace (or personal scope when --org is omitted).

Resource IDs in show / rotate-secret / revoke are the server UUID (id field),
not the public client_id string.`,
}

var oauthClientsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OAuth clients for your account or an org namespace",
	RunE:  runOAuthClientsList,
}

var oauthClientsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Register a new OAuth client",
	Long: `Requires --name and at least one --redirect-uri. Confidential clients are
default; pass --public for a PKCE-only public client.

The client_secret is printed once (and included in --output json).`,
	RunE: runOAuthClientsCreate,
}

var oauthClientsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show one OAuth client by resource UUID",
	Args:  cobra.ExactArgs(1),
	RunE:  runOAuthClientsShow,
}

var oauthClientsRotateSecretCmd = &cobra.Command{
	Use:   "rotate-secret <id>",
	Short: "Rotate a confidential client's secret (step-up / recent MFA required)",
	Long: `POSTs to /api/oauth/clients/{id}/rotate-secret. Requires a recent aal2 JWT
(within ~5 minutes) or a recent MFA marker from the web app.

Human output prints only the new secret on stdout (one line). Use --output json
for the full client payload including client_secret.`,
	Args: cobra.ExactArgs(1),
	RunE: runOAuthClientsRotateSecret,
}

var oauthClientsRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Soft-revoke an OAuth client by resource UUID",
	Long: `Deletes the client registration (revoked_at). Idempotent: already-revoked
clients yield success.

Requires typing the client UUID unless --yes.`,
	Args: cobra.ExactArgs(1),
	RunE: runOAuthClientsRevoke,
}

type oauthClient struct {
	ID                   string     `json:"id"`
	ClientID             string     `json:"client_id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description,omitempty"`
	LogoURL              string     `json:"logo_url,omitempty"`
	HomepageURL          string     `json:"homepage_url,omitempty"`
	RedirectURIs         []string   `json:"redirect_uris"`
	AllowedScopes        []string   `json:"allowed_scopes"`
	IsPublic             bool       `json:"is_public"`
	OwnerUserID          *string    `json:"owner_user_id,omitempty"`
	OwnerNamespaceID     *string    `json:"owner_namespace_id,omitempty"`
	OwnerSlug            string     `json:"owner_slug,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	Dcr                  bool       `json:"dcr,omitempty"`
	DcrSponsoredByUserID *string    `json:"dcr_sponsored_by_user_id,omitempty"`
}

type oauthClientWithSecret struct {
	oauthClient
	ClientSecret string `json:"client_secret,omitempty"`
}

// requireUUID validates the cobra arg is a UUID and returns the trimmed form.
func requireUUID(arg string) (string, error) {
	id := strings.TrimSpace(arg)
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("id must be a UUID: %w", err)
	}
	return id, nil
}

func runOAuthClientsList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	orgSlug, _ := cmd.Flags().GetString("org")
	output := outputFlag(cmd)
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all && output == "json" {
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if orgSlug != "" {
			q.Set("namespace", orgSlug)
		}
		var payload struct {
			Clients    []oauthClient `json:"clients"`
			NextCursor string        `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), "/oauth/clients?"+q.Encode(), &payload); err != nil {
			return err
		}
		clients := payload.Clients
		next := strings.TrimSpace(payload.NextCursor)

		if len(clients) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(clients) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON([]oauthClient{})
			case "ndjson":
				return nil
			default:
				fmt.Println("No OAuth clients.")
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(clients)
		case "ndjson":
			if err := emitNDJSONLines(clients); err != nil {
				return err
			}
		default:
			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "CLIENT ID\tNAME\tSCOPES")
			for _, oc := range clients {
				scopes := strings.Join(oc.AllowedScopes, ",")
				if scopes == "" {
					scopes = "—"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", oc.ClientID, oc.Name, scopes)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !all {
			if output == "" && next != "" {
				fmt.Println("(use --cursor " + next + " for more, or --all to fetch everything)")
			}
			return nil
		}
		if next == "" {
			return nil
		}
		cursor = next
	}
}

func runOAuthClientsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nameRaw, _ := cmd.Flags().GetString("name")
	name := strings.TrimSpace(nameRaw)
	redirects, _ := cmd.Flags().GetStringSlice("redirect-uri")
	orgSlug, _ := cmd.Flags().GetString("org")
	isPublic, _ := cmd.Flags().GetBool("public")
	desc, _ := cmd.Flags().GetString("description")
	scopes, _ := cmd.Flags().GetStringSlice("scope")
	output := outputFlag(cmd)

	if len(redirects) == 0 {
		return errors.New("at least one --redirect-uri is required")
	}

	payload := map[string]any{
		"name":          name,
		"redirect_uris": redirects,
		"is_public":     isPublic,
	}
	if desc != "" {
		payload["description"] = desc
	}
	if orgSlug != "" {
		payload["owner_namespace_slug"] = orgSlug
	}
	if len(scopes) > 0 {
		payload["allowed_scopes"] = scopes
	}

	var created oauthClientWithSecret
	if err := c.Post(cmd.Context(), "/oauth/clients", payload, &created); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(created)
	}

	fmt.Fprintf(os.Stderr, "Created OAuth client %q\n", created.Name)
	fmt.Fprintf(os.Stderr, "  id:         %s\n", created.ID)
	fmt.Fprintf(os.Stderr, "  client_id:  %s\n", created.ClientID)
	if created.ClientSecret != "" {
		fmt.Println(created.ClientSecret)
		fmt.Fprintf(os.Stderr, "(client_secret printed once above — store it securely)\n")
	}
	return nil
}

func runOAuthClientsShow(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id, err := requireUUID(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)

	var row oauthClient
	if err := c.Get(cmd.Context(), "/oauth/clients/"+url.PathEscape(id), &row); err != nil {
		return err
	}

	return emitOne(output, row, func(w *tabwriter.Writer, row oauthClient) {
		_, _ = fmt.Fprintf(w, "id:\t%s\n", row.ID)
		_, _ = fmt.Fprintf(w, "client_id:\t%s\n", row.ClientID)
		_, _ = fmt.Fprintf(w, "name:\t%s\n", row.Name)
		_, _ = fmt.Fprintf(w, "is_public:\t%v\n", row.IsPublic)
		_, _ = fmt.Fprintf(w, "redirect_uris:\t%s\n", strings.Join(row.RedirectURIs, ", "))
		_, _ = fmt.Fprintf(w, "allowed_scopes:\t%s\n", strings.Join(row.AllowedScopes, ", "))
		if row.RevokedAt != nil {
			_, _ = fmt.Fprintf(w, "revoked_at:\t%s\n", row.RevokedAt.Format(time.RFC3339))
		}
		_, _ = fmt.Fprintf(w, "created_at:\t%s\n", row.CreatedAt.Format(time.RFC3339))
		_, _ = fmt.Fprintf(w, "updated_at:\t%s\n", row.UpdatedAt.Format(time.RFC3339))
	})
}

func runOAuthClientsRotateSecret(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id, err := requireUUID(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	copyClip, _ := cmd.Flags().GetBool("copy-to-clipboard")

	var out oauthClientWithSecret
	err = c.Post(cmd.Context(), "/oauth/clients/"+url.PathEscape(id)+"/rotate-secret", nil, &out)
	if err != nil {
		// 428 Precondition Required carries an `mfa_required` payload; surface
		// the canonical "obtain a fresh aal2 JWT" hint.
		if apiclient.IsStatus(err, http.StatusPreconditionRequired) {
			return fmt.Errorf("recent MFA required: obtain an aal2 JWT within ~5 minutes (re-login with MFA) or complete recent-verify in the web app, then retry")
		}
		return err
	}

	if output == "json" {
		return emitJSON(out)
	}
	if out.ClientSecret == "" {
		return fmt.Errorf("server returned no client_secret (public clients have no secret)")
	}
	fmt.Println(out.ClientSecret)
	if copyClip {
		if err := copySecretToClipboard(out.ClientSecret); err != nil {
			fmt.Fprintf(os.Stderr, "clipboard: %v\n", err)
		}
	}
	return nil
}

func runOAuthClientsRevoke(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id, err := requireUUID(args[0])
	if err != nil {
		return err
	}

	if dryRunFlag(cmd) {
		fmt.Printf("Would DELETE /oauth/clients/%s (skipped; --dry-run)\n", id)
		return nil
	}
	if err := confirmTypedValue(yesFlag(cmd), "revoke OAuth client", id); err != nil {
		return err
	}
	output := outputFlag(cmd)

	if err := c.Delete(cmd.Context(), "/oauth/clients/"+url.PathEscape(id)); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(map[string]string{"status": "revoked", "id": id})
	}
	fmt.Fprintf(os.Stderr, "OAuth client %s revoked.\n", id)
	return nil
}

func copySecretToClipboard(s string) error {
	c, err := clipboardCommand()
	if err != nil {
		return err
	}
	c.Stdin = strings.NewReader(s)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// clipboardCommand picks the right OS clipboard tool, or returns an error
// describing what to install.
func clipboardCommand() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbcopy"), nil
	case "windows":
		return exec.Command("cmd", "/c", "clip"), nil
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		return exec.Command("wl-copy"), nil
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return exec.Command("xclip", "-selection", "clipboard"), nil
	}
	return nil, fmt.Errorf("install wl-copy or xclip, or copy manually")
}

func completeOAuthClientIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeyOAuthClients, completion.FetchOAuthClientIDs)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	OauthCmd.AddCommand(oauthClientsCmd)
	oauthClientsCmd.AddCommand(oauthClientsListCmd)
	oauthClientsCmd.AddCommand(oauthClientsCreateCmd)
	oauthClientsCmd.AddCommand(oauthClientsShowCmd)
	oauthClientsCmd.AddCommand(oauthClientsRotateSecretCmd)
	oauthClientsCmd.AddCommand(oauthClientsRevokeCmd)

	addOutputFlag(oauthClientsListCmd, oauthClientsCreateCmd, oauthClientsShowCmd,
		oauthClientsRotateSecretCmd, oauthClientsRevokeCmd)
	addPaginationFlags(oauthClientsListCmd)
	addYesFlag(oauthClientsRevokeCmd)
	addDryRunFlag(oauthClientsRevokeCmd)

	oauthClientsListCmd.Flags().String("org", "", "Org namespace slug (omit for personal-scope clients)")

	oauthClientsCreateCmd.Flags().String("name", "", "Display name (required)")
	oauthClientsCreateCmd.Flags().StringSlice("redirect-uri", nil, "Redirect URI (repeat flag for multiple)")
	_ = oauthClientsCreateCmd.MarkFlagRequired("name")
	oauthClientsCreateCmd.Flags().String("org", "", "Register under this org namespace slug")
	oauthClientsCreateCmd.Flags().Bool("public", false, "Register a public (PKCE-only) client")
	oauthClientsCreateCmd.Flags().String("description", "", "Optional description")
	oauthClientsCreateCmd.Flags().StringSlice("scope", nil, "Allowed OAuth scope (repeatable; default server set if omitted)")

	oauthClientsRotateSecretCmd.Flags().Bool("copy-to-clipboard", false, "Copy rotated secret to clipboard (after printing)")

	oauthClientsShowCmd.ValidArgsFunction = completeOAuthClientIDs
	oauthClientsRevokeCmd.ValidArgsFunction = completeOAuthClientIDs

	oauthClientsCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOAuthClients)
	}
	oauthClientsRevokeCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOAuthClients)
	}
	oauthClientsRotateSecretCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeyOAuthClients)
	}
}
