package cmd

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Frozen CSV column contracts documented in README.md "Output formats".
// Excel/Sheets paste compatibility is enforced at the column-contract level;
// RFC 4180 field serialization is covered in output_formats_test.go.
func TestCSVHeaders_matchREADMEFrozenContract(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "repo list",
			got:  (repoRow{}).CSVHeader(),
			want: []string{"slug", "path", "visibility", "default_branch", "description", "namespace_id", "parent_slug", "created_at"},
		},
		{
			name: "repo ref list",
			got:  (repoRefRow{}).CSVHeader(),
			want: []string{"name", "sha", "date"},
		},
		{
			name: "issue list",
			got:  (issueListRow{}).CSVHeader(),
			want: []string{"number", "namespace_path", "title", "state", "author_id", "created_at", "updated_at", "closed_at"},
		},
		{
			name: "agent list",
			got:  (agentRow{}).CSVHeader(),
			want: []string{"id", "owner_user_id", "name", "model_hint"},
		},
		{
			name: "token list",
			got:  (token{}).CSVHeader(),
			want: []string{"id", "agent_id", "created_at", "expires_at", "revoked_at", "scopes"},
		},
		{
			name: "oauth clients list",
			got:  (oauthClient{}).CSVHeader(),
			want: []string{"id", "client_id", "name", "allowed_scopes", "is_public", "owner_slug", "created_at", "updated_at", "revoked_at"},
		},
		{
			name: "namespace list",
			got:  (nsOrgRow{}).CSVHeader(),
			want: []string{"namespace_id", "slug", "display_name", "legal_entity_name", "created_at"},
		},
		{
			name: "namespace members",
			got:  (nsMemberRow{}).CSVHeader(),
			want: []string{"user_id", "email", "slug", "display_name", "is_owner", "permissions", "joined_at"},
		},
		{
			name: "namespace transfer list-pending",
			got:  (nsTransferRow{}).CSVHeader(),
			want: []string{"id", "org_namespace_id", "org_slug", "org_name", "from_user_id", "from_user_slug", "to_user_id", "to_user_slug", "expires_at", "created_at"},
		},
		{
			name: "ssh-key list",
			got:  (sshKeyRow{}).CSVHeader(),
			want: []string{"id", "fingerprint", "public_key", "label", "created_at"},
		},
		{
			name: "org invitation list/pending",
			got:  (orgInvitationRow{}).CSVHeader(),
			want: []string{"id", "org_slug", "email", "user_slug", "status", "permissions", "created_at", "expires_at"},
		},
		{
			name: "audit list",
			got:  (auditEventPayload{}).CSVHeader(),
			want: []string{"id", "ts", "kind", "actor_slug", "actor_id", "namespace_slug", "namespace_id", "subject_id", "actor_type"},
		},
		{
			name: "audit sessions list",
			got:  (auditSessionSummary{}).CSVHeader(),
			want: []string{"session_id", "id", "actor_slug", "actor_id", "actor_type", "namespace_slug", "namespace_id", "started_at", "last_event_at", "event_count"},
		},
		{
			name: "commit list",
			got:  (commitItem{}).CSVHeader(),
			want: []string{"sha", "author", "author_email", "committer", "committer_email", "timestamp", "message"},
		},
		{
			name: "auth provider list",
			got:  (authProviderRow{}).CSVHeader(),
			want: []string{"id", "label"},
		},
		{
			name: "deploy token list",
			got:  (deployTokenRow{}).CSVHeader(),
			want: []string{"id", "name", "namespace_path", "created_at", "expires_at", "revoked_at"},
		},
		{
			name: "webhook list",
			got:  (webhookRow{}).CSVHeader(),
			want: []string{"id", "name", "namespace_path", "target_url", "event_kinds", "include_descendants", "active", "created_at", "updated_at", "last_delivery_at", "last_delivery_state", "secret_hint"},
		},
		{
			name: "milestone list",
			got:  (milestoneListRow{}).CSVHeader(),
			want: []string{"id", "title", "state", "due_on", "progress", "created_at"},
		},
		{
			name: "pr list",
			got:  (prListRow{}).CSVHeader(),
			want: []string{"number", "title", "state", "source_ref", "target_ref", "author_id", "created_at", "updated_at"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !slices.Equal(tc.got, tc.want) {
				t.Fatalf("CSVHeader mismatch\ngot:  %q\nwant: %q", tc.got, tc.want)
			}
		})
	}
}

