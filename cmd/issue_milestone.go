package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var issueMilestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Manage issue milestones for a namespace path",
	Long: `Milestones group issues into release buckets for any Citadel namespace path.

Examples:
  citadel-cli issue milestone list -R acme/demo
  citadel-cli issue milestone view -R acme/demo <milestone-uuid>
  citadel-cli issue milestone create -R acme/demo --title "v1.0" --due-on 2026-06-01`,
}

var issueMilestoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones for a namespace path",
	RunE:  runIssueMilestoneList,
}

var issueMilestoneViewCmd = &cobra.Command{
	Use:               "view <id>",
	Short:             "Show one milestone by UUID",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueMilestoneIDs,
	RunE:              runIssueMilestoneView,
}

var issueMilestoneCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a milestone",
	RunE:  runIssueMilestoneCreate,
}

var issueMilestoneEditCmd = &cobra.Command{
	Use:               "edit <id>",
	Short:             "Edit a milestone",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueMilestoneIDs,
	RunE:              runIssueMilestoneEdit,
}

var issueMilestoneDeleteCmd = &cobra.Command{
	Use:               "delete <id>",
	Short:             "Delete a milestone",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueMilestoneIDs,
	RunE:              runIssueMilestoneDelete,
}

type milestoneProgress struct {
	OpenCount   int `json:"open_count"`
	ClosedCount int `json:"closed_count"`
	Total       int `json:"total"`
	Percent     int `json:"percent"`
}

type milestoneRow struct {
	ID          string            `json:"id"`
	NamespaceID string            `json:"namespace_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	DueOn       *time.Time        `json:"due_on,omitempty"`
	State       string            `json:"state"`
	ClosedAt    *time.Time        `json:"closed_at,omitempty"`
	CreatedBy   *string           `json:"created_by,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Progress    milestoneProgress `json:"progress"`
}

type milestoneListRow struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	DueOn     *time.Time `json:"due_on,omitempty"`
	Progress  string     `json:"progress"`
	CreatedAt time.Time  `json:"created_at"`
}

func (milestoneListRow) CSVHeader() []string {
	return []string{"id", "title", "state", "due_on", "progress", "created_at"}
}

func (r milestoneListRow) CSVRecord() []string {
	return []string{
		r.ID,
		r.Title,
		r.State,
		formatMilestoneDate(r.DueOn),
		r.Progress,
		formatRFC3339UTC(r.CreatedAt),
	}
}

func milestoneBasePath(nsPath string) string {
	return "/namespaces/" + url.PathEscape(nsPath) + "/milestones"
}

func milestoneCompletionKey(namespacePath string) string {
	return "milestones:" + strings.Trim(strings.TrimSpace(namespacePath), "/")
}

func parseMilestoneID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", fmt.Errorf("milestone id must be a UUID")
	}
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("milestone id must be a UUID")
	}
	return id, nil
}

func parseMilestoneDueOn(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", raw); err != nil {
		return "", fmt.Errorf("--due-on must use YYYY-MM-DD")
	}
	return raw, nil
}

func milestoneProgressSummary(p milestoneProgress) string {
	return fmt.Sprintf("%d%% (%d/%d closed)", p.Percent, p.ClosedCount, p.Total)
}

func milestoneProgressBar(p milestoneProgress) string {
	const width = 10
	filled := 0
	if p.Percent > 0 {
		filled = (p.Percent * width) / 100
		if filled == 0 {
			filled = 1
		}
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "] " + milestoneProgressSummary(p)
}

func formatMilestoneDate(v *time.Time) string {
	if v == nil || v.IsZero() {
		return "—"
	}
	return v.UTC().Format("2006-01-02")
}

func milestoneListRows(rows []milestoneRow) []milestoneListRow {
	out := make([]milestoneListRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, milestoneListRow{
			ID:        row.ID,
			Title:     row.Title,
			State:     row.State,
			DueOn:     row.DueOn,
			Progress:  milestoneProgressSummary(row.Progress),
			CreatedAt: row.CreatedAt,
		})
	}
	return out
}

func filterMilestonesByState(rows []milestoneRow, state string) []milestoneRow {
	if state == "all" {
		return rows
	}
	out := make([]milestoneRow, 0, len(rows))
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.State), state) {
			out = append(out, row)
		}
	}
	return out
}

