package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// PrCmd is the exported root for `citadel-cli pr`.
var PrCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	Long: `Create, view, and manage pull requests across Citadel namespace paths.

Subcommands:
  list       List pull requests
  view       View a pull request
  create     Open a pull request
  close      Close a pull request (without merging)
  merge      Merge a pull request
  diff       Show changed files (--file <path> for unified text)
  check      Check mergeability
  comment    Manage PR comments
  reviewer   Manage PR reviewers
  review     Submit a review

Target namespace via -R <org/repo> (or CITADEL_REPO env, or CWD git origin).`,
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests for a namespace",
	Long: `Lists pull requests for the target namespace. Defaults to --state open.

Examples:
  citadel-cli pr list -R myorg/myrepo
  citadel-cli pr list -R myorg/myrepo --state all --output json`,
	RunE: runPRList,
}

var prViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View a pull request",
	Long: `Shows full details for a pull request, including reviewers.

Examples:
  citadel-cli pr view -R myorg/myrepo 42
  citadel-cli pr view -R myorg/myrepo 42 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runPRView,
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open a new pull request",
	Long: `Creates a pull request from --source into --target.

--body reads from the flag, stdin (when not a TTY), or launches $EDITOR.

Examples:
  citadel-cli pr create -R myorg/myrepo --title "My change" --source feature --target main
  citadel-cli pr create -R myorg/myrepo --title "Fix" --source fix/123 --target main --body "Fixes #123"`,
	RunE: runPRCreate,
}

var prCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close a pull request without merging",
	Long: `Closes a pull request (sets state to closed). Does not merge.

Examples:
  citadel-cli pr close -R myorg/myrepo 42
  citadel-cli pr close -R myorg/myrepo 42 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runPRClose,
}

var prMergeCmd = &cobra.Command{
	Use:   "merge <number>",
	Short: "Merge a pull request",
	Long: `Merges the source branch into the target branch. Surfaces actionable
errors for merge conflicts, missing approvals, and invalid state transitions.

Examples:
  citadel-cli pr merge -R myorg/myrepo 42`,
	Args: cobra.ExactArgs(1),
	RunE: runPRMerge,
}

// ── domain types ──────────────────────────────────────────────────────────────

type prRow struct {
	ID           string     `json:"id"`
	NamespaceID  string     `json:"namespace_id"`
	Number       int64      `json:"number"`
	Title        string     `json:"title"`
	BodyMarkdown string     `json:"body_markdown"`
	State        string     `json:"state"`
	SourceRef    string     `json:"source_ref"`
	TargetRef    string     `json:"target_ref"`
	HeadSHA      string     `json:"head_sha"`
	BaseSHA      string     `json:"base_sha"`
	MergeSHA     *string    `json:"merge_sha,omitempty"`
	AuthorID     string     `json:"author_id"`
	MergedBy     *string    `json:"merged_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

type prReviewer struct {
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type prComment struct {
	ID            string    `json:"id"`
	PRID          string    `json:"pr_id"`
	AuthorID      string    `json:"author_id"`
	BodyMarkdown  string    `json:"body_markdown"`
	ThreadID      *string   `json:"thread_id,omitempty"`
	DiffCommitSHA *string   `json:"diff_commit_sha,omitempty"`
	DiffFile      *string   `json:"diff_file,omitempty"`
	DiffLine      *int      `json:"diff_line,omitempty"`
	DiffSide      *string   `json:"diff_side,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type prListRow struct {
	Number    int64     `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	SourceRef string    `json:"source_ref"`
	TargetRef string    `json:"target_ref"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r prListRow) CSVHeader() []string {
	return []string{"number", "title", "state", "source_ref", "target_ref", "author_id", "created_at", "updated_at"}
}

func (r prListRow) CSVRecord() []string {
	return []string{
		strconv.FormatInt(r.Number, 10), r.Title, r.State,
		r.SourceRef, r.TargetRef, r.AuthorID,
		formatRFC3339UTC(r.CreatedAt), formatRFC3339UTC(r.UpdatedAt),
	}
}

type prDiffFileStat struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type prDiffResult struct {
	Files   []prDiffFileStat `json:"files"`
	BaseRef string           `json:"base_ref"`
	HeadRef string           `json:"head_ref"`
	BaseSHA string           `json:"base_sha"`
	HeadSHA string           `json:"head_sha"`
}

