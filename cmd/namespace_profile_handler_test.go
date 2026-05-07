package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func makeProfile(slug, kind, visibility string) map[string]any {
	return map[string]any{
		"namespace_id":    "11111111-1111-1111-1111-111111111111",
		"slug":            slug,
		"kind":            kind,
		"visibility":      visibility,
		"display_name":    "Test " + slug,
		"bio":             "A test namespace",
		"location":        "The Internet",
		"website_url":     "https://example.com",
		"social_links":    map[string]any{"github": "testuser"},
		"stats":           map[string]any{"repos": 5},
		"repos_preview":   []any{},
		"members_preview": []any{},
	}
}

// ── namespace profile get ─────────────────────────────────────────────────────

func TestNamespaceProfileGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeProfile("myorg", "org", "public"))
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "profile", "get", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNamespaceProfileGet_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeProfile("myorg", "org", "public"))
		},
	}))
	if err := rootForOut(cmd.NamespaceCmd, &buf, "profile", "get", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if out["slug"] != "myorg" {
		t.Fatalf("expected slug=myorg, got %v", out["slug"])
	}
	if out["visibility"] != "public" {
		t.Fatalf("expected visibility=public, got %v", out["visibility"])
	}
}

func TestNamespaceProfileGet_UserNamespace(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/alice/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeProfile("alice", "user", "public"))
		},
	}))
	if err := rootForOut(cmd.NamespaceCmd, &buf, "profile", "get", "alice").Execute(); err != nil {
		t.Fatal(err)
	}
	// Human table should include the slug
	if !strings.Contains(buf.String(), "alice") {
		t.Fatalf("expected slug in output, got: %s", buf.String())
	}
}

func TestNamespaceProfileGet_SocialLinks(t *testing.T) {
	var buf bytes.Buffer
	profile := makeProfile("testorg", "org", "public")
	profile["social_links"] = map[string]any{
		"github":   "testorg",
		"linkedin": "testorg-ltd",
	}
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/testorg/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, profile)
		},
	}))
	if err := rootForOut(cmd.NamespaceCmd, &buf, "profile", "get", "testorg").Execute(); err != nil {
		t.Fatal(err)
	}
	// Social links should appear in human output
	if !strings.Contains(buf.String(), "github") {
		t.Fatalf("expected social links in output, got: %s", buf.String())
	}
}

func TestNamespaceProfileGet_OwnerFields(t *testing.T) {
	profile := makeProfile("myorg", "org", "private")
	profile["billing_email"] = "billing@myorg.example"
	profile["verified_domains"] = []any{"myorg.example"}
	profile["is_owner"] = true

	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, profile)
		},
	}))
	var buf bytes.Buffer
	if err := rootForOut(cmd.NamespaceCmd, &buf, "profile", "get", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
	// Owner-only fields should appear
	if !strings.Contains(buf.String(), "billing@myorg.example") {
		t.Fatalf("expected billing email in owner output, got: %s", buf.String())
	}
}

func TestNamespaceProfileGet_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/nosuchns/profile": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"code":"not_found"}`, 404)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "profile", "get", "nosuchns").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestNamespaceProfileGet_NoAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://nope")
	err := rootFor(cmd.NamespaceCmd, "profile", "get", "myorg").Execute()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("want not-authenticated, got %v", err)
	}
}

func TestNamespaceProfileGet_YAML(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/myorg/profile": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeProfile("myorg", "org", "public"))
		},
	}))
	if err := rootForOut(cmd.NamespaceCmd, &buf, "profile", "get", "myorg", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "slug: myorg") {
		t.Fatalf("expected YAML with slug, got: %s", buf.String())
	}
}
