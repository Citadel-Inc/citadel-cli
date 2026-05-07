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

var repoBranchCmd = &cobra.Command{
	Use:   "branch",
	Short: "List, delete, and change the default branch for a repository",
}

var repoBranchListCmd = &cobra.Command{
	Use:               "list [<namespace>/<repo>]",
	Short:             "List repository branches",
	Args:              cobra.RangeArgs(0, 1),
	RunE:              runRepoBranchList,
	ValidArgsFunction: completeRepoSlugs,
}

var repoBranchDeleteCmd = &cobra.Command{
	Use:               "delete [<namespace>/<repo>] <name>",
	Short:             "Delete a repository branch",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoBranchDelete,
	ValidArgsFunction: completeRepoBranchNames,
}

var repoBranchSetDefaultCmd = &cobra.Command{
	Use:               "set-default [<namespace>/<repo>] <name>",
	Short:             "Change the default branch for a repository",
	Args:              cobra.RangeArgs(1, 2),
	RunE:              runRepoBranchSetDefault,
	ValidArgsFunction: completeRepoBranchNames,
}

func runRepoBranchList(cmd *cobra.Command, args []string) error {
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
		q.Set("kind", "branch")
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No branches for repository '%s/%s'.\n", ns, slug)
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
			_, _ = fmt.Fprintln(w, "BRANCH\tSHA\tDATE")
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

func runRepoBranchDelete(cmd *cobra.Command, args []string) error {
	ns, slug, name, err := parseRepoScopedNameArgs(cmd, args)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "delete"); err != nil {
		return err
	}
	path := "/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/refs/branches?name=" + url.QueryEscape(name)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("branch %q not found in %s/%s", name, ns, slug)
		}
		if apiclient.IsStatus(err, http.StatusConflict) {
			return fmt.Errorf("branch %q is the default branch for %s/%s; set a new default first", name, ns, slug)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, map[string]string{"status": "deleted", "kind": "branch", "name": name})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted branch %s from %s/%s.\n", name, ns, slug)
	return nil
}

func runRepoBranchSetDefault(cmd *cobra.Command, args []string) error {
	ns, slug, name, err := parseRepoScopedNameArgs(cmd, args)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	if err := validateMutationOutput(output, "set-default"); err != nil {
		return err
	}
	path := "/namespaces/" + url.PathEscape(ns) + "/repos/" + url.PathEscape(slug) + "/default-branch"
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would PATCH %s name=%q (skipped; --dry-run)\n", path, name)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var payload struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.Patch(cmd.Context(), path, map[string]string{"name": name}, &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("branch %q not found in %s/%s", name, ns, slug)
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(output), "json") {
		return emitJSON(cmd, payload)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default branch for %s/%s is now %s.\n", ns, slug, payload.DefaultBranch)
	return nil
}

func completeRepoBranchNames(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	return completeRepoExistingRefNames(cmd, args, completion.RepoBranchKey, completion.FetchRepoBranchNames)
}

func init() {
	repoBranchCmd.AddCommand(repoBranchListCmd)
	repoBranchCmd.AddCommand(repoBranchDeleteCmd)
	repoBranchCmd.AddCommand(repoBranchSetDefaultCmd)

	addOutputFlag(repoBranchListCmd, repoBranchDeleteCmd, repoBranchSetDefaultCmd)
	addPaginationFlags(repoBranchListCmd)
	addRepoFlag(repoBranchListCmd, repoBranchDeleteCmd, repoBranchSetDefaultCmd)
	addDryRunFlag(repoBranchDeleteCmd, repoBranchSetDefaultCmd)

	repoBranchDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		ns, slug, _, err := parseRepoScopedNameArgs(cmd, cmd.Flags().Args())
		if err != nil {
			return
		}
		scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoBranchKey(ns+"/"+slug))
	}
}
