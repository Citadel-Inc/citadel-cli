package cmd

import (
	"reflect"
	"testing"
)

func TestAuditEventPayload_CSVRow(t *testing.T) {
	t.Parallel()
	r := auditEventPayload{
		ID: "42", TS: "2026-01-01T00:00:00Z", Kind: "agent.created",
		ActorSlug: "alice", ActorID: "00000000-0000-0000-0000-000000000001",
		NamespaceSlug: "ns", NamespaceID: "00000000-0000-0000-0000-000000000002",
		SubjectID: "subj", ActorType: "user",
	}
	wantH := []string{"id", "ts", "kind", "actor_slug", "actor_id", "namespace_slug", "namespace_id", "subject_id", "actor_type"}
	if h := r.CSVHeader(); !reflect.DeepEqual(h, wantH) {
		t.Fatalf("CSVHeader: %v", h)
	}
	wantR := []string{"42", "2026-01-01T00:00:00Z", "agent.created", "alice", "00000000-0000-0000-0000-000000000001", "ns", "00000000-0000-0000-0000-000000000002", "subj", "user"}
	if rec := r.CSVRecord(); !reflect.DeepEqual(rec, wantR) {
		t.Fatalf("CSVRecord: %v", rec)
	}
}
