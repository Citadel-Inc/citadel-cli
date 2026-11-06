package cmd

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
)

// KgCmd is the parent for `citadel-cli kg ...`. Talks to the authenticated
// /api/kg/{slug}/* endpoints via internal/apiclient.
var KgCmd = &cobra.Command{
	Use:     "kg",
	GroupID: "repo",
	Short:   "Knowledge-graph queries (search, symbols, impact, …)",
	Long: `Commands for querying the Citadel knowledge-graph JSON API.

Use kg search for cross-namespace fulltext; namespace-scoped verbs (symbols, files,
walk, fulltext, diff, impact) resolve repository context from -R / CITADEL_REPO /
git origin unless you pass an explicit namespace/repo prefix.

Authentication uses your saved Citadel CLI session from 'citadel-cli auth login'
(normally an opaque agent token).`,
}

var kgImpactCmd = &cobra.Command{
	Use:   "impact [<owner>[/<repo>]] <symbol>",
	Short: "Find direct + transitive callers of a symbol",
	Long: `Projects a callers-direction BFS into the rename-impact shape:
direct callers + transitive callers + affected files. Default depth = 2 server-side.

Uses GET /api/namespaces/{namespace}/kg/impact (JWT required).

The repository may be given as <owner>, <owner>/<repo>, or omitted when -R/--repo,
` + "`CITADEL_REPO`" + `, or git origin in the current directory supplies it.
<symbol> accepts a UUID or a name: Name, Recv.Name, pkg.Name, pkg.Recv.Name, path/to/file.go:Name or
path/to/dir:Recv.Name. If more than one symbol matches, prints candidates to pass back.

Edges other than a resolved call are labelled: [dispatch] is a concrete method an interface call may reach,
[http] a client call to a route, [bridge] a generated desktop binding. Same-named method calls on an unknown
receiver are listed apart and not counted. Compact by default; use --json for the raw HTTP response.`,
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

// kgImpactRepoSlug resolves owner/repo from an optional leading repo argument,
// --repo, or the current repository, in that order of precedence for 2 args.
func kgImpactRepoSlug(cmd *cobra.Command, args []string, repoFlag string) (string, error) {
	if len(args) != 1 && len(args) != 2 {
		return "", fmt.Errorf("expected 1 or 2 arguments")
	}
	if repoFlag == "" && len(args) == 2 {
		return strings.TrimSpace(args[0]), nil
	}
	var ns, rslug string
	var err error
	if repoFlag != "" {
		ns, rslug, err = splitRepoArg(repoFlag)
	} else {
		ns, rslug, err = resolveRepoFlag(cmd)
	}
	if err != nil {
		return "", err
	}
	return ns + "/" + rslug, nil
}

func runKgImpact(cmd *cobra.Command, args []string) error {
	symbol := strings.TrimSpace(args[len(args)-1])
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	cmd.SilenceUsage = true
	depth, _ := cmd.Flags().GetInt("depth")
	if cmd.Flags().Changed("depth") && (depth < 1 || depth > 3) {
		return fmt.Errorf("--depth must be between 1 and 3")
	}
	rawJSON := jsonFlag(cmd)

	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)

	slug, err := kgImpactRepoSlug(cmd, args, repoFlag)
	if err != nil {
		return err
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
	if noTests, _ := cmd.Flags().GetBool("no-tests"); noTests {
		q.Set("tests", "exclude")
	}
	reqPath := "/api/namespaces/" + url.PathEscape(owner) + "/kg/impact?" + q.Encode()

	if rawJSON {
		var pretty any
		if err := c.Get(cmd.Context(), reqPath, &pretty); err != nil {
			return upgradeUnauthorized(err)
		}
		return emitJSON(cmd, pretty)
	}

	var ir impactResp
	if err := c.Get(cmd.Context(), reqPath, &ir); err != nil {
		return upgradeUnauthorized(err)
	}
	printImpactTree(cmd, ir)
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
	Unresolved        []impactNode `json:"unresolved"`
	UnresolvedTotal   int          `json:"unresolved_total"`
}

type impactNode struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name,omitempty"`
	Qualifier string `json:"qualifier,omitempty"`
	Path      string `json:"path,omitempty"`
	Via       string `json:"via,omitempty"`
}

// printImpactTree renders the response grouped by file, which spends far fewer
// tokens than one line per caller when an agent reads it.
//
//	impact Store.Load (function) internal/store.go: 3 direct, 1 transitive, 3 files
//	direct:
//	  internal/a.go: Handler.Show, run
//	  internal/mem.go: Mem.Reload [dispatch]
//	transitive:
//	  web/api.ts: fetchIt [http]
//	unresolved, same method name on an unknown receiver, not counted (2):
//	  internal/b.go: Worker.Tick
//
// `--json` skips this and prints the raw response.
func printImpactTree(cmd *cobra.Command, ir impactResp) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "impact %s (%s) %s: %d direct, %d transitive, %d files\n",
		qualifiedName(ir.Symbol.Qualifier, ir.Symbol.Name), ir.Symbol.Kind, ir.Symbol.Path,
		len(ir.DirectCallers), len(ir.TransitiveCallers), len(ir.AffectedFiles))
	printImpactGroup(out, "direct", ir.DirectCallers)
	printImpactGroup(out, "transitive", ir.TransitiveCallers)
	if len(ir.Unresolved) > 0 {
		total := max(ir.UnresolvedTotal, len(ir.Unresolved))
		nodes := make([]impactNode, 0, len(ir.Unresolved))
		for _, u := range ir.Unresolved {
			nodes = append(nodes, impactNode{Name: u.Name, Qualifier: u.Qualifier, Path: u.Path})
		}
		printImpactGroup(out, fmt.Sprintf("unresolved, same method name on an unknown receiver, not counted (%d)", total), nodes)
	}
	if ir.Truncated {
		_, _ = fmt.Fprintln(out, "TRUNCATED: more callers exist than were walked; lower --depth or use --no-tests")
	}
}

