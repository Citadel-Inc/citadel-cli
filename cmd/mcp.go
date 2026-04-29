package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Interact with MCP tools",
	Long:  `Commands for listing and calling MCP tools via the server.`,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long: `Retrieves and displays all available MCP tools from the configured server,
including tool name and description.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

var callCmd = &cobra.Command{
	Use:   "call <tool>",
	Short: "Call an MCP tool with arguments",
	Long: `Invokes a named MCP tool with optional key=value arguments.
Results are pretty-printed by default; use --json for raw output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	McpCmd.AddCommand(toolsCmd)
	McpCmd.AddCommand(callCmd)

	// Add flags to call command (stub; actual parsing in Phase B/C)
	callCmd.Flags().StringSlice("arg", []string{}, "Tool arguments as key=value pairs")
	callCmd.Flags().Bool("json", false, "Output raw JSON")
}