func fetchMilestones(ctx context.Context, c *apiclient.Client, nsPath string) ([]milestoneRow, error) {
	var payload struct {
		Milestones []milestoneRow `json:"milestones"`
	}
	if err := c.Get(ctx, milestoneBasePath(nsPath), &payload); err != nil {
		return nil, err
	}
	if payload.Milestones == nil {
		return []milestoneRow{}, nil
	}
	return payload.Milestones, nil
}

func renderMilestoneDetailTable(cmd *cobra.Command, nsPath string, row milestoneRow) error {
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "ID\t%s\n", row.ID)
	_, _ = fmt.Fprintf(w, "NAMESPACE\t%s\n", nsPath)
	_, _ = fmt.Fprintf(w, "STATE\t%s\n", row.State)
	_, _ = fmt.Fprintf(w, "TITLE\t%s\n", row.Title)
	_, _ = fmt.Fprintf(w, "DUE ON\t%s\n", formatMilestoneDate(row.DueOn))
	_, _ = fmt.Fprintf(w, "PROGRESS\t%s\n", milestoneProgressBar(row.Progress))
	_, _ = fmt.Fprintf(w, "CREATED\t%s\n", formatRFC3339UTC(row.CreatedAt))
	_, _ = fmt.Fprintf(w, "CLOSED\t%s\n", optionalTime(row.ClosedAt))
	if row.CreatedBy != nil && strings.TrimSpace(*row.CreatedBy) != "" {
		_, _ = fmt.Fprintf(w, "CREATED BY\t%s\n", strings.TrimSpace(*row.CreatedBy))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if desc := strings.TrimSpace(row.Description); desc != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), desc)
	}
	return nil
}

func completeIssueMilestoneIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return lookupMilestoneIDs(cmd, nsPath)
}

func lookupMilestoneIDs(cmd *cobra.Command, nsPath string) ([]string, cobra.ShellCompDirective) {
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), milestoneCompletionKey(nsPath), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		rows, err := fetchMilestones(ctx, c, nsPath)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			if id := strings.TrimSpace(row.ID); id != "" {
				out = append(out, id)
			}
		}
		return out, nil
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func runIssueMilestoneList(cmd *cobra.Command, _ []string) error {
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
	rows, err := fetchMilestones(cmd.Context(), c, nsPath)
	if err != nil {
		return err
	}
	rows = filterMilestonesByState(rows, state)
	if len(rows) == 0 {
		switch output {
		case "json":
			return emitJSON(cmd, []milestoneRow{})
		case "yaml":
			return emitYAML(cmd, []milestoneRow{})
		case "ndjson":
			return nil
		case "csv":
			return emitCSVHeaderOnly[milestoneListRow](cmd)
		default:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No %s milestones for namespace '%s'.\n", state, nsPath)
			return nil
		}
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		header := false
		return emitCSVRows(cmd, &header, milestoneListRows(rows))
	default:
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tSTATE\tTITLE\tDUE_ON\tPROGRESS\tCREATED")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				row.ID,
				row.State,
				row.Title,
				formatMilestoneDate(row.DueOn),
				milestoneProgressSummary(row.Progress),
				formatRFC3339UTC(row.CreatedAt),
			)
		}
		return w.Flush()
	}
}

func runIssueMilestoneView(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	id, err := parseMilestoneID(args[0])
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	var row milestoneRow
	if err := c.Get(cmd.Context(), milestoneBasePath(nsPath)+"/"+url.PathEscape(id), &row); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("milestone %s not found in %s", id, nsPath)
		}
		return err
	}
	switch output {
	case "json":
		return emitJSON(cmd, row)
	case "yaml":
		return emitYAML(cmd, row)
	default:
		return renderMilestoneDetailTable(cmd, nsPath, row)
	}
}

func runIssueMilestoneCreate(cmd *cobra.Command, _ []string) error {
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
	if title == "" {
		return fmt.Errorf("title required")
	}
	payload := map[string]any{"title": title}
	if f := cmd.Flags().Lookup("description"); f != nil && f.Changed {
		desc, _ := cmd.Flags().GetString("description")
		payload["description"] = desc
	}
	if f := cmd.Flags().Lookup("due-on"); f != nil && f.Changed {
		raw, _ := cmd.Flags().GetString("due-on")
		dueOn, err := parseMilestoneDueOn(raw)
		if err != nil {
			return err
		}
		if dueOn != "" {
			payload["due_on"] = dueOn
		}
	}
	var created milestoneRow
	if err := c.Post(cmd.Context(), milestoneBasePath(nsPath), payload, &created); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created milestone %s for %s.\n", created.ID, nsPath)
	return nil
}

