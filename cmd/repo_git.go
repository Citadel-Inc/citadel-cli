package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

var (
	execCommandContext = exec.CommandContext
	execLookPath       = exec.LookPath
	stdinIsTerminal    = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)

var repoCloneCmd = &cobra.Command{
	Use:   "clone <namespace>/<repo> [local-dir]",
	Short: "Clone a repository over SSH",
	Long: `Runs the system git binary and clones the repository over SSH.

Authentication is handled by SSH keys registered with Citadel
(see: citadel-cli ssh-key add).

Examples:
  citadel-cli repo clone myorg/myrepo
  citadel-cli repo clone myorg/myrepo ./local-dir`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeRepoPaths,
	RunE:              runRepoClone,
}

var repoPushCmd = &cobra.Command{
	Use:   "push [<namespace>/<repo>]",
	Short: "Push the current checkout over SSH",
	Long: `Runs the system git binary in the current checkout over SSH.

When no repo path is passed, the CLI infers the target from -R/--repo, CITADEL_REPO,
or the configured git remote URL. If the target repo does not exist on Citadel yet,
the CLI prompts to create it first; pass --create to skip that prompt.

Authentication is handled by SSH keys registered with Citadel
(see: citadel-cli ssh-key add).

Examples:
  citadel-cli repo push
  citadel-cli repo push --remote upstream
  citadel-cli repo push --create`,
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: completeRepoPaths,
	RunE:              runRepoPush,
}

var repoPullCmd = &cobra.Command{
	Use:   "pull [<namespace>/<repo>]",
	Short: "Pull into the current checkout over SSH",
	Long: `Runs the system git binary in the current checkout over SSH.

When no repo path is passed, the CLI infers the target from -R/--repo, CITADEL_REPO,
or the configured git remote URL.

Authentication is handled by SSH keys registered with Citadel
(see: citadel-cli ssh-key add).

Examples:
  citadel-cli repo pull
  citadel-cli repo pull --remote upstream`,
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: completeRepoPaths,
	RunE:              runRepoPull,
}

func runRepoClone(cmd *cobra.Command, args []string) error {
	if err := ensureGitOnPath(); err != nil {
		return err
	}
	_, serverURL, err := loadGitConfig(cmd)
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ns, slug, err := splitRepoArg(strings.TrimSpace(args[0]))
	if err != nil {
		return err
	}
	sshRemote, err := resolveRepoSSHRemote(cmd.Context(), c, serverURL, ns, slug)
	if err != nil {
		return err
	}
	gitArgs := []string{"clone", sshRemote}
	if len(args) == 2 {
		gitArgs = append(gitArgs, strings.TrimSpace(args[1]))
	}
	if err := runGit(cmd, "", gitArgs...); err != nil {
		return err
	}
	localDir := strings.TrimSpace(filepath.Base(slug))
	if len(args) == 2 && strings.TrimSpace(args[1]) != "" {
		localDir = strings.TrimSpace(args[1])
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), localDir)
	return nil
}

func runRepoPush(cmd *cobra.Command, args []string) error {
	return runRepoSync(cmd, args, http.MethodPut)
}

func runRepoPull(cmd *cobra.Command, args []string) error {
	return runRepoSync(cmd, args, http.MethodGet)
}

