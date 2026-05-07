package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

var IssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues across Citadel namespaces",
	Long: `Issue workflows target any Citadel namespace path via -R <ns/path>.

Examples:
  citadel-cli issue list -R acme/demo
  citadel-cli issue view -R acme/demo 42
  citadel-cli issue create -R acme/demo --title "Ship it" --body "..."`,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues for a namespace path",
	RunE:  runIssueList,
}

var issueViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "Show one issue by number",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueView,
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	RunE:  runIssueCreate,
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment <number>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueComment,
}

var issueCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueClose,
}

var issueReopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueReopen,
}

var issueLabelCmd = &cobra.Command{
	Use:   "label <number>",
	Short: "Add or remove labels on an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueLabel,
}

var issueCloseRefsCmd = &cobra.Command{
	Use:   "close-refs <number>",
	Short: "List close-ref state for one issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueCloseRefs,
}

type issueAssignee struct {
	UserID      string `json:"user_id"`
	Slug        string `json:"slug,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type issueLabel struct {
	ID           string `json:"id"`
	NamespaceID  string `json:"namespace_id"`
	Slug         string `json:"slug"`
	DisplayName  string `json:"display_name"`
	Color        string `json:"color"`
	Description  string `json:"description"`
	IsDefault    bool   `json:"is_default"`
	SemanticRole string `json:"semantic_role,omitempty"`
}

type issueComment struct {
	ID           string    `json:"id"`
	IssueID      string    `json:"issue_id"`
	AuthorID     string    `json:"author_id"`
	BodyMarkdown string    `json:"body_markdown"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	EditCount    int       `json:"edit_count"`
}

type issueRow struct {
	ID               string            `json:"id"`
	NamespaceID      string            `json:"namespace_id"`
	NamespacePath    string            `json:"namespace_path"`
	Number           int64             `json:"number"`
	Title            string            `json:"title"`
	BodyMarkdown     string            `json:"body_markdown"`
	BodyHTML         string            `json:"body_html,omitempty"`
	State            string            `json:"state"`
	AuthorID         string            `json:"author_id"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ClosedAt         *time.Time        `json:"closed_at,omitempty"`
	ClosedBy         *string           `json:"closed_by,omitempty"`
	MilestoneID      *string           `json:"milestone_id,omitempty"`
	MilestoneTitle   string            `json:"milestone_title,omitempty"`
	Assignees        []issueAssignee   `json:"assignees,omitempty"`
	Reactions        map[string]int    `json:"reactions,omitempty"`
	ActorReactions   []string          `json:"actor_reactions,omitempty"`
	ActorReactionIDs map[string]string `json:"actor_reaction_ids,omitempty"`
}

type issueDetailPayload struct {
	Issue    issueRow       `json:"issue"`
	Comments []issueComment `json:"comments"`
	Labels   []issueLabel   `json:"labels"`
}

type issueListRow struct {
	Number        int64      `json:"number"`
	NamespacePath string     `json:"namespace_path"`
	Title         string     `json:"title"`
	State         string     `json:"state"`
	AuthorID      string     `json:"author_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
}

type issueCloseRef struct {
	ReferencedNamespacePath string     `json:"referenced_namespace_path"`
	ClosingCommitSHA        *string    `json:"closing_commit_sha,omitempty"`
	ResolvedAt              *time.Time `json:"resolved_at,omitempty"`
}

func addIssuePathFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringP("repo", "R", "", "Issue namespace path (e.g. org/repo or org/team/project)")
	}
}

func normalizeNamespacePath(raw string) (string, error) {
	path := strings.Trim(strings.TrimSpace(raw), "/")
	if path == "" {
		return "", fmt.Errorf("namespace path required")
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("namespace path must not contain empty segments")
		}
	}
	return strings.Join(parts, "/"), nil
}

func resolveIssueNamespacePath(cmd *cobra.Command) (string, error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	if strings.TrimSpace(repoFlag) != "" {
		return normalizeNamespacePath(repoFlag)
	}
	if ev := strings.TrimSpace(os.Getenv(citadelRepoEnv)); ev != "" {
		return normalizeNamespacePath(ev)
	}

	path, wdErr := os.Getwd()
	if wdErr != nil {
		path = ""
	}
	rawURL, err := gitOriginURL(cmd.Context(), path)
	if err != nil {
		if err == exec.ErrNotFound {
			return "", fmt.Errorf("namespace path required: pass -R <ns/path> or set %s", citadelRepoEnv)
		}
		return "", fmt.Errorf("namespace path required: pass -R <ns/path>, set %s, or run from a git checkout with a Citadel origin", citadelRepoEnv)
	}
	ns, slug, err := parseOriginIntoRepo(rawURL, mergeCitadelHosts())
	if err != nil {
		return "", fmt.Errorf("namespace path required: pass -R <ns/path>, set %s, or run from a git checkout with a Citadel origin", citadelRepoEnv)
	}
	if inferenceHintWorthy(cmd) {
		_, _ = fmt.Fprintf(os.Stderr, "Inferred -R %s/%s from CWD\n", ns, slug)
	}
	return ns + "/" + slug, nil
}

