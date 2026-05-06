package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var kgSearchCmd = &cobra.Command{
	Use:   "search [query...]",
	Short: "Cross-namespace fulltext search",
	Long: `Calls GET /api/kg/search with scope=cross-namespace (required by the server).

Pass the query as positional words or use --query.`,
	Args: cobra.ArbitraryArgs,
	RunE: runKgSearch,
}

var kgSymbolsCmd = &cobra.Command{
	Use:   "symbols [namespace/repo]",
	Short: "Symbol substring lookup within a namespace",
	Long: `Calls GET /api/namespaces/{slug}/kg/symbols.

Namespace/repo defaults from -R, CITADEL_REPO, or git origin when omitted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKgSymbols,
}

var kgFilesCmd = &cobra.Command{
	Use:   "files [namespace/repo]",
	Short: "List KG-indexed files for a namespace",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKgFiles,
}

var kgWalkCmd = &cobra.Command{
	Use:   "walk [namespace/repo]",
	Short: "Bounded graph walk from a seed symbol",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKgWalk,
}

var kgFulltextCmd = &cobra.Command{
	Use:   "fulltext [namespace/repo]",
	Short: "Per-namespace fulltext search",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKgFulltext,
}

var kgDiffCmd = &cobra.Command{
	Use:   "diff [namespace/repo]",
	Short: "Structural diff for a namespace/repository",
	Long: `Calls GET /api/namespaces/{namespace}/kg/diff with optional repo/ref filters.

Refs are passed as query parameters from-ref / to-ref (server naming may vary).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKgDiff,
}

func resolveKgNamespace(cmd *cobra.Command, positional string) (ns, repo string, err error) {
	positional = strings.TrimSpace(positional)
	if positional != "" {
		return resolveRepoFromPosOrFlag(cmd, positional)
	}
	return resolveRepoFlag(cmd)
}

func kgWritePayload(cmd *cobra.Command, payload any) error {
	out := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	switch out {
	case "", "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	default:
		return fmt.Errorf("--output supports json or yaml for kg queries; got %q", out)
	}
}

func runKgSearch(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("scope", "cross-namespace")

	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		query, _ = cmd.Flags().GetString("query")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("query required: pass positional text or use --query")
	}
	q.Set("q", query)

	if s := strings.TrimSpace(mustFlag(cmd, "mode")); s != "" {
		q.Set("mode", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "path-prefix")); s != "" {
		q.Set("path_prefix", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "language")); s != "" {
		q.Set("language", s)
	}

	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all {
		return fmt.Errorf("kg search does not support --all yet")
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}

	path := "/api/kg/search?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

func runKgSymbols(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	qstr, _ := cmd.Flags().GetString("q")
	qstr = strings.TrimSpace(qstr)
	if qstr == "" {
		return fmt.Errorf("--q is required")
	}

	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, rslug, err := resolveKgNamespace(cmd, pos)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("q", qstr)
	if rslug != "" {
		q.Set("repo", rslug)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "kind")); s != "" {
		q.Set("kind", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "path-prefix")); s != "" {
		q.Set("path_prefix", s)
	}

	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all {
		return fmt.Errorf("kg symbols does not support --all yet")
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}

	path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/symbols?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

