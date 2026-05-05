package cmd

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFormatRFC3339PtrUTC_nil(t *testing.T) {
	t.Parallel()
	if formatRFC3339PtrUTC(nil) != "" {
		t.Fatal()
	}
}

func TestListCSVRowMethods(t *testing.T) {
	t.Parallel()
	var rr repoRow
	_ = rr.CSVHeader()
	_ = rr.CSVRecord()

	a := agentRow{ID: uuid.New(), OwnerID: "owner-1", Name: "n"}
	_ = a.CSVHeader()
	_ = a.CSVRecord()
	h := "hint"
	a.ModelHint = &h
	_ = a.CSVRecord()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tok := token{
		ID:        uuid.New(),
		AgentID:   uuid.New(),
		CreatedAt: now,
		Scopes:    []string{"a", "b"},
	}
	_ = tok.CSVHeader()
	_ = tok.CSVRecord()
	exp := now.Add(time.Hour)
	tok.ExpiresAt = &exp
	rev := now.Add(2 * time.Hour)
	tok.RevokedAt = &rev
	_ = tok.CSVRecord()

	oc := oauthClient{
		ID: "id", ClientID: "cid", Name: "nm",
		AllowedScopes: []string{"s"}, IsPublic: true, OwnerSlug: "os",
		CreatedAt: now, UpdatedAt: now,
	}
	_ = oc.CSVHeader()
	_ = oc.CSVRecord()
	rat := now.Add(time.Minute)
	oc.RevokedAt = &rat
	_ = oc.CSVRecord()

	org := nsOrgRow{NamespaceID: "nid", Slug: "sl", DisplayName: "dn", LegalEntityName: "le", CreatedAt: now}
	_ = org.CSVHeader()
	_ = org.CSVRecord()

	mem := nsMemberRow{
		UserID: "u", Email: "e", Slug: "s", DisplayName: "d",
		IsOwner: true, Permissions: []string{"p"}, JoinedAt: now,
	}
	_ = mem.CSVHeader()
	_ = mem.CSVRecord()

	tr := nsTransferRow{
		ID: "tid", OrgID: "oid", OrgSlug: "os", OrgName: "on",
		FromUserID: "f", FromUserSlug: "fs", ToUserID: "t", ToUserSlug: "ts",
		ExpiresAt: now, CreatedAt: now,
	}
	_ = tr.CSVHeader()
	_ = tr.CSVRecord()
}
