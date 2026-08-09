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

var repoWebhookEditCmd = &cobra.Command{
	Use:               "edit [<namespace>/<repo>] <id>",
	Short:             "Edit a repository webhook",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoWebhookEdit,
	ValidArgsFunction: completeRepoWebhookIDs,
}

var repoWebhookDeleteCmd = &cobra.Command{
	Use:               "delete [<namespace>/<repo>] <id>",
	Short:             "Delete a repository webhook",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoWebhookDelete,
	ValidArgsFunction: completeRepoWebhookIDs,
}

var repoWebhookDeliveriesCmd = &cobra.Command{
	Use:   "deliveries",
	Short: "Manage repository webhook deliveries",
}

var repoWebhookDeliveryListCmd = &cobra.Command{
	Use:   "list [<namespace>/<repo>]",
	Short: "List repository webhook deliveries",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  runRepoWebhookDeliveryList,
}

var repoWebhookDeliveryGetCmd = &cobra.Command{
	Use:   "get [<namespace>/<repo>] <delivery-id>",
	Short: "Get a repository webhook delivery",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoWebhookDeliveryGet,
}

var repoWebhookDeliveryRedeliverCmd = &cobra.Command{
	Use:   "redeliver [<namespace>/<repo>] <delivery-id>",
	Short: "Redeliver a repository webhook delivery",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoWebhookDeliveryRedeliver,
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

var namespaceWebhookEditCmd = &cobra.Command{
	Use:               "edit <slug> <id>",
	Short:             "Edit a namespace webhook",
	Args:              cobra.ExactArgs(2),
	RunE:              runNamespaceWebhookEdit,
	ValidArgsFunction: completeNamespaceWebhookIDs,
}

var namespaceWebhookDeleteCmd = &cobra.Command{
	Use:               "delete <slug> <id>",
	Short:             "Delete a namespace webhook",
	Args:              cobra.ExactArgs(2),
	RunE:              runNamespaceWebhookDelete,
	ValidArgsFunction: completeNamespaceWebhookIDs,
}

var namespaceWebhookDeliveriesCmd = &cobra.Command{
	Use:   "deliveries",
	Short: "Manage namespace webhook deliveries",
}

var namespaceWebhookDeliveryListCmd = &cobra.Command{
	Use:   "list <slug>",
	Short: "List namespace webhook deliveries",
	Args:  cobra.ExactArgs(1),
	RunE:  runNamespaceWebhookDeliveryList,
}

var namespaceWebhookDeliveryGetCmd = &cobra.Command{
	Use:   "get <slug> <delivery-id>",
	Short: "Get a namespace webhook delivery",
	Args:  cobra.ExactArgs(2),
	RunE:  runNamespaceWebhookDeliveryGet,
}

var namespaceWebhookDeliveryRedeliverCmd = &cobra.Command{
	Use:   "redeliver <slug> <delivery-id>",
	Short: "Redeliver a namespace webhook delivery",
	Args:  cobra.ExactArgs(2),
	RunE:  runNamespaceWebhookDeliveryRedeliver,
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

type webhookDeliveryRow struct {
	ID                   string         `json:"id"`
	WebhookID            string         `json:"webhook_id"`
	WebhookName          string         `json:"webhook_name,omitempty"`
	WebhookURL           string         `json:"webhook_url,omitempty"`
	EventID              string         `json:"event_id,omitempty"`
	EventKind            string         `json:"event_kind"`
	WebhookNamespacePath string         `json:"webhook_namespace_path,omitempty"`
	SourceNamespacePath  string         `json:"source_namespace_path,omitempty"`
	State                string         `json:"state"`
	AttemptCount         int            `json:"attempt_count"`
	LastAttemptAt        *time.Time     `json:"last_attempt_at,omitempty"`
	DeliveredAt          *time.Time     `json:"delivered_at,omitempty"`
	HTTPStatus           *int           `json:"http_status,omitempty"`
	ResponseBody         string         `json:"response_body,omitempty"`
	ResponseHeaders      map[string]any `json:"response_headers,omitempty"`
	ErrorMessage         string         `json:"error_message,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	Payload              any            `json:"payload,omitempty"`
}

type webhookDeliveryResponse struct {
	Delivery webhookDeliveryRow `json:"delivery"`
	Sent     bool               `json:"sent,omitempty"`
	State    string             `json:"state,omitempty"`
}

func (r webhookDeliveryRow) CSVHeader() []string {
	return []string{"id", "event_kind", "state", "attempt_count", "http_status", "created_at"}
}

func (r webhookDeliveryRow) CSVRecord() []string {
	httpStatus := ""
	if r.HTTPStatus != nil {
		httpStatus = fmt.Sprintf("%d", *r.HTTPStatus)
	}
	return []string{
		r.ID,
		r.EventKind,
		r.State,
		fmt.Sprintf("%d", r.AttemptCount),
		httpStatus,
		r.CreatedAt.Format(time.RFC3339),
	}
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

type webhookPatchRequest struct {
	Name               *string  `json:"name,omitempty"`
	TargetURL          *string  `json:"target_url,omitempty"`
	EventKinds         []string `json:"event_kinds,omitempty"`
	IncludeDescendants *bool    `json:"include_descendants,omitempty"`
	Active             *bool    `json:"active,omitempty"`
	RotateSecret       bool     `json:"rotate_secret,omitempty"`
}

func webhookAPIPath(namespacePath string) string {
	return "/api/namespaces/" + url.PathEscape(strings.Trim(strings.TrimSpace(namespacePath), "/")) + "/webhooks"
}

func webhookDeliveryAPIPath(namespacePath, rawID string) (string, error) {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return "", fmt.Errorf("invalid delivery id: %w", err)
	}
	return webhookAPIPath(namespacePath) + "/deliveries/" + url.PathEscape(id.String()), nil
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

func runRepoWebhookEdit(cmd *cobra.Command, args []string) error {
	namespacePath, id, err := parseRepoWebhookIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runWebhookEdit(cmd, namespacePath, id, false)
}

func runNamespaceWebhookEdit(cmd *cobra.Command, args []string) error {
	return runWebhookEdit(cmd, args[0], args[1], true)
}

func runWebhookEdit(cmd *cobra.Command, namespacePath, rawID string, allowDescendants bool) error {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return fmt.Errorf("invalid webhook id: %w", err)
	}
	body, err := readWebhookPatchRequest(cmd, allowDescendants)
	if err != nil {
		return err
	}
	path := webhookAPIPath(namespacePath) + "/" + url.PathEscape(id.String())
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would PATCH %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var updated webhookRow
	if err := c.Patch(cmd.Context(), path, body, &updated); err != nil {
		return decorateWebhookError(err, namespacePath, "edit")
	}
	if updated.NamespacePath == "" {
		updated.NamespacePath = namespacePath
	}

	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, updated)
	case "yaml":
		return emitYAML(cmd, updated)
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated webhook %s for %s.\n", updated.ID, updated.NamespacePath)
		if updated.CleartextSecret != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret (save now; shown once): %s\n", updated.CleartextSecret)
		}
		return nil
	}
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

func runRepoWebhookDeliveryList(cmd *cobra.Command, args []string) error {
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	return runWebhookDeliveryList(cmd, ns+"/"+slug)
}

func runNamespaceWebhookDeliveryList(cmd *cobra.Command, args []string) error {
	return runWebhookDeliveryList(cmd, args[0])
}

func runWebhookDeliveryList(cmd *cobra.Command, namespacePath string) error {
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
	webhookID, _ := cmd.Flags().GetString("webhook-id")
	state, _ := cmd.Flags().GetString("state")

	var yamlAccum []webhookDeliveryRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if strings.TrimSpace(webhookID) != "" {
			q.Set("webhook_id", strings.TrimSpace(webhookID))
		}
		if strings.TrimSpace(state) != "" {
			q.Set("state", strings.TrimSpace(state))
		}
		var payload struct {
			Deliveries []webhookDeliveryRow `json:"deliveries"`
			Next       string               `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), webhookAPIPath(namespacePath)+"/deliveries?"+q.Encode(), &payload); err != nil {
			return decorateWebhookDeliveryError(err, namespacePath, "list")
		}
		rows := payload.Deliveries
		next := strings.TrimSpace(payload.Next)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []webhookDeliveryRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[webhookDeliveryRow](cmd)
			case "yaml":
				return emitYAML(cmd, []webhookDeliveryRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No deliveries for namespace '%s'.\n", namespacePath)
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
			_, _ = fmt.Fprintln(w, "ID\tEVENT_KIND\tSTATE\tATTEMPT_COUNT\tHTTP_STATUS\tCREATED_AT")
			for _, row := range rows {
				httpStatus := "-"
				if row.HTTPStatus != nil {
					httpStatus = fmt.Sprintf("%d", *row.HTTPStatus)
				}
				_, _ = fmt.Fprintf(
					w, "%s\t%s\t%s\t%d\t%s\t%s\n",
					row.ID, row.EventKind, row.State, row.AttemptCount, httpStatus, row.CreatedAt.Format(time.RFC3339),
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

func runRepoWebhookDeliveryGet(cmd *cobra.Command, args []string) error {
	namespacePath, id, err := parseRepoWebhookDeliveryIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runWebhookDeliveryGet(cmd, namespacePath, id)
}

func runNamespaceWebhookDeliveryGet(cmd *cobra.Command, args []string) error {
	return runWebhookDeliveryGet(cmd, args[0], args[1])
}

func runWebhookDeliveryGet(cmd *cobra.Command, namespacePath, rawID string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	path, err := webhookDeliveryAPIPath(namespacePath, rawID)
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var response webhookDeliveryResponse
	if err := c.Get(cmd.Context(), path, &response); err != nil {
		return decorateWebhookDeliveryError(err, namespacePath, "get")
	}
	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, response.Delivery)
	case "yaml":
		return emitYAML(cmd, response.Delivery)
	default:
		return emitWebhookDeliveryHuman(cmd, response.Delivery)
	}
}

func runRepoWebhookDeliveryRedeliver(cmd *cobra.Command, args []string) error {
	namespacePath, id, err := parseRepoWebhookDeliveryIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runWebhookDeliveryRedeliver(cmd, namespacePath, id)
}

func runNamespaceWebhookDeliveryRedeliver(cmd *cobra.Command, args []string) error {
	return runWebhookDeliveryRedeliver(cmd, args[0], args[1])
}

func runWebhookDeliveryRedeliver(cmd *cobra.Command, namespacePath, rawID string) error {
	path, err := webhookDeliveryAPIPath(namespacePath, rawID)
	if err != nil {
		return err
	}
	path += "/redeliver"
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would POST %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var response webhookDeliveryResponse
	if err := c.Post(cmd.Context(), path, nil, &response); err != nil {
		return decorateWebhookDeliveryError(err, namespacePath, "redeliver")
	}
	switch out := strings.TrimSpace(strings.ToLower(outputFlag(cmd))); out {
	case "json":
		return emitJSON(cmd, response.Delivery)
	case "yaml":
		return emitYAML(cmd, response.Delivery)
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Redelivered delivery %s for %s.\n", response.Delivery.ID, strings.Trim(strings.TrimSpace(namespacePath), "/"))
		if response.Delivery.EventKind != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Event kind: %s\n", response.Delivery.EventKind)
		}
		return nil
	}
}

func parseRepoWebhookDeliveryIDArgs(cmd *cobra.Command, args []string) (string, string, error) {
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
		return "", "", fmt.Errorf("expected <delivery-id> with -R/--repo, or <namespace>/<repo> <delivery-id>")
	}
}

