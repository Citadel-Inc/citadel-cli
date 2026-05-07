package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

// RepoCmd is the top-level `citadel repo` command.
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories and repository git workflows",
	Long:  `CRUD operations against the Citadel repository API, plus thin wrappers for clone/push/pull via the system git binary.`,
}

var repoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new repository",
	Long: `Creates a new repository under the given parent namespace.

Examples:
  citadel-cli repo create --namespace myorg --slug myrepo
  citadel-cli repo create --namespace myorg --slug myrepo --visibility public --init-with-readme`,
	RunE: runRepoCreate,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in a namespace",
	Long: `Lists all repositories owned by the authenticated user in the given namespace.

Examples:
  citadel-cli repo list --namespace myorg
  citadel-cli repo list --namespace myorg --output json`,
	RunE: runRepoList,
}

var repoGetCmd = &cobra.Command{
	Use:   "get [<namespace>/<repo>]",
	Short: "Get details of a single repository",
	Long: `Fetches metadata for a single repository by its full path.

If <namespace>/<repo> is omitted, the CLI resolves it from -R/--repo, ` + "`CITADEL_REPO`" + `,
or the git origin remote in the current directory (Citadel hosts only).

Examples:
  citadel-cli repo get myorg/myrepo
  citadel-cli repo get -R myorg/myrepo
  citadel-cli repo get --output json
  citadel-cli repo get`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runRepoGet,
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete [<namespace>/<repo>]",
	Short: "Hard-purge a repository",
	Long: `Hard-purges a repository: drops the repo namespace + every FK-cascaded
child (kg_files, kg_symbols, kg_file_content, kg_edges, repos, repo_submodule_pins,
pg_edges, namespace_pins (kind=repo), repo_topics, repo_stars, issues, milestones,
issue_labels, issue_close_refs, namespace_grants, namespace_profiles, …)
in one tx, removes the bare repo dir on disk, and inserts a slug-hold
tombstone in namespace_aliases (default 30 + 30 days). Search index
(searchable_namespaces) refreshes after commit.

Requires typed-slug confirmation unless --yes is set.

Examples:
  citadel-cli repo delete myorg/myrepo
  citadel-cli repo delete -R myorg/myrepo --yes
  citadel-cli repo delete --yes`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runRepoDelete,
}

type repoRow struct {
	NamespaceID   string `json:"namespace_id"`
	ParentSlug    string `json:"parent_slug"`
	Slug          string `json:"slug"`
	Visibility    string `json:"visibility"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description,omitempty"`
	Path          string `json:"path"`
	CreatedAt     string `json:"created_at"`
	GitSSHRemote  string `json:"git_ssh_remote,omitempty"`
}

// splitRepoArg parses the canonical "<namespace>/<repo>" cobra arg shape.
func splitRepoArg(arg string) (ns, slug string, err error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("argument must be <namespace>/<repo>")
	}
	return parts[0], parts[1], nil
}

func runRepoCreate(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := outputFlag(cmd)
	ns, _ := cmd.Flags().GetString("namespace")
	slug, _ := cmd.Flags().GetString("slug")

	desc, _ := cmd.Flags().GetString("description")
	visibility, _ := cmd.Flags().GetString("visibility")
	defaultBranch, _ := cmd.Flags().GetString("default-branch")
	initReadme, _ := cmd.Flags().GetBool("init-with-readme")

	reqBody := struct {
		Slug           string  `json:"slug"`
		Description    *string `json:"description,omitempty"`
		DefaultBranch  *string `json:"default_branch,omitempty"`
		Visibility     string  `json:"visibility"`
		InitWithReadme bool    `json:"init_with_readme"`
	}{
		Slug:           slug,
		Visibility:     visibility,
		InitWithReadme: initReadme,
	}
	if desc != "" {
		reqBody.Description = &desc
	}
	if defaultBranch != "" {
		reqBody.DefaultBranch = &defaultBranch
	}

	var row repoRow
	if err := c.Post(cmd.Context(), "/namespaces/"+url.PathEscape(ns)+"/repos", reqBody, &row); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(cmd, row)
	}
	fmt.Printf("Created %s/%s (%s)\n", row.ParentSlug, row.Slug, row.Visibility)
	return nil
}

func runRepoList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ns, _ := cmd.Flags().GetString("namespace")
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
		return runRepoListWatch(cmd, c, ns, limit, cursor, all)
	}
	if err := validateDescCursor(cursor); err != nil {
		return fmt.Errorf("invalid --cursor: %w", err)
	}

	var yamlAccum []repoRow
	csvHdr := false
	first := true
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Repos      []repoRow `json:"repos"`
			NextCursor string    `json:"next_cursor"`
		}
		path := "/namespaces/" + url.PathEscape(ns) + "/repos?" + q.Encode()
		if err := c.Get(cmd.Context(), path, &payload); err != nil {
			return err
		}
		rows := payload.Repos
		next := strings.TrimSpace(payload.NextCursor)

		if len(rows) == 0 && cursor != "" && next == "" {
			return nil
		}
		if first && len(rows) == 0 && cursor == "" {
			empty := fmt.Sprintf("No repositories in namespace '%s'", ns)
			switch output {
			case "json":
				return emitJSON(cmd, []repoRow{})
			case "ndjson":
				return nil
			case "csv":
				return emitCSVHeaderOnly[repoRow](cmd)
			case "yaml":
				return emitYAML(cmd, []repoRow{})
			default:
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), empty)
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
			_, _ = fmt.Fprintln(w, "PATH\tVISIBILITY\tBRANCH\tCREATED")
			for _, r := range rows {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Path, r.Visibility, r.DefaultBranch, r.CreatedAt)
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
			yamlAccum = []repoRow{}
		}
		return emitYAML(cmd, yamlAccum)
	}
	return nil
}

