package cmd

import (
	"context"
	"errors"
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

const webhookCompletionPrefix = "webhooks:"

var webhookEventKinds = []string{
	"comment.created",
	"comment.edited",
	"issue.assigned",
	"issue.closed",
	"issue.labeled",
	"issue.opened",
	"issue.reopened",
	"issue.unassigned",
	"issue.unlabeled",
}

var repoWebhookCmd = &cobra.Command{
	Use:     "webhook",
	Aliases: []string{"webhooks"},
	Short:   "Manage webhooks for a repository namespace",
}

var repoWebhookListCmd = &cobra.Command{
	Use:               "list [<namespace>/<repo>]",
	Short:             "List repository webhooks",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoWebhookList,
	ValidArgsFunction: completeRepoSlugs,
}

var repoWebhookCreateCmd = &cobra.Command{
	Use:               "create [<namespace>/<repo>]",
	Short:             "Create a repository webhook",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoWebhookCreate,
	ValidArgsFunction: completeRepoSlugs,
}

var repoWebhookGetCmd = &cobra.Command{
	Use:               "get [<namespace>/<repo>] <id>",
	Short:             "Get a repository webhook",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoWebhookGet,
	ValidArgsFunction: completeRepoWebhookIDs,
}

var repoWebhookDeleteCmd = &cobra.Command{
	Use:               "delete [<namespace>/<repo>] <id>",
	Short:             "Delete a repository webhook",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoWebhookDelete,
	ValidArgsFunction: completeRepoWebhookIDs,
}

var namespaceWebhookCmd = &cobra.Command{
	Use:     "webhook",
	Aliases: []string{"webhooks"},
	Short:   "Manage webhooks for a namespace",
}

var namespaceWebhookListCmd = &cobra.Command{
	Use:               "list <slug>",
	Short:             "List namespace webhooks",
	Args:              cobra.ExactArgs(1),
	RunE:              runNamespaceWebhookList,
	ValidArgsFunction: completeOrgNamespaceSlugs,
}

var namespaceWebhookCreateCmd = &cobra.Command{
	Use:               "create <slug>",
	Short:             "Create a namespace webhook",
	Args:              cobra.ExactArgs(1),
	RunE:              runNamespaceWebhookCreate,
	ValidArgsFunction: completeOrgNamespaceSlugs,
}

var namespaceWebhookGetCmd = &cobra.Command{
	Use:               "get <slug> <id>",
	Short:             "Get a namespace webhook",
	Args:              cobra.ExactArgs(2),
	RunE:              runNamespaceWebhookGet,
	ValidArgsFunction: completeNamespaceWebhookIDs,
}

var namespaceWebhookDeleteCmd = &cobra.Command{
	Use:               "delete <slug> <id>",
	Short:             "Delete a namespace webhook",
	Args:              cobra.ExactArgs(2),
	RunE:              runNamespaceWebhookDelete,
	ValidArgsFunction: completeNamespaceWebhookIDs,
}

type webhookRow struct {
	ID                 string     `json:"id"`
	NamespaceID        string     `json:"namespace_id"`
	NamespacePath      string     `json:"namespace_path"`
	Name               string     `json:"name,omitempty"`
	TargetURL          string     `json:"target_url"`
	EventKinds         []string   `json:"event_kinds"`
	IncludeDescendants bool       `json:"include_descendants"`
	Active             bool       `json:"active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastDeliveryAt     *time.Time `json:"last_delivery_at,omitempty"`
	LastDeliveryState  string     `json:"last_delivery_state,omitempty"`
	SecretHint         string     `json:"secret_hint,omitempty"`
	CleartextSecret    string     `json:"cleartext_secret,omitempty"`
}

func (r webhookRow) CSVHeader() []string {
	return []string{
		"id", "name", "namespace_path", "target_url", "event_kinds",
		"include_descendants", "active", "created_at", "updated_at",
		"last_delivery_at", "last_delivery_state", "secret_hint",
	}
}

func (r webhookRow) CSVRecord() []string {
	lastDeliveryAt := ""
	if r.LastDeliveryAt != nil {
		lastDeliveryAt = r.LastDeliveryAt.Format(time.RFC3339)
	}
	return []string{
		r.ID,
		r.Name,
		r.NamespacePath,
		r.TargetURL,
		strings.Join(r.EventKinds, ","),
		fmt.Sprintf("%t", r.IncludeDescendants),
		fmt.Sprintf("%t", r.Active),
		r.CreatedAt.Format(time.RFC3339),
		r.UpdatedAt.Format(time.RFC3339),
		lastDeliveryAt,
		r.LastDeliveryState,
		r.SecretHint,
	}
}

type webhookCreateRequest struct {
	Name               string   `json:"name,omitempty"`
	TargetURL          string   `json:"target_url"`
	EventKinds         []string `json:"event_kinds"`
	IncludeDescendants bool     `json:"include_descendants"`
	Active             bool     `json:"active"`
}

func webhookAPIPath(namespacePath string) string {
	return "/api/namespaces/" + url.PathEscape(strings.Trim(strings.TrimSpace(namespacePath), "/")) + "/webhooks"
}

func webhookCompletionKey(namespacePath string) string {
	return webhookCompletionPrefix + strings.Trim(strings.TrimSpace(namespacePath), "/")
}

func runRepoWebhookList(cmd *cobra.Command, args []string) error {
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	return runWebhookList(cmd, ns+"/"+slug)
}

func runNamespaceWebhookList(cmd *cobra.Command, args []string) error {
	return runWebhookList(cmd, args[0])
}

func runWebhookList(cmd *cobra.Command, namespacePath string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
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
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []webhookRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Webhooks []webhookRow `json:"webhooks"`
			Next     string       `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), webhookAPIPath(namespacePath)+"?"+q.Encode(), &payload); err != nil {
			return decorateWebhookError(err, namespacePath, "list")
		}
		rows := payload.Webhooks
		next := strings.TrimSpace(payload.Next)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []webhookRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[webhookRow](cmd)
			case "yaml":
				return emitYAML(cmd, []webhookRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No webhooks for namespace '%s'.\n", namespacePath)
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
			_, _ = fmt.Fprintln(w, "ID\tNAME\tACTIVE\tEVENTS\tTARGET\tLAST")
			for _, row := range rows {
				name := strings.TrimSpace(row.Name)
				if name == "" {
					name = "-"
				}
				last := strings.TrimSpace(row.LastDeliveryState)
				if last == "" {
					last = "-"
				}
				_, _ = fmt.Fprintf(
					w, "%s\t%s\t%t\t%s\t%s\t%s\n",
					row.ID, name, row.Active, strings.Join(row.EventKinds, ","), row.TargetURL, last,
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

	if output == "yaml" {
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runRepoWebhookCreate(cmd *cobra.Command, args []string) error {
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	return runWebhookCreate(cmd, ns+"/"+slug, false)
}

func runNamespaceWebhookCreate(cmd *cobra.Command, args []string) error {
	return runWebhookCreate(cmd, args[0], true)
}

func runWebhookCreate(cmd *cobra.Command, namespacePath string, allowDescendants bool) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	body, err := readWebhookCreateRequest(cmd, allowDescendants)
	if err != nil {
		return err
	}
	var created webhookRow
	if err := c.Post(cmd.Context(), webhookAPIPath(namespacePath), body, &created); err != nil {
		return decorateWebhookError(err, namespacePath, "create")
	}
	if created.NamespacePath == "" {
		created.NamespacePath = namespacePath
	}

	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, created)
	case "yaml":
		return emitYAML(cmd, created)
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created webhook %s for %s.\n", created.ID, created.NamespacePath)
		if created.Name != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", created.Name)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target: %s\n", created.TargetURL)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Events: %s\n", strings.Join(created.EventKinds, ", "))
		if created.CleartextSecret != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret (save now; shown once): %s\n", created.CleartextSecret)
		}
		return nil
	}
}