func emitWebhookDeliveryHuman(cmd *cobra.Command, delivery webhookDeliveryRow) error {
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintln(w, "FIELD\tVALUE")
	_, _ = fmt.Fprintf(w, "ID\t%s\n", delivery.ID)
	_, _ = fmt.Fprintf(w, "Webhook ID\t%s\n", delivery.WebhookID)
	if delivery.WebhookName != "" {
		_, _ = fmt.Fprintf(w, "Webhook name\t%s\n", delivery.WebhookName)
	}
	if delivery.WebhookURL != "" {
		_, _ = fmt.Fprintf(w, "Webhook URL\t%s\n", delivery.WebhookURL)
	}
	_, _ = fmt.Fprintf(w, "Event ID\t%s\n", delivery.EventID)
	_, _ = fmt.Fprintf(w, "Event kind\t%s\n", delivery.EventKind)
	_, _ = fmt.Fprintf(w, "State\t%s\n", delivery.State)
	_, _ = fmt.Fprintf(w, "Attempt count\t%d\n", delivery.AttemptCount)
	if delivery.HTTPStatus != nil {
		_, _ = fmt.Fprintf(w, "HTTP status\t%d\n", *delivery.HTTPStatus)
	}
	_, _ = fmt.Fprintf(w, "Created\t%s\n", delivery.CreatedAt.Format(time.RFC3339))
	if delivery.LastAttemptAt != nil {
		_, _ = fmt.Fprintf(w, "Last attempt\t%s\n", delivery.LastAttemptAt.Format(time.RFC3339))
	}
	if delivery.DeliveredAt != nil {
		_, _ = fmt.Fprintf(w, "Delivered\t%s\n", delivery.DeliveredAt.Format(time.RFC3339))
	}
	if delivery.ErrorMessage != "" {
		_, _ = fmt.Fprintf(w, "Error\t%s\n", delivery.ErrorMessage)
	}
	return w.Flush()
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

func readWebhookPatchRequest(cmd *cobra.Command, allowDescendants bool) (webhookPatchRequest, error) {
	var body webhookPatchRequest
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		name = strings.TrimSpace(name)
		body.Name = &name
	}
	if cmd.Flags().Changed("url") {
		targetURL, _ := cmd.Flags().GetString("url")
		targetURL = strings.TrimSpace(targetURL)
		if targetURL == "" {
			return webhookPatchRequest{}, fmt.Errorf("--url cannot be empty")
		}
		body.TargetURL = &targetURL
	}
	if cmd.Flags().Changed("events") {
		events, _ := cmd.Flags().GetStringSlice("events")
		events = normaliseCLIEventKinds(events)
		if len(events) == 0 {
			return webhookPatchRequest{}, fmt.Errorf("--events must include at least one event kind")
		}
		body.EventKinds = events
	}
	if allowDescendants && cmd.Flags().Changed("include-descendants") {
		includeDescendants, _ := cmd.Flags().GetBool("include-descendants")
		body.IncludeDescendants = &includeDescendants
	}
	if cmd.Flags().Changed("active") {
		active, _ := cmd.Flags().GetBool("active")
		body.Active = &active
	}
	if cmd.Flags().Changed("rotate-secret") {
		rotateSecret, _ := cmd.Flags().GetBool("rotate-secret")
		body.RotateSecret = rotateSecret
	}
	if body.Name == nil && body.TargetURL == nil && len(body.EventKinds) == 0 &&
		body.IncludeDescendants == nil && body.Active == nil && !body.RotateSecret {
		return webhookPatchRequest{}, fmt.Errorf("at least one changing flag is required")
	}
	return body, nil
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

func decorateWebhookDeliveryError(err error, namespacePath, action string) error {
	if err == nil {
		return nil
	}
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return err
	}
	switch he.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: missing permission to %s webhook deliveries in %s", action, namespacePath)
	case http.StatusNotFound:
		return fmt.Errorf("webhook delivery not found in %s", namespacePath)
	case http.StatusBadRequest:
		return fmt.Errorf("invalid webhook delivery request for %s", namespacePath)
	}
	return err
}