func runRepoSync(cmd *cobra.Command, args []string, method string) error {
	if err := ensureGitOnPath(); err != nil {
		return err
	}
	_, serverURL, err := loadGitConfig(cmd)
	if err != nil {
		return err
	}
	target, explicit, err := resolveGitTarget(cmd, args)
	if err != nil {
		return err
	}
	remote, _ := cmd.Flags().GetString("remote")
	remote = strings.TrimSpace(remote)
	if remote == "" {
		remote = "origin"
	}
	if explicit && remote != "origin" {
		return fmt.Errorf("--remote cannot be used with an explicit repo path; omit --remote or omit the repo path")
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var sshRemote string
	if method == http.MethodPut {
		row, rerr := ensureRemoteRepoForPush(cmd, c, target.ns, target.slug)
		if rerr != nil {
			return rerr
		}
		if explicit {
			if row != nil && row.GitSSHRemote != "" {
				sshRemote = row.GitSSHRemote
			} else {
				sshRemote, err = gitSSHRemote(serverURL, target.ns, target.slug)
				if err != nil {
					return err
				}
			}
		}
	} else if explicit {
		sshRemote, err = resolveRepoSSHRemote(cmd.Context(), c, serverURL, target.ns, target.slug)
		if err != nil {
			return err
		}
	}

	var gitArgs []string
	switch method {
	case http.MethodPut:
		gitArgs = []string{"push"}
	case http.MethodGet:
		gitArgs = []string{"pull"}
	default:
		return fmt.Errorf("unsupported git sync method %q", method)
	}
	if explicit {
		branch := currentBranchOrDefault(cmd.Context(), "HEAD")
		switch method {
		case http.MethodPut:
			gitArgs = append(gitArgs, "--set-upstream", sshRemote, branch)
		case http.MethodGet:
			gitArgs = append(gitArgs, sshRemote, branch)
		}
	} else {
		gitArgs = append(gitArgs, remote)
	}
	return runGit(cmd, "", gitArgs...)
}

type gitTarget struct {
	ns   string
	slug string
}

func resolveGitTarget(cmd *cobra.Command, args []string) (gitTarget, bool, error) {
	positional := ""
	if len(args) > 0 {
		positional = strings.TrimSpace(args[0])
	}
	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)
	if repoFlag != "" || positional != "" {
		ns, slug, err := resolveRepoFromPosOrFlag(cmd, positional)
		if err != nil {
			return gitTarget{}, false, err
		}
		return gitTarget{ns: ns, slug: slug}, true, nil
	}

	remote, _ := cmd.Flags().GetString("remote")
	remote = strings.TrimSpace(remote)
	if remote == "" {
		remote = "origin"
	}
	wd, err := os.Getwd()
	if err != nil {
		return gitTarget{}, false, err
	}
	rawURL, err := gitRemoteURL(cmd.Context(), wd, remote)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return gitTarget{}, false, fmt.Errorf("git is not available on PATH")
		}
		return gitTarget{}, false, fmt.Errorf("could not resolve git remote %q: %w", remote, err)
	}
	ns, slug, err := parseOriginIntoRepo(rawURL, mergeCitadelHosts())
	if err != nil {
		return gitTarget{}, false, fmt.Errorf("could not infer repo from git remote %q: %w", remote, err)
	}
	return gitTarget{ns: ns, slug: slug}, false, nil
}

func ensureRemoteRepoForPush(cmd *cobra.Command, c *apiclient.Client, ns, slug string) (*repoRow, error) {
	var row repoRow
	path := "/namespaces/" + url.PathEscape(ns) + "/" + url.PathEscape(slug)
	err := c.Get(cmd.Context(), path, &row)
	if err == nil {
		return &row, nil
	}
	if !apiclient.IsStatus(err, http.StatusNotFound) {
		return nil, err
	}

	create, _ := cmd.Flags().GetBool("create")
	repoPath := ns + "/" + slug
	if err := confirmCreateRepo(create, repoPath); err != nil {
		return nil, err
	}
	defaultBranch := currentBranchOrDefault(cmd.Context(), "main")
	reqBody := struct {
		Slug           string  `json:"slug"`
		Description    *string `json:"description,omitempty"`
		DefaultBranch  *string `json:"default_branch,omitempty"`
		Visibility     string  `json:"visibility"`
		InitWithReadme bool    `json:"init_with_readme"`
	}{
		Slug:           slug,
		DefaultBranch:  &defaultBranch,
		Visibility:     "private",
		InitWithReadme: false,
	}
	var created repoRow
	if err := c.Post(cmd.Context(), "/namespaces/"+url.PathEscape(ns)+"/repos", reqBody, &created); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Created %s/%s (%s).\n", created.ParentSlug, created.Slug, created.Visibility)
	scheduleCompletionInvalidate(serverFlag(cmd), completion.RepoKey(ns))
	return &created, nil
}

func confirmCreateRepo(force bool, repoPath string) error {
	if force {
		return nil
	}
	if !stdinIsTerminal() {
		return fmt.Errorf("repository %s does not exist; re-run with --create to create it before pushing", repoPath)
	}
	fmt.Fprintf(os.Stderr, "Repository %s does not exist. Create it now? [y/N]: ", repoPath)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		return fmt.Errorf("repository %s does not exist; operation aborted", repoPath)
	}
	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("repository %s does not exist; operation aborted", repoPath)
	}
}