func runRepoWebhookGet(cmd *cobra.Command, args []string) error {
	namespacePath, id, err := parseRepoWebhookIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runWebhookGet(cmd, namespacePath, id)
}

func runNamespaceWebhookGet(cmd *cobra.Command, args []string) error {
	return runWebhookGet(cmd, args[0], args[1])
}

func runWebhookGet(cmd *cobra.Command, namespacePath, rawID string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	hook, err := fetchWebhookByID(cmd.Context(), cmd, namespacePath, rawID)
	if err != nil {
		return err
	}
	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, hook)
	case "yaml":
		return emitYAML(cmd, hook)
	default:
		return emitWebhookHuman(cmd, hook)
	}
}

func runRepoWebhookDelete(cmd *cobra.Command, args []string) error {
	namespacePath, id, err := parseRepoWebhookIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runWebhookDelete(cmd, namespacePath, id)
}

func runNamespaceWebhookDelete(cmd *cobra.Command, args []string) error {
	return runWebhookDelete(cmd, args[0], args[1])
}

func runWebhookDelete(cmd *cobra.Command, namespacePath, rawID string) error {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return fmt.Errorf("invalid webhook id: %w", err)
	}
	path := webhookAPIPath(namespacePath) + "/" + url.PathEscape(id.String())
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		return decorateWebhookError(err, namespacePath, "delete")
	}

	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, map[string]string{"status": "deleted", "id": id.String(), "namespace_path": namespacePath})
	case "yaml":
		return emitYAML(cmd, map[string]string{"status": "deleted", "id": id.String(), "namespace_path": namespacePath})
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted webhook %s from %s.\n", id.String(), namespacePath)
		return nil
	}
}

