package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── namespace profile subtree ─────────────────────────────────────────────────

// nsProfileCmd is the `citadel namespace profile` group.
var nsProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Inspect namespace profile identity",
	Long:  `Read-only access to namespace identity metadata (display name, bio, social links, etc.).`,
}

var nsProfileGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get the identity profile for a namespace",
	Long: `Fetches the public identity profile for a namespace.

For private namespaces, only the owner can read the full profile; non-owners
receive a not-found error (the server does not leak private namespace existence).

Examples:
  citadel-cli namespace profile get myorg
  citadel-cli namespace profile get myorg --output json
  citadel-cli namespace profile get myorg --output yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runNsProfileGet,
}

// ── domain types ─────────────────────────────────────────────────────────────

// nsProfile mirrors the profileapi profileResponse JSON (identity fields only;
// stats/preview/avatar blocks included for JSON/YAML pass-through).
type nsProfile struct {
	NamespaceID string `json:"namespace_id"`
	Slug        string `json:"slug"`
	Kind        string `json:"kind,omitempty"`
	Visibility  string `json:"visibility"`

	DisplayName     string `json:"display_name,omitempty"`
	LegalEntityName string `json:"legal_entity_name,omitempty"`
	Bio             string `json:"bio,omitempty"`
	Location        string `json:"location,omitempty"`
	WebsiteURL      string `json:"website_url,omitempty"`
	PublicEmail     string `json:"public_email,omitempty"`
	Pronouns        string `json:"pronouns,omitempty"`
	Company         string `json:"company,omitempty"`
	Timezone        string `json:"timezone,omitempty"`
	SponsorURL      string `json:"sponsor_url,omitempty"`

	// Social links is a map of provider→handle (e.g. "github" → "octocat").
	SocialLinks map[string]string `json:"social_links,omitempty"`

	Stats struct {
		Repos   int  `json:"repos"`
		Members *int `json:"members,omitempty"`
	} `json:"stats"`

	// Owner-only fields (omitted for non-owners)
	BillingEmail    string   `json:"billing_email,omitempty"`
	VerifiedDomains []string `json:"verified_domains,omitempty"`

	// Pass-through preview lists (useful in JSON/YAML)
	ReposPreview []struct {
		Slug       string `json:"slug"`
		Visibility string `json:"visibility"`
	} `json:"repos_preview,omitempty"`
	MembersPreview []struct {
		Slug        string `json:"slug,omitempty"`
		DisplayName string `json:"display_name,omitempty"`
		IsOwner     bool   `json:"is_owner"`
	} `json:"members_preview,omitempty"`
}

// ── handler ───────────────────────────────────────────────────────────────────

func runNsProfileGet(cmd *cobra.Command, args []string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	slug := args[0]

	var profile nsProfile
	if err := c.Get(cmd.Context(), "/api/namespaces/"+url.PathEscape(slug)+"/profile", &profile); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("namespace '%s' not found (or private)", slug)
		}
		return err
	}

	return emitOne(cmd, output, profile, renderNsProfile)
}

// renderNsProfile renders the identity fields as a human-readable tab table.
func renderNsProfile(w *tabwriter.Writer, p nsProfile) {
	row := func(label, value string) {
		if value != "" {
			_, _ = fmt.Fprintf(w, "%s:\t%s\n", label, value)
		}
	}

	row("Slug", p.Slug)
	row("Kind", p.Kind)
	row("Visibility", p.Visibility)
	row("Display name", p.DisplayName)
	row("Legal entity", p.LegalEntityName)
	row("Bio", truncate(p.Bio, 120))
	row("Location", p.Location)
	row("Website", p.WebsiteURL)
	row("Email", p.PublicEmail)
	row("Pronouns", p.Pronouns)
	row("Company", p.Company)
	row("Timezone", p.Timezone)
	row("Sponsor URL", p.SponsorURL)

	if len(p.SocialLinks) > 0 {
		_, _ = fmt.Fprintf(w, "Social:\t%s\n", formatSocialLinks(p.SocialLinks))
	}

	_, _ = fmt.Fprintf(w, "Repos:\t%d\n", p.Stats.Repos)
	if p.Stats.Members != nil {
		_, _ = fmt.Fprintf(w, "Members:\t%d\n", *p.Stats.Members)
	}

	// Owner-only
	row("Billing email", p.BillingEmail)
	if len(p.VerifiedDomains) > 0 {
		_, _ = fmt.Fprintf(w, "Verified domains:\t%s\n", strings.Join(p.VerifiedDomains, ", "))
	}
}

// formatSocialLinks renders a map[string]string as sorted "provider: handle" pairs.
func formatSocialLinks(links map[string]string) string {
	keys := make([]string, 0, len(links))
	for k := range links {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v := links[k]; v != "" {
			parts = append(parts, k+": "+v)
		}
	}
	return strings.Join(parts, "  ")
}

// truncate clips s to max bytes, appending "…" if trimmed.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// init wires the profile subtree into NamespaceCmd.
func init() {
	nsProfileCmd.AddCommand(nsProfileGetCmd)
	NamespaceCmd.AddCommand(nsProfileCmd)

	addOutputFlag(nsProfileGetCmd)
	nsProfileGetCmd.ValidArgsFunction = completeOrgNamespaceSlugs
}
