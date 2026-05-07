package cmd

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── domain types ─────────────────────────────────────────────────────────────

type commitItem struct {
	SHA            string    `json:"sha"`
	Message        string    `json:"message"`
	Author         string    `json:"author"`
	AuthorEmail    string    `json:"author_email"`
	Committer      string    `json:"committer"`
	CommitterEmail string    `json:"committer_email"`
	Timestamp      time.Time `json:"timestamp"`
}

func (c commitItem) subject() string {
	if idx := strings.IndexByte(c.Message, '\n'); idx >= 0 {
		return strings.TrimSpace(c.Message[:idx])
	}
	return strings.TrimSpace(c.Message)
}

type commitFileStat struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type commitSignature struct {
	Present  bool   `json:"present"`
	Kind     string `json:"kind,omitempty"`
	Verified bool   `json:"verified"`
}

type commitDetail struct {
	commitItem
	Parents   []string         `json:"parents"`
	Files     []commitFileStat `json:"files"`
	Signature commitSignature  `json:"signature"`
}

// ── command tree ─────────────────────────────────────────────────────────────

var repoCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Browse commits in a repository",
}

var repoCommitListCmd = &cobra.Command{
	Use:               "list [<namespace>/<repo>]",
	Short:             "List commits in a repository",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoCommitList,
	ValidArgsFunction: completeRepoSlugs,
}

var repoCommitGetCmd = &cobra.Command{
	Use:   "get [<namespace>/<repo>] <sha>",
	Short: "Get details of a single commit",
	Long: `Fetches metadata, parent SHAs, per-file change stats, and signature presence for a commit.

Pass --path to print the per-file unified diff for a specific file in that commit.

Examples:
  citadel-cli repo commit get myorg/myrepo abc123
  citadel-cli repo commit get abc123 --path src/main.go
  citadel-cli repo commit get abc123 --output json`,
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoCommitGet,
	ValidArgsFunction: completeRepoSlugs,
}

// ── list ─────────────────────────────────────────────────────────────────────

