package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	q := url.Values{}
	q.Set("symbol", symbol)
	if depth > 0 {
		q.Set("depth", fmt.Sprintf("%d", depth))
	}
	endpoint := fmt.Sprintf("%s/api/kg/%s/impact?%s", server, url.PathEscape(slug), q.Encode())

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

func init() {
	KgCmd.AddCommand(kgImpactCmd)
	kgImpactCmd.Flags().Int("depth", 0, "BFS depth (1-3, default 2 server-side)")
	kgImpactCmd.Flags().Bool("json", false, "Output raw JSON response")
}
