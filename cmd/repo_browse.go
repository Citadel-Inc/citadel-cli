package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

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

var repoBrowseRawCmd = &cobra.Command{
	Use:   "raw [<namespace>/<repo>] <path>",
	Short: "Stream a file's raw content from a repository",
	Long: `Stream a repository file's raw bytes to stdout or an output file.

Use --ref to target a specific branch, tag, or commit SHA.
Binary files are refused on a terminal unless --output-file is used.`,
	Example: `  # Stream a file from the repo selected by -R
  citadel-cli repo browse raw README.md -R acme/myrepo

  # Stream a file from a specific branch
  citadel-cli repo browse raw acme/myrepo src/main.go --ref feature/x

  # Save binary content to a file
  citadel-cli repo browse raw acme/myrepo image.png --output-file image.png`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRepoBrowseRaw,
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
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
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

	switch output {
	case "json":
		return emitJSON(cmd, resp)
	case "yaml":
		return emitYAML(cmd, resp)
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
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
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

	switch output {
	case "json":
		return emitJSON(cmd, resp)
	case "yaml":
		return emitYAML(cmd, resp)
	}

	if resp.Binary {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Binary file (%d bytes), SHA %s\n", resp.Size, resp.SHA)
		return nil
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), resp.Content)
	return nil
}

func runRepoBrowseRaw(cmd *cobra.Command, args []string) error {
	posArg := ""
	path := args[0]
	if len(args) == 2 {
		posArg = args[0]
		path = args[1]
	}
	if path == "" {
		return fmt.Errorf("file path required")
	}

	ns, slug, err := resolveRepoFromPosOrFlag(cmd, posArg)
	if err != nil {
		return err
	}
	ref, _ := cmd.Flags().GetString("ref")
	outputFile, _ := cmd.Flags().GetString("output-file")

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("path", path)
	if ref != "" {
		q.Set("ref", ref)
	}
	apiPath := fmt.Sprintf("/api/namespaces/%s/repos/%s/raw?%s", ns, slug, q.Encode())
	resp, err := client.GetStream(cmd.Context(), apiPath)
	if err != nil {
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
	defer func() { _ = resp.Body.Close() }()

	var dst = cmd.OutOrStdout()
	var file *os.File
	if outputFile != "" && outputFile != "-" {
		file, err = os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		dst = file
	}

	if file == nil && downloadOutputIsTTY(dst) {
		prefix := make([]byte, 512)
		n, readErr := io.ReadFull(resp.Body, prefix)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read repository file: %w", readErr)
		}
		if downloadLooksBinary(resp.Header.Get("Content-Type"), prefix[:n]) {
			return fmt.Errorf("refusing to write binary repository file to a terminal; redirect stdout or pass --output-file")
		}
		_, err = io.Copy(dst, io.MultiReader(bytes.NewReader(prefix[:n]), resp.Body))
	} else {
		_, err = io.Copy(dst, resp.Body)
	}
	if err != nil {
		return fmt.Errorf("write repository file: %w", err)
	}
	return nil
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	repoBrowseCmd.AddCommand(repoBrowseTreeCmd)
	repoBrowseCmd.AddCommand(repoBrowseBlobCmd)
	repoBrowseCmd.AddCommand(repoBrowseRawCmd)

	addOutputFlag(repoBrowseTreeCmd, repoBrowseBlobCmd)
	addRepoFlag(repoBrowseTreeCmd, repoBrowseBlobCmd)

	repoBrowseTreeCmd.Flags().String("ref", "", "Branch, tag, or commit SHA (default: repo default branch)")
	repoBrowseTreeCmd.Flags().String("path", "", "Directory path to list (default: repo root)")

	repoBrowseBlobCmd.Flags().String("ref", "", "Branch, tag, or commit SHA (default: repo default branch)")
	repoBrowseBlobCmd.Flags().String("path", "", "File path to read (required)")

	addRepoFlag(repoBrowseRawCmd)
	repoBrowseRawCmd.Flags().String("ref", "", "Branch, tag, or commit SHA (default: repo default branch)")
	repoBrowseRawCmd.Flags().StringP("output-file", "o", "", "Write raw content to this file instead of stdout")
}
