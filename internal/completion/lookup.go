package completion

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/pagination"
)

// Resource cache keys (logical); paired with resolved server URL for paths.
const (
	KeyOrgs         = "orgs"
	KeyReposPrefix  = "repos:" // + namespace slug
	KeyAgents       = "agents"
	KeyOAuthClients = "oauth_clients"
	KeyAgentTokens  = "agent_tokens"
	KeySSHKeys      = "ssh_keys"
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
	var all []struct {
		Slug string `json:"slug"`
	}
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(pagination.MaxLimit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Orgs []struct {
				Slug string `json:"slug"`
			} `json:"orgs"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.Get(ctx, "/orgs?"+q.Encode(), &payload); err != nil {
			return nil, err
		}
		all = append(all, payload.Orgs...)
		if strings.TrimSpace(payload.NextCursor) == "" {
			break
		}
		cursor = payload.NextCursor
	}
	out := make([]string, 0, len(all))
	for _, o := range all {
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
	var all []struct {
		Slug string `json:"slug"`
	}
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(pagination.MaxLimit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		path := "/namespaces/" + url.PathEscape(ns) + "/repos?" + q.Encode()
		var payload struct {
			Repos []struct {
				Slug string `json:"slug"`
			} `json:"repos"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.Get(ctx, path, &payload); err != nil {
			return nil, err
		}
		all = append(all, payload.Repos...)
		if strings.TrimSpace(payload.NextCursor) == "" {
			break
		}
		cursor = payload.NextCursor
	}
	out := make([]string, 0, len(all))
	for _, r := range all {
		if s := strings.TrimSpace(r.Slug); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchAgentNames lists agent display names from GET /agents.
func FetchAgentNames(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var all []struct {
		Name string `json:"name"`
	}
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(pagination.MaxLimit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Agents []struct {
				Name string `json:"name"`
			} `json:"agents"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.Get(ctx, "/agents?"+q.Encode(), &payload); err != nil {
			return nil, err
		}
		all = append(all, payload.Agents...)
		if strings.TrimSpace(payload.NextCursor) == "" {
			break
		}
		cursor = payload.NextCursor
	}
	out := make([]string, 0, len(all))
	for _, r := range all {
		if s := strings.TrimSpace(r.Name); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}

// FetchOAuthClientIDs lists OAuth client resource UUIDs from GET /oauth/clients.
func FetchOAuthClientIDs(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var all []struct {
		ID string `json:"id"`
	}
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(pagination.MaxLimit))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var payload struct {
			Clients []struct {
				ID string `json:"id"`
			} `json:"clients"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.Get(ctx, "/oauth/clients?"+q.Encode(), &payload); err != nil {
			return nil, err
		}
		all = append(all, payload.Clients...)
		if strings.TrimSpace(payload.NextCursor) == "" {
			break
		}
		cursor = payload.NextCursor
	}
	out := make([]string, 0, len(all))
	for _, r := range all {
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

// FetchSSHKeyIDs lists SSH public key resource UUIDs from GET /account/ssh-keys.
func FetchSSHKeyIDs(ctx context.Context, c *apiclient.Client) ([]string, error) {
	var payload struct {
		Keys []struct {
			ID string `json:"id"`
		} `json:"keys"`
	}
	if err := c.Get(ctx, "/account/ssh-keys", &payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Keys))
	for _, k := range payload.Keys {
		if s := strings.TrimSpace(k.ID); s != "" {
			out = append(out, s)
		}
	}
	return sortDedupe(out), nil
}
