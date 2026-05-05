// Package pagination mirrors the Citadel server cursor codec for client-side
// validation of --cursor before issuing HTTP requests.
package pagination

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
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

var ErrInvalidCursor = errors.New("invalid_cursor")

type DescCursor struct {
	TimeUnixNano int64
	ID           uuid.UUID
}

func EncodeDesc(t time.Time, id uuid.UUID) string {
	buf := make([]byte, 1+8+16)
	buf[0] = cursorV1Desc
	binary.BigEndian.PutUint64(buf[1:9], uint64(t.UTC().UnixNano()))
	copy(buf[9:25], id[:])
	return base64.RawURLEncoding.EncodeToString(buf)
}

func EncodeMemberAsc(notOwner bool, joinedAt time.Time, userID uuid.UUID) string {
	buf := make([]byte, 1+1+8+16)
	buf[0] = cursorV2Mem
	if notOwner {
		buf[1] = 1
	}
	binary.BigEndian.PutUint64(buf[2:10], uint64(joinedAt.UTC().UnixNano()))
	copy(buf[10:26], userID[:])
	return base64.RawURLEncoding.EncodeToString(buf)
}

func DecodeDesc(s string) (DescCursor, error) {
	var z DescCursor
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(raw) != 25 || raw[0] != cursorV1Desc {
		return z, fmt.Errorf("%w: desc", ErrInvalidCursor)
	}
	z.TimeUnixNano = int64(binary.BigEndian.Uint64(raw[1:9]))
	copy(z.ID[:], raw[9:25])
	return z, nil
}

// AuditDescCursor orders audit_log rows by ts DESC, id DESC.
type AuditDescCursor struct {
	TimeUnixNano int64
	ID           int64
}

// EncodeAuditDesc encodes a V3 audit list cursor (mirrors the Citadel server).
func EncodeAuditDesc(t time.Time, id int64) string {
	buf := make([]byte, 1+8+8)
	buf[0] = cursorV3Audit
	binary.BigEndian.PutUint64(buf[1:9], uint64(t.UTC().UnixNano()))
	binary.BigEndian.PutUint64(buf[9:17], uint64(id))
	return base64.RawURLEncoding.EncodeToString(buf)
}

// DecodeAuditDesc decodes a V3 audit list cursor.
func DecodeAuditDesc(s string) (AuditDescCursor, error) {
	var z AuditDescCursor
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(raw) != 17 || raw[0] != cursorV3Audit {
		return z, fmt.Errorf("%w: audit", ErrInvalidCursor)
	}
	z.TimeUnixNano = int64(binary.BigEndian.Uint64(raw[1:9]))
	z.ID = int64(binary.BigEndian.Uint64(raw[9:17]))
	return z, nil
}

type MemberAscCursor struct {
	NotOwner   bool
	JoinedNano int64
	UserID     uuid.UUID
}

func DecodeMemberAsc(s string) (MemberAscCursor, error) {
	var z MemberAscCursor
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(raw) != 26 || raw[0] != cursorV2Mem {
		return z, fmt.Errorf("%w: members", ErrInvalidCursor)
	}
	z.NotOwner = raw[1] != 0
	z.JoinedNano = int64(binary.BigEndian.Uint64(raw[2:10]))
	copy(z.UserID[:], raw[10:26])
	return z, nil
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