func loadGitConfig(cmd *cobra.Command) (clicfg.Config, string, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return clicfg.Config{}, "", err
	}
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return clicfg.Config{}, "", errors.New("not authenticated; run 'citadel-cli auth login' first")
	}
	return cfg, cfg.ResolveServerURL(serverFlag(cmd)), nil
}

func ensureGitOnPath() error {
	if _, err := execLookPath("git"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("git is not available on PATH")
		}
		return err
	}
	return nil
}

func gitRemoteURL(ctx context.Context, dir, remote string) (string, error) {
	cmd := execCommandContext(ctx, "git", "remote", "get-url", remote)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func currentBranchOrDefault(ctx context.Context, fallback string) string {
	cmd := execCommandContext(ctx, "git", "symbolic-ref", "--quiet", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return fallback
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return fallback
	}
	return branch
}

// canonicalSSHHost maps known Citadel API hostnames to their canonical git SSH hostname.
func canonicalSSHHost(serverURL string) (string, error) {
	raw := strings.TrimSpace(serverURL)
	if raw == "" {
		return "src.land", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	switch host {
	case "api.src.land", "mcp.src.land", "git.src.land", "src.land":
		return "src.land", nil
	}
	return host, nil
}

// gitSSHRemote constructs a git SSH remote URL (git@host:ns/slug.git) from
// the configured server URL and a repo namespace/slug pair.
func gitSSHRemote(serverURL, ns, slug string) (string, error) {
	host, err := canonicalSSHHost(serverURL)
	if err != nil {
		return "", err
	}
	return "git@" + host + ":" + url.PathEscape(ns) + "/" + url.PathEscape(slug) + ".git", nil
}

// resolveRepoSSHRemote fetches git_ssh_remote from the repo API, falling back
// to a constructed SSH URL when the field is absent or the API is unreachable.
func resolveRepoSSHRemote(ctx context.Context, c *apiclient.Client, serverURL, ns, slug string) (string, error) {
	var row repoRow
	if err := c.Get(ctx, "/namespaces/"+url.PathEscape(ns)+"/"+url.PathEscape(slug), &row); err == nil && row.GitSSHRemote != "" {
		return row.GitSSHRemote, nil
	}
	return gitSSHRemote(serverURL, ns, slug)
}

func runGit(cmd *cobra.Command, dir string, args ...string) error {
	gitCmd := execCommandContext(cmd.Context(), "git", args...)
	if dir != "" {
		gitCmd.Dir = dir
	}
	baseEnv := gitCmd.Env
	if len(baseEnv) == 0 {
		baseEnv = os.Environ()
	}
	gitCmd.Env = append(baseEnv, "GIT_TERMINAL_PROMPT=0")
	gitCmd.Stdout = cmd.OutOrStdout()
	gitCmd.Stderr = cmd.ErrOrStderr()
	gitCmd.Stdin = os.Stdin
	if err := gitCmd.Run(); err != nil {
		return err
	}
	return nil
}

func completeRepoPaths(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	prefix := strings.TrimSpace(toComplete)
	if !strings.Contains(prefix, "/") {
		vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeyOrgs, completion.FetchOrgNamespaceSlugs)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		out := make([]string, 0, len(vals))
		for _, ns := range vals {
			if strings.HasPrefix(ns, prefix) {
				out = append(out, ns+"/")
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	parts := strings.SplitN(prefix, "/", 2)
	ns := strings.TrimSpace(parts[0])
	repoPrefix := ""
	if len(parts) == 2 {
		repoPrefix = parts[1]
	}
	if ns == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.RepoKey(ns), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		return completion.FetchRepoSlugs(ctx, c, ns)
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(vals))
	for _, slug := range vals {
		if strings.HasPrefix(slug, repoPrefix) {
			out = append(out, ns+"/"+slug)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	RepoCmd.AddCommand(repoCloneCmd)
	RepoCmd.AddCommand(repoPushCmd)
	RepoCmd.AddCommand(repoPullCmd)

	addRepoFlag(repoPushCmd, repoPullCmd)
	repoPushCmd.Flags().String("remote", "origin", "Git remote name to push when no explicit repo path is given")
	repoPullCmd.Flags().String("remote", "origin", "Git remote name to pull when no explicit repo path is given")
	repoPushCmd.Flags().Bool("create", false, "Create the remote repository first when it does not exist")
}
