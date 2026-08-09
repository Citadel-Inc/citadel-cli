package cmd

import "github.com/spf13/cobra"

// GistCmd is the top-level command for standalone gist management.
var GistCmd = &cobra.Command{
	Use:     "gist",
	GroupID: "repo",
	Short:   "Manage standalone gists",
}