func issueBasePath(nsPath string) string {
	return "/namespaces/" + url.PathEscape(nsPath) + "/issues"
}

func issueBrowserURL(server, nsPath string, number int64) string {
	parts := strings.Split(strings.Trim(nsPath, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.TrimRight(server, "/") + "/" + strings.Join(parts, "/") + "/issues/" + strconv.FormatInt(number, 10)
}

func parseIssueNumber(arg string) (int64, error) {
	num, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	if err != nil || num < 1 {
		return 0, fmt.Errorf("issue number must be a positive integer")
	}
	return num, nil
}

func normalizeStringSlice(raw []string) []string {
	var out []string
	for _, chunk := range raw {
		for _, part := range strings.Split(chunk, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func issueListRows(rows []issueRow) []issueListRow {
	out := make([]issueListRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, issueListRow{
			Number:        row.Number,
			NamespacePath: row.NamespacePath,
			Title:         row.Title,
			State:         row.State,
			AuthorID:      row.AuthorID,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			ClosedAt:      row.ClosedAt,
		})
	}
	return out
}

func assigneeSummary(xs []issueAssignee) string {
	if len(xs) == 0 {
		return "—"
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		switch {
		case strings.TrimSpace(x.Slug) != "":
			out = append(out, x.Slug)
		case strings.TrimSpace(x.DisplayName) != "":
			out = append(out, x.DisplayName)
		case strings.TrimSpace(x.UserID) != "":
			out = append(out, x.UserID)
		}
	}
	if len(out) == 0 {
		return "—"
	}
	return strings.Join(out, ",")
}

func labelSummary(xs []issueLabel) string {
	if len(xs) == 0 {
		return "—"
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if strings.TrimSpace(x.Slug) != "" {
			out = append(out, x.Slug)
		}
	}
	if len(out) == 0 {
		return "—"
	}
	return strings.Join(out, ",")
}

func optionalString(v *string) string {
	if v == nil {
		return "—"
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return "—"
	}
	return s
}

func optionalTime(v *time.Time) string {
	if v == nil {
		return "—"
	}
	return formatRFC3339PtrUTC(v)
}

func readIssueBody(cmd *cobra.Command, flagName string) (string, error) {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		v, _ := cmd.Flags().GetString(flagName)
		return v, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	}
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return "", fmt.Errorf("body required: pass --body, pipe markdown on stdin, or set $EDITOR")
	}
	tmp, err := os.CreateTemp("", "citadel-issue-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	name := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close temp file: %w", err)
	}
	defer func() { _ = os.Remove(name) }()

	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid $EDITOR command")
	}
	ecmd := exec.Command(parts[0], append(parts[1:], name)...)
	ecmd.Stdin = os.Stdin
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	if err := ecmd.Run(); err != nil {
		return "", fmt.Errorf("run editor: %w", err)
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read edited body: %w", err)
	}
	return string(b), nil
}

