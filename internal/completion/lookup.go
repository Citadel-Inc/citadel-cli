package completion

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// Resource cache keys (logical); paired with resolved server URL for paths.
const (
	KeyOrgs         = "orgs"
	KeyReposPrefix  = "repos:" // + namespace slug
	KeyAgents       = "agents"
	KeyOAuthClients = "oauth_clients"
	KeyAgentTokens  = "agent_tokens"
)

// RepoKey returns the cache resource key for repo slugs in a namespace.
func RepoKey(namespace string) string { return KeyReposPrefix + namespace }

// Lookup loads cached values or calls fetch with a quiet apiclient. Any error
// from fetch (including missing auth) is returned to the caller for shell
// completion handling.
func Lookup(ctx context.Context, serverFlag string, resourceKey string, fetch func(context.Context, *apiclient.Client) ([]string, error)) ([]string, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return nil, errors.New("not authenticated")
	}
	resolved := strings.TrimRight(cfg.ResolveServerURL(serverFlag), "/")
	if vals, ok := readCache(resolved, resourceKey); ok {
		return vals, nil
	}
	c, err := apiclient.New(cfg, apiclient.Options{
		Server:    serverFlag,
		Verbose:   false,
		DebugHTTP: false,
	})
	if err != nil {
		return nil, err
	}
	vals, err := fetch(ctx, c)
	if err != nil {
		return nil, err
	}
	vals = sortDedupe(vals)
	writeCache(resolved, resourceKey, vals)
	return vals, nil
}

func sortDedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	slices.Sort(out)
	out = slices.Compact(out)
	return out
}

// FetchOrgNamespaceSlugs lists org namespace slugs from GET /orgs.
func FetchOrgNamespaceSlugs(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var payload struct {
		Orgs []struct {
			Slug string `json:"slug"`
		} `json:"orgs"`
	}
	if err := c.Get(ctx, "/orgs", &payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Orgs))
	for _, o := range payload.Orgs {
		if s := strings.TrimSpace(o.Slug); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchRepoSlugs lists repo slugs for a parent namespace.
func FetchRepoSlugs(ctx context.Context, c *apiclient.Client, parentNamespace string) ([]string, error) {
	ns := strings.TrimSpace(parentNamespace)
	if ns == "" {
		return nil, errors.New("missing namespace for repo completion")
	}
	var payload struct {
		Repos []struct {
			Slug string `json:"slug"`
		} `json:"repos"`
	}
	path := "/namespaces/" + url.PathEscape(ns) + "/repos"
	if err := c.Get(ctx, path, &payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Repos))
	for _, r := range payload.Repos {
		if s := strings.TrimSpace(r.Slug); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchAgentNames lists agent display names from GET /agents.
func FetchAgentNames(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var rows []struct {
		Name string `json:"name"`
	}
	if err := c.Get(ctx, "/agents", &rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if s := strings.TrimSpace(r.Name); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchOAuthClientIDs lists OAuth client resource UUIDs from GET /oauth/clients.
func FetchOAuthClientIDs(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var rows []struct {
		ID string `json:"id"`
	}
	if err := c.Get(ctx, "/oauth/clients", &rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if s := strings.TrimSpace(r.ID); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchAgentTokenIDs lists token UUIDs from GET /agent-tokens.
func FetchAgentTokenIDs(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var rows []struct {
		ID string `json:"id"`
	}
	if err := c.Get(ctx, "/agent-tokens", &rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if s := strings.TrimSpace(r.ID); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}
