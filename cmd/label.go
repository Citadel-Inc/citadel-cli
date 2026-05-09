package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var LabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage namespace labels",
	Long: `Create, list, edit, and delete labels for any Citadel namespace.

Examples:
  citadel-cli label list -R acme/demo
  citadel-cli label create -R acme/demo --name "Good First Issue" --color a2eeef
  citadel-cli label edit -R acme/demo good-first-issue --color d73a4a
  citadel-cli label delete -R acme/demo good-first-issue`,
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels for a namespace",
	RunE:  runLabelList,
}

var labelCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a label",
	RunE:  runLabelCreate,
}

var labelEditCmd = &cobra.Command{
	Use:               "edit <slug>",
	Short:             "Edit a label",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeLabelSlugs,
	RunE:              runLabelEdit,
}

var labelDeleteCmd = &cobra.Command{
	Use:               "delete <slug>",
	Short:             "Delete a label",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeLabelSlugs,
	RunE:              runLabelDelete,
}

func labelBasePath(nsPath string) string {
	return "/namespaces/" + url.PathEscape(nsPath) + "/labels"
}

func labelCompletionKey(nsPath string) string {
	return "labels:" + strings.Trim(strings.TrimSpace(nsPath), "/")
}

// slugify converts a display name to a URL-safe slug: lowercase ASCII, spaces
// and underscores become hyphens, non-alnum characters are stripped.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// normalizeColor accepts a 6-char hex string with or without a leading '#'
// and returns the canonical #rrggbb form.
func normalizeColor(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("--color is required")
	}
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return "", fmt.Errorf("--color must be a 6-character hex string (e.g. a2eeef or #a2eeef)")
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return "", fmt.Errorf("--color must be a valid hex color string")
		}
	}
	return "#" + strings.ToLower(s), nil
}

func fetchLabels(ctx context.Context, c *apiclient.Client, nsPath string) ([]issueLabel, error) {
	var payload struct {
		Labels []issueLabel `json:"labels"`
	}
	if err := c.Get(ctx, labelBasePath(nsPath), &payload); err != nil {
		return nil, err
	}
	if payload.Labels == nil {
		return []issueLabel{}, nil
	}
	return payload.Labels, nil
}

func completeLabelSlugs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), labelCompletionKey(nsPath), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		labels, err := fetchLabels(ctx, c, nsPath)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(labels))
		for _, l := range labels {
			if s := strings.TrimSpace(l.Slug); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func runLabelList(cmd *cobra.Command, _ []string) error {
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
	labels, err := fetchLabels(cmd.Context(), c, nsPath)
	if err != nil {
		return err
	}
	if len(labels) == 0 {
		switch output {
		case "json":
			return emitJSON(cmd, []issueLabel{})
		case "yaml":
			return emitYAML(cmd, []issueLabel{})
		default:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No labels for namespace '%s'.\n", nsPath)
			return nil
		}
	}
	switch output {
	case "json":
		return emitJSON(cmd, labels)
	case "yaml":
		return emitYAML(cmd, labels)
	case "ndjson":
		return emitNDJSONLines(cmd, labels)
	default:
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "SLUG\tNAME\tCOLOR\tDEFAULT\tDESCRIPTION")
		for _, l := range labels {
			def := ""
			if l.IsDefault {
				def = "yes"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				l.Slug, l.DisplayName, l.Color, def, l.Description)
		}
		return w.Flush()
	}
}

