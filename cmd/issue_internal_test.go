package cmd

import (
	"testing"
	"time"
)

func TestIssueBrowserURL(t *testing.T) {
	got := issueBrowserURL("https://mcp.src.land/", "acme/demo", 7)
	want := "https://mcp.src.land/acme/demo/issues/7"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeNamespacePath(t *testing.T) {
	got, err := normalizeNamespacePath("/acme/platform/demo/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "acme/platform/demo" {
		t.Fatalf("got %q", got)
	}
}

func TestIssueListRows(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []issueRow{
		{Number: 1, NamespacePath: "acme/demo", Title: "bug", State: "open", AuthorID: "u1", CreatedAt: now, UpdatedAt: now},
		{Number: 2, NamespacePath: "acme/demo", Title: "feat", State: "closed", AuthorID: "u2", CreatedAt: now, UpdatedAt: now},
	}
	out := issueListRows(rows)
	if len(out) != 2 {
		t.Fatalf("got %d rows, want 2", len(out))
	}
	if out[0].Number != 1 || out[1].State != "closed" {
		t.Fatalf("unexpected issueListRows output: %+v", out)
	}
}
