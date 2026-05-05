package cmd

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateListOutput_unknown(t *testing.T) {
	if err := validateListOutput("protobuf"); err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateGetOutput_rejectsCSV(t *testing.T) {
	if err := validateGetOutput("csv"); err == nil {
		t.Fatal("want error")
	}
}

func TestEmitCSVRowsTo_twoBatchesOneHeader(t *testing.T) {
	var buf bytes.Buffer
	hdr := false
	r1 := repoRow{Slug: "a", Path: "ns/a", Visibility: "private", DefaultBranch: "main", CreatedAt: "2026-01-01T00:00:00Z"}
	r2 := repoRow{Slug: "b", Path: "ns/b", Visibility: "public", DefaultBranch: "main", CreatedAt: "2026-01-02T00:00:00Z"}
	if err := emitCSVRowsTo(&buf, &hdr, []repoRow{r1}); err != nil {
		t.Fatal(err)
	}
	if err := emitCSVRowsTo(&buf, &hdr, []repoRow{r2}); err != nil {
		t.Fatal(err)
	}
	cr := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := cr.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("want header + 2 data rows, got %d", len(rows))
	}
	if rows[0][0] != "slug" {
		t.Fatalf("header: %q", rows[0])
	}
	if rows[1][0] != "a" || rows[2][0] != "b" {
		t.Fatalf("data: %#v", rows)
	}
}

func TestEmitCSVHeaderOnlyTo_empty(t *testing.T) {
	var buf bytes.Buffer
	if err := emitCSVHeaderOnlyTo[repoRow](&buf); err != nil {
		t.Fatal(err)
	}
	cr := csv.NewReader(strings.NewReader(buf.String()))
	row, err := cr.Read()
	if err != nil {
		t.Fatal(err)
	}
	if row[0] != "slug" {
		t.Fatalf("got %v", row)
	}
}

func TestTokenCSVRecord_RFC3339UTC(t *testing.T) {
	ts := time.Date(2026, 3, 4, 15, 30, 0, 0, time.FixedZone("CST", -6*3600))
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tok := token{
		ID:        id1,
		AgentID:   id2,
		CreatedAt: ts,
		Scopes:    []string{"a", "b"},
	}
	rec := tok.CSVRecord()
	if rec[2] != ts.UTC().Format(time.RFC3339) {
		t.Fatalf("created_at: got %q want RFC3339 UTC", rec[2])
	}
}