func TestCSVRecord_columnCountMatchesHeader(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	check := func(t *testing.T, name string, hdr []string, rec []string) {
		t.Helper()
		if len(hdr) != len(rec) {
			t.Fatalf("%s: header len %d record len %d", name, len(hdr), len(rec))
		}
	}

	var rr repoRow
	check(t, "repo", rr.CSVHeader(), rr.CSVRecord())

	ref := repoRefRow{Name: "main", SHA: "abc", Date: now}
	check(t, "repo ref", ref.CSVHeader(), ref.CSVRecord())

	iss := issueListRow{Number: 7, NamespacePath: "acme/demo", Title: "hello", State: "open", AuthorID: "u1", CreatedAt: now, UpdatedAt: now}
	check(t, "issue list", iss.CSVHeader(), iss.CSVRecord())

	a := agentRow{ID: id1, OwnerID: "o", Name: "n"}
	check(t, "agent", a.CSVHeader(), a.CSVRecord())

	tok := token{ID: id1, AgentID: id2, CreatedAt: now, Scopes: []string{"s"}}
	check(t, "token", tok.CSVHeader(), tok.CSVRecord())

	oc := oauthClient{ID: "i", ClientID: "c", Name: "n", AllowedScopes: []string{"x"}, IsPublic: false, OwnerSlug: "o", CreatedAt: now, UpdatedAt: now}
	check(t, "oauth", oc.CSVHeader(), oc.CSVRecord())

	org := nsOrgRow{NamespaceID: "n", Slug: "s", CreatedAt: now}
	check(t, "ns org", org.CSVHeader(), org.CSVRecord())

	mem := nsMemberRow{UserID: "u", JoinedAt: now}
	check(t, "member", mem.CSVHeader(), mem.CSVRecord())

	tr := nsTransferRow{ID: "t", ExpiresAt: now, CreatedAt: now}
	check(t, "transfer", tr.CSVHeader(), tr.CSVRecord())

	sk := sshKeyRow{ID: "k", Fingerprint: "f", PublicKey: "p", CreatedAt: now}
	check(t, "ssh-key", sk.CSVHeader(), sk.CSVRecord())

	inv := orgInvitationRow{ID: "i", CreatedAt: now}
	check(t, "invitation", inv.CSVHeader(), inv.CSVRecord())

	as := auditSessionSummary{SessionID: "s", EventCount: 3}
	check(t, "audit session", as.CSVHeader(), as.CSVRecord())

	ev := auditEventPayload{ID: "e", TS: "t", Kind: "k", ActorType: "user"}
	check(t, "audit event", ev.CSVHeader(), ev.CSVRecord())

	ci := commitItem{SHA: "abc123", Message: "init\nbody", Author: "A", AuthorEmail: "a@x", Committer: "C", CommitterEmail: "c@x", Timestamp: now}
	check(t, "commit", ci.CSVHeader(), ci.CSVRecord())

	ap := authProviderRow{ID: "github", Label: "GitHub"}
	check(t, "auth provider", ap.CSVHeader(), ap.CSVRecord())

	dt := deployTokenRow{ID: "d", Name: "ci", NamespacePath: "acme/demo", CreatedAt: now}
	check(t, "deploy token nil expiry", dt.CSVHeader(), dt.CSVRecord())

	expiresAt := now.Add(24 * 3600 * 1e9)
	dt2 := deployTokenRow{ID: "d2", Name: "ci2", NamespacePath: "acme/demo", CreatedAt: now, ExpiresAt: &expiresAt}
	check(t, "deploy token with expiry", dt2.CSVHeader(), dt2.CSVRecord())

	wh := webhookRow{ID: "w1", NamespacePath: "acme/demo", TargetURL: "https://x.test/h", EventKinds: []string{"issue.opened"}, CreatedAt: now, UpdatedAt: now}
	check(t, "webhook", wh.CSVHeader(), wh.CSVRecord())

	ml := milestoneListRow{ID: "m1", Title: "v1", State: "open", CreatedAt: now}
	check(t, "milestone", ml.CSVHeader(), ml.CSVRecord())

	pr := prListRow{Number: 3, Title: "fix", State: "open", SourceRef: "feat", TargetRef: "main", AuthorID: "u1", CreatedAt: now, UpdatedAt: now}
	check(t, "pr list", pr.CSVHeader(), pr.CSVRecord())
}