func init() {
	repoWebhookCmd.AddCommand(repoWebhookListCmd)
	repoWebhookCmd.AddCommand(repoWebhookCreateCmd)
	repoWebhookCmd.AddCommand(repoWebhookGetCmd)
	repoWebhookCmd.AddCommand(repoWebhookEditCmd)
	repoWebhookCmd.AddCommand(repoWebhookDeleteCmd)
	repoWebhookDeliveriesCmd.AddCommand(repoWebhookDeliveryListCmd)
	repoWebhookDeliveriesCmd.AddCommand(repoWebhookDeliveryGetCmd)
	repoWebhookDeliveriesCmd.AddCommand(repoWebhookDeliveryRedeliverCmd)
	repoWebhookCmd.AddCommand(repoWebhookDeliveriesCmd)
	RepoCmd.AddCommand(repoWebhookCmd)

	namespaceWebhookCmd.AddCommand(namespaceWebhookListCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookCreateCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookGetCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookEditCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookDeleteCmd)
	namespaceWebhookDeliveriesCmd.AddCommand(namespaceWebhookDeliveryListCmd)
	namespaceWebhookDeliveriesCmd.AddCommand(namespaceWebhookDeliveryGetCmd)
	namespaceWebhookDeliveriesCmd.AddCommand(namespaceWebhookDeliveryRedeliverCmd)
	namespaceWebhookCmd.AddCommand(namespaceWebhookDeliveriesCmd)
	NamespaceCmd.AddCommand(namespaceWebhookCmd)

	addOutputFlag(
		repoWebhookListCmd, repoWebhookCreateCmd, repoWebhookGetCmd, repoWebhookEditCmd, repoWebhookDeleteCmd,
		namespaceWebhookListCmd, namespaceWebhookCreateCmd, namespaceWebhookGetCmd, namespaceWebhookEditCmd, namespaceWebhookDeleteCmd,
		repoWebhookDeliveryListCmd, repoWebhookDeliveryGetCmd, repoWebhookDeliveryRedeliverCmd,
		namespaceWebhookDeliveryListCmd, namespaceWebhookDeliveryGetCmd, namespaceWebhookDeliveryRedeliverCmd,
	)
	addPaginationFlags(repoWebhookListCmd, namespaceWebhookListCmd, repoWebhookDeliveryListCmd, namespaceWebhookDeliveryListCmd)
	addRepoFlag(repoWebhookListCmd, repoWebhookCreateCmd, repoWebhookGetCmd, repoWebhookEditCmd, repoWebhookDeleteCmd)
	addDryRunFlag(repoWebhookDeleteCmd, namespaceWebhookDeleteCmd)
	addDryRunFlag(repoWebhookEditCmd, namespaceWebhookEditCmd)
	addRepoFlag(repoWebhookDeliveryListCmd, repoWebhookDeliveryGetCmd, repoWebhookDeliveryRedeliverCmd)
	addDryRunFlag(repoWebhookDeliveryRedeliverCmd, namespaceWebhookDeliveryRedeliverCmd)
	for _, c := range []*cobra.Command{repoWebhookDeliveryListCmd, namespaceWebhookDeliveryListCmd} {
		c.Flags().String("webhook-id", "", "Filter deliveries by webhook ID")
		c.Flags().String("state", "", "Filter deliveries by state")
	}

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

	for _, c := range []*cobra.Command{repoWebhookEditCmd, namespaceWebhookEditCmd} {
		c.Flags().String("name", "", "Webhook name (empty clears the name)")
		c.Flags().String("url", "", "Target URL for webhook delivery")
		c.Flags().StringSlice("events", nil, "Comma-separated or repeated event kinds to deliver")
		c.Flags().Bool("active", true, "Set whether the webhook is active")
		c.Flags().Bool("rotate-secret", false, "Rotate the webhook secret")
		_ = c.RegisterFlagCompletionFunc("events", completeWebhookEvents)
	}
	namespaceWebhookEditCmd.Flags().Bool("include-descendants", false, "Set whether descendant namespace events are included")

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
	repoWebhookEditCmd.PostRun = func(cmd *cobra.Command, args []string) {
		namespacePath, _, err := parseRepoWebhookIDArgs(cmd, args)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(namespacePath))
		}
	}
	namespaceWebhookEditCmd.PostRun = func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			scheduleCompletionInvalidate(serverFlag(cmd), webhookCompletionKey(args[0]))
		}
	}
}
