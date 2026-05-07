package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── domain types ─────────────────────────────────────────────────────────────

type treeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Kind string `json:"kind"` // "blob" or "tree"
	Size int64  `json:"size"`
	SHA  string `json:"sha"`
}

type treeResponse struct {
	Ref     string      `json:"ref"`
	Path    string      `json:"path"`
	Entries []treeEntry `json:"entries"`
}

type blobResponse struct {
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Binary   bool   `json:"binary"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

// ── command tree ─────────────────────────────────────────────────────────────

var repoBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse repository file tree and file contents",
}

var repoBrowseTreeCmd = &cobra.Command{
	Use:   "tree [<namespace>/<repo>]",
	Short: "List directory entries in a repository at a given ref and path",
	Long: `List directory entries (files and subdirectories) in a repository.

Defaults to the root directory of the repository's default branch.
Use --ref to target a specific branch, tag, or commit SHA.
Use --path to list a subdirectory.`,
	Example: `  # List the root of the default branch
  citadel-cli repo browse tree acme/myrepo

  # List a subdirectory on a specific branch
  citadel-cli repo browse tree acme/myrepo --ref main --path cmd

  # Output as JSON
  citadel-cli repo browse tree acme/myrepo --output json`,
	RunE: runRepoBrowseTree,
}

var repoBrowseBlobCmd = &cobra.Command{
	Use:   "blob [<namespace>/<repo>]",
	Short: "Read a file's content from a repository at a given ref",
	Long: `Read the content of a file in a repository.

In human mode, the file content is printed directly to stdout (suitable for
piping). Binary files print an informational line instead of raw bytes.
Use --output json to get the full metadata envelope (sha, size, binary, content).`,
	Example: `  # Read a file on the default branch
  citadel-cli repo browse blob acme/myrepo --path README.md

  # Read a file on a specific branch
  citadel-cli repo browse blob acme/myrepo --path src/main.go --ref feature/x

  # Get metadata as JSON
  citadel-cli repo browse blob acme/myrepo --path go.mod --output json`,
	RunE: runRepoBrowseBlob,
}

// ── handlers ─────────────────────────────────────────────────────────────────

func runRepoBrowseTree(cmd *cobra.Command, args []string) error {
	posArg := ""
	if len(args) > 0 {
		posArg = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, posArg)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	if err := validateGetOutput(output); err != nil {
		return err
	}

	ref, _ := cmd.Flags().GetString("ref")
	path, _ := cmd.Flags().GetString("path")

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	q := url.Values{}
	if ref != "" {
		q.Set("ref", ref)
	}
	if path != "" {
		q.Set("path", path)
	}
	apiPath := fmt.Sprintf("/api/namespaces/%s/repos/%s/tree", ns, slug)
	if len(q) > 0 {
		apiPath += "?" + q.Encode()
	}

	var resp treeResponse
	if err := client.Get(cmd.Context(), apiPath, &resp); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			if ref != "" {
				return fmt.Errorf("ref or path not found: %s", ref)
			}
			return fmt.Errorf("repository not found: %s/%s", ns, slug)
		}
		if apiclient.IsStatus(err, http.StatusUnauthorized) {
			return fmt.Errorf("authentication required — run: citadel-cli auth login")
		}
		return err
	}

	if output != "" && output != "table" {
		return emitJSON(cmd, resp)
	}

	renderTreeEntries(cmd, resp)
	return nil
}

func renderTreeEntries(cmd *cobra.Command, resp treeResponse) {
	w := newTabWriter(cmd)
	for _, e := range resp.Entries {
		icon := "📄"
		sizeStr := formatFileSize(e.Size)
		if e.Kind == "tree" {
			icon = "📁"
			sizeStr = ""
		}
		_, _ = fmt.Fprintf(w, "%s  %-40s\t%8s\t%s\n",
			icon,
			e.Path,
			sizeStr,
			shortSHA(e.SHA),
		)
	}
	if err := w.Flush(); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: flush: %v\n", err)
	}
}

func formatFileSize(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func runRepoBrowseBlob(cmd *cobra.Command, args []string) error {
	posArg := ""
	if len(args) > 0 {
		posArg = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, posArg)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	if err := validateGetOutput(output); err != nil {
		return err
	}

	path, _ := cmd.Flags().GetString("path")
	if path == "" {
		return fmt.Errorf("--path is required for blob")
	}
	ref, _ := cmd.Flags().GetString("ref")

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("path", path)
	if ref != "" {
		q.Set("ref", ref)
	}
	apiPath := fmt.Sprintf("/api/namespaces/%s/repos/%s/blob?%s", ns, slug, q.Encode())

	var resp blobResponse
	if err := client.Get(cmd.Context(), apiPath, &resp); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("file not found: %s", path)
		}
		if apiclient.IsStatus(err, http.StatusBadRequest) {
			return fmt.Errorf("invalid path: %s", path)
		}
		if apiclient.IsStatus(err, http.StatusUnauthorized) {
			return fmt.Errorf("authentication required — run: citadel-cli auth login")
		}
		return err
	}

	if output != "" && output != "table" {
		return emitJSON(cmd, resp)
	}

	if resp.Binary {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Binary file (%d bytes), SHA %s\n", resp.Size, resp.SHA)
		return nil
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), resp.Content)
	return nil
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	repoBrowseCmd.AddCommand(repoBrowseTreeCmd)
	repoBrowseCmd.AddCommand(repoBrowseBlobCmd)

	addOutputFlag(repoBrowseTreeCmd, repoBrowseBlobCmd)
	addRepoFlag(repoBrowseTreeCmd, repoBrowseBlobCmd)

	repoBrowseTreeCmd.Flags().String("ref", "", "Branch, tag, or commit SHA (default: repo default branch)")
	repoBrowseTreeCmd.Flags().String("path", "", "Directory path to list (default: repo root)")

	repoBrowseBlobCmd.Flags().String("ref", "", "Branch, tag, or commit SHA (default: repo default branch)")
	repoBrowseBlobCmd.Flags().String("path", "", "File path to read (required)")
}
