package cmd

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// KgCmd is the parent for `citadel-cli kg ...`. Talks to the JWT-gated
// /api/kg/{slug}/* endpoints via internal/apiclient.
var KgCmd = &cobra.Command{
	Use:   "kg",
	Short: "Knowledge-graph queries (impact analysis)",
	Long: `Commands for querying the knowledge-graph substrate populated by go-kg-indexer.

Authentication uses your Supabase JWT from 'citadel-cli auth login'.`,
}

var kgImpactCmd = &cobra.Command{
	Use:   "impact [<owner>[/<repo>]] <symbol>",
	Short: "Find direct + transitive callers of a symbol",
	Long: `Projects a callers-direction BFS into the rename-impact shape:
direct callers + transitive callers + affected files. Default depth = 2.

The repository may be given as <owner>, <owner>/<repo>, or omitted when -R/--repo,
` + "`CITADEL_REPO`" + `, or git origin in the current directory supplies it.
<symbol> accepts a UUID (direct call) or a name (resolved via /kg/<owner>/symbols first; if more
than one symbol matches, prints the candidates so you can disambiguate).

Pretty-printed by default; use --json for the raw HTTP response.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runKgImpact,
}

// splitOwnerRepo parses an "<owner>" or "<owner>/<repo>" slug.
func splitOwnerRepo(slug string) (owner, repo string) {
	if i := strings.Index(slug, "/"); i > 0 {
		return slug[:i], slug[i+1:]
	}
	return slug, ""
}

// upgradeUnauthorized maps any *HTTPError 401 to the friendlier login hint.
func upgradeUnauthorized(err error) error {
	if apiclient.IsStatus(err, http.StatusUnauthorized) {
		return fmt.Errorf("unauthorized: run `citadel-cli auth login` to refresh your session")
	}
	return err
}

func runKgImpact(cmd *cobra.Command, args []string) error {
	symbol := strings.TrimSpace(args[len(args)-1])
	depth, _ := cmd.Flags().GetInt("depth")
	rawJSON := jsonFlag(cmd)

	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)

	var slug string
	switch len(args) {
	case 1:
		if repoFlag != "" {
			ns, rslug, err := splitRepoArg(repoFlag)
			if err != nil {
				return err
			}
			slug = ns + "/" + rslug
		} else {
			ns, rslug, err := resolveRepoFlag(cmd)
			if err != nil {
				return err
			}
			slug = ns + "/" + rslug
		}
	case 2:
		if repoFlag != "" {
			ns, rslug, err := splitRepoArg(repoFlag)
			if err != nil {
				return err
			}
			slug = ns + "/" + rslug
		} else {
			slug = strings.TrimSpace(args[0])
		}
	default:
		return fmt.Errorf("expected 1 or 2 arguments")
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	symbolID := symbol
	if _, err := uuid.Parse(symbol); err != nil {
		// Resolve <name> → UUID via /kg/<owner>/symbols.
		resolved, err := resolveSymbolID(cmd.Context(), c, slug, symbol)
		if err != nil {
			return err
		}
		symbolID = resolved
	}

	owner, repo := splitOwnerRepo(slug)
	q := url.Values{}
	q.Set("symbol", symbolID)
	if depth > 0 {
		q.Set("depth", strconv.Itoa(depth))
	}
	if repo != "" {
		q.Set("repo", repo)
	}
	path := "/kg/" + url.PathEscape(owner) + "/impact?" + q.Encode()

	if rawJSON {
		var pretty any
		if err := c.Get(cmd.Context(), path, &pretty); err != nil {
			return upgradeUnauthorized(err)
		}
		return emitJSON(pretty)
	}

	var ir impactResp
	if err := c.Get(cmd.Context(), path, &ir); err != nil {
		return upgradeUnauthorized(err)
	}
	printImpactTree(ir)
	return nil
}

// impactResp mirrors kgapi.ImpactResponse without importing the server
// package (the CLI lives in the same module but the import would pull
// pgx + Supabase deps into the CLI binary).
type impactResp struct {
	Symbol            impactNode   `json:"symbol"`
	DirectCallers     []impactNode `json:"direct_callers"`
	TransitiveCallers []impactNode `json:"transitive_callers"`
	AffectedFiles     []string     `json:"affected_files"`
	Truncated         bool         `json:"truncated"`
}

type impactNode struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
}

// printImpactTree renders the response as a human-readable tree.
//
//	rename-impact for foo (function) at file-1.go
//	  direct callers (3):
//	    - bar  in file-2.go
//	    - baz  in file-3.go
//	  transitive callers (1):
//	    - qux  in file-4.go
//	  affected files (4):
//	    file-1.go, file-2.go, file-3.go, file-4.go
//
// `--json` skips this and prints the raw response.
func printImpactTree(ir impactResp) {
	header := "rename-impact for " + ir.Symbol.Name
	if ir.Symbol.Kind != "" {
		header += " (" + ir.Symbol.Kind + ")"
	}
	if ir.Symbol.Path != "" {
		header += " at " + ir.Symbol.Path
	}
	fmt.Println(header)

	fmt.Printf("  direct callers (%d):\n", len(ir.DirectCallers))
	for _, n := range ir.DirectCallers {
		fmt.Printf("    - %s\n", formatCaller(n))
	}
	fmt.Printf("  transitive callers (%d):\n", len(ir.TransitiveCallers))
	for _, n := range ir.TransitiveCallers {
		fmt.Printf("    - %s\n", formatCaller(n))
	}
	fmt.Printf("  affected files (%d):\n", len(ir.AffectedFiles))
	if len(ir.AffectedFiles) > 0 {
		fmt.Printf("    %s\n", strings.Join(ir.AffectedFiles, ", "))
	}
	if ir.Truncated {
		fmt.Println("  (truncated — narrow depth or scope to see the full set)")
	}
}

func formatCaller(n impactNode) string {
	label := cmp.Or(n.Name, n.ID)
	if n.Path != "" {
		return label + "  in " + n.Path
	}
	return label
}

// symbolMatch is the kgapi /symbols result row we care about.
type symbolMatch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// resolveSymbolID looks up a symbol UUID by name via /kg/<owner>/symbols.
// slug accepts "<owner>" (search across owner's repos) or "<owner>/<repo>"
// (the suffix is sent as the repo filter so duplicate names across repos
// don't mask the user's intent).
func resolveSymbolID(ctx context.Context, c *apiclient.Client, slug, name string) (string, error) {
	owner, repo := splitOwnerRepo(slug)
	q := url.Values{}
	q.Set("q", name)
	if repo != "" {
		q.Set("repo", repo)
	}
	path := "/kg/" + url.PathEscape(owner) + "/symbols?" + q.Encode()

	var sr struct {
		Matches []symbolMatch `json:"matches"`
	}
	if err := c.Get(ctx, path, &sr); err != nil {
		return "", upgradeUnauthorized(fmt.Errorf("symbols lookup: %w", err))
	}

	exact := sr.Matches[:0:0]
	for _, m := range sr.Matches {
		if m.Name == name {
			exact = append(exact, m)
		}
	}
	if len(exact) == 0 {
		if len(sr.Matches) == 0 {
			return "", fmt.Errorf("no symbol matches %q in %s — try a broader query or pass a UUID", name, slug)
		}
		return "", fmt.Errorf("no exact match for %q; closest: %s — pass a UUID to disambiguate", name, listSymbolCandidates(sr.Matches))
	}
	if len(exact) > 1 {
		return "", fmt.Errorf("symbol %q is ambiguous (%d hits): %s — pass a UUID", name, len(exact), listSymbolCandidates(exact))
	}
	return exact[0].ID, nil
}

func listSymbolCandidates(ms []symbolMatch) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		parts = append(parts, fmt.Sprintf("%s (%s @ %s id=%s)", m.Name, m.Kind, m.Path, m.ID))
	}
	return strings.Join(parts, "; ")
}

func init() {
	KgCmd.AddCommand(kgImpactCmd)
	kgImpactCmd.Flags().Int("depth", 0, "BFS depth (1-3, default 2 server-side)")
	addJSONFlag(kgImpactCmd)
	addRepoFlag(kgImpactCmd)
}