func runIssueMilestoneEdit(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	id, err := parseMilestoneID(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "edit"); err != nil {
		return err
	}
	payload := map[string]any{}
	if f := cmd.Flags().Lookup("title"); f != nil && f.Changed {
		title, _ := cmd.Flags().GetString("title")
		title = strings.TrimSpace(title)
		if title == "" {
			return fmt.Errorf("title cannot be empty")
		}
		payload["title"] = title
	}
	if f := cmd.Flags().Lookup("description"); f != nil && f.Changed {
		desc, _ := cmd.Flags().GetString("description")
		payload["description"] = desc
	}
	if f := cmd.Flags().Lookup("due-on"); f != nil && f.Changed {
		raw, _ := cmd.Flags().GetString("due-on")
		dueOn, err := parseMilestoneDueOn(raw)
		if err != nil {
			return err
		}
		payload["due_on"] = dueOn
	}
	if f := cmd.Flags().Lookup("state"); f != nil && f.Changed {
		state, _ := cmd.Flags().GetString("state")
		state = strings.TrimSpace(strings.ToLower(state))
		switch state {
		case "open", "closed":
			payload["state"] = state
		default:
			return fmt.Errorf("--state must be open or closed")
		}
	}
	if len(payload) == 0 {
		return fmt.Errorf("set at least one field to update")
	}
	var updated milestoneRow
	if err := c.Put(cmd.Context(), milestoneBasePath(nsPath)+"/"+url.PathEscape(id), payload, &updated); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("milestone %s not found in %s", id, nsPath)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, updated)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated milestone %s in %s.\n", updated.ID, nsPath)
	return nil
}

func runIssueMilestoneDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	id, err := parseMilestoneID(args[0])
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "delete"); err != nil {
		return err
	}
	path := milestoneBasePath(nsPath) + "/" + url.PathEscape(id)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if !yesFlag(cmd) {
		var row milestoneRow
		if err := c.Get(cmd.Context(), path, &row); err != nil {
			if apiclient.IsStatus(err, http.StatusNotFound) {
				return fmt.Errorf("milestone %s not found in %s", id, nsPath)
			}
			return err
		}
		value := strings.TrimSpace(row.Title)
		if value == "" {
			value = id
		}
		if err := confirmTypedValue(false, "delete milestone", value); err != nil {
			return err
		}
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("milestone %s not found in %s", id, nsPath)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, map[string]any{
			"status":         "ok",
			"namespace_path": nsPath,
			"id":             id,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted milestone %s from %s.\n", id, nsPath)
	return nil
}

func init() {
	IssueCmd.AddCommand(issueMilestoneCmd)
	issueMilestoneCmd.AddCommand(
		issueMilestoneListCmd,
		issueMilestoneViewCmd,
		issueMilestoneCreateCmd,
		issueMilestoneEditCmd,
		issueMilestoneDeleteCmd,
	)

	addIssuePathFlag(issueMilestoneListCmd, issueMilestoneViewCmd, issueMilestoneCreateCmd, issueMilestoneEditCmd, issueMilestoneDeleteCmd)
	addOutputFlag(issueMilestoneListCmd, issueMilestoneViewCmd, issueMilestoneCreateCmd, issueMilestoneEditCmd, issueMilestoneDeleteCmd)
	addYesFlag(issueMilestoneDeleteCmd)
	addDryRunFlag(issueMilestoneDeleteCmd)

	issueMilestoneListCmd.Flags().String("state", "open", "Milestone state filter: open, closed, or all")

	issueMilestoneCreateCmd.Flags().String("title", "", "Milestone title")
	issueMilestoneCreateCmd.Flags().String("description", "", "Milestone description")
	issueMilestoneCreateCmd.Flags().String("due-on", "", "Milestone due date (YYYY-MM-DD)")
	_ = issueMilestoneCreateCmd.MarkFlagRequired("title")

	issueMilestoneEditCmd.Flags().String("title", "", "New milestone title")
	issueMilestoneEditCmd.Flags().String("description", "", "New milestone description (empty clears)")
	issueMilestoneEditCmd.Flags().String("due-on", "", "New milestone due date (YYYY-MM-DD; empty clears)")
	issueMilestoneEditCmd.Flags().String("state", "", "Milestone state: open or closed")

	issueMilestoneCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		nsPath, err := resolveIssueNamespacePath(cmd)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), milestoneCompletionKey(nsPath))
		}
	}
	issueMilestoneDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		nsPath, err := resolveIssueNamespacePath(cmd)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), milestoneCompletionKey(nsPath))
		}
	}
}
