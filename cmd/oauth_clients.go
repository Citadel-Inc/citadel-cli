package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
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
	output, _ := cmd.Flags().GetString("output")

	path := "/oauth/clients"
	if orgSlug != "" {
		path += "?namespace=" + url.QueryEscape(orgSlug)
	}

	var clients []oauthClient
	if err := c.Get(cmd.Context(), path, &clients); err != nil {
		return err
	}

	return emitList(output, clients, "No OAuth clients.", func(w *tabwriter.Writer, clients []oauthClient) {
		_, _ = fmt.Fprintln(w, "CLIENT ID\tNAME\tSCOPES\tLAST USED")
		for _, oc := range clients {
			scopes := strings.Join(oc.AllowedScopes, ",")
			if scopes == "" {
				scopes = "—"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", oc.ClientID, oc.Name, scopes, "—")
		}
	})
}

func runOAuthClientsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(must(cmd.Flags().GetString("name")))
	redirects, _ := cmd.Flags().GetStringSlice("redirect-uri")
	orgSlug, _ := cmd.Flags().GetString("org")
	isPublic, _ := cmd.Flags().GetBool("public")
	desc, _ := cmd.Flags().GetString("description")
	scopes, _ := cmd.Flags().GetStringSlice("scope")
	output, _ := cmd.Flags().GetString("output")

	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if len(redirects) == 0 {
		return fmt.Errorf("at least one --redirect-uri is required")
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
	output, _ := cmd.Flags().GetString("output")

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
	output, _ := cmd.Flags().GetString("output")
	copyClip, _ := cmd.Flags().GetBool("copy-to-clipboard")

	var out oauthClientWithSecret
	err = c.Post(cmd.Context(), "/oauth/clients/"+url.PathEscape(id)+"/rotate-secret", nil, &out)
	if err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && he.StatusCode == 428 && strings.Contains(he.Body, "mfa_required") {
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

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmTypedValue(yes, "revoke OAuth client", id); err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")

	if err := c.Delete(cmd.Context(), "/oauth/clients/"+url.PathEscape(id)); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(map[string]string{"status": "revoked", "id": id})
	}
	fmt.Fprintf(os.Stderr, "OAuth client %s revoked.\n", id)
	return nil
}

// must is a helper for cobra Get* lookups whose only failure mode is
// "flag not registered" — programmer error caught at init().
func must[T any](v T, _ error) T { return v }

func copySecretToClipboard(s string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("pbcopy")
	case "windows":
		c = exec.Command("cmd", "/c", "clip")
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil && path != "" {
			c = exec.Command("wl-copy")
		} else if path, err := exec.LookPath("xclip"); err == nil && path != "" {
			c = exec.Command("xclip", "-selection", "clipboard")
		} else {
			return fmt.Errorf("install wl-copy or xclip, or copy manually")
		}
	}
	c.Stdin = strings.NewReader(s)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func init() {
	OauthCmd.AddCommand(oauthClientsCmd)
	oauthClientsCmd.AddCommand(oauthClientsListCmd)
	oauthClientsCmd.AddCommand(oauthClientsCreateCmd)
	oauthClientsCmd.AddCommand(oauthClientsShowCmd)
	oauthClientsCmd.AddCommand(oauthClientsRotateSecretCmd)
	oauthClientsCmd.AddCommand(oauthClientsRevokeCmd)

	oauthClientsListCmd.Flags().String("org", "", "Org namespace slug (omit for personal-scope clients)")
	oauthClientsListCmd.Flags().String("output", "", "Output format: json")

	oauthClientsCreateCmd.Flags().String("name", "", "Display name (required)")
	oauthClientsCreateCmd.Flags().StringSlice("redirect-uri", nil, "Redirect URI (repeat flag for multiple)")
	oauthClientsCreateCmd.Flags().String("org", "", "Register under this org namespace slug")
	oauthClientsCreateCmd.Flags().Bool("public", false, "Register a public (PKCE-only) client")
	oauthClientsCreateCmd.Flags().String("description", "", "Optional description")
	oauthClientsCreateCmd.Flags().StringSlice("scope", nil, "Allowed OAuth scope (repeatable; default server set if omitted)")
	oauthClientsCreateCmd.Flags().String("output", "", "Output format: json")

	oauthClientsShowCmd.Flags().String("output", "", "Output format: json")

	oauthClientsRotateSecretCmd.Flags().String("output", "", "Output format: json")
	oauthClientsRotateSecretCmd.Flags().Bool("copy-to-clipboard", false, "Copy rotated secret to clipboard (after printing)")

	oauthClientsRevokeCmd.Flags().Bool("yes", false, "Skip typed-UUID confirmation")
	oauthClientsRevokeCmd.Flags().String("output", "", "Output format: json")
}
