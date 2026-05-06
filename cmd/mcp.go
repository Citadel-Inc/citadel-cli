package cmd

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/mcpclient"
)

// ErrToolCallFailed signals that a tools/call returned isError=true; main
// translates it to exit code 2.
var ErrToolCallFailed = errors.New("tool call returned isError")

// McpCmd is the parent for `citadel-cli mcp ...`. Speaks the MCP Streamable
// HTTP protocol against /mcp.
//
// Authentication: defaults to cfg.AccessToken (normally the opaque agent token
// minted by `citadel-cli auth login`). Override with --token or
// CITADEL_AGENT_TOKEN for agent / CI use. The MCP server accepts both OAuth
// JWTs and opaque agent tokens where the route supports them.
var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Interact with MCP tools, resources, and prompts",
	Long: `Commands for listing MCP tools, resources, and prompts and invoking
tool / resource / prompt RPCs via the Citadel MCP server.

Authentication defaults to your saved Citadel CLI session from 'citadel-cli auth
login' (normally an opaque agent token). Override with --token or
CITADEL_AGENT_TOKEN for explicit agent / CI workflows.`,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	RunE:  runMcpTools,
}

var callCmd = &cobra.Command{
	Use:   "call <tool>",
	Short: "Call an MCP tool with arguments",
	Long: `Invokes a named MCP tool with optional --arg key=value pairs.
Results are pretty-printed by default; use --json for the raw JSON-RPC
response. Args coerce by default: digits→number, CSV→array, else string.
Use --arg-string key=value to force string for a single arg.`,
	Args: cobra.ExactArgs(1),
	RunE: runMcpCall,
}

var mcpResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List and read MCP resources (citadel://, repo://)",
}

var mcpResourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources (resources/list)",
	RunE:  runMcpResourcesList,
}

var mcpResourcesReadCmd = &cobra.Command{
	Use:   "read <uri>",
	Short: "Read a resource URI (resources/read)",
	Args:  cobra.ExactArgs(1),
	RunE:  runMcpResourcesRead,
}

var mcpPromptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "List and fetch MCP prompts",
}

var mcpPromptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List prompts (prompts/list)",
	RunE:  runMcpPromptsList,
}

var mcpPromptsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Fetch a prompt template (prompts/get)",
	Args:  cobra.ExactArgs(1),
	RunE:  runMcpPromptsGet,
}

func runMcpTools(cmd *cobra.Command, _ []string) error {
	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	tools, err := c.ToolsList(cmd.Context())
	if err != nil {
		return surfaceErr(err)
	}
	w := newTabWriter(cmd)
	for _, t := range tools {
		if t.Description == "" {
			_, _ = fmt.Fprintln(w, t.Name)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", t.Name, t.Description)
		}
	}
	return w.Flush()
}

func runMcpCall(cmd *cobra.Command, args []string) error {
	toolName := args[0]
	rawJSON := jsonFlag(cmd)
	argPairs, _ := cmd.Flags().GetStringSlice("arg")
	stringArgPairs, _ := cmd.Flags().GetStringSlice("arg-string")

	toolArgs, err := parseArgPairs(argPairs, stringArgPairs)
	if err != nil {
		return err
	}

	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	res, err := c.ToolsCall(cmd.Context(), toolName, toolArgs)
	if err != nil {
		return surfaceErr(err)
	}
	if rawJSON {
		if err := emitRawJSON(cmd, res.Raw); err != nil {
			return err
		}
	} else {
		printToolResult(res)
	}
	if res.IsError {
		return ErrToolCallFailed
	}
	return nil
}

// parseArgPairs splits --arg key=value pairs (with type coercion) and
// --arg-string key=value pairs (raw string) into a single argument map.
func parseArgPairs(coerced, raw []string) (map[string]any, error) {
	out := make(map[string]any, len(coerced)+len(raw))
	for _, p := range coerced {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("bad --arg %q (expected key=value)", p)
		}
		out[k] = coerceArg(v)
	}
	for _, p := range raw {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("bad --arg-string %q (expected key=value)", p)
		}
		out[k] = v
	}
	return out, nil
}