func readWebhookCreateRequest(cmd *cobra.Command, allowDescendants bool) (webhookCreateRequest, error) {
	targetURL, _ := cmd.Flags().GetString("url")
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return webhookCreateRequest{}, fmt.Errorf("--url is required")
	}
	events, _ := cmd.Flags().GetStringSlice("events")
	events = normaliseCLIEventKinds(events)
	if len(events) == 0 {
		return webhookCreateRequest{}, fmt.Errorf("--events is required")
	}
	name, _ := cmd.Flags().GetString("name")
	active, _ := cmd.Flags().GetBool("active")
	includeDescendants := false
	if allowDescendants {
		includeDescendants, _ = cmd.Flags().GetBool("include-descendants")
	}
	return webhookCreateRequest{
		Name:               strings.TrimSpace(name),
		TargetURL:          targetURL,
		EventKinds:         events,
		IncludeDescendants: includeDescendants,
		Active:             active,
	}, nil
}

func normaliseCLIEventKinds(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			v := strings.TrimSpace(strings.ToLower(part))
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func parseRepoWebhookIDArgs(cmd *cobra.Command, args []string) (string, string, error) {
	switch len(args) {
	case 1:
		ns, slug, err := resolveRepoFromPosOrFlag(cmd, "")
		if err != nil {
			return "", "", err
		}
		return ns + "/" + slug, strings.TrimSpace(args[0]), nil
	case 2:
		ns, slug, err := resolveRepoFromPosOrFlag(cmd, args[0])
		if err != nil {
			return "", "", err
		}
		return ns + "/" + slug, strings.TrimSpace(args[1]), nil
	default:
		return "", "", fmt.Errorf("expected <id> with -R/--repo, or <namespace>/<repo> <id>")
	}
}

func fetchWebhookByID(ctx context.Context, cmd *cobra.Command, namespacePath, rawID string) (webhookRow, error) {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return webhookRow{}, fmt.Errorf("invalid webhook id: %w", err)
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return webhookRow{}, err
	}
	hooks, err := fetchWebhookRows(ctx, c, namespacePath)
	if err != nil {
		return webhookRow{}, decorateWebhookError(err, namespacePath, "get")
	}
	for _, hook := range hooks {
		if hook.ID == id.String() {
			return hook, nil
		}
	}
	return webhookRow{}, fmt.Errorf("webhook %s not found in %s", id.String(), namespacePath)
}

func fetchWebhookRows(ctx context.Context, c *apiclient.Client, namespacePath string) ([]webhookRow, error) {
	var payload struct {
		Webhooks []webhookRow `json:"webhooks"`
	}
	if err := c.Get(ctx, webhookAPIPath(namespacePath), &payload); err != nil {
		return nil, err
	}
	return payload.Webhooks, nil
}

