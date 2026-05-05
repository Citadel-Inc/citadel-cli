package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const citadelRepoEnv = "CITADEL_REPO"
const citadelGitHostsEnv = "CITADEL_GIT_HOSTS"

var (
	repoHTTPS = regexp.MustCompile(`^https?://(?:[^@]+@)?(?P<host>[^/:]+)(?::[0-9]+)?/(?P<ns>[^/]+)/(?P<slug>[^/.]+)(?:\.git)?/?$`)
	repoSSH   = regexp.MustCompile(`^(?:git\+ssh://|ssh://)?(?:[^@]+@)?(?P<host>[^/:]+)(?::[0-9]+)?[:/](?P<ns>[^/]+)/(?P<slug>[^/.]+?)(?:\.git)?$`)
)

func defaultCitadelGitHosts() []string {
	return []string{
		"api.src.land",
		"src.land",
		"mcp.src.land",
		"git.src.land",
	}
}

func mergeCitadelHosts() map[string]struct{} {
	out := make(map[string]struct{})
	for _, h := range defaultCitadelGitHosts() {
		out[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}
	raw := strings.TrimSpace(os.Getenv(citadelGitHostsEnv))
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		h := strings.ToLower(strings.TrimSpace(part))
		if h != "" {
			out[h] = struct{}{}
		}
	}
	return out
}

// parseOriginIntoRepo extracts namespace/slug from a git remote URL if the host is in hosts.
func parseOriginIntoRepo(raw string, hosts map[string]struct{}) (ns, slug string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", errors.New("empty remote URL")
	}

	try := func(host, ns0, slug0 string) (string, string, error) {
		h := strings.ToLower(strings.TrimSpace(host))
		if _, ok := hosts[h]; !ok {
			return "", "", fmt.Errorf("origin remote points at %q; pass -R explicitly or add the host to "+citadelGitHostsEnv, host)
		}
		if ns0 == "" || slug0 == "" {
			return "", "", errors.New("could not parse namespace/slug from remote URL")
		}
		return ns0, slug0, nil
	}

	if m := repoHTTPS.FindStringSubmatch(s); m != nil {
		return try(matchNamed(repoHTTPS, m, "host"), matchNamed(repoHTTPS, m, "ns"), matchNamed(repoHTTPS, m, "slug"))
	}
	if m := repoSSH.FindStringSubmatch(s); m != nil {
		return try(matchNamed(repoSSH, m, "host"), matchNamed(repoSSH, m, "ns"), matchNamed(repoSSH, m, "slug"))
	}
	return "", "", errors.New("unsupported git remote URL shape")
}

func matchNamed(re *regexp.Regexp, m []string, name string) string {
	idx := re.SubexpIndex(name)
	if idx < 0 || idx >= len(m) {
		return ""
	}
	return m[idx]
}

func gitOriginURL(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
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

func inferenceHintWorthy(cmd *cobra.Command) bool {
	if quietFlag(cmd) {
		return false
	}
	if strings.TrimSpace(os.Getenv("CI")) != "" {
		return false
	}
	w := cmd.ErrOrStderr()
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// resolveRepoFlag resolves repository ns/slug from -R/--repo, CITADEL_REPO, or git origin in CWD.
func resolveRepoFlag(cmd *cobra.Command) (ns, slug string, err error) {
	noCWD, _ := cmd.Flags().GetBool("no-cwd-repo")
	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)

	if repoFlag != "" {
		return splitRepoArg(repoFlag)
	}

	if ev := strings.TrimSpace(os.Getenv(citadelRepoEnv)); ev != "" {
		return splitRepoArg(ev)
	}

	if noCWD {
		return "", "", fmt.Errorf(`repository required: pass -R <namespace>/<slug>, set %s, or omit --no-cwd-repo to infer from git`, citadelRepoEnv)
	}

	path, wdErr := os.Getwd()
	if wdErr != nil {
		path = ""
	}

	rawURL, err := gitOriginURL(cmd.Context(), path)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", fmt.Errorf("git is not available on PATH; pass -R <namespace>/<slug> or set %s", citadelRepoEnv)
		}
		return "", "", fmt.Errorf("could not infer repo from CWD (git remote): %v — pass -R <namespace>/<slug>", err)
	}

	hosts := mergeCitadelHosts()
	ns, slug, err = parseOriginIntoRepo(rawURL, hosts)
	if err != nil {
		return "", "", fmt.Errorf("could not infer repo from CWD: %v — pass -R <namespace>/<slug>", err)
	}

	if inferenceHintWorthy(cmd) {
		_, _ = fmt.Fprintf(os.Stderr, "Inferred -R %s/%s from CWD\n", ns, slug)
	}
	return ns, slug, nil
}

// resolveRepoFromPosOrFlag prefers -R/--repo (canonical), then an optional positional
// "<namespace>/<repo>", then resolveRepoFlag (env + CWD inference).
func resolveRepoFromPosOrFlag(cmd *cobra.Command, positional string) (ns, slug string, err error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)
	if repoFlag != "" {
		return splitRepoArg(repoFlag)
	}
	positional = strings.TrimSpace(positional)
	if positional != "" {
		return splitRepoArg(positional)
	}
	return resolveRepoFlag(cmd)
}

// ResolveRepoNamespaceForCompletion returns the parent namespace slug used to
// scope repo slug completion (mirrors resolveRepoFromPosOrFlag without a
// positional path).
func ResolveRepoNamespaceForCompletion(cmd *cobra.Command) (string, error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	repoFlag = strings.TrimSpace(repoFlag)
	if repoFlag != "" {
		if ns, _, err := splitRepoArg(repoFlag); err == nil {
			return ns, nil
		}
		// Partial "-R myorg" without repo segment: still scope listings to myorg.
		if i := strings.Index(repoFlag, "/"); i > 0 {
			return repoFlag[:i], nil
		}
		return "", errors.New("namespace not available for completion")
	}
	if ev := strings.TrimSpace(os.Getenv(citadelRepoEnv)); ev != "" {
		if ns, _, err := splitRepoArg(ev); err == nil {
			return ns, nil
		}
	}
	ns, _, err := resolveRepoFlag(cmd)
	return ns, err
}
