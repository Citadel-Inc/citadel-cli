package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

// KgCmd is the parent for `citadel-cli kg ...`. Talks to the JWT-gated
// /api/kg/{slug}/* endpoints directly using cfg.AccessToken from
// `citadel-cli auth login`. Server URL precedence inherits from the root
// `--server` flag + CITADEL_SERVER env var via clicfg.
var KgCmd = &cobra.Command{
	Use:   "kg",
	Short: "Knowledge-graph queries (impact analysis)",
	Long: `Commands for querying the knowledge-graph substrate populated by go-kg-indexer.

Authentication uses your Supabase JWT from 'citadel-cli auth login'.`,
}

var kgImpactCmd = &cobra.Command{
	Use:   "impact <slug> <symbol>",
	Short: "Find direct + transitive callers of a symbol",
	Long: `Projects a callers-direction BFS into the rename-impact shape:
direct callers + transitive callers + affected files. Default depth = 2.

<slug> accepts <owner> or <owner>/<repo>. <symbol> accepts a UUID
(direct call) or a name (resolved via /kg/<owner>/symbols first; if more
than one symbol matches, prints the candidates so you can disambiguate).

Pretty-printed by default; use --json for the raw HTTP response.`,
	Args: cobra.ExactArgs(2),
	RunE: runKgImpact,
}

func runKgImpact(cmd *cobra.Command, args []string) error {
	slug := strings.TrimSpace(args[0])
	symbol := strings.TrimSpace(args[1])
	depth, _ := cmd.Flags().GetInt("depth")
	rawJSON, _ := cmd.Flags().GetBool("json")

	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("not authenticated: run `citadel-cli auth login`")
	}
	flagServer, _ := cmd.Flags().GetString("server")
	server := strings.TrimRight(cfg.ResolveServerURL(flagServer), "/")

	symbolID := symbol
	if _, err := uuid.Parse(symbol); err != nil {
		// Resolve <name> → UUID via /kg/<owner>/symbols.
		resolved, err := resolveSymbolID(cmd.Context(), server, cfg.AccessToken, slug, symbol)
		if err != nil {
			return err
		}
		symbolID = resolved
	}

	owner, repo := slug, ""
	if i := strings.Index(slug, "/"); i > 0 {
		owner, repo = slug[:i], slug[i+1:]
	}
	q := url.Values{}
	q.Set("symbol", symbolID)
	if depth > 0 {
		q.Set("depth", fmt.Sprintf("%d", depth))
	}
	if repo != "" {
		q.Set("repo", repo)
	}
	endpoint := fmt.Sprintf("%s/kg/%s/impact?%s", server, url.PathEscape(owner), q.Encode())

	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: run `citadel-cli auth login` to refresh your session")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if rawJSON {
		var pretty any
		_ = json.Unmarshal(body, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	var ir impactResp
	if err := json.Unmarshal(body, &ir); err != nil {
		return fmt.Errorf("decode response: %w", err)
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
	label := n.Name
	if label == "" {
		label = n.ID
	}
	if n.Path != "" {
		return label + "  in " + n.Path
	}
	return label
}

// resolveSymbolID looks up a symbol UUID by name via /kg/<owner>/symbols.
// slug accepts "<owner>" (search across owner's repos) or "<owner>/<repo>"
// (the suffix is sent as the repo filter so duplicate names across repos
// don't mask the user's intent).
func resolveSymbolID(ctx context.Context, server, accessToken, slug, name string) (string, error) {
	owner, repo := slug, ""
	if i := strings.Index(slug, "/"); i > 0 {
		owner, repo = slug[:i], slug[i+1:]
	}
	q := url.Values{}
	q.Set("q", name)
	if repo != "" {
		q.Set("repo", repo)
	}
	endpoint := fmt.Sprintf("%s/kg/%s/symbols?%s", server, url.PathEscape(owner), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("symbols lookup failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("unauthorized: run `citadel-cli auth login` to refresh your session")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("symbols lookup HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var sr struct {
		Matches []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Kind string `json:"kind"`
			Path string `json:"path"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", fmt.Errorf("decode symbols response: %w", err)
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

func listSymbolCandidates(ms []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		parts = append(parts, fmt.Sprintf("%s (%s @ %s id=%s)", m.Name, m.Kind, m.Path, m.ID))
	}
	return strings.Join(parts, "; ")
}

func init() {
	KgCmd.AddCommand(kgImpactCmd)
	kgImpactCmd.Flags().Int("depth", 0, "BFS depth (1-3, default 2 server-side)")
	kgImpactCmd.Flags().Bool("json", false, "Output raw JSON response")
}