func emitWebhookHuman(cmd *cobra.Command, hook webhookRow) error {
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintln(w, "FIELD\tVALUE")
	_, _ = fmt.Fprintf(w, "ID\t%s\n", hook.ID)
	_, _ = fmt.Fprintf(w, "Namespace\t%s\n", hook.NamespacePath)
	if hook.Name != "" {
		_, _ = fmt.Fprintf(w, "Name\t%s\n", hook.Name)
	}
	_, _ = fmt.Fprintf(w, "Target\t%s\n", hook.TargetURL)
	_, _ = fmt.Fprintf(w, "Events\t%s\n", strings.Join(hook.EventKinds, ", "))
	_, _ = fmt.Fprintf(w, "Include descendants\t%t\n", hook.IncludeDescendants)
	_, _ = fmt.Fprintf(w, "Active\t%t\n", hook.Active)
	_, _ = fmt.Fprintf(w, "Created\t%s\n", hook.CreatedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "Updated\t%s\n", hook.UpdatedAt.Format(time.RFC3339))
	if hook.LastDeliveryAt != nil {
		_, _ = fmt.Fprintf(w, "Last delivery\t%s\n", hook.LastDeliveryAt.Format(time.RFC3339))
	}
	if hook.LastDeliveryState != "" {
		_, _ = fmt.Fprintf(w, "Last delivery state\t%s\n", hook.LastDeliveryState)
	}
	if hook.SecretHint != "" {
		_, _ = fmt.Fprintf(w, "Secret hint\t%s\n", hook.SecretHint)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func completeRepoWebhookIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, "")
	if err == nil && len(args) == 0 {
		return lookupWebhookIDs(cmd, ns+"/"+slug)
	}
	if len(args) == 0 {
		return completeRepoSlugs(cmd, args, "")
	}
	ns, slug, err = resolveRepoFromPosOrFlag(cmd, args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return lookupWebhookIDs(cmd, ns+"/"+slug)
}

func completeNamespaceWebhookIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeOrgNamespaceSlugs(cmd, args, "")
	case 1:
		return lookupWebhookIDs(cmd, args[0])
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func lookupWebhookIDs(cmd *cobra.Command, namespacePath string) ([]string, cobra.ShellCompDirective) {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), webhookCompletionKey(namespacePath), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		rows, err := fetchWebhookRows(ctx, c, namespacePath)
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

func completeWebhookEvents(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	prefix := strings.TrimSpace(strings.ToLower(toComplete))
	out := make([]string, 0, len(webhookEventKinds))
	for _, event := range webhookEventKinds {
		if strings.HasPrefix(event, prefix) {
			out = append(out, event)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func decorateWebhookError(err error, namespacePath, action string) error {
	if err == nil {
		return nil
	}
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return err
	}
	switch he.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: missing permission to %s webhooks in %s", action, namespacePath)
	case http.StatusNotFound:
		return fmt.Errorf("namespace or webhook not found in %s", namespacePath)
	case http.StatusConflict:
		return fmt.Errorf("namespace webhook limit reached for %s", namespacePath)
	case http.StatusBadRequest:
		return fmt.Errorf("invalid webhook request for %s", namespacePath)
	}
	return err
}

func init() {
	repoWebhookCmd.AddCommand(repoWebhookListCmd)
	repoWebhookCmd.AddCommand(repoWebhookCreateCmd)
	repoWebhookCmd.AddCommand(repoWebhookGetCmd)
	repoWebhookCmd.AddCommand(repoWebhookDeleteCmd)
	RepoCmd.AddCommand(repoWebhookCmd)

	namespaceWebhookCmd.AddCommand(namespaceWebhookListCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookCreateCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookGetCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookDeleteCmd)
	NamespaceCmd.AddCommand(namespaceWebhookCmd)

	addOutputFlag(
		repoWebhookListCmd, repoWebhookCreateCmd, repoWebhookGetCmd, repoWebhookDeleteCmd,
		namespaceWebhookListCmd, namespaceWebhookCreateCmd, namespaceWebhookGetCmd, namespaceWebhookDeleteCmd,
	)
	addPaginationFlags(repoWebhookListCmd, namespaceWebhookListCmd)
	addRepoFlag(repoWebhookListCmd, repoWebhookCreateCmd, repoWebhookGetCmd, repoWebhookDeleteCmd)
	addDryRunFlag(repoWebhookDeleteCmd, namespaceWebhookDeleteCmd)

	for _, c := range []*cobra.Command{repoWebhookCreateCmd, namespaceWebhookCreateCmd} {
		c.Flags().String("name", "", "Optional webhook name")
		c.Flags().String("url", "", "Target URL for webhook delivery (required)")
		c.Flags().StringSlice("events", nil, "Comma-separated or repeated event kinds to deliver (required)")
		c.Flags().Bool("active", true, "Create the webhook in active state")
		_ = c.MarkFlagRequired("url")
		_ = c.MarkFlagRequired("events")
		_ = c.RegisterFlagCompletionFunc("events", completeWebhookEvents)
	}
	namespaceWebhookCreateCmd.Flags().Bool("include-descendants", false, "Deliver matching events from descendant namespaces as well")

	repoWebhookCreateCmd.PostRun = func(cmd *cobra.Command, args []string) {
		pos := ""
		if len(args) > 0 {
			pos = args[0]
		}
		ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(ns+"/"+slug))
		}
	}
	namespaceWebhookCreateCmd.PostRun = func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(args[0]))
		}
	}
	repoWebhookDeleteCmd.PostRun = func(cmd *cobra.Command, args []string) {
		namespacePath, _, err := parseRepoWebhookIDArgs(cmd, args)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(namespacePath))
		}
	}
	namespaceWebhookDeleteCmd.PostRun = func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(args[0]))
		}
	}
}
