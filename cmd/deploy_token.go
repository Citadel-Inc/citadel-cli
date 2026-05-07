package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var repoDeployTokenCmd = &cobra.Command{
	Use:     "deploy-token",
	Aliases: []string{"deploy-tokens"},
	Short:   "Manage deploy tokens for a repository",
}

var repoDeployTokenListCmd = &cobra.Command{
	Use:               "list [<namespace>/<repo>]",
	Short:             "List deploy tokens for a repository",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoDeployTokenList,
	ValidArgsFunction: completeRepoSlugs,
}

var repoDeployTokenCreateCmd = &cobra.Command{
	Use:               "create [<namespace>/<repo>]",
	Short:             "Create a deploy token for a repository",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoDeployTokenCreate,
	ValidArgsFunction: completeRepoSlugs,
}

var repoDeployTokenRevokeCmd = &cobra.Command{
	Use:               "revoke [<namespace>/<repo>] <id>",
	Short:             "Revoke a repository deploy token",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoDeployTokenRevoke,
	ValidArgsFunction: completeRepoDeployTokenIDs,
}

var namespaceDeployTokenCmd = &cobra.Command{
	Use:     "deploy-token",
	Aliases: []string{"deploy-tokens"},
	Short:   "Manage deploy tokens for a namespace",
}

var namespaceDeployTokenListCmd = &cobra.Command{
	Use:               "list <slug>",
	Short:             "List deploy tokens for a namespace",
	Args:              cobra.ExactArgs(1),
	RunE:              runNamespaceDeployTokenList,
	ValidArgsFunction: completeOrgNamespaceSlugs,
}

var namespaceDeployTokenCreateCmd = &cobra.Command{
	Use:               "create <slug>",
	Short:             "Create a deploy token for a namespace",
	Args:              cobra.ExactArgs(1),
	RunE:              runNamespaceDeployTokenCreate,
	ValidArgsFunction: completeOrgNamespaceSlugs,
}

var namespaceDeployTokenRevokeCmd = &cobra.Command{
	Use:               "revoke <slug> <id>",
	Short:             "Revoke a namespace deploy token",
	Args:              cobra.ExactArgs(2),
	RunE:              runNamespaceDeployTokenRevoke,
	ValidArgsFunction: completeNamespaceDeployTokenIDs,
}