func runRepoGet(cmd *cobra.Command, args []string) error {
	if err := validateGetOutput(outputFlag(cmd)); err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	pos := ""
	if len(args) > 0 {
		pos = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, pos)
	if err != nil {
		return err
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var r repoRow
	path := "/namespaces/" + url.PathEscape(ns) + "/" + url.PathEscape(slug)
	if err := c.Get(cmd.Context(), path, &r); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("repository '%s/%s' not found", ns, slug)
		}
		return err
	}

	return emitOne(cmd, output, r, func(w *tabwriter.Writer, r repoRow) {
		_, _ = fmt.Fprintf(w, "Path:\t%s\n", r.Path)
		_, _ = fmt.Fprintf(w, "Visibility:\t%s\n", r.Visibility)
		_, _ = fmt.Fprintf(w, "Default branch:\t%s\n", r.DefaultBranch)
		if r.Description != "" {
			_, _ = fmt.Fprintf(w, "Description:\t%s\n", r.Description)
		}
		_, _ = fmt.Fprintf(w, "Created:\t%s\n", r.CreatedAt)
	})
}

func runRepoDelete(cmd *cobra.Command, args []string) error {
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

	if dryRunFlag(cmd) {
		fmt.Printf("Would DELETE /namespaces/%s/%s (skipped; --dry-run)\n", ns, slug)
		return nil
	}

	if err := confirmSlug(yesFlag(cmd), "delete", slug); err != nil {
		return err
	}

	// DELETE route has no /repos segment: /namespaces/{parent}/{repo}.
	if err := c.Delete(cmd.Context(), "/namespaces/"+url.PathEscape(ns)+"/"+url.PathEscape(slug)); err != nil {
		return err
	}
	fmt.Printf("Deleted %s/%s\n", ns, slug)
	return nil
}

func completeRepoSlugs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ns, err := ResolveRepoNamespaceForCompletion(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.RepoKey(ns), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		return completion.FetchRepoSlugs(ctx, c, ns)
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	RepoCmd.AddCommand(repoCreateCmd)
	RepoCmd.AddCommand(repoListCmd)
	RepoCmd.AddCommand(repoGetCmd)
	RepoCmd.AddCommand(repoDeleteCmd)
	RepoCmd.AddCommand(repoBranchCmd)
	RepoCmd.AddCommand(repoTagCmd)
	RepoCmd.AddCommand(repoCommitCmd)
	RepoCmd.AddCommand(repoBrowseCmd)
	RepoCmd.AddCommand(repoTopicCmd)
	RepoCmd.AddCommand(repoInsightsCmd)

	addOutputFlag(repoCreateCmd, repoListCmd, repoGetCmd, repoDeleteCmd)
	addPaginationFlags(repoListCmd)
	addWatchFlag(repoListCmd)
	addRepoFlag(repoGetCmd, repoDeleteCmd)
	addYesFlag(repoDeleteCmd)
	addDryRunFlag(repoDeleteCmd)

	repoCreateCmd.Flags().String("namespace", "", "Parent namespace slug (required)")
	repoCreateCmd.Flags().String("slug", "", "Repository slug (required)")
	repoCreateCmd.Flags().String("description", "", "Repository description")
	repoCreateCmd.Flags().String("visibility", "private", "Visibility: public or private")
	repoCreateCmd.Flags().String("default-branch", "main", "Default branch name")
	repoCreateCmd.Flags().Bool("init-with-readme", false, "Initialize with a README")
	_ = repoCreateCmd.MarkFlagRequired("namespace")
	_ = repoCreateCmd.MarkFlagRequired("slug")

	repoListCmd.Flags().String("namespace", "", "Parent namespace slug (required)")
	_ = repoListCmd.MarkFlagRequired("namespace")

	repoGetCmd.ValidArgsFunction = completeRepoSlugs
	repoDeleteCmd.ValidArgsFunction = completeRepoSlugs

	repoCreateCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		ns, _ := cmd.Flags().GetString("namespace")
		scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoKey(strings.TrimSpace(ns)))
	}
	repoDeleteCmd.PostRun = func(cmd *cobra.Command, args []string) {
		pos := ""
		if len(args) > 0 {
			pos = args[0]
		}
		ns, _, err := resolveRepoFromPosOrFlag(cmd, pos)
		if err != nil {
			return
		}
		scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoKey(ns))
	}
}