func runLabelCreate(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	rawColor, _ := cmd.Flags().GetString("color")
	color, err := normalizeColor(rawColor)
	if err != nil {
		return err
	}
	slug := slugify(name)
	if f := cmd.Flags().Lookup("slug"); f != nil && f.Changed {
		explicit, _ := cmd.Flags().GetString("slug")
		slug = strings.TrimSpace(explicit)
	}
	if slug == "" {
		return fmt.Errorf("slug is empty — provide --slug explicitly")
	}
	desc, _ := cmd.Flags().GetString("description")
	payload := map[string]any{
		"slug":         slug,
		"display_name": name,
		"color":        color,
		"description":  desc,
	}
	var created issueLabel
	if err := c.Post(cmd.Context(), labelBasePath(nsPath), payload, &created); err != nil {
		if apiclient.IsStatus(err, http.StatusConflict) {
			return fmt.Errorf("label '%s' already exists in %s", slug, nsPath)
		}
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if output == "json" {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created label '%s' in %s.\n", created.Slug, nsPath)
	return nil
}

func runLabelEdit(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	targetSlug := strings.TrimSpace(args[0])
	if targetSlug == "" {
		return fmt.Errorf("label slug required")
	}

	nameChanged := cmd.Flags().Lookup("name") != nil && cmd.Flags().Lookup("name").Changed
	colorChanged := cmd.Flags().Lookup("color") != nil && cmd.Flags().Lookup("color").Changed
	descChanged := cmd.Flags().Lookup("description") != nil && cmd.Flags().Lookup("description").Changed
	if !nameChanged && !colorChanged && !descChanged {
		return fmt.Errorf("set at least one of --name, --color, --description")
	}

	// PATCH does a full SQL UPDATE; GET the current label first to preserve
	// fields the user did not supply.
	labels, err := fetchLabels(cmd.Context(), c, nsPath)
	if err != nil {
		return err
	}
	var existing *issueLabel
	for i := range labels {
		if labels[i].Slug == targetSlug {
			existing = &labels[i]
			break
		}
	}
	if existing == nil {
		return fmt.Errorf("label '%s' not found in %s", targetSlug, nsPath)
	}

	displayName := existing.DisplayName
	color := existing.Color
	desc := existing.Description

	if nameChanged {
		n, _ := cmd.Flags().GetString("name")
		displayName = strings.TrimSpace(n)
	}
	if colorChanged {
		raw, _ := cmd.Flags().GetString("color")
		norm, err := normalizeColor(raw)
		if err != nil {
			return err
		}
		color = norm
	}
	if descChanged {
		desc, _ = cmd.Flags().GetString("description")
	}

	payload := map[string]any{
		"display_name": displayName,
		"color":        color,
		"description":  desc,
	}
	var updated issueLabel
	path := labelBasePath(nsPath) + "/" + url.PathEscape(targetSlug)
	if err := c.Patch(cmd.Context(), path, payload, &updated); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("label '%s' not found in %s", targetSlug, nsPath)
		}
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if output == "json" {
		return emitJSON(cmd, updated)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated label '%s' in %s.\n", updated.Slug, nsPath)
	return nil
}

func runLabelDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	targetSlug := strings.TrimSpace(args[0])
	if targetSlug == "" {
		return fmt.Errorf("label slug required")
	}
	path := labelBasePath(nsPath) + "/" + url.PathEscape(targetSlug)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE label '%s' from %s (skipped; --dry-run)\n", targetSlug, nsPath)
		return nil
	}
	if !yesFlag(cmd) {
		if err := confirmTypedValue(false, "delete label", targetSlug); err != nil {
			return err
		}
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("label '%s' not found in %s", targetSlug, nsPath)
		}
		if apiclient.IsStatus(err, http.StatusConflict) {
			return fmt.Errorf("cannot delete label '%s': it is the last default label for its semantic role", targetSlug)
		}
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if output == "json" {
		return emitJSON(cmd, map[string]any{
			"status":         "ok",
			"namespace_path": nsPath,
			"slug":           targetSlug,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted label '%s' from %s.\n", targetSlug, nsPath)
	return nil
}

func init() {
	LabelCmd.AddCommand(labelListCmd, labelCreateCmd, labelEditCmd, labelDeleteCmd)

	addIssuePathFlag(labelListCmd, labelCreateCmd, labelEditCmd, labelDeleteCmd)
	addOutputFlag(labelListCmd, labelCreateCmd, labelEditCmd, labelDeleteCmd)
	addYesFlag(labelDeleteCmd)
	addDryRunFlag(labelDeleteCmd)

	labelCreateCmd.Flags().String("name", "", "Label display name")
	labelCreateCmd.Flags().String("color", "", "Label color as 6-character hex (e.g. a2eeef or #a2eeef)")
	labelCreateCmd.Flags().String("slug", "", "Label slug override (default: auto-derived from --name)")
	labelCreateCmd.Flags().String("description", "", "Label description")
	_ = labelCreateCmd.MarkFlagRequired("name")
	_ = labelCreateCmd.MarkFlagRequired("color")

	labelEditCmd.Flags().String("name", "", "New display name")
	labelEditCmd.Flags().String("color", "", "New color as 6-character hex")
	labelEditCmd.Flags().String("description", "", "New description (empty clears)")

	labelCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		nsPath, err := resolveIssueNamespacePath(cmd)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), labelCompletionKey(nsPath))
		}
	}
	labelEditCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		nsPath, err := resolveIssueNamespacePath(cmd)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), labelCompletionKey(nsPath))
		}
	}
	labelDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		nsPath, err := resolveIssueNamespacePath(cmd)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), labelCompletionKey(nsPath))
		}
	}
}