type prFileDiffResult struct {
	Path      string `json:"path"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Unified   string `json:"unified"`
}

type prMergeabilityResult struct {
	Mergeable bool   `json:"mergeable"`
	Reason    string `json:"reason,omitempty"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func prBasePath(nsPath string) string {
	return "/namespaces/" + url.PathEscape(nsPath) + "/pulls"
}

func parsePRNumber(arg string) (int64, error) {
	num, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	if err != nil || num < 1 {
		return 0, fmt.Errorf("PR number must be a positive integer")
	}
	return num, nil
}

func prFriendlyError(err error) error {
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return err
	}
	var body struct {
		Error string `json:"error"`
	}
	if decErr := json.Unmarshal([]byte(he.Body), &body); decErr != nil || body.Error == "" {
		return err
	}
	switch body.Error {
	case "already_merged":
		return fmt.Errorf("PR is already merged")
	case "merge_conflict":
		return fmt.Errorf("merge conflict detected — resolve conflicts in the source branch before merging")
	case "approval_required":
		return fmt.Errorf("required reviewer approval missing — get approval before merging")
	case "invalid_state":
		return fmt.Errorf("PR is not in a state that allows this action (check current state with 'pr view')")
	case "missing_required_fields":
		return fmt.Errorf("title, --source, and --target are required")
	case "invalid_refs":
		return fmt.Errorf("one or both refs could not be resolved in the repository")
	case "invalid_inline_anchor":
		return fmt.Errorf("--diff-file and --diff-line must be supplied together")
	case "thread_not_found":
		return fmt.Errorf("thread not found on this PR")
	case "invalid_diff_side":
		return fmt.Errorf("--diff-side must be \"left\" or \"right\"")
	}
	return err
}

func prToListRow(pr prRow) prListRow {
	return prListRow{
		Number:    pr.Number,
		Title:     pr.Title,
		State:     pr.State,
		SourceRef: pr.SourceRef,
		TargetRef: pr.TargetRef,
		AuthorID:  pr.AuthorID,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
	}
}

// ── runners ───────────────────────────────────────────────────────────────────

