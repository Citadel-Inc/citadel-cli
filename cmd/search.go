package cmd

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// SearchCmd is the top-level `citadel-cli search` command (dashboard Cmd-K parity).
var SearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search namespaces and repositories on Citadel",
	Long: `Runs authenticated GET /api/search against the Citadel API.

Like every citadel-cli verb, search requires a login ('citadel-cli auth login').
Requiring an identity lets Citadel enforce per-user rate limits and protect shared
service quality.

By default the scope is namespaces you can access (owned or member). Pass --public
to widen discovery to unrelated namespaces (still authenticated; server uses scope=all).

Examples:
  citadel-cli search "my team"
  citadel-cli search "acme" --public
  citadel-cli search "foo" --scope repos --limit 15 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	SearchCmd.Flags().Bool("public", false, "Include broader namespace discovery beyond those you belong to (authenticated scope=all)")
	SearchCmd.Flags().String("scope", "", "Search scope: namespaces, repos, or all (overrides default; default is namespaces, or all with --public)")
	SearchCmd.Flags().Int("limit", 0, "Maximum results per request (1–25; omit for server default)")
	addOutputFlag(SearchCmd)
	addJSONFlag(SearchCmd)
}

type searchAPIResponse struct {
	Query   string         `json:"query"`
	Scope   string         `json:"scope"`
	Results []searchResult `json:"results"`
}

type searchResult struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Kind         string `json:"kind"`
	ParentSlug   string `json:"parent_slug"`
	DisplayName  string `json:"display_name"`
	Path         string `json:"path"`
	Score        any    `json:"score"`
	AvatarURL    any    `json:"avatar_url"`
	GravatarHash string `json:"gravatar_hash"`
}

func runSearch(cmd *cobra.Command, args []string) error {
	q := strings.TrimSpace(args[0])
	if utf8.RuneCountInString(q) < 2 {
		return fmt.Errorf("query must be at least 2 characters")
	}

	outMode := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if jsonFlag(cmd) {
		outMode = "json"
	}
	if err := validateGetOutput(outMode); err != nil {
		return err
	}

	scopeExplicit, err := cmd.Flags().GetString("scope")
	if err != nil {
		return err
	}
	scopeExplicit = strings.TrimSpace(strings.ToLower(scopeExplicit))

	public, err := cmd.Flags().GetBool("public")
	if err != nil {
		return err
	}

	var scope string
	switch {
	case scopeExplicit != "":
		scope = scopeExplicit
	case public:
		scope = "all"
	default:
		scope = "namespaces"
	}
	switch scope {
	case "namespaces", "repos", "all":
	default:
		return fmt.Errorf("invalid scope %q (use namespaces, repos, or all)", scope)
	}

	limitSet := cmd.Flags().Changed("limit")
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return err
	}
	if limitSet {
		if limit < 1 || limit > 25 {
			return fmt.Errorf("--limit must be between 1 and 25")
		}
	}

	v := url.Values{}
	v.Set("q", q)
	v.Set("scope", scope)
	if limitSet {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/search?" + v.Encode()

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var resp searchAPIResponse
	if err := c.Get(cmd.Context(), path, &resp); err != nil {
		return upgradeUnauthorized(err)
	}

	switch outMode {
	case "json":
		return emitJSON(cmd, resp)
	case "yaml":
		return emitYAML(cmd, resp)
	default:
		return writeSearchTable(cmd.OutOrStdout(), resp.Results)
	}
}

func writeSearchTable(w io.Writer, rows []searchResult) error {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(w, "(no results)")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "TYPE\tKIND\tPATH\tDISPLAY NAME\tSCORE")
	for _, r := range rows {
		path := pickSearchPath(r)
		disp := r.DisplayName
		if disp == "" {
			disp = "—"
		}
		typeCol := mapSearchType(r.Type, r.Kind)
		kindCol := r.Kind
		if kindCol == "" {
			kindCol = "—"
		}
		score := "—"
		if r.Score != nil {
			score = fmt.Sprint(r.Score)
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", typeCol, kindCol, path, disp, score)
	}
	return tw.Flush()
}

func pickSearchPath(r searchResult) string {
	if r.Path != "" {
		return r.Path
	}
	if r.ParentSlug != "" && r.Slug != "" {
		return r.ParentSlug + "/" + r.Slug
	}
	if r.Slug != "" {
		return r.Slug
	}
	return "—"
}

func mapSearchType(apiType, kind string) string {
	t := strings.TrimSpace(strings.ToLower(apiType))
	switch t {
	case "":
		if k := strings.TrimSpace(strings.ToLower(kind)); k != "" {
			return k
		}
		return "—"
	case "namespace", "repo", "repository", "org", "organization", "user":
		return t
	default:
		return apiType
	}
}
