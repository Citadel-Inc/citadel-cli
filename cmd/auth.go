package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication (login, logout, status)",
	Long:  `Commands for managing authentication with the Citadel server.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the server via OAuth/PKCE flow",
	Long: `Starts an OAuth/PKCE authentication flow with Supabase Auth.
Opens a browser to the authorization endpoint and stores the resulting
access and refresh tokens in ~/.config/citadel/config.toml (mode 0600).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long: `Prints whether a session is active, the bound user UUID,
the access-token expiry, and the configured server URL.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the local authentication session",
	Long:  `Removes the local config file, clearing the stored tokens and session state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(statusCmd)
	AuthCmd.AddCommand(logoutCmd)
}
