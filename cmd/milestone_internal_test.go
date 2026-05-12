package cmd

import (
	"testing"
	"time"
)

func TestMilestoneListRows(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []milestoneRow{
		{ID: "m1", Title: "v1", State: "open", CreatedAt: now, Progress: milestoneProgress{OpenCount: 2, Total: 5}},
		{ID: "m2", Title: "v2", State: "closed", CreatedAt: now},
	}
	out := milestoneListRows(rows)
	if len(out) != 2 {
		t.Fatalf("got %d rows, want 2", len(out))
	}
	if out[0].ID != "m1" || out[1].State != "closed" {
		t.Fatalf("unexpected output: %+v", out)
	}
}
