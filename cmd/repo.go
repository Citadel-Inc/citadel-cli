package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// RepoCmd is the top-level `citadel repo` command.
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories (create, list, get, delete)",
	Long:  `CRUD operations against the Citadel repository API.`,
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
	Use:   "get <namespace>/<repo>",
	Short: "Get details of a single repository",
	Long: `Fetches metadata for a single repository by its full path.

Examples:
  citadel-cli repo get myorg/myrepo
  citadel-cli repo get myorg/myrepo --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runRepoGet,
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <namespace>/<repo>",
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
  citadel-cli repo delete myorg/myrepo --yes`,
	Args: cobra.ExactArgs(1),
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
	output, _ := cmd.Flags().GetString("output")
	ns, _ := cmd.Flags().GetString("namespace")
	slug, _ := cmd.Flags().GetString("slug")
	if ns == "" {
		return fmt.Errorf("--namespace is required")
	}
	if slug == "" {
		return fmt.Errorf("--slug is required")
	}

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
		return emitJSON(row)
	}
	fmt.Printf("Created %s/%s (%s)\n", row.ParentSlug, row.Slug, row.Visibility)
	return nil
}

// listRepos fetches all repos in a parent namespace.
func listRepos(cmd *cobra.Command, ns string) ([]repoRow, error) {
	c, err := newAPIClient(cmd)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Repos []repoRow `json:"repos"`
	}
	if err := c.Get(cmd.Context(), "/namespaces/"+url.PathEscape(ns)+"/repos", &payload); err != nil {
		return nil, err
	}
	return payload.Repos, nil
}

func runRepoList(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output")
	ns, _ := cmd.Flags().GetString("namespace")
	if ns == "" {
		return fmt.Errorf("--namespace is required")
	}

	repos, err := listRepos(cmd, ns)
	if err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(repos)
	}
	if len(repos) == 0 {
		fmt.Printf("No repositories in namespace '%s'\n", ns)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tVISIBILITY\tBRANCH\tCREATED")
	for _, r := range repos {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Path, r.Visibility, r.DefaultBranch, r.CreatedAt)
	}
	return w.Flush()
}

func runRepoGet(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	ns, slug, err := splitRepoArg(args[0])
	if err != nil {
		return err
	}

	repos, err := listRepos(cmd, ns)
	if err != nil {
		return err
	}
	for _, r := range repos {
		if strings.EqualFold(r.Slug, slug) {
			if output == "json" {
				return emitJSON(r)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "Path:\t%s\n", r.Path)
			_, _ = fmt.Fprintf(w, "Visibility:\t%s\n", r.Visibility)
			_, _ = fmt.Fprintf(w, "Default branch:\t%s\n", r.DefaultBranch)
			if r.Description != "" {
				_, _ = fmt.Fprintf(w, "Description:\t%s\n", r.Description)
			}
			_, _ = fmt.Fprintf(w, "Created:\t%s\n", r.CreatedAt)
			return w.Flush()
		}
	}
	return fmt.Errorf("repository '%s/%s' not found", ns, slug)
}

func runRepoDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ns, slug, err := splitRepoArg(args[0])
	if err != nil {
		return err
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmSlug(yes, "delete", slug); err != nil {
		return err
	}

	// DELETE route has no /repos segment: /namespaces/{parent}/{repo}.
	if err := c.Delete(cmd.Context(), "/namespaces/"+url.PathEscape(ns)+"/"+url.PathEscape(slug)); err != nil {
		return err
	}
	fmt.Printf("Deleted %s/%s\n", ns, slug)
	return nil
}

func init() {
	RepoCmd.AddCommand(repoCreateCmd)
	RepoCmd.AddCommand(repoListCmd)
	RepoCmd.AddCommand(repoGetCmd)
	RepoCmd.AddCommand(repoDeleteCmd)

	repoCreateCmd.Flags().String("namespace", "", "Parent namespace slug (required)")
	repoCreateCmd.Flags().String("slug", "", "Repository slug (required)")
	repoCreateCmd.Flags().String("description", "", "Repository description")
	repoCreateCmd.Flags().String("visibility", "private", "Visibility: public or private")
	repoCreateCmd.Flags().String("default-branch", "main", "Default branch name")
	repoCreateCmd.Flags().Bool("init-with-readme", false, "Initialize with a README")
	repoCreateCmd.Flags().String("output", "", "Output format: json")

	repoListCmd.Flags().String("namespace", "", "Parent namespace slug (required)")
	repoListCmd.Flags().String("output", "", "Output format: json")

	repoGetCmd.Flags().String("output", "", "Output format: json")

	repoDeleteCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	repoDeleteCmd.Flags().String("output", "", "Output format: json")
}
