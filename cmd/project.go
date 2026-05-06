package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ProjectCmd is `citadel-cli project` — namespace-scoped project graph API (JWT).
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Inspect and update the Citadel project graph (namespaces, pins, edges)",
	Long: `Talks to GET/POST /api/projectgraph/... with your saved session.

Namespace paths may be multi-segment (for example org/repo or org/project/repo); the
graph is anchored on namespaces whether they represent organizations, projects, or
repositories.

HTTP 404 from these routes may mean "not found" or insufficient projectgraph:read /
projectgraph:manage scope — the server sometimes uses opaque denials.

Large responses (walk, neighbors): prefer --output json and filter with jq.`,
}

// ── pin-chain ────────────────────────────────────────────────────────────────

var projectPinChainCmd = &cobra.Command{
	Use:   "pin-chain <namespace-path>",
	Short: "GET pin chain for a repo namespace",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectPinChain,
}

func runProjectPinChain(cmd *cobra.Command, args []string) error {
	slug := strings.TrimSpace(args[0])
	path := projectgraphPath(slug, "pin-chain")
	return projectGET(cmd, path)
}

// ── walk ─────────────────────────────────────────────────────────────────────

var projectWalkCmd = &cobra.Command{
	Use:   "walk <namespace-path>",
	Short: "GET dependency walk from a namespace",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectWalk,
}

func runProjectWalk(cmd *cobra.Command, args []string) error {
	kind, _ := cmd.Flags().GetString("kind")
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return fmt.Errorf("--kind is required")
	}
	maxDepth, _ := cmd.Flags().GetInt("max-depth")

	slug := strings.TrimSpace(args[0])
	q := url.Values{}
	q.Set("kind", kind)
	if cmd.Flags().Changed("max-depth") && maxDepth > 0 {
		q.Set("max_depth", strconv.Itoa(maxDepth))
	}
	suffix := "walk"
	if enc := q.Encode(); enc != "" {
		suffix += "?" + enc
	}
	path := projectgraphPath(slug, suffix)
	return projectGET(cmd, path)
}

// ── neighbors ──────────────────────────────────────────────────────────────────

var projectNeighborsCmd = &cobra.Command{
	Use:   "neighbors <namespace-path>",
	Short: "GET neighbors for a namespace",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectNeighbors,
}

func runProjectNeighbors(cmd *cobra.Command, args []string) error {
	kind, _ := cmd.Flags().GetString("kind")
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return fmt.Errorf("--kind is required")
	}
	slug := strings.TrimSpace(args[0])
	q := url.Values{}
	q.Set("kind", kind)
	if ns, _ := cmd.Flags().GetString("ns"); strings.TrimSpace(ns) != "" {
		q.Set("ns", strings.TrimSpace(ns))
	}
	if d, _ := cmd.Flags().GetString("direction"); strings.TrimSpace(d) != "" {
		q.Set("direction", strings.TrimSpace(d))
	}
	if inc, _ := cmd.Flags().GetBool("include-deleted"); inc {
		q.Set("include_deleted", "true")
	}
	suffix := "neighbors"
	if enc := q.Encode(); enc != "" {
		suffix += "?" + enc
	}
	path := projectgraphPath(slug, suffix)
	return projectGET(cmd, path)
}

// ── status ─────────────────────────────────────────────────────────────────────

var projectStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status rollup and drilldown helpers",
}

var projectStatusRollupCmd = &cobra.Command{
	Use:   "rollup <namespace-path>",
	Short: "GET aggregated status rollup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := strings.TrimSpace(args[0])
		path := projectgraphPath(slug, "status-rollup")
		return projectGET(cmd, path)
	},
}

var projectStatusDrilldownCmd = &cobra.Command{
	Use:   "drilldown <namespace-path>",
	Short: "GET status rollup drilldown",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := strings.TrimSpace(args[0])
		path := projectgraphPath(slug, "status-rollup/drilldown")
		return projectGET(cmd, path)
	},
}

// ── edges ──────────────────────────────────────────────────────────────────────

var projectEdgeCmd = &cobra.Command{
	Use:   "edge",
	Short: "Create, delete, or restore graph edges",
}

var projectEdgeAddCmd = &cobra.Command{
	Use:   "add <namespace-path>",
	Short: "POST a manual edge from this namespace scope",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectEdgeAdd,
}