func renderIssueDetailTable(cmd *cobra.Command, payload issueDetailPayload) error {
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "NUMBER\t%d\n", payload.Issue.Number)
	_, _ = fmt.Fprintf(w, "NAMESPACE\t%s\n", payload.Issue.NamespacePath)
	_, _ = fmt.Fprintf(w, "STATE\t%s\n", payload.Issue.State)
	_, _ = fmt.Fprintf(w, "TITLE\t%s\n", payload.Issue.Title)
	_, _ = fmt.Fprintf(w, "AUTHOR\t%s\n", payload.Issue.AuthorID)
	_, _ = fmt.Fprintf(w, "ASSIGNEES\t%s\n", assigneeSummary(payload.Issue.Assignees))
	_, _ = fmt.Fprintf(w, "LABELS\t%s\n", labelSummary(payload.Labels))
	_, _ = fmt.Fprintf(w, "CREATED\t%s\n", formatRFC3339UTC(payload.Issue.CreatedAt))
	_, _ = fmt.Fprintf(w, "UPDATED\t%s\n", formatRFC3339UTC(payload.Issue.UpdatedAt))
	_, _ = fmt.Fprintf(w, "CLOSED\t%s\n", optionalTime(payload.Issue.ClosedAt))
	_, _ = fmt.Fprintf(w, "MILESTONE\t%s\n", strings.TrimSpace(payload.Issue.MilestoneTitle))
	if err := w.Flush(); err != nil {
		return err
	}
	if body := strings.TrimSpace(payload.Issue.BodyMarkdown); body != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), body)
	}
	if len(payload.Comments) == 0 {
		return nil
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "COMMENTS")
	for _, comment := range payload.Comments {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", comment.AuthorID, formatRFC3339UTC(comment.CreatedAt))
		if body := strings.TrimSpace(comment.BodyMarkdown); body != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), body)
		}
	}
	return nil
}