func runKgFiles(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, rslug, err := resolveKgNamespace(cmd, pos)
	if err != nil {
		return err
	}
	q := url.Values{}
	if rslug != "" {
		q.Set("repo", rslug)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "path-prefix")); s != "" {
		q.Set("path_prefix", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "language")); s != "" {
		q.Set("language", s)
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all {
		return fmt.Errorf("kg files does not support --all yet")
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/files?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

func runKgWalk(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	seed, _ := cmd.Flags().GetString("seed-id")
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return fmt.Errorf("--seed-id is required")
	}
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, rslug, err := resolveKgNamespace(cmd, pos)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("seed_id", seed)
	if rslug != "" {
		q.Set("repo", rslug)
	}
	if depth, err := cmd.Flags().GetInt("depth"); err == nil && cmd.Flags().Changed("depth") {
		q.Set("depth", strconv.Itoa(depth))
	}
	if s := strings.TrimSpace(mustFlag(cmd, "direction")); s != "" {
		q.Set("direction", s)
	}
	path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/walk?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

func runKgFulltext(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	qstr, _ := cmd.Flags().GetString("q")
	qstr = strings.TrimSpace(qstr)
	if qstr == "" {
		return fmt.Errorf("--q is required")
	}
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, rslug, err := resolveKgNamespace(cmd, pos)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("q", qstr)
	if rslug != "" {
		q.Set("repo", rslug)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "mode")); s != "" {
		q.Set("mode", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "path-prefix")); s != "" {
		q.Set("path_prefix", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "language")); s != "" {
		q.Set("language", s)
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all {
		return fmt.Errorf("kg fulltext does not support --all yet")
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/fulltext?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

func runKgDiff(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, rslug, err := resolveKgNamespace(cmd, pos)
	if err != nil {
		return err
	}
	q := url.Values{}
	if rslug != "" {
		q.Set("repo", rslug)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "from-ref")); s != "" {
		q.Set("from_ref", s)
	}
	if s := strings.TrimSpace(mustFlag(cmd, "to-ref")); s != "" {
		q.Set("to_ref", s)
	}
	path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/diff?" + q.Encode()
	var payload any
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return upgradeUnauthorized(err)
	}
	return kgWritePayload(cmd, payload)
}

// mustFlag returns the flag value when registered on cmd; empty string if absent.
func mustFlag(cmd *cobra.Command, name string) string {
	if f := cmd.Flags().Lookup(name); f == nil {
		return ""
	}
	s, _ := cmd.Flags().GetString(name)
	return s
}

func init() {
	KgCmd.AddCommand(kgSearchCmd)
	KgCmd.AddCommand(kgSymbolsCmd)
	KgCmd.AddCommand(kgFilesCmd)
	KgCmd.AddCommand(kgWalkCmd)
	KgCmd.AddCommand(kgFulltextCmd)
	KgCmd.AddCommand(kgDiffCmd)

	addOutputFlag(kgSearchCmd, kgSymbolsCmd, kgFilesCmd, kgWalkCmd, kgFulltextCmd, kgDiffCmd)
	addPaginationFlags(kgSearchCmd, kgSymbolsCmd, kgFilesCmd, kgFulltextCmd)

	kgSearchCmd.Flags().String("query", "", "Search query (alternative to positional args)")
	kgSearchCmd.Flags().String("mode", "", "Search mode (e.g. fts; regex may be unsupported cross-namespace)")
	kgSearchCmd.Flags().String("path-prefix", "", "Restrict to paths with this prefix")
	kgSearchCmd.Flags().String("language", "", "Language filter")

	kgSymbolsCmd.Flags().String("q", "", "Substring query")
	_ = kgSymbolsCmd.MarkFlagRequired("q")
	kgSymbolsCmd.Flags().String("kind", "", "Symbol kind filter")
	kgSymbolsCmd.Flags().String("path-prefix", "", "Path prefix filter")

	kgFilesCmd.Flags().String("path-prefix", "", "Path prefix filter")
	kgFilesCmd.Flags().String("language", "", "Language filter")

	kgWalkCmd.Flags().String("seed-id", "", "Seed symbol UUID")
	_ = kgWalkCmd.MarkFlagRequired("seed-id")
	kgWalkCmd.Flags().Int("depth", 0, "Walk depth cap (server-enforced max)")
	kgWalkCmd.Flags().String("direction", "", "Graph direction (server-defined)")

	kgFulltextCmd.Flags().String("q", "", "Fulltext query")
	_ = kgFulltextCmd.MarkFlagRequired("q")
	kgFulltextCmd.Flags().String("mode", "", "fts or regex")
	kgFulltextCmd.Flags().String("path-prefix", "", "Path prefix filter")
	kgFulltextCmd.Flags().String("language", "", "Language filter")

	kgDiffCmd.Flags().String("from-ref", "", "From revision/ref")
	kgDiffCmd.Flags().String("to-ref", "", "To revision/ref")

	addRepoFlag(kgSymbolsCmd, kgFilesCmd, kgWalkCmd, kgFulltextCmd, kgDiffCmd)
}