type deployTokenRow struct {
	ID            string     `json:"id"`
	NamespaceID   string     `json:"namespace_id,omitempty"`
	NamespacePath string     `json:"namespace_path,omitempty"`
	Name          string     `json:"name,omitempty"`
	Scopes        []string   `json:"scopes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

type deployTokenWithCleartext struct {
	deployTokenRow
	CleartextToken string `json:"cleartext_token"`
}

func (r deployTokenRow) CSVHeader() []string {
	return []string{"id", "name", "namespace_path", "created_at", "expires_at", "revoked_at"}
}

func (r deployTokenRow) CSVRecord() []string {
	return []string{
		r.ID,
		r.Name,
		r.NamespacePath,
		r.CreatedAt.Format(time.RFC3339),
		formatTimePtr(r.ExpiresAt),
		formatTimePtr(r.RevokedAt),
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func deployTokenAPIPath(namespacePath string) string {
	return "/namespaces/" + url.PathEscape(strings.TrimSpace(namespacePath)) + "/deploy-tokens"
}

func parseRepoDeployTokenIDArgs(cmd *cobra.Command, args []string) (string, string, error) {
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

func parseExpiresFlag(cmd *cobra.Command) (*int64, error) {
	expiresStr, _ := cmd.Flags().GetString("expires")
	expiresStr = strings.TrimSpace(expiresStr)
	if expiresStr == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(expiresStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --expires: %w", err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("--expires must be greater than zero")
	}
	sec := int64(d.Seconds())
	return &sec, nil
}

func runRepoDeployTokenList(cmd *cobra.Command, args []string) error {
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	return runDeployTokenList(cmd, ns+"/"+slug)
}

func runNamespaceDeployTokenList(cmd *cobra.Command, args []string) error {
	return runDeployTokenList(cmd, args[0])
}

func runDeployTokenList(cmd *cobra.Command, namespacePath string) error {
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
	if err := validateWatchOutput(cmd); err != nil {
		return err
	}
	if watchFlag(cmd) {
		if err := validateDescCursor(cursor); err != nil {
			return fmt.Errorf("invalid --cursor: %w", err)
		}
		return runDeployTokenListWatch(cmd, c, namespacePath, limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []deployTokenRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Tokens []deployTokenRow `json:"deploy_tokens"`
			Next   string           `json:"next_cursor"`
		}
		if err := c.Get(cmd.Context(), deployTokenAPIPath(namespacePath)+"?"+q.Encode(), &payload); err != nil {
			return decorateDeployTokenError(err, namespacePath, "list")
		}
		rows := payload.Tokens
		next := strings.TrimSpace(payload.Next)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []deployTokenRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[deployTokenRow](cmd)
			case "yaml":
				return emitYAML(cmd, []deployTokenRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No deploy tokens for namespace '%s'.\n", namespacePath)
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
			_, _ = fmt.Fprintln(w, "ID\tNAME\tCREATED\tEXPIRES\tREVOKED")
			for _, row := range rows {
				name := row.Name
				if name == "" {
					name = "—"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					row.ID,
					name,
					row.CreatedAt.Format("2006-01-02 15:04:05"),
					formatTimePtr(row.ExpiresAt),
					formatTimePtr(row.RevokedAt),
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
			yamlAccum = []deployTokenRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runRepoDeployTokenCreate(cmd *cobra.Command, args []string) error {
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}
	return runDeployTokenCreate(cmd, ns+"/"+slug)
}

func runNamespaceDeployTokenCreate(cmd *cobra.Command, args []string) error {
	return runDeployTokenCreate(cmd, args[0])
}

func runDeployTokenCreate(cmd *cobra.Command, namespacePath string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "create"); err != nil {
		return err
	}
	expiresIn, err := parseExpiresFlag(cmd)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)

	body := struct {
		Name             string `json:"name,omitempty"`
		ExpiresInSeconds *int64 `json:"expires_in_seconds,omitempty"`
	}{
		Name:             name,
		ExpiresInSeconds: expiresIn,
	}

	var created deployTokenWithCleartext
	if err := c.Post(cmd.Context(), deployTokenAPIPath(namespacePath), body, &created); err != nil {
		return decorateDeployTokenError(err, namespacePath, "create")
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, created)
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Created deploy token for %s\n", namespacePath)
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  id:    %s\n", created.ID)
	if created.Name != "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  name:  %s\n", created.Name)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), created.CleartextToken)
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "(deploy token printed once above — store it securely)")
	return nil
}

func runRepoDeployTokenRevoke(cmd *cobra.Command, args []string) error {
	namespacePath, tokenID, err := parseRepoDeployTokenIDArgs(cmd, args)
	if err != nil {
		return err
	}
	return runDeployTokenRevoke(cmd, namespacePath, tokenID)
}

func runNamespaceDeployTokenRevoke(cmd *cobra.Command, args []string) error {
	return runDeployTokenRevoke(cmd, args[0], args[1])
}

func runDeployTokenRevoke(cmd *cobra.Command, namespacePath, tokenID string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	tokenID = strings.TrimSpace(tokenID)
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "revoke"); err != nil {
		return err
	}
	path := deployTokenAPIPath(namespacePath) + "/" + url.PathEscape(tokenID)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		return decorateDeployTokenError(err, namespacePath, "revoke", tokenID)
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, map[string]string{"status": "revoked", "id": tokenID, "namespace_path": namespacePath})
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Deploy token %s revoked for %s.\n", tokenID, namespacePath)
	return nil
}

func decorateDeployTokenError(err error, namespacePath, action string, extra ...string) error {
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return err
	}
	switch he.StatusCode {
	case http.StatusNotFound:
		if len(extra) > 0 && strings.TrimSpace(extra[0]) != "" {
			return fmt.Errorf("deploy token %s not found in %s", strings.TrimSpace(extra[0]), namespacePath)
		}
		return fmt.Errorf("namespace %s not found", namespacePath)
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: missing permission to %s deploy tokens in %s", action, namespacePath)
	}
	return err
}

func completeRepoDeployTokenIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, "")
	if err == nil && len(args) == 0 {
		return lookupDeployTokenIDs(cmd, ns+"/"+slug)
	}
	if len(args) == 0 {
		return completeRepoSlugs(cmd, args, "")
	}
	ns, slug, err = resolveRepoFromPosOrFlag(cmd, args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return lookupDeployTokenIDs(cmd, ns+"/"+slug)
}

func completeNamespaceDeployTokenIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeOrgNamespaceSlugs(cmd, args, "")
	case 1:
		return lookupDeployTokenIDs(cmd, args[0])
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func lookupDeployTokenIDs(cmd *cobra.Command, namespacePath string) ([]string, cobra.ShellCompDirective) {
	namespacePath = strings.Trim(strings.TrimSpace(namespacePath), "/")
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.DeployTokenKey(namespacePath), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		return completion.FetchNamespaceDeployTokenIDs(ctx, c, namespacePath)
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func runDeployTokenListWatch(cmd *cobra.Command, c *apiclient.Client, namespacePath string, limit int, cursor string, all bool) error {
	path := deployTokenAPIPath(namespacePath) + "?" + sseWatchQuery(limit, cursor, all, nil)
	h, err := newWatchSSEHandler(cmd, watchDeployTokens, watchTableCtx{})
	if err != nil {
		return err
	}
	return consumeSSEWatch(cmd, c, path, h)
}

func init() {
	repoDeployTokenCmd.AddCommand(repoDeployTokenListCmd, repoDeployTokenCreateCmd, repoDeployTokenRevokeCmd)
	namespaceDeployTokenCmd.AddCommand(namespaceDeployTokenListCmd, namespaceDeployTokenCreateCmd, namespaceDeployTokenRevokeCmd)
	RepoCmd.AddCommand(repoDeployTokenCmd)
	NamespaceCmd.AddCommand(namespaceDeployTokenCmd)

	addOutputFlag(repoDeployTokenListCmd, repoDeployTokenCreateCmd, repoDeployTokenRevokeCmd,
		namespaceDeployTokenListCmd, namespaceDeployTokenCreateCmd, namespaceDeployTokenRevokeCmd)
	addPaginationFlags(repoDeployTokenListCmd, namespaceDeployTokenListCmd)
	addWatchFlag(repoDeployTokenListCmd, namespaceDeployTokenListCmd)
	addDryRunFlag(repoDeployTokenRevokeCmd, namespaceDeployTokenRevokeCmd)
	addRepoFlag(repoDeployTokenListCmd, repoDeployTokenCreateCmd, repoDeployTokenRevokeCmd)

	for _, c := range []*cobra.Command{repoDeployTokenCreateCmd, namespaceDeployTokenCreateCmd} {
		c.Flags().String("name", "", "Optional human-readable label for the token")
		c.Flags().String("expires", "", "Expiration duration (e.g. 24h)")
	}

	repoDeployTokenRevokeCmd.PostRun = func(cmd *cobra.Command, args []string) {
		namespacePath, _, err := parseRepoDeployTokenIDArgs(cmd, args)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), completion.DeployTokenKey(namespacePath))
		}
	}
	namespaceDeployTokenCreateCmd.PostRun = func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			scheduleCompletionInvalidate(serverFlag(cmd), completion.DeployTokenKey(args[0]))
		}
	}
	namespaceDeployTokenRevokeCmd.PostRun = func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			scheduleCompletionInvalidate(serverFlag(cmd), completion.DeployTokenKey(args[0]))
		}
	}
	repoDeployTokenCreateCmd.PostRun = func(cmd *cobra.Command, args []string) {
		pos := ""
		if len(args) > 0 {
			pos = args[0]
		}
		ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
		if err == nil {
			scheduleCompletionInvalidate(serverFlag(cmd), completion.DeployTokenKey(ns+"/"+slug))
		}
	}
}