// emitRawJSON re-encodes a json.RawMessage with indentation. Surfaces decode
// errors instead of swallowing them.
func emitRawJSON(cmd *cobra.Command, raw json.RawMessage) error {
	var pretty any
	if err := json.Unmarshal(raw, &pretty); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return emitJSON(cmd, pretty)
}

func runMcpResourcesList(cmd *cobra.Command, _ []string) error {
	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	rows, err := c.ResourcesList(cmd.Context())
	if err != nil {
		return surfaceErr(err)
	}
	w := newTabWriter(cmd)
	for _, r := range rows {
		if r.Description == "" {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", r.URI, r.Name)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.URI, r.Name, r.Description)
		}
	}
	return w.Flush()
}

func runMcpResourcesRead(cmd *cobra.Command, args []string) error {
	uri := args[0]
	rawJSON := jsonFlag(cmd)
	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	raw, err := c.ResourcesRead(cmd.Context(), uri)
	if err != nil {
		return surfaceErr(err)
	}
	if rawJSON {
		return emitRawJSON(cmd, raw)
	}
	var parsed struct {
		Contents []map[string]any `json:"contents"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("decode resources/read: %w", err)
	}
	for _, block := range parsed.Contents {
		printContentBlock(block)
	}
	return nil
}

func runMcpPromptsList(cmd *cobra.Command, _ []string) error {
	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	rows, err := c.PromptsList(cmd.Context())
	if err != nil {
		return surfaceErr(err)
	}
	w := newTabWriter(cmd)
	for _, p := range rows {
		if p.Description == "" {
			_, _ = fmt.Fprintln(w, p.Name)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", p.Name, p.Description)
		}
	}
	return w.Flush()
}

func runMcpPromptsGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	rawJSON := jsonFlag(cmd)
	argPairs, _ := cmd.Flags().GetStringSlice("arg")
	stringArgPairs, _ := cmd.Flags().GetStringSlice("arg-string")

	promptArgs, err := parseArgPairs(argPairs, stringArgPairs)
	if err != nil {
		return err
	}

	c, err := dialMCP(cmd)
	if err != nil {
		return err
	}
	raw, err := c.PromptsGet(cmd.Context(), name, promptArgs)
	if err != nil {
		return surfaceErr(err)
	}
	if rawJSON {
		return emitRawJSON(cmd, raw)
	}
	var parsed struct {
		Description string           `json:"description"`
		Messages    []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("decode prompts/get: %w", err)
	}
	if parsed.Description != "" {
		fmt.Println(parsed.Description)
		fmt.Println()
	}
	for _, m := range parsed.Messages {
		role, _ := m["role"].(string)
		content, _ := m["content"].(map[string]any)
		if content != nil && content["type"] == "text" {
			if text, ok := content["text"].(string); ok {
				if role != "" {
					fmt.Printf("[%s]\n", role)
				}
				fmt.Println(text)
				continue
			}
		}
		if err := emitJSON(cmd, m); err != nil {
			fmt.Fprintf(os.Stderr, "encode prompt message: %v\n", err)
		}
	}
	return nil
}

// dialMCP loads config + flags + env into a connected (Initialize-d)
// mcpclient. Token precedence: --token > CITADEL_AGENT_TOKEN >
// cfg.AccessToken.
func dialMCP(cmd *cobra.Command) (*mcpclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	flagServer, _ := cmd.Flags().GetString("server")
	flagToken, _ := cmd.Flags().GetString("token")
	timeoutSecs, _ := cmd.Flags().GetInt("timeout")

	mcpURL := resolveMCPURL(cfg.ResolveServerURL(flagServer))
	token := pickToken(flagToken, cfg.AccessToken)
	if token == "" {
		return nil, errors.New("no auth token: run `citadel-cli auth login` (or pass --token / set CITADEL_AGENT_TOKEN)")
	}

	c := mcpclient.New(mcpURL, token, time.Duration(timeoutSecs)*time.Second, mcpclient.Options{Verbose: verboseFlag(cmd), DebugHTTP: debugHTTPFlag(cmd)})
	if err := c.Initialize(cmd.Context()); err != nil {
		return nil, surfaceErr(err)
	}
	return c, nil
}

// resolveMCPURL turns the resolved server base URL into the /mcp path.
// If the operator pointed --server at api.src.land, swap to mcp.src.land
// since the production MCP endpoint lives on its own subdomain.
func resolveMCPURL(server string) string {
	url := strings.TrimRight(server, "/") + "/mcp"
	return strings.Replace(url, "api.src.land/mcp", "mcp.src.land/mcp", 1)
}

// pickToken applies the token-precedence chain: --token > env > JWT.
func pickToken(flagToken, jwt string) string {
	return cmp.Or(flagToken, os.Getenv("CITADEL_AGENT_TOKEN"), jwt)
}

// surfaceErr maps mcpclient errors to user copy. Auth failures point at
// `citadel-cli auth login` per spec §Auth; everything else passes through.
func surfaceErr(err error) error {
	if mcpclient.IsUnauthorized(err) {
		return errors.New("unauthorized: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN")
	}
	return err
}

// printToolResult pretty-prints a tools/call result. Text content blocks
// emit one per line; non-text content falls through to JSON.
func printToolResult(res *mcpclient.ToolCallResult) {
	for _, c := range res.Content {
		printContentBlock(c)
	}
}

// printContentBlock prints one MCP content block: type=text → text on its own
// line; everything else → indented JSON. Errors during JSON encoding go to
// stderr instead of being silently dropped.
func printContentBlock(m map[string]any) {
	if m["type"] == "text" {
		if text, ok := m["text"].(string); ok {
			fmt.Println(text)
			return
		}
	}
	if err := emitJSON(nil, m); err != nil {
		fmt.Fprintf(os.Stderr, "encode content block: %v\n", err)
	}
}

// coerceArg implements the spec A6 / Q2 coercion: digit-only → number,
// CSV → array (with each element coerced recursively), bare boolean
// literals → bool, everything else → string. --arg-string opts out.
func coerceArg(v string) any {
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if n, ok := parseInt(v); ok {
		return n
	}
	if f, ok := parseFloat(v); ok {
		return f
	}
	if strings.Contains(v, ",") {
		parts := strings.Split(v, ",")
		out := make([]any, 0, len(parts))
		for _, p := range parts {
			out = append(out, coerceArg(p))
		}
		return out
	}
	return v
}

// parseInt accepts only the canonical integer form: optional leading
// minus, then digits with no leading zero (except literal "0"). This
// avoids treating zip codes / IDs like "07823" as numbers and clobbering
// their leading zeros.
func parseInt(v string) (int64, bool) {
	if v == "" {
		return 0, false
	}
	s := strings.TrimPrefix(v, "-")
	if s == "" {
		return 0, false
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseFloat accepts a decimal with at least one dot and digits on both
// sides; rejects scientific notation, hex, and ambiguous forms.
func parseFloat(v string) (float64, bool) {
	if !strings.Contains(v, ".") {
		return 0, false
	}
	s := strings.TrimPrefix(v, "-")
	dot := strings.Index(s, ".")
	if dot <= 0 || dot == len(s)-1 {
		return 0, false
	}
	for _, r := range s {
		if r != '.' && (r < '0' || r > '9') {
			return 0, false
		}
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func init() {
	McpCmd.AddCommand(toolsCmd)
	McpCmd.AddCommand(callCmd)
	McpCmd.AddCommand(mcpResourcesCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesListCmd, mcpResourcesReadCmd)
	McpCmd.AddCommand(mcpPromptsCmd)
	mcpPromptsCmd.AddCommand(mcpPromptsListCmd, mcpPromptsGetCmd)

	McpCmd.PersistentFlags().String("token", "", "Auth token (overrides CITADEL_AGENT_TOKEN env var; defaults to your `citadel-cli auth login` session JWT)")
	McpCmd.PersistentFlags().Int("timeout", 60, "Per-call HTTP timeout in seconds")

	callCmd.Flags().StringSlice("arg", []string{}, "Tool arguments as key=value pairs (digits→number, CSV→array, else string)")
	callCmd.Flags().StringSlice("arg-string", []string{}, "Tool arguments forced to string (no coercion)")
	mcpPromptsGetCmd.Flags().StringSlice("arg", []string{}, "Prompt arguments as key=value (same coercion as mcp call)")
	mcpPromptsGetCmd.Flags().StringSlice("arg-string", []string{}, "Prompt arguments forced to string")
	addJSONFlag(callCmd, mcpResourcesReadCmd, mcpPromptsGetCmd)
}
