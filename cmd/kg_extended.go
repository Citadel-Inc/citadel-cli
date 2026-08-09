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

func kgFetchPages(fetch func(cursor string) (any, error), cursor string, all bool) ([]any, error) {
	pages := make([]any, 0, 1)
	for {
		payload, err := fetch(cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, payload)
		if !all {
			return pages, nil
		}
		next := kgNextCursor(payload)
		if next == "" {
			return pages, nil
		}
		cursor = next
	}
}

func kgNextCursor(payload any) string {
	switch v := payload.(type) {
	case map[string]any:
		if next, ok := v["next_cursor"].(string); ok {
			return strings.TrimSpace(next)
		}
		for _, key := range []string{"pagination", "meta"} {
			if next := kgNextCursor(v[key]); next != "" {
				return next
			}
		}
	}
	return ""
}

func kgQueryWithPagination(base url.Values, limit int, cursor string) string {
	q := url.Values{}
	for key, values := range base {
		q[key] = append([]string(nil), values...)
	}
	q.Set("limit", strconv.Itoa(limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	} else {
		q.Del("cursor")
	}
	return q.Encode()
}

func kgPayloadRows(payload any, preferred ...string) []any {
	if rows, ok := payload.([]any); ok {
		return rows
	}
	m, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	keys := append(append([]string{}, preferred...), "results", "items", "rows", "data")
	for _, key := range keys {
		if rows, ok := m[key].([]any); ok {
			return rows
		}
		if nested, ok := m[key].(map[string]any); ok {
			if rows := kgPayloadRows(nested, preferred...); rows != nil {
				return rows
			}
		}
	}
	for key, value := range m {
		if key == "next_cursor" {
			continue
		}
		if rows := kgPayloadRows(value, preferred...); rows != nil {
			return rows
		}
	}
	return nil
}

func kgWritePages(cmd *cobra.Command, pages []any, all bool, rowKeys ...string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if all && output == "json" {
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	if !all {
		if output == "table" {
			return kgWriteTable(cmd, pages, rowKeys...)
		}
		if output == "ndjson" {
			return kgWriteNDJSON(cmd, pages, rowKeys...)
		}
		return kgWritePayload(cmd, pages[0])
	}
	if output == "" {
		output = "table"
	}
	switch output {
	case "ndjson":
		return kgWriteNDJSON(cmd, pages, rowKeys...)
	case "table":
		return kgWriteTable(cmd, pages, rowKeys...)
	case "yaml":
		rows := make([]any, 0)
		for _, page := range pages {
			rows = append(rows, kgPayloadRows(page, rowKeys...)...)
		}
		return emitYAML(cmd, rows)
	default:
		return fmt.Errorf("--output supports json, yaml, ndjson, or table for kg queries; got %q", output)
	}
}

func kgWriteNDJSON(cmd *cobra.Command, pages []any, rowKeys ...string) error {
	for _, page := range pages {
		rows := kgPayloadRows(page, rowKeys...)
		if len(rows) == 0 {
			continue
		}
		if err := emitNDJSONLines(cmd, rows); err != nil {
			return err
		}
	}
	return nil
}

func kgWriteTable(cmd *cobra.Command, pages []any, rowKeys ...string) error {
	rows := make([]map[string]any, 0)
	for _, page := range pages {
		for _, row := range kgPayloadRows(page, rowKeys...) {
			if m, ok := row.(map[string]any); ok {
				rows = append(rows, m)
			}
		}
	}
	w := newTabWriter(cmd)
	if len(rowKeys) > 0 && rowKeys[0] == "symbols" {
		_, _ = fmt.Fprintln(w, "NAME\tKIND\tPATH\tLINE\tSIGNATURE")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				kgCell(row, "name", "symbol_name", "symbol", "qualified_name"),
				kgCell(row, "kind", "symbol_kind", "type"),
				kgCell(row, "path", "file_path", "file"),
				kgCell(row, "line", "line_start", "start_line", "line_number"),
				kgCell(row, "signature", "signature_text", "detail"),
			)
		}
	} else {
		_, _ = fmt.Fprintln(w, "NAMESPACE\tREPOSITORY\tPATH\tLINE\tSYMBOL\tSNIPPET")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				kgCell(row, "namespace", "namespace_slug", "namespace_name", "owner"),
				kgCell(row, "repo", "repo_slug", "repository", "repository_slug", "repo_name"),
				kgCell(row, "path", "file_path", "file"),
				kgCell(row, "line", "line_start", "start_line", "line_number"),
				kgCell(row, "symbol", "name", "qualified_name"),
				kgCell(row, "snippet", "match", "content", "text"),
			)
		}
	}
	return w.Flush()
}

func kgCell(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
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
	pages, err := kgFetchPages(func(pageCursor string) (any, error) {
		var payload any
		path := "/api/kg/search?" + kgQueryWithPagination(q, limit, pageCursor)
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return nil, upgradeUnauthorized(err)
		}
		return payload, nil
	}, cursor, all)
	if err != nil {
		return err
	}
	return kgWritePages(cmd, pages, all, "results")
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
	pages, err := kgFetchPages(func(pageCursor string) (any, error) {
		var payload any
		path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/symbols?" + kgQueryWithPagination(q, limit, pageCursor)
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return nil, upgradeUnauthorized(err)
		}
		return payload, nil
	}, cursor, all)
	if err != nil {
		return err
	}
	return kgWritePages(cmd, pages, all, "symbols")
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
	pages, err := kgFetchPages(func(pageCursor string) (any, error) {
		var payload any
		path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/files?" + kgQueryWithPagination(q, limit, pageCursor)
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return nil, upgradeUnauthorized(err)
		}
		return payload, nil
	}, cursor, all)
	if err != nil {
		return err
	}
	return kgWritePages(cmd, pages, all, "files")
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
	pages, err := kgFetchPages(func(pageCursor string) (any, error) {
		var payload any
		path := "/api/namespaces/" + url.PathEscape(ns) + "/kg/fulltext?" + kgQueryWithPagination(q, limit, pageCursor)
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return nil, upgradeUnauthorized(err)
		}
		return payload, nil
	}, cursor, all)
	if err != nil {
		return err
	}
	return kgWritePages(cmd, pages, all, "results", "matches")
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