func runProjectEdgeAdd(cmd *cobra.Command, args []string) error {
	outMode := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if jsonFlag(cmd) {
		outMode = "json"
	}
	if err := validateGetOutput(outMode); err != nil {
		return err
	}

	slug := strings.TrimSpace(args[0])
	fromNS, err := cmd.Flags().GetString("from-namespace-id")
	if err != nil {
		return err
	}
	fromNS = strings.TrimSpace(fromNS)
	if _, err := uuid.Parse(fromNS); err != nil {
		return fmt.Errorf("--from-namespace-id must be a UUID")
	}
	fromKind, _ := cmd.Flags().GetString("from-kind")
	fromKind = strings.TrimSpace(fromKind)
	if fromKind == "" {
		return fmt.Errorf("--from-kind is required")
	}
	toKind, _ := cmd.Flags().GetString("to-kind")
	toKind = strings.TrimSpace(toKind)
	if toKind == "" {
		return fmt.Errorf("--to-kind is required")
	}
	edgeType, _ := cmd.Flags().GetString("edge-type")
	edgeType = strings.TrimSpace(edgeType)
	if edgeType == "" {
		return fmt.Errorf("--edge-type is required")
	}
	attrStr, _ := cmd.Flags().GetString("attrs-json")
	attrStr = strings.TrimSpace(attrStr)
	var attrs json.RawMessage
	switch {
	case attrStr == "":
		attrs = json.RawMessage([]byte("{}"))
	default:
		if !json.Valid([]byte(attrStr)) {
			return fmt.Errorf("--attrs-json must be valid JSON")
		}
		attrs = json.RawMessage([]byte(attrStr))
	}
	toNS, _ := cmd.Flags().GetString("to-namespace-id")
	toNS = strings.TrimSpace(toNS)
	toExt, _ := cmd.Flags().GetString("to-external-id")
	toExt = strings.TrimSpace(toExt)

	body := map[string]any{
		"from_namespace_id": fromNS,
		"from_kind":         fromKind,
		"to_kind":           toKind,
		"edge_type":         edgeType,
		"attrs":             attrs,
		"source":            "manual",
	}
	if toNS != "" {
		if _, err := uuid.Parse(toNS); err != nil {
			return fmt.Errorf("--to-namespace-id must be a UUID")
		}
		body["to_namespace_id"] = toNS
	}
	if toExt != "" {
		body["to_external_id"] = toExt
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := projectgraphPath(slug, "edges")
	var out map[string]any
	if err := c.Post(cmd.Context(), path, body, &out); err != nil {
		return upgradeUnauthorized(err)
	}
	return projectEmitMutationOK(cmd, out)
}

var projectEdgeDeleteCmd = &cobra.Command{
	Use:   "delete <namespace-path> <edge-id>",
	Short: "DELETE an edge by id",
	Args:  cobra.ExactArgs(2),
	RunE:  runProjectEdgeDelete,
}

func runProjectEdgeDelete(cmd *cobra.Command, args []string) error {
	slug := strings.TrimSpace(args[0])
	edgeID := strings.TrimSpace(args[1])
	if _, err := uuid.Parse(edgeID); err != nil {
		return fmt.Errorf("edge-id must be a UUID")
	}
	if err := confirmSlug(yesFlag(cmd), "delete graph edge", edgeID); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := projectgraphPath(slug, "edges/"+url.PathEscape(edgeID))
	if err := c.Delete(cmd.Context(), path); err != nil {
		return upgradeUnauthorized(err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ok")
	return nil
}

var projectEdgeRestoreCmd = &cobra.Command{
	Use:   "restore <namespace-path> <edge-id>",
	Short: "Restore a deleted edge",
	Args:  cobra.ExactArgs(2),
	RunE:  runProjectEdgeRestore,
}

func runProjectEdgeRestore(cmd *cobra.Command, args []string) error {
	slug := strings.TrimSpace(args[0])
	edgeID := strings.TrimSpace(args[1])
	if _, err := uuid.Parse(edgeID); err != nil {
		return fmt.Errorf("edge-id must be a UUID")
	}
	if err := confirmSlug(yesFlag(cmd), "restore graph edge", edgeID); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := projectgraphPath(slug, "edges/"+url.PathEscape(edgeID)+"/restore")
	var out map[string]any
	if err := c.Post(cmd.Context(), path, map[string]any{}, &out); err != nil {
		return upgradeUnauthorized(err)
	}
	return projectEmitMutationOK(cmd, out)
}

// ── reindex ────────────────────────────────────────────────────────────────────

var projectReindexCmd = &cobra.Command{
	Use:   "reindex <namespace-path>",
	Short: "POST reindex hook for a namespace (destructive-adjacent)",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectReindex,
}

func runProjectReindex(cmd *cobra.Command, args []string) error {
	slug := strings.TrimSpace(args[0])
	if err := confirmSlug(yesFlag(cmd), "reindex project graph", slug); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := projectgraphPath(slug, "reindex")
	var out map[string]any
	if err := c.Post(cmd.Context(), path, map[string]any{}, &out); err != nil {
		return upgradeUnauthorized(err)
	}
	return projectEmitMutationOK(cmd, out)
}

// projectgraphPath builds /api/projectgraph/<escaped slug>/<suffix>.
func projectgraphPath(slug, suffix string) string {
	slug = strings.Trim(slug, "/")
	parts := strings.Split(slug, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	enc := strings.Join(parts, "/")
	suffix = strings.TrimPrefix(strings.TrimSpace(suffix), "/")
	return "/api/projectgraph/" + enc + "/" + suffix
}

func projectGET(cmd *cobra.Command, path string) error {
	outMode := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if jsonFlag(cmd) {
		outMode = "json"
	}
	if err := validateGetOutput(outMode); err != nil {
		return err
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	body, err := projectGETRaw(cmd.Context(), c, path)
	if err != nil {
		return upgradeUnauthorized(err)
	}

	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	switch outMode {
	case "json":
		return emitJSON(cmd, v)
	case "yaml":
		return emitYAML(cmd, v)
	default:
		return projectHumanPreview(cmd, v)
	}
}

func projectGETRaw(ctx context.Context, c *apiclient.Client, path string) ([]byte, error) {
	var raw json.RawMessage
	if err := c.Get(ctx, path, &raw); err != nil {
		return nil, err
	}
	return []byte(raw), nil
}

func projectHumanPreview(cmd *cobra.Command, v any) error {
	switch x := v.(type) {
	case []any:
		if len(x) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(empty)")
			return nil
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d row(s). Use --output json for the full response.\n", len(x))
		preview := x
		if len(preview) > 8 {
			preview = preview[:8]
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "IDX\tPREVIEW")
		for i, row := range preview {
			_, _ = fmt.Fprintf(w, "%d\t%s\n", i, projectPreviewCell(row))
		}
		return w.Flush()
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		w := newTabWriter(cmd)
		for _, k := range keys {
			_, _ = fmt.Fprintf(w, "%s:\t%s\n", k, projectPreviewCell(x[k]))
		}
		return w.Flush()
	default:
		return emitJSON(cmd, v)
	}
}

func projectPreviewCell(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	s := string(b)
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

func projectEmitMutationOK(cmd *cobra.Command, out map[string]any) error {
	outMode := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if jsonFlag(cmd) {
		outMode = "json"
	}
	if err := validateGetOutput(outMode); err != nil {
		return err
	}
	switch outMode {
	case "", "table":
		if status, ok := out["status"].(string); ok && status != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		}
		return emitJSON(cmd, out)
	case "json":
		return emitJSON(cmd, out)
	case "yaml":
		return emitYAML(cmd, out)
	default:
		return fmt.Errorf("--output: only json|yaml|table supported for this command")
	}
}

func init() {
	addOutputFlag(
		projectPinChainCmd, projectWalkCmd, projectNeighborsCmd,
		projectStatusRollupCmd, projectStatusDrilldownCmd,
		projectEdgeAddCmd, projectEdgeRestoreCmd,
	)
	addJSONFlag(
		projectPinChainCmd, projectWalkCmd, projectNeighborsCmd,
		projectStatusRollupCmd, projectStatusDrilldownCmd,
		projectEdgeAddCmd, projectEdgeRestoreCmd,
	)
	addYesFlag(projectEdgeDeleteCmd, projectEdgeRestoreCmd, projectReindexCmd)

	projectWalkCmd.Flags().String("kind", "", "Graph kind (required)")
	projectWalkCmd.Flags().Int("max-depth", 0, "Optional positive walk depth")
	_ = projectWalkCmd.MarkFlagRequired("kind")

	projectNeighborsCmd.Flags().String("kind", "", "Neighbor kind filter (required)")
	projectNeighborsCmd.Flags().String("ns", "", "Override target namespace (optional)")
	projectNeighborsCmd.Flags().String("direction", "", "Edge direction filter")
	projectNeighborsCmd.Flags().Bool("include-deleted", false, "Include tombstoned edges")
	_ = projectNeighborsCmd.MarkFlagRequired("kind")

	projectEdgeAddCmd.Flags().String("from-namespace-id", "", "From namespace UUID")
	projectEdgeAddCmd.Flags().String("from-kind", "", "From namespace kind")
	projectEdgeAddCmd.Flags().String("to-namespace-id", "", "To namespace UUID (optional)")
	projectEdgeAddCmd.Flags().String("to-kind", "", "To namespace kind")
	projectEdgeAddCmd.Flags().String("to-external-id", "", "External target id (optional)")
	projectEdgeAddCmd.Flags().String("edge-type", "", "Edge type atom")
	projectEdgeAddCmd.Flags().String("attrs-json", "", "JSON object merged into attrs (default {})")
	_ = projectEdgeAddCmd.MarkFlagRequired("from-namespace-id")
	_ = projectEdgeAddCmd.MarkFlagRequired("from-kind")
	_ = projectEdgeAddCmd.MarkFlagRequired("to-kind")
	_ = projectEdgeAddCmd.MarkFlagRequired("edge-type")

	projectEdgeCmd.AddCommand(projectEdgeAddCmd, projectEdgeDeleteCmd, projectEdgeRestoreCmd)

	ProjectCmd.AddCommand(
		projectPinChainCmd,
		projectWalkCmd,
		projectNeighborsCmd,
		projectStatusCmd,
		projectEdgeCmd,
		projectReindexCmd,
	)
	projectStatusCmd.AddCommand(projectStatusRollupCmd, projectStatusDrilldownCmd)
}
