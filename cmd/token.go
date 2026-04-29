package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var TokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage agent tokens (list, issue, revoke)",
	Long:  `Commands for managing agent authentication tokens.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent tokens owned by the authenticated user",
	Long: `Retrieves and displays all agent tokens, including token ID, agent name,
scopes, creation time, expiry, and revocation status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue a new agent token",
	Long: `Creates or finds an agent with the given name and issues a new token.
Prints the clear-text token once to stdout (it is never stored or cached).
Subsequent 'token list' calls will show only metadata, not the secret.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

var revokeCmd = &cobra.Command{
	Use:   "revoke <token-id>",
	Short: "Revoke an agent token",
	Long:  `Sets the revoked_at timestamp on the token; idempotent if already revoked.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	TokenCmd.AddCommand(listCmd)
	TokenCmd.AddCommand(issueCmd)
	TokenCmd.AddCommand(revokeCmd)

	// Add flags to issue command (stub; actual parsing in Phase B/C)
	issueCmd.Flags().String("agent", "", "Agent name (required)")
	issueCmd.Flags().StringSlice("scopes", []string{}, "Token scopes (optional)")
	issueCmd.Flags().String("expires", "", "Expiration duration (optional)")
}
