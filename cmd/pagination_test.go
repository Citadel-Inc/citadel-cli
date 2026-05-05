package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Rethunk-Tech/citadel-cli/internal/pagination"
)

func TestValidateDescCursor_emptyOK(t *testing.T) {
	if err := validateDescCursor(""); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDescCursor_roundTrip(t *testing.T) {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	s := pagination.EncodeDesc(ts, id)
	if err := validateDescCursor(s); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDescCursor_invalid(t *testing.T) {
	err := validateDescCursor("not-base64!!!")
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("want invalid cursor, got %v", err)
	}
	// v2 member cursor must fail desc decode
	s := pagination.EncodeMemberAsc(false, time.Unix(0, 0).UTC(), uuid.Nil)
	if err := validateDescCursor(s); err == nil {
		t.Fatal("member cursor must not validate as desc")
	}
}

func TestValidateMemberCursor_emptyOK(t *testing.T) {
	if err := validateMemberCursor(""); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMemberCursor_roundTrip(t *testing.T) {
	uid := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	joined := time.Date(2021, 2, 3, 4, 5, 6, 7, time.UTC)
	s := pagination.EncodeMemberAsc(true, joined, uid)
	if err := validateMemberCursor(s); err != nil {
		t.Fatal(err)
	}
}

func TestValidateMemberCursor_invalid(t *testing.T) {
	err := validateMemberCursor("%%%")
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("want invalid cursor, got %v", err)
	}
	// v1 desc cursor must fail member decode
	s := pagination.EncodeDesc(time.Unix(0, 0).UTC(), uuid.Nil)
	if err := validateMemberCursor(s); err == nil {
		t.Fatal("desc cursor must not validate as member")
	}
}
