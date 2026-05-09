package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── commands ──────────────────────────────────────────────────────────────────

var prDiffCmd = &cobra.Command{
	Use:   "diff <number>",
	Short: "Show changed files for a pull request",
	Long: `Without --file: prints a stat table (files changed, additions, deletions).
With --file <path>: prints the raw unified diff for that file.

Examples:
  citadel-cli pr diff 42 -R myorg/myrepo
  citadel-cli pr diff 42 -R myorg/myrepo --file src/main.go`,
	Args: cobra.ExactArgs(1),
	RunE: runPRDiff,
}

var prCheckCmd = &cobra.Command{
	Use:   "check <number>",
	Short: "Check mergeability of a pull request",
	Long: `Reports whether the source branch can be cleanly merged into the target.
Reason values: fast_forward, clean, conflict, no_merge_base, resolve_error.

Examples:
  citadel-cli pr check 42 -R myorg/myrepo
  citadel-cli pr check 42 -R myorg/myrepo --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runPRCheck,
}

var prCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage PR comments",
}

var prCommentListCmd = &cobra.Command{
	Use:   "list <number>",
	Short: "List comments on a pull request",
	Args:  cobra.ExactArgs(1),
	RunE:  runPRCommentList,
}

var prCommentAddCmd = &cobra.Command{
	Use:   "add <number>",
	Short: "Add a comment to a pull request",
	Long: `Adds a general (non-diff) comment to the pull request.

Examples:
  citadel-cli pr comment add 42 -R myorg/myrepo --body "Looks good!"
  echo "LGTM" | citadel-cli pr comment add 42 -R myorg/myrepo`,
	Args: cobra.ExactArgs(1),
	RunE: runPRCommentAdd,
}

var prReviewerCmd = &cobra.Command{
	Use:   "reviewer",
	Short: "Manage PR reviewers",
}

var prReviewerListCmd = &cobra.Command{
	Use:   "list <number>",
	Short: "List reviewers for a pull request",
	Args:  cobra.ExactArgs(1),
	RunE:  runPRReviewerList,
}

var prReviewerAddCmd = &cobra.Command{
	Use:   "add <number>",
	Short: "Add a reviewer to a pull request",
	Long: `Adds a reviewer by user UUID. The user must have pull_requests:read on the namespace.

Examples:
  citadel-cli pr reviewer add 42 -R myorg/myrepo --reviewer <user-uuid>`,
	Args: cobra.ExactArgs(1),
	RunE: runPRReviewerAdd,
}

var prReviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Submit a review on a pull request",
	Long: `Submit a review: approve, request changes, or add a comment.
At least one of --approve, --request-changes, or --comment must be given.

  --approve            Set review status to approved
  --request-changes    Set review status to changes_requested
  --comment <body>     Add a general comment (can be combined with --approve or --request-changes)

Examples:
  citadel-cli pr review 42 -R myorg/myrepo --approve
  citadel-cli pr review 42 -R myorg/myrepo --request-changes --comment "Please fix the tests"
  citadel-cli pr review 42 -R myorg/myrepo --comment "Looking good so far"`,
	Args: cobra.ExactArgs(1),
	RunE: runPRReview,
}

// ── runners ───────────────────────────────────────────────────────────────────

func runPRDiff(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	filePath, _ := cmd.Flags().GetString("file")
	filePath = strings.TrimSpace(filePath)

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	base := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10)

	if filePath != "" {
		// File-scoped unified diff.
		q := url.Values{}
		q.Set("path", filePath)
		var result prFileDiffResult
		if err := c.Get(cmd.Context(), base+"/diff/file?"+q.Encode(), &result); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("PR %s#%d or file '%s' not found", nsPath, num, filePath)
			}
			return err
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(), result.Unified)
		return nil
	}

	// Whole-PR stat table.
	var result prDiffResult
	if err := c.Get(cmd.Context(), base+"/diff", &result); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return err
	}
	if len(result.Files) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No changes.")
		return nil
	}
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintln(w, "FILE\tSTATUS\t+\t-")
	for _, f := range result.Files {
		path := f.Path
		if f.OldPath != "" && f.OldPath != f.Path {
			path = f.OldPath + " → " + f.Path
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t+%d\t-%d\n", path, f.Status, f.Additions, f.Deletions)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s..%s  (use --file <path> for unified diff)\n",
		result.BaseSHA[:min(8, len(result.BaseSHA))],
		result.HeadSHA[:min(8, len(result.HeadSHA))])
	return nil
}

func runPRCheck(cmd *cobra.Command, args []string) error {
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

	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/mergeability"
	var result prMergeabilityResult
	if err := c.Get(cmd.Context(), path, &result); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return err
	}

	switch output {
	case "json":
		return emitJSON(cmd, result)
	case "yaml":
		return emitYAML(cmd, result)
	}

	if result.Mergeable {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "MERGEABLE  reason: %s\n", result.Reason)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "NOT MERGEABLE  reason: %s\n", result.Reason)
	}
	return nil
}

func runPRCommentList(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	onlyInline, _ := cmd.Flags().GetBool("inline")
	onlyGeneral, _ := cmd.Flags().GetBool("general")
	if onlyInline && onlyGeneral {
		return fmt.Errorf("--inline and --general are mutually exclusive")
	}

	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/comments"
	var payload struct {
		Comments []prComment `json:"comments"`
	}
	if err := client.Get(cmd.Context(), path, &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return err
	}

	comments := payload.Comments
	if onlyInline || onlyGeneral {
		var filtered []prComment
		for _, c := range comments {
			isInline := c.DiffFile != nil
			if (onlyInline && isInline) || (onlyGeneral && !isInline) {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	}

	if len(comments) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No comments on PR %s#%d.\n", nsPath, num)
		return nil
	}

	// Separate standalone comments from threaded comments.
	var standalone []prComment
	threads := make(map[string][]prComment)
	var threadOrder []string
	for _, c := range comments {
		if c.ThreadID != nil {
			tid := *c.ThreadID
			if _, exists := threads[tid]; !exists {
				threadOrder = append(threadOrder, tid)
			}
			threads[tid] = append(threads[tid], c)
		} else {
			standalone = append(standalone, c)
		}
	}

	for _, c := range standalone {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "— %s (%s)\n", c.AuthorID, formatRFC3339UTC(c.CreatedAt))
		if body := strings.TrimSpace(c.BodyMarkdown); body != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), body)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	for _, tid := range threadOrder {
		tComments := threads[tid]
		first := tComments[0]
		header := "▸ Thread " + tid[:min(8, len(tid))]
		if first.DiffFile != nil {
			header += " — " + *first.DiffFile
			if first.DiffLine != nil {
				header += ":" + strconv.Itoa(*first.DiffLine)
			}
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), header)
		for _, c := range tComments {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  — %s (%s)\n", c.AuthorID, formatRFC3339UTC(c.CreatedAt))
			if body := strings.TrimSpace(c.BodyMarkdown); body != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  "+body)
			}
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func runPRCommentAdd(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	body, err := readIssueBody(cmd, "body")
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body required: pass --body, pipe markdown on stdin, or set $EDITOR")
	}

	diffFile, _ := cmd.Flags().GetString("diff-file")
	diffLine, _ := cmd.Flags().GetInt("diff-line")
	diffSide, _ := cmd.Flags().GetString("diff-side")
	diffSHA, _ := cmd.Flags().GetString("diff-sha")
	threadID, _ := cmd.Flags().GetString("thread-id")

	diffFile = strings.TrimSpace(diffFile)
	diffSHA = strings.TrimSpace(diffSHA)
	threadID = strings.TrimSpace(threadID)

	if (diffFile == "") != (diffLine == 0) {
		return fmt.Errorf("--diff-file and --diff-line must be supplied together")
	}
	if diffFile != "" {
		if diffSide != "left" && diffSide != "right" {
			return fmt.Errorf("--diff-side must be \"left\" or \"right\"")
		}
	} else if cmd.Flags().Changed("diff-side") {
		return fmt.Errorf("--diff-side requires --diff-file")
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/comments"

	reqBody := map[string]any{"body_markdown": body}
	if diffFile != "" {
		reqBody["diff_file"] = diffFile
		reqBody["diff_line"] = diffLine
		reqBody["diff_side"] = diffSide
	}
	if diffSHA != "" {
		reqBody["diff_commit_sha"] = diffSHA
	}
	if threadID != "" {
		reqBody["thread_id"] = threadID
	}

	var created prComment
	if err := c.Post(cmd.Context(), path, reqBody, &created); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		var he *apiclient.HTTPError
		if errors.As(err, &he) {
			var errBody struct {
				Error string `json:"error"`
			}
			if decErr := json.Unmarshal([]byte(he.Body), &errBody); decErr == nil {
				switch errBody.Error {
				case "invalid_inline_anchor":
					return fmt.Errorf("--diff-file and --diff-line must be supplied together")
				case "thread_not_found":
					return fmt.Errorf("thread %q not found on this PR", threadID)
				case "invalid_diff_side":
					return fmt.Errorf("--diff-side must be \"left\" or \"right\"")
				}
			}
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added comment %s to %s#%d.\n", created.ID, nsPath, num)
	return nil
}

func runPRReviewerList(cmd *cobra.Command, args []string) error {
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

	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/reviewers"
	var payload struct {
		Reviewers []prReviewer `json:"reviewers"`
	}
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		return err
	}
	if len(payload.Reviewers) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No reviewers on PR %s#%d.\n", nsPath, num)
		return nil
	}
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintln(w, "USER_ID\tSTATUS\tUPDATED")
	for _, rv := range payload.Reviewers {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			rv.UserID, rv.Status, formatRFC3339UTC(rv.UpdatedAt))
	}
	return w.Flush()
}

func runPRReviewerAdd(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	reviewerID, _ := cmd.Flags().GetString("reviewer")
	reviewerID = strings.TrimSpace(reviewerID)
	if reviewerID == "" {
		return fmt.Errorf("--reviewer <user-uuid> is required")
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/reviewers"
	if err := c.Post(cmd.Context(), path, map[string]string{"user_id": reviewerID}, nil); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("PR %s#%d not found", nsPath, num)
		}
		var he *apiclient.HTTPError
		if errors.As(err, &he) {
			var body struct {
				Error string `json:"error"`
			}
			if decErr := json.Unmarshal([]byte(he.Body), &body); decErr == nil {
				switch body.Error {
				case "invalid_user_id":
					return fmt.Errorf("invalid reviewer UUID %q — pass a valid user UUID", reviewerID)
				}
			}
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added reviewer %s to PR %s#%d.\n", reviewerID, nsPath, num)
	return nil
}

func runPRReview(cmd *cobra.Command, args []string) error {
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return err
	}
	approve, _ := cmd.Flags().GetBool("approve")
	requestChanges, _ := cmd.Flags().GetBool("request-changes")
	comment, _ := cmd.Flags().GetString("comment")
	comment = strings.TrimSpace(comment)

	if !approve && !requestChanges && comment == "" {
		return fmt.Errorf("at least one of --approve, --request-changes, or --comment <body> is required")
	}
	if approve && requestChanges {
		return fmt.Errorf("--approve and --request-changes are mutually exclusive")
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	base := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10)

	// Post comment if given.
	if comment != "" {
		commentPath := base + "/comments"
		if err := c.Post(cmd.Context(), commentPath, map[string]string{"body_markdown": comment}, nil); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("PR %s#%d not found", nsPath, num)
			}
			return err
		}
	}

	// Submit review status if approve or request-changes given.
	if approve || requestChanges {
		status := "approved"
		if requestChanges {
			status = "changes_requested"
		}
		reviewPath := base + "/reviews/me"
		if err := c.Put(cmd.Context(), reviewPath, map[string]string{"status": status}, nil); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("PR %s#%d not found", nsPath, num)
			}
			return prFriendlyError(err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Review submitted for PR %s#%d: %s.\n", nsPath, num, status)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Comment added to PR %s#%d.\n", nsPath, num)
	}
	return nil
}

// ── completion ────────────────────────────────────────────────────────────────

func completePRDiffFiles(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	num, err := parsePRNumber(args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	path := prBasePath(nsPath) + "/" + strconv.FormatInt(num, 10) + "/diff"
	var result prDiffResult
	if err := c.Get(cmd.Context(), path, &result); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	paths := make([]string, 0, len(result.Files))
	for _, f := range result.Files {
		paths = append(paths, f.Path)
	}
	return paths, cobra.ShellCompDirectiveNoFileComp
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	PrCmd.AddCommand(prDiffCmd)
	PrCmd.AddCommand(prCheckCmd)
	PrCmd.AddCommand(prCommentCmd)
	PrCmd.AddCommand(prReviewerCmd)
	PrCmd.AddCommand(prReviewCmd)

	prCommentCmd.AddCommand(prCommentListCmd)
	prCommentCmd.AddCommand(prCommentAddCmd)

	prReviewerCmd.AddCommand(prReviewerListCmd)
	prReviewerCmd.AddCommand(prReviewerAddCmd)

	addIssuePathFlag(prDiffCmd, prCheckCmd,
		prCommentListCmd, prCommentAddCmd,
		prReviewerListCmd, prReviewerAddCmd,
		prReviewCmd)
	addOutputFlag(prCheckCmd)

	prDiffCmd.Flags().String("file", "", "Narrow diff to a single file path (emits raw unified text)")

	prCommentAddCmd.Flags().String("body", "", "Comment body markdown (reads stdin or $EDITOR when omitted)")
	prCommentAddCmd.Flags().String("diff-file", "", "File path for inline comment (requires --diff-line)")
	prCommentAddCmd.Flags().Int("diff-line", 0, "Hunk line number for inline comment (requires --diff-file)")
	prCommentAddCmd.Flags().String("diff-side", "right", "Diff side for inline comment: left or right (default right)")
	prCommentAddCmd.Flags().String("diff-sha", "", "Commit SHA to anchor inline comment")
	prCommentAddCmd.Flags().String("thread-id", "", "Thread UUID to reply to an existing thread")
	_ = prCommentAddCmd.RegisterFlagCompletionFunc("diff-file", completePRDiffFiles)

	prCommentListCmd.Flags().Bool("inline", false, "Show only inline/thread comments")
	prCommentListCmd.Flags().Bool("general", false, "Show only general (non-diff) comments")

	prReviewerAddCmd.Flags().String("reviewer", "", "User UUID to add as reviewer (required)")
	_ = prReviewerAddCmd.MarkFlagRequired("reviewer")

	prReviewCmd.Flags().Bool("approve", false, "Approve the pull request")
	prReviewCmd.Flags().Bool("request-changes", false, "Request changes on the pull request")
	prReviewCmd.Flags().String("comment", "", "Add a comment (can be combined with --approve or --request-changes)")

	prDiffCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prCheckCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prCommentListCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prCommentAddCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prReviewerListCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prReviewerAddCmd.ValidArgsFunction = completeOrgNamespaceSlugs
	prReviewCmd.ValidArgsFunction = completeOrgNamespaceSlugs
}
