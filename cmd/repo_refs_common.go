package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

type repoRefRow struct {
	Name string    `json:"name"`
	SHA  string    `json:"sha"`
	Kind string    `json:"kind,omitempty"`
	Date time.Time `json:"date"`
}

func parseRepoScopedNameArgs(cmd *cobra.Command, args []string) (ns, slug, name string, err error) {
	switch len(args) {
	case 1:
		ns, slug, err = resolveRepoFromPosOrFlag(cmd, "")
		if err != nil {
			return "", "", "", err
		}
		return ns, slug, strings.TrimSpace(args[0]), nil
	case 2:
		ns, slug, err = resolveRepoFromPosOrFlag(cmd, args[0])
		if err != nil {
			return "", "", "", err
		}
		return ns, slug, strings.TrimSpace(args[1]), nil
	default:
		return "", "", "", fmt.Errorf("expected <name> with -R/--repo, or <namespace>/<repo> <name>")
	}
}

func completeRepoExistingRefNames(
	cmd *cobra.Command,
	args []string,
	lookupKey func(string) string,
	fetch func(context.Context, *apiclient.Client, string, string) ([]string, error),
) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ns, slug, err := resolveRepoFromPosOrFlag(cmd, "")
	if err == nil && len(args) == 0 {
		return lookupRepoRefNames(cmd, ns, slug, lookupKey, fetch)
	}
	if len(args) == 0 {
		return completeRepoSlugs(cmd, args, "")
	}

	ns, slug, err = resolveRepoFromPosOrFlag(cmd, args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return lookupRepoRefNames(cmd, ns, slug, lookupKey, fetch)
}

func lookupRepoRefNames(
	cmd *cobra.Command,
	ns, slug string,
	lookupKey func(string) string,
	fetch func(context.Context, *apiclient.Client, string, string) ([]string, error),
) ([]string, cobra.ShellCompDirective) {
	repoPath := ns + "/" + slug
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), lookupKey(repoPath), func(ctx context.Context, c *apiclient.Client) ([]string, error) {
		return fetch(ctx, c, ns, slug)
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func validateMutationOutput(output string, action string) error {
	switch strings.TrimSpace(strings.ToLower(output)) {
	case "", "json":
		return nil
	default:
		return fmt.Errorf("--output for %s supports json or default human summary only; got %q", action, output)
	}
}

func shortSHA(sha string) string {
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}

func formatRepoRefDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
