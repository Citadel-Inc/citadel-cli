// Package pagination holds list limits and test-fixture cursor encoders.
// Cursors are opaque to the CLI: the server alone validates them.
package pagination

import (
	"encoding/base64"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
)

const (
	cursorV1Desc  = byte(1)
	cursorV2Mem   = byte(2)
	cursorV3Audit = byte(3)
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

func encodeCursorInt64(value int64) uint64 {
	return uint64(value) //nolint:gosec // Fixed-width cursor fields preserve signed integer bits.
}

func EncodeDesc(t time.Time, id uuid.UUID) string {
	buf := make([]byte, 1+8+16)
	buf[0] = cursorV1Desc
	binary.BigEndian.PutUint64(buf[1:9], encodeCursorInt64(t.UTC().UnixNano()))
	copy(buf[9:25], id[:])
	return base64.RawURLEncoding.EncodeToString(buf)
}

func EncodeMemberAsc(notOwner bool, joinedAt time.Time, userID uuid.UUID) string {
	buf := make([]byte, 1+1+8+16)
	buf[0] = cursorV2Mem
	if notOwner {
		buf[1] = 1
	}
	binary.BigEndian.PutUint64(buf[2:10], encodeCursorInt64(joinedAt.UTC().UnixNano()))
	copy(buf[10:26], userID[:])
	return base64.RawURLEncoding.EncodeToString(buf)
}

// EncodeAuditDesc encodes a V3 audit list cursor (mirrors the Citadel server).
func EncodeAuditDesc(t time.Time, id int64) string {
	buf := make([]byte, 1+8+8)
	buf[0] = cursorV3Audit
	binary.BigEndian.PutUint64(buf[1:9], encodeCursorInt64(t.UTC().UnixNano()))
	binary.BigEndian.PutUint64(buf[9:17], encodeCursorInt64(id))
	return base64.RawURLEncoding.EncodeToString(buf)
}

func ClampLimit(n int) int {
	if n <= 0 {
		return DefaultLimit
	}
	if n > MaxLimit {
		return MaxLimit
	}
	return n
}

// ParseLimit parses the raw "limit" query value; empty or invalid yields DefaultLimit.
func ParseLimit(raw string) int {
	if raw == "" {
		return DefaultLimit
	}
	var n int
	for _, c := range raw {
		if c < '0' || c > '9' {
			return DefaultLimit
		}
		n = n*10 + int(c-'0')
		if n > MaxLimit*10 {
			return MaxLimit
		}
	}
	return ClampLimit(n)
}
