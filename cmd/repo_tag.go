package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var repoTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "List, create, and delete repository tags",
}

var repoTagListCmd = &cobra.Command{
	Use:               "list [<namespace>/<repo>]",
	Short:             "List repository tags",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoTagList,
	ValidArgsFunction: completeRepoSlugs,
}

var repoTagCreateCmd = &cobra.Command{
	Use:   "create [<namespace>/<repo>] <name>",
	Short: "Create a lightweight or annotated tag",
	Long: `Create a lightweight tag by default. Supplying --message creates an
annotated tag, matching git's default behavior.`,
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoTagCreate,
	ValidArgsFunction: completeRepoTagCreateArgs,
}

var repoTagDeleteCmd = &cobra.Command{
	Use:               "delete [<namespace>/<repo>] <name>",
	Short:             "Delete a repository tag",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoTagDelete,
	ValidArgsFunction: completeRepoTagNames,
}

func runRepoTagList(cmd *cobra.Command, args []string) error {
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

	var yamlAccum []repoRefRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("kind", "tag")
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("after", cursor)
		}
		var payload struct {
			Refs       []repoRefRow `json:"refs"`
			NextCursor string       `json:"next_cursor"`
		}
		path := "/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/refs?" + q.Encode()
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return err
		}
		rows := payload.Refs
		next := strings.TrimSpace(payload.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			switch output {
			case "json":
				return emitJSON(cmd, []repoRefRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[repoRefRow](cmd)
			case "yaml":
				return emitYAML(cmd, []repoRefRow{})
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No tags for repository '%s/%s'.\n", ns, slug)
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
			_, _ = fmt.Fprintln(w, "TAG\tSHA\tDATE")
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", row.Name, shortSHA(row.SHA), formatRepoRefDate(row.Date))
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
			yamlAccum = []repoRefRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runRepoTagCreate(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ns, slug, name, err := parseRepoScopedNameArgs(cmd, args)
	if err != nil {
		return err
	}
	ref, _ := cmd.Flags().GetString("ref")
	message, _ := cmd.Flags().GetString("message")
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "create"); err != nil {
		return err
	}
	var payload struct {
		Name      string `json:"name"`
		SHA       string `json:"sha"`
		Annotated bool   `json:"annotated"`
	}
	path := "/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/refs/tags"
	if err := c.Post(cmd.Context(), path, map[string]string{
		"name":    name,
		"ref":     strings.TrimSpace(ref),
		"message": strings.TrimSpace(message),
	}, &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusConflict) {
			return fmt.Errorf("tag %q already exists in %s/%s", name, ns, slug)
		}
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("ref %q not found in %s/%s", ref, ns, slug)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, payload)
	}
	tagType := "lightweight"
	if payload.Annotated {
		tagType = "annotated"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s tag %s at %s.\n", tagType, payload.Name, shortSHA(payload.SHA))
	return nil
}

func runRepoTagDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ns, slug, name, err := parseRepoScopedNameArgs(cmd, args)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "delete"); err != nil {
		return err
	}
	path := "/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/refs/tags?name=" + url.QueryEscape(name)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("tag %q not found in %s/%s", name, ns, slug)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, map[string]string{"status": "deleted", "kind": "tag", "name": name})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted tag %s from %s/%s.\n", name, ns, slug)
	return nil
}

func completeRepoTagCreateArgs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if _, _, err := resolveRepoFromPosOrFlag(cmd, ""); err == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeRepoSlugs(cmd, args, "")
}

func completeRepoTagNames(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	return completeRepoExistingRefNames(cmd, args, completion.RepoTagKey, completion.FetchRepoTagNames)
}

func init() {
	repoTagCmd.AddCommand(repoTagListCmd)
	repoTagCmd.AddCommand(repoTagCreateCmd)
	repoTagCmd.AddCommand(repoTagDeleteCmd)

	addOutputFlag(repoTagListCmd, repoTagCreateCmd, repoTagDeleteCmd)
	addPaginationFlags(repoTagListCmd)
	addRepoFlag(repoTagListCmd, repoTagCreateCmd, repoTagDeleteCmd)
	addDryRunFlag(repoTagDeleteCmd)

	repoTagCreateCmd.Flags().String("ref", "", "Commit SHA, branch, or tag to point at (required)")
	repoTagCreateCmd.Flags().String("message", "", "Create an annotated tag with this message")
	_ = repoTagCreateCmd.MarkFlagRequired("ref")

	repoTagCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		ns, slug, _, err := parseRepoScopedNameArgs(cmd, cmd.Flags().Args())
		if err != nil {
			return
		}
		scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoTagKey(ns+"/"+slug))
	}
	repoTagDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		ns, slug, _, err := parseRepoScopedNameArgs(cmd, cmd.Flags().Args())
		if err != nil {
			return
		}
		scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoTagKey(ns+"/"+slug))
	}
}
