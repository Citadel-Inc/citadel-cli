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
			name: "account passkey list",
			got:  (passkeyRow{}).CSVHeader(),
			want: []string{"id", "name", "created_at"},
		},
		{
			name: "account device list",
			got:  (deviceRow{}).CSVHeader(),
			want: []string{"id", "name", "user_agent", "last_seen_at", "created_at"},
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

	pk := passkeyRow{ID: "p", Name: "n", CreatedAt: now}
	check(t, "account passkey", pk.CSVHeader(), pk.CSVRecord())

	dv := deviceRow{ID: "d", Name: "n", UserAgent: "ua", LastSeenAt: now, CreatedAt: now}
	check(t, "account device", dv.CSVHeader(), dv.CSVRecord())

	inv := orgInvitationRow{ID: "i", CreatedAt: now}
	check(t, "invitation", inv.CSVHeader(), inv.CSVRecord())

	as := auditSessionSummary{SessionID: "s", EventCount: 3}
	check(t, "audit session", as.CSVHeader(), as.CSVRecord())

	ev := auditEventPayload{ID: "e", TS: "t", Kind: "k", ActorType: "user"}
	check(t, "audit event", ev.CSVHeader(), ev.CSVRecord())
}