func printImpactGroup(out io.Writer, title string, nodes []impactNode) {
	if len(nodes) == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "%s:\n", title)
	byFile := map[string][]string{}
	files := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if _, seen := byFile[n.Path]; !seen {
			files = append(files, n.Path)
		}
		label := qualifiedName(n.Qualifier, cmp.Or(n.Name, n.ID))
		if n.Via != "" && n.Via != "call" {
			label += " [" + n.Via + "]"
		}
		byFile[n.Path] = append(byFile[n.Path], label)
	}
	sort.Strings(files)
	for _, f := range files {
		_, _ = fmt.Fprintf(out, "  %s: %s\n", f, strings.Join(byFile[f], ", "))
	}
}

func qualifiedName(qualifier, name string) string {
	if qualifier == "" {
		return name
	}
	return qualifier + "." + name
}

// symbolMatch is the kgapi /symbols result row we care about.
type symbolMatch struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Qualifier string `json:"qualifier"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
}

// symbolQuery is what a user wrote for a symbol: Name, Recv.Name, pkg.Name,
// pkg.Recv.Name or path/file.go:Name, with Recv.Name allowed after the colon.
type symbolQuery struct {
	path, pkg, qualifier, name string
	ambiguousDot               bool // the part before the dot may be a package or a receiver
}

func parseSymbolQuery(s string) symbolQuery {
	var q symbolQuery
	if i := strings.LastIndexByte(s, ':'); i > 0 {
		q.path, s = s[:i], s[i+1:]
	}
	parts := strings.Split(s, ".")
	q.name = parts[len(parts)-1]
	switch len(parts) {
	case 1:
	case 2:
		q.qualifier, q.ambiguousDot = parts[0], q.path == ""
	default:
		q.pkg, q.qualifier = parts[len(parts)-3], parts[len(parts)-2]
	}
	return q
}

func (q symbolQuery) matches(m symbolMatch) bool {
	if m.Name != q.name {
		return false
	}
	if q.path != "" && m.Path != q.path && !strings.HasPrefix(m.Path, strings.TrimSuffix(q.path, "/")+"/") {
		return false
	}
	if q.pkg != "" && path.Base(path.Dir(m.Path)) != q.pkg {
		return false
	}
	if q.qualifier == "" {
		return true
	}
	if m.Qualifier == q.qualifier {
		return true
	}
	return q.ambiguousDot && path.Base(path.Dir(m.Path)) == q.qualifier
}

// resolveSymbolID looks a symbol up by any form parseSymbolQuery accepts. slug
// accepts "<owner>" (search across owner's repos) or "<owner>/<repo>". When the
// name does not pin down one symbol the error lists candidates in a form that
// can be passed back as is.
func resolveSymbolID(ctx context.Context, c *apiclient.Client, slug, name string) (string, error) {
	owner, repo := splitOwnerRepo(slug)
	want := parseSymbolQuery(name)
	q := url.Values{}
	q.Set("q", want.name)
	q.Set("limit", "200")
	if repo != "" {
		q.Set("repo", repo)
	}
	if want.path != "" {
		q.Set("path_prefix", want.path)
	}
	reqPath := "/api/namespaces/" + url.PathEscape(owner) + "/kg/symbols?" + q.Encode()

	var sr struct {
		Matches []symbolMatch `json:"matches"`
	}
	if err := c.Get(ctx, reqPath, &sr); err != nil {
		return "", upgradeUnauthorized(fmt.Errorf("symbols lookup: %w", err))
	}

	exact := sr.Matches[:0:0]
	for _, m := range sr.Matches {
		if want.matches(m) {
			exact = append(exact, m)
		}
	}
	if len(exact) == 1 {
		return exact[0].ID, nil
	}
	if fns := onlyFunctions(exact); len(fns) == 1 {
		return fns[0].ID, nil
	}
	if len(exact) > 1 {
		return "", fmt.Errorf("%q matches %d symbols; rerun with one of:\n%s", name, len(exact), listSymbolCandidates(exact))
	}
	if len(sr.Matches) == 0 {
		return "", fmt.Errorf("no symbol matches %q in %s; it may not be indexed, so search the source to confirm", want.name, slug)
	}
	return "", fmt.Errorf("no symbol is exactly %q; closest:\n%s", name, listSymbolCandidates(sr.Matches))
}

func onlyFunctions(ms []symbolMatch) []symbolMatch {
	out := ms[:0:0]
	for _, m := range ms {
		if m.Kind == "function" {
			out = append(out, m)
		}
	}
	return out
}

const maxSymbolCandidates = 12

func listSymbolCandidates(ms []symbolMatch) string {
	var b strings.Builder
	for i, m := range ms {
		if i == maxSymbolCandidates {
			fmt.Fprintf(&b, "  ... %d more; narrow with path/to/dir:%s\n", len(ms)-i, m.Name)
			break
		}
		fmt.Fprintf(&b, "  %s:%s  (%s, line %d)\n", m.Path, qualifiedName(m.Qualifier, m.Name), m.Kind, m.Line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func registerKGCommands() {
	KgCmd.AddCommand(kgImpactCmd)
	kgImpactCmd.Flags().Int("depth", 0, "BFS depth (1-3, default 2 server-side)")
	kgImpactCmd.Flags().Bool("no-tests", false, "Leave out callers in test files")
	addJSONFlag(kgImpactCmd)
	addRepoFlag(kgImpactCmd)
}
