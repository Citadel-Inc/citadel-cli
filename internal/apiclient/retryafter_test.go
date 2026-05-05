package apiclient

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfterSeconds_Integer(t *testing.T) {
	if got := ParseRetryAfterSeconds("  42  "); got != 42 {
		t.Fatalf("got %d", got)
	}
}

func TestParseRetryAfterSeconds_HTTPDate(t *testing.T) {
	when := time.Now().UTC().Add(90 * time.Second).Format(http.TimeFormat)
	got := ParseRetryAfterSeconds(when)
	if got < 85 || got > 95 {
		t.Fatalf("got %d for when %q", got, when)
	}
}

func TestParseRetryAfterSeconds_Malformed(t *testing.T) {
	if got := ParseRetryAfterSeconds("nope"); got != 0 {
		t.Fatalf("got %d", got)
	}
}
