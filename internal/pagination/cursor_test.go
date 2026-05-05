package pagination

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecodeDesc(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC)
	s := EncodeDesc(ts, id)
	got, err := DecodeDesc(s)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.TimeUnixNano != ts.UTC().UnixNano() {
		t.Fatalf("%+v", got)
	}
}

func TestDecodeDesc_errors(t *testing.T) {
	for _, s := range []string{
		"not-valid-base64!!!",
		base64.RawURLEncoding.EncodeToString([]byte{1}), // too short
	} {
		_, err := DecodeDesc(s)
		if err == nil || !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("DecodeDesc(%q) = _, %v; want wrapped %v", s, err, ErrInvalidCursor)
		}
	}
	wrongVer := make([]byte, 25)
	wrongVer[0] = 99
	_, err := DecodeDesc(base64.RawURLEncoding.EncodeToString(wrongVer))
	if err == nil || !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("wrong version: %v", err)
	}
}

func TestEncodeDecodeMemberAsc(t *testing.T) {
	uid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	joined := time.Date(2025, 6, 7, 8, 9, 10, 11, time.UTC)

	for _, tc := range []struct {
		notOwner bool
	}{
		{notOwner: false},
		{notOwner: true},
	} {
		s := EncodeMemberAsc(tc.notOwner, joined, uid)
		got, err := DecodeMemberAsc(s)
		if err != nil {
			t.Fatal(err)
		}
		if got.NotOwner != tc.notOwner || got.JoinedNano != joined.UTC().UnixNano() || got.UserID != uid {
			t.Fatalf("notOwner=%v: %+v", tc.notOwner, got)
		}
	}
}

func TestDecodeMemberAsc_errors(t *testing.T) {
	_, err := DecodeMemberAsc(base64.RawURLEncoding.EncodeToString([]byte{cursorV2Mem})) // too short
	if err == nil || !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("short: %v", err)
	}
	raw := make([]byte, 26)
	raw[0] = cursorV1Desc
	_, err = DecodeMemberAsc(base64.RawURLEncoding.EncodeToString(raw))
	if err == nil || !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("wrong kind: %v", err)
	}
}

func TestClampLimit(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want int
	}{
		{0, DefaultLimit},
		{-1, DefaultLimit},
		{1, 1},
		{50, 50},
		{MaxLimit, MaxLimit},
		{MaxLimit + 1, MaxLimit},
		{9999, MaxLimit},
	} {
		if got := ClampLimit(tc.in); got != tc.want {
			t.Fatalf("ClampLimit(%d) = %d; want %d", tc.in, got, tc.want)
		}
	}
}

func TestEncodeDecodeAuditDesc(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC)
	id := int64(9007199254740991)
	s := EncodeAuditDesc(ts, id)
	got, err := DecodeAuditDesc(s)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.TimeUnixNano != ts.UTC().UnixNano() {
		t.Fatalf("%+v", got)
	}
}

func TestDecodeAuditDesc_errors(t *testing.T) {
	t.Parallel()
	_, err := DecodeAuditDesc("not-base64!!!")
	if err == nil || !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("got %v", err)
	}
	// v1 desc cursor must fail audit decode
	raw := make([]byte, 25)
	raw[0] = cursorV1Desc
	_, err = DecodeAuditDesc(base64.RawURLEncoding.EncodeToString(raw))
	if err == nil || !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("wrong kind: %v", err)
	}
}

func TestParseLimit(t *testing.T) {
	t.Parallel()
	if ParseLimit("") != DefaultLimit {
		t.Fatal()
	}
	if ParseLimit("abc") != DefaultLimit {
		t.Fatal()
	}
	if ParseLimit("100") != 100 {
		t.Fatal()
	}
	if ParseLimit("500") != MaxLimit {
		t.Fatal()
	}
	if ParseLimit("999999999999999999999") != MaxLimit {
		t.Fatal("expected early clamp when digit run exceeds bound")
	}
}
