package pagination

import (
	"encoding/base64"
	"testing"
	"time"
	"uuid"
)

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

func TestEncodersEmitVersionedCursors(t *testing.T) {
	now := time.Unix(1, 0)
	for name, c := range map[string]struct {
		got  string
		size int
		ver  byte
	}{
		"desc":   {EncodeDesc(now, uuid.Nil()), 25, cursorV1Desc},
		"member": {EncodeMemberAsc(true, now, uuid.Nil()), 26, cursorV2Mem},
		"audit":  {EncodeAuditDesc(now, 7), 17, cursorV3Audit},
	} {
		raw, err := base64.RawURLEncoding.DecodeString(c.got)
		if err != nil || len(raw) != c.size || raw[0] != c.ver {
			t.Fatalf("%s: %d bytes, version %d, err %v", name, len(raw), raw[0], err)
		}
	}
}