func runPRList(cmd *cobra.Command, _ []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
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
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson or omit --all")
	}
	state, _ := cmd.Flags().GetString("state")
	state = strings.TrimSpace(strings.ToLower(state))

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var yamlAccum []prListRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if state != "" {
			q.Set("state", state)
		}
		var page struct {
			PullRequests []prRow `json:"pull_requests"`
			NextCursor   string  `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), prBasePath(nsPath)+"?"+q.Encode(), &page); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("namespace '%s' not found", nsPath)
			}
			return err
		}
		rows := page.PullRequests
		next := strings.TrimSpace(page.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []prListRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[prListRow](cmd)
			case "yaml":
				return emitYAML(cmd, []prListRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No pull requests found for '%s'.\n", nsPath)
				return nil
			}
		}
		first = false

		listRows := make([]prListRow, len(rows))
		for i, pr := range rows {
			listRows[i] = prToListRow(pr)
		}

		switch output {
		case "json":
			return emitJSON(cmd, listRows)
		case "ndjson":
			if err := emitNDJSONLines(cmd, listRows); err != nil {
				return err
			}
		case "csv":
			if err := emitCSVRows(cmd, &csvHdr, listRows); err != nil {
				return err
			}
		case "yaml":
			if all {
				yamlAccum = append(yamlAccum, listRows...)
			} else {
				return emitYAML(cmd, listRows)
			}
		default:
			w := newTabWriter(cmd)
			_, _ = fmt.Fprintln(w, "NUMBER\tTITLE\tSTATE\tSOURCE\tTARGET\tAUTHOR\tCREATED")
			for _, r := range listRows {
				title := r.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				_, _ = fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
					r.Number, title, r.State, r.SourceRef, r.TargetRef,
					r.AuthorID[:min(8, len(r.AuthorID))]+"…",
					r.CreatedAt.Format("2006-01-02"))
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if !all && next != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(use --cursor %s for more, or --all to fetch everything)\n", next)
			}
		}

		if !all || next == "" {
			break
		}
		cursor = next
	}

	if all && output == "yaml" {
		if yamlAccum == nil {
			yamlAccum = []prListRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runPRView(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}

	var payload struct {
		PR        prRow        `json:"pull_request"`
		Reviewers []prReviewer `json:"reviewers"`
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10)
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return err
	}

	type viewPayload struct {
		PR        prRow        `json:"pull_request"`
		Reviewers []prReviewer `json:"reviewers"`
	}
	vp := viewPayload{PR: payload.PR, Reviewers: payload.Reviewers}

	switch output {
	case "json":
		return emitJSON(cmd, vp)
	case "yaml":
		return emitYAML(cmd, vp)
	}

	pr := payload.PR
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "NUMBER\t#%d\n", pr.Number)
	_, _ = fmt.Fprintf(w, "STATE\t%s\n", pr.State)
	_, _ = fmt.Fprintf(w, "TITLE\t%s\n", pr.Title)
	_, _ = fmt.Fprintf(w, "SOURCE\t%s\n", pr.SourceRef)
	_, _ = fmt.Fprintf(w, "TARGET\t%s\n", pr.TargetRef)
	_, _ = fmt.Fprintf(w, "AUTHOR\t%s\n", pr.AuthorID)
	_, _ = fmt.Fprintf(w, "CREATED\t%s\n", formatRFC3339UTC(pr.CreatedAt))
	_, _ = fmt.Fprintf(w, "UPDATED\t%s\n", formatRFC3339UTC(pr.UpdatedAt))
	if pr.MergedAt != nil {
		_, _ = fmt.Fprintf(w, "MERGED\t%s\n", formatRFC3339UTC(*pr.MergedAt))
	}
	if pr.ClosedAt != nil {
		_, _ = fmt.Fprintf(w, "CLOSED\t%s\n", formatRFC3339UTC(*pr.ClosedAt))
	}
	if len(payload.Reviewers) > 0 {
		parts := make([]string, 0, len(payload.Reviewers))
		for _, rv := range payload.Reviewers {
			parts = append(parts, rv.UserID[:min(8, len(rv.UserID))]+":"+rv.Status)
		}
		_, _ = fmt.Fprintf(w, "REVIEWERS\t%s\n", strings.Join(parts, ", "))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if body := strings.TrimSpace(pr.BodyMarkdown); body != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), body)
	}
	return nil
}

func runPRCreate(cmd *cobra.Command, _ []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	title, _ := cmd.Flags().GetString("title")
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("--title is required")
	}
	source, _ := cmd.Flags().GetString("source")
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("--source ref is required")
	}
	target, _ := cmd.Flags().GetString("target")
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("--target ref is required")
	}
	body, err := readIssueBody(cmd, "body")
	if err != nil {
		return err
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"title":         title,
		"body_markdown": body,
		"source_ref":    source,
		"target_ref":    target,
	}
	var created prRow
	if err := c.Post(cmd.Context(), prBasePath(nsPath), payload, &created); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("namespace '%s' not found", nsPath)
		}
		return prFriendlyError(err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pull request #%d created in %s: %s\n", created.Number, nsPath, created.Title)
	return nil
}

func runPRClose(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	if err := confirmSlug(yesFlag(cmd), "PR close", fmt.Sprintf("%s#%d", nsPath, num)); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10)
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return prFriendlyError(err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PR %s#%d closed.\n", nsPath, num)
	return nil
}

func runPRMerge(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/merge"
	var merged prRow
	if err := c.Post(cmd.Context(), path, nil, &merged); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return prFriendlyError(err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PR %s#%d merged.\n", nsPath, num)
	return nil
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	PrCmd.AddCommand(prListCmd)
	PrCmd.AddCommand(prViewCmd)
	PrCmd.AddCommand(prCreateCmd)
	PrCmd.AddCommand(prCloseCmd)
	PrCmd.AddCommand(prMergeCmd)

	addIssuePathFlag(prListCmd, prViewCmd, prCreateCmd, prCloseCmd, prMergeCmd)
	addPaginationFlags(prListCmd)
	addOutputFlag(prListCmd, prViewCmd)
	addYesFlag(prCloseCmd)

	prListCmd.Flags().String("state", "open", "Filter by state: open, closed, merged, or all")

	prCreateCmd.Flags().String("title", "", "PR title (required)")
	prCreateCmd.Flags().String("body", "", "PR body markdown (reads stdin or $EDITOR when omitted)")
	prCreateCmd.Flags().String("source", "", "Source branch or ref (required)")
	prCreateCmd.Flags().String("target", "", "Target branch or ref (required)")
	_ = prCreateCmd.MarkFlagRequired("title")
	_ = prCreateCmd.MarkFlagRequired("source")
	_ = prCreateCmd.MarkFlagRequired("target")

	prListCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prViewCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prCreateCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prCloseCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prMergeCmd.ValidArgsFunction = completeOrgNamespaceSlugs
}