func runRepoCommitList(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	limit, cursor, all, err := readPagination(cmd)
	if err != nil {
		return err
	}
	if all && output == "json" {
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	ref, _ := cmd.Flags().GetString("ref")
	pathFilter, _ := cmd.Flags().GetString("path")

	var yamlAccum []commitItem
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("after", cursor)
		}
		if ref != "" {
			q.Set("ref", ref)
		}
		if pathFilter != "" {
			q.Set("path", pathFilter)
		}
		var payload struct {
			Commits    []commitItem `json:"commits"`
			NextCursor string       `json:"next_cursor"`
			Ref        string       `json:"ref"`
		}
		apiPath := "/api/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/commits?" + q.Encode()
		if err := c.Get(cmd.Context(), apiPath, &payload); err != nil {
			return err
		}
		rows := payload.Commits
		next := strings.TrimSpace(payload.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []commitItem{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[commitItem](cmd)
			case "yaml":
				return emitYAML(cmd, []commitItem{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No commits in repository '%s/%s'.\n", ns, slug)
				return nil
			}
		}
		first = false

		switch output {
		case "json":
			return emitJSON(cmd, rows)
		case "ndjson":
			if err := emitNDJSONLines(cmd, rows); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, rows); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, rows...)
			} else {
				return emitYAML(cmd, rows)
			}
		default:
			w := newTabWriter(cmd)
			_, _ = fmt.Fprintln(w, "SHA\tAUTHOR\tDATE\tSUBJECT")
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					shortSHA(row.SHA),
					row.Author,
					formatCommitDate(row.Timestamp),
					truncateSubject(row.subject(), 72),
				)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		if !all {
			if isHumanListOutput(output) && next != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(use --cursor "+next+" for more, or --all to fetch everything)")
			}
			return nil
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if all && output == "yaml" {
		if yamlAccum == nil {
			yamlAccum = []commitItem{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

// ── get ───────────────────────────────────────────────────────────────────────

func runRepoCommitGet(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	// args is 1 or 2: optional <ns>/<repo> then <sha>
	var ns, slug, sha string
	switch len(args) {
	case 1:
		var rerr error
		ns, slug, rerr = resolveRepoFromPosOrFlag(cmd, "")
		if rerr != nil {
			return rerr
		}
		sha = strings.TrimSpace(args[0])
	case 2:
		var rerr error
		ns, slug, rerr = resolveRepoFromPosOrFlag(cmd, args[0])
		if rerr != nil {
			return rerr
		}
		sha = strings.TrimSpace(args[1])
	}

	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	filePath, _ := cmd.Flags().GetString("path")

	// If --path is given, fetch the per-file unified diff
	if filePath != "" {
		return runRepoCommitDiff(cmd, c, ns, slug, sha, filePath, output)
	}

	var payload struct {
		Commit commitDetail `json:"commit"`
	}
	apiPath := "/api/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/commits/" + url.PathEscape(sha)
	if err := c.Get(cmd.Context(), apiPath, &payload); err != nil {
		return err
	}
	detail := payload.Commit

	switch output {
	case "json":
		return emitJSON(cmd, detail)
	case "yaml":
		return emitYAML(cmd, detail)
	default:
		return renderCommitDetail(cmd.OutOrStdout(), detail)
	}
}

func runRepoCommitDiff(cmd *cobra.Command, c *apiclient.Client, ns, slug, sha, filePath, output string) error {
	var payload struct {
		Unified       string `json:"unified"`
		Truncated     bool   `json:"truncated"`
		Binary        bool   `json:"binary"`
		InitialCommit bool   `json:"initial_commit"`
	}
	apiPath := "/api/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) +
		"/commits/" + url.PathEscape(sha) + "/diff?path=" + url.QueryEscape(filePath)
	if err := c.Get(cmd.Context(), apiPath, &payload); err != nil {
		return err
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	default:
		if payload.Binary {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(binary file %s)\n", filePath)
			return nil
		}
		if payload.InitialCommit {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(initial commit — no parent to diff)\n")
			return nil
		}
		_, _ = io.WriteString(cmd.OutOrStdout(), payload.Unified)
		if payload.Truncated {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n... (diff truncated by server)")
		}
		return nil
	}
}

// ── render helpers ────────────────────────────────────────────────────────────

func renderCommitDetail(out io.Writer, d commitDetail) error {
	_, _ = fmt.Fprintf(out, "commit  %s\n", d.SHA)
	if d.Author != "" {
		_, _ = fmt.Fprintf(out, "Author: %s <%s>\n", d.Author, d.AuthorEmail)
	}
	if d.Committer != "" && d.Committer != d.Author {
		_, _ = fmt.Fprintf(out, "Commit: %s <%s>\n", d.Committer, d.CommitterEmail)
	}
	if !d.Timestamp.IsZero() {
		_, _ = fmt.Fprintf(out, "Date:   %s\n", d.Timestamp.UTC().Format(time.RFC3339))
	}
	if len(d.Parents) > 0 {
		_, _ = fmt.Fprintf(out, "Parent: %s\n", strings.Join(d.Parents, " "))
	}
	if d.Signature.Present {
		kind := d.Signature.Kind
		if kind == "" {
			kind = "unknown"
		}
		verified := "unverified"
		if d.Signature.Verified {
			verified = "verified"
		}
		_, _ = fmt.Fprintf(out, "GPG:    %s (%s)\n", kind, verified)
	}
	_, _ = fmt.Fprintln(out)
	for _, line := range strings.Split(strings.TrimRight(d.Message, "\n"), "\n") {
		_, _ = fmt.Fprintf(out, "    %s\n", line)
	}
	if len(d.Files) > 0 {
		_, _ = fmt.Fprintln(out)
		for _, f := range d.Files {
			path := f.Path
			if f.OldPath != "" {
				path = f.OldPath + " -> " + f.Path
			}
			_, _ = fmt.Fprintf(out, " %s  +%d -%d  %s\n", padStatus(f.Status), f.Additions, f.Deletions, path)
		}
	}
	return nil
}

func padStatus(s string) string {
	switch s {
	case "added":
		return "A"
	case "modified":
		return "M"
	case "deleted":
		return "D"
	case "renamed":
		return "R"
	default:
		return "?"
	}
}

func formatCommitDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func truncateSubject(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	repoCommitCmd.AddCommand(repoCommitListCmd)
	repoCommitCmd.AddCommand(repoCommitGetCmd)

	addOutputFlag(repoCommitListCmd, repoCommitGetCmd)
	addPaginationFlags(repoCommitListCmd)
	addRepoFlag(repoCommitListCmd, repoCommitGetCmd)

	repoCommitListCmd.Flags().String("ref", "", "Branch or tag to list commits from (default: repo default branch)")
	repoCommitListCmd.Flags().String("path", "", "Filter commits that touch this file path")
	repoCommitGetCmd.Flags().String("path", "", "Print unified diff for this file in the commit")
}