func runIssueList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("--all cannot be used with --output json; use --output ndjson to stream all rows, or omit --all for a single JSON array page")
	}
	state, _ := cmd.Flags().GetString("state")
	state = strings.TrimSpace(strings.ToLower(state))
	if state == "" {
		state = "open"
	}
	switch state {
	case "open", "closed", "all":
	default:
		return fmt.Errorf("--state must be open, closed, or all")
	}
	labels, _ := cmd.Flags().GetStringSlice("label")
	assignees, _ := cmd.Flags().GetStringSlice("assignee")
	labels = normalizeStringSlice(labels)
	assignees = normalizeStringSlice(assignees)

	var yamlAccum []issueRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		q.Set("state", state)
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		for _, label := range labels {
			q.Add("label", label)
		}
		for _, assignee := range assignees {
			q.Add("assignee", assignee)
		}
		var payload struct {
			Issues     []issueRow `json:"issues"`
			NextCursor string     `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), issueBasePath(nsPath)+"?"+q.Encode(), &payload); err != nil {
			return err
		}
		rows := payload.Issues
		next := strings.TrimSpace(payload.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []issueRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[issueListRow](cmd)
			case "yaml":
				return emitYAML(cmd, []issueRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No issues for namespace '%s'.\n", nsPath)
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
			if err := emitCSVRows(cmd, &csvHdr, issueListRows(rows)); err != nil {
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
			_, _ = fmt.Fprintln(w, "#\tSTATE\tTITLE\tASSIGNEES\tUPDATED")
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", row.Number, row.State, row.Title, assigneeSummary(row.Assignees), formatRFC3339UTC(row.UpdatedAt))
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
			yamlAccum = []issueRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runIssueView(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	var payload issueDetailPayload
	if err := c.Get(cmd.Context(), issueBasePath(nsPath)+"/"+strconv.FormatInt(num, 10), &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("issue %s#%d not found", nsPath, num)
		}
		return err
	}
	if web, _ := cmd.Flags().GetBool("web"); web {
		launchBrowser(issueBrowserURL(c.Server(), payload.Issue.NamespacePath, payload.Issue.Number))
		return nil
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	default:
		return renderIssueDetailTable(cmd, payload)
	}
}

func runIssueCreate(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "create"); err != nil {
		return err
	}
	title, _ := cmd.Flags().GetString("title")
	title = strings.TrimSpace(title)
	body, err := readIssueBody(cmd, "body")
	if err != nil {
		return err
	}
	labels, _ := cmd.Flags().GetStringSlice("label")
	payload := map[string]any{
		"title":         title,
		"body_markdown": body,
	}
	if labels = normalizeStringSlice(labels); len(labels) > 0 {
		payload["labels"] = labels
	}
	var created issueRow
	if err := c.Post(cmd.Context(), issueBasePath(nsPath), payload, &created); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created issue %s#%d.\n", created.NamespacePath, created.Number)
	return nil
}

func runIssueComment(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "comment"); err != nil {
		return err
	}
	body, err := readIssueBody(cmd, "body")
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body cannot be empty")
	}
	var created issueComment
	if err := c.Post(cmd.Context(), issueBasePath(nsPath)+"/"+strconv.FormatInt(num, 10)+"/comments", map[string]string{
		"body_markdown": body,
	}, &created); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("issue %s#%d not found", nsPath, num)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added comment %s to %s#%d.\n", created.ID, nsPath, num)
	return nil
}

func runIssueStateMutation(cmd *cobra.Command, args []string, state string, verb string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, verb); err != nil {
		return err
	}
	var resp map[string]any
	if err := c.Patch(cmd.Context(), issueBasePath(nsPath)+"/"+strconv.FormatInt(num, 10), map[string]string{
		"state": state,
	}, &resp); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("issue %s#%d not found", nsPath, num)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, resp)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s issue %s#%d.\n", strings.Title(verb), nsPath, num)
	return nil
}

func runIssueClose(cmd *cobra.Command, args []string) error {
	return runIssueStateMutation(cmd, args, "closed", "close")
}

func runIssueReopen(cmd *cobra.Command, args []string) error {
	return runIssueStateMutation(cmd, args, "open", "reopen")
}

func runIssueLabel(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "label"); err != nil {
		return err
	}
	add, _ := cmd.Flags().GetStringSlice("add")
	remove, _ := cmd.Flags().GetStringSlice("remove")
	add = normalizeStringSlice(add)
	remove = normalizeStringSlice(remove)
	if len(add) == 0 && len(remove) == 0 {
		return fmt.Errorf("set at least one --add or --remove label")
	}
	if err := c.Post(cmd.Context(), issueBasePath(nsPath)+"/"+strconv.FormatInt(num, 10)+"/labels", map[string]any{
		"add":    add,
		"remove": remove,
	}, nil); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("issue %s#%d not found", nsPath, num)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, map[string]any{
			"status":         "ok",
			"namespace_path": nsPath,
			"number":         num,
			"add":            add,
			"remove":         remove,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated labels on issue %s#%d.\n", nsPath, num)
	return nil
}

func runIssueCloseRefs(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	num, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	var payload struct {
		CloseRefs []issueCloseRef `json:"close_refs"`
	}
	if err := c.Get(cmd.Context(), issueBasePath(nsPath)+"/"+strconv.FormatInt(num, 10)+"/close-refs", &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("issue %s#%d not found", nsPath, num)
		}
		return err
	}
	rows := payload.CloseRefs
	if rows == nil {
		rows = []issueCloseRef{}
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No close refs for issue %s#%d.\n", nsPath, num)
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "NAMESPACE\tSHA\tRESOLVED")
		for _, row := range rows {
			sha := "—"
			if row.ClosingCommitSHA != nil && strings.TrimSpace(*row.ClosingCommitSHA) != "" {
				sha = shortSHA(*row.ClosingCommitSHA)
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", row.ReferencedNamespacePath, sha, optionalTime(row.ResolvedAt))
		}
		return w.Flush()
	}
}

func init() {
	IssueCmd.AddCommand(issueListCmd)
	IssueCmd.AddCommand(issueViewCmd)
	IssueCmd.AddCommand(issueCreateCmd)
	IssueCmd.AddCommand(issueCommentCmd)
	IssueCmd.AddCommand(issueCloseCmd)
	IssueCmd.AddCommand(issueReopenCmd)
	IssueCmd.AddCommand(issueLabelCmd)
	IssueCmd.AddCommand(issueCloseRefsCmd)

	addIssuePathFlag(issueListCmd, issueViewCmd, issueCreateCmd, issueCommentCmd, issueCloseCmd, issueReopenCmd, issueLabelCmd, issueCloseRefsCmd)
	addOutputFlag(issueListCmd, issueViewCmd, issueCreateCmd, issueCommentCmd, issueCloseCmd, issueReopenCmd, issueLabelCmd, issueCloseRefsCmd)
	addPaginationFlags(issueListCmd)

	issueListCmd.Flags().String("state", "open", "Issue state filter: open, closed, or all")
	issueListCmd.Flags().StringSlice("label", nil, "Filter by one or more label slugs")
	issueListCmd.Flags().StringSlice("assignee", nil, "Filter by one or more assignee slugs or UUIDs")

	issueViewCmd.Flags().Bool("web", false, "Open the issue in the default browser")

	issueCreateCmd.Flags().String("title", "", "Issue title")
	issueCreateCmd.Flags().String("body", "", "Issue body markdown (defaults to stdin or $EDITOR)")
	issueCreateCmd.Flags().StringSlice("label", nil, "Apply one or more labels at creation time")
	_ = issueCreateCmd.MarkFlagRequired("title")

	issueCommentCmd.Flags().String("body", "", "Comment body markdown (defaults to stdin or $EDITOR)")

	issueLabelCmd.Flags().StringSlice("add", nil, "Add one or more label slugs")
	issueLabelCmd.Flags().StringSlice("remove", nil, "Remove one or more label slugs")
}
