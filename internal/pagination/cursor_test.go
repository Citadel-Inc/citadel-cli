package pagination

import (
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
