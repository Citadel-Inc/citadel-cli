package cmd

import (
	"strconv"
	"strings"
	"time"
)

func formatRFC3339UTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatRFC3339PtrUTC(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatRFC3339UTC(*t)
}

// ── repo list ────────────────────────────────────────────────────────────────

func (repoRow) CSVHeader() []string {
	return []string{"slug", "path", "visibility", "default_branch", "description", "namespace_id", "parent_slug", "created_at"}
}

func (r repoRow) CSVRecord() []string {
	return []string{
		r.Slug,
		r.Path,
		r.Visibility,
		r.DefaultBranch,
		r.Description,
		r.NamespaceID,
		r.ParentSlug,
		r.CreatedAt,
	}
}

// ── agent list ───────────────────────────────────────────────────────────────

func (agentRow) CSVHeader() []string {
	return []string{"id", "owner_user_id", "name", "model_hint"}
}

func (a agentRow) CSVRecord() []string {
	hint := ""
	if a.ModelHint != nil {
		hint = *a.ModelHint
	}
	return []string{a.ID.String(), a.OwnerID, a.Name, hint}
}

// ── token list ───────────────────────────────────────────────────────────────

func (token) CSVHeader() []string {
	return []string{"id", "agent_id", "created_at", "expires_at", "revoked_at", "scopes"}
}

func (t token) CSVRecord() []string {
	return []string{
		t.ID.String(),
		t.AgentID.String(),
		formatRFC3339UTC(t.CreatedAt),
		formatRFC3339PtrUTC(t.ExpiresAt),
		formatRFC3339PtrUTC(t.RevokedAt),
		strings.Join(t.Scopes, ","),
	}
}

// ── oauth clients list ───────────────────────────────────────────────────────

func (oauthClient) CSVHeader() []string {
	return []string{"id", "client_id", "name", "allowed_scopes", "is_public", "owner_slug", "created_at", "updated_at", "revoked_at"}
}

func (o oauthClient) CSVRecord() []string {
	return []string{
		o.ID,
		o.ClientID,
		o.Name,
		strings.Join(o.AllowedScopes, ","),
		strconv.FormatBool(o.IsPublic),
		o.OwnerSlug,
		formatRFC3339UTC(o.CreatedAt),
		formatRFC3339UTC(o.UpdatedAt),
		formatRFC3339PtrUTC(o.RevokedAt),
	}
}

// ── namespace list (orgs) ────────────────────────────────────────────────────

func (nsOrgRow) CSVHeader() []string {
	return []string{"namespace_id", "slug", "display_name", "legal_entity_name", "created_at"}
}

func (o nsOrgRow) CSVRecord() []string {
	return []string{o.NamespaceID, o.Slug, o.DisplayName, o.LegalEntityName, formatRFC3339UTC(o.CreatedAt)}
}

// ── namespace members ─────────────────────────────────────────────────────────

func (nsMemberRow) CSVHeader() []string {
	return []string{"user_id", "email", "slug", "display_name", "is_owner", "permissions", "joined_at"}
}

func (m nsMemberRow) CSVRecord() []string {
	return []string{
		m.UserID,
		m.Email,
		m.Slug,
		m.DisplayName,
		strconv.FormatBool(m.IsOwner),
		strings.Join(m.Permissions, ","),
		formatRFC3339UTC(m.JoinedAt),
	}
}

// ── namespace transfer list-pending ──────────────────────────────────────────

func (nsTransferRow) CSVHeader() []string {
	return []string{"id", "org_namespace_id", "org_slug", "org_name", "from_user_id", "from_user_slug", "to_user_id", "to_user_slug", "expires_at", "created_at"}
}

func (tr nsTransferRow) CSVRecord() []string {
	return []string{
		tr.ID,
		tr.OrgID,
		tr.OrgSlug,
		tr.OrgName,
		tr.FromUserID,
		tr.FromUserSlug,
		tr.ToUserID,
		tr.ToUserSlug,
		formatRFC3339UTC(tr.ExpiresAt),
		formatRFC3339UTC(tr.CreatedAt),
	}
}

// ── ssh keys ─────────────────────────────────────────────────────────────────

func (sshKeyRow) CSVHeader() []string {
	return []string{"id", "fingerprint", "public_key", "label", "created_at"}
}

func (r sshKeyRow) CSVRecord() []string {
	lbl := ""
	if r.Label != nil {
		lbl = *r.Label
	}
	return []string{r.ID, r.Fingerprint, r.PublicKey, lbl, formatRFC3339UTC(r.CreatedAt)}
}

// ── org invitations ───────────────────────────────────────────────────────────

func (orgInvitationRow) CSVHeader() []string {
	return []string{"id", "org_slug", "email", "user_slug", "status", "permissions", "created_at", "expires_at"}
}

func (r orgInvitationRow) CSVRecord() []string {
	return []string{
		r.ID,
		r.OrgSlug,
		r.Email,
		r.UserSlug,
		r.Status,
		strings.Join(r.Permissions, ","),
		formatRFC3339UTC(r.CreatedAt),
		formatRFC3339PtrUTC(r.ExpiresAt),
	}
}

// ── audit sessions ────────────────────────────────────────────────────────────

func (auditSessionSummary) CSVHeader() []string {
	return []string{"session_id", "id", "actor_slug", "actor_id", "actor_type", "namespace_slug", "namespace_id", "started_at", "last_event_at", "event_count"}
}

func (s auditSessionSummary) CSVRecord() []string {
	sid := s.SessionID
	if sid == "" {
		sid = s.ID
	}
	return []string{
		sid,
		s.ID,
		s.ActorSlug,
		s.ActorID,
		s.ActorType,
		s.NamespaceSlug,
		s.NamespaceID,
		s.StartedAt,
		s.LastEventAt,
		strconv.Itoa(s.EventCount),
	}
}
