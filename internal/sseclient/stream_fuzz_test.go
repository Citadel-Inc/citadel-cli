package sseclient

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func FuzzReadEvent(f *testing.F) {
	f.Add("event: message\ndata: hello\n\n")
	f.Add("id: 1\ndata: {\"ok\":true}\n\n")
	f.Add(": comment\ndata: x\n\n")
	f.Add("data: line1\ndata: line2\n\n")
	f.Add("event: ping\n\n")
	f.Add("id: abc\nevent: message\ndata: {}\n\n")
	f.Fuzz(func(t *testing.T, raw string) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		ev, err := readEvent(ctx, bufio.NewReader(strings.NewReader(raw)))
		if err != nil {
			return
		}
		var buf bytes.Buffer
		if ev.ID != "" {
			buf.WriteString("id: ")
			buf.WriteString(ev.ID)
			buf.WriteByte('\n')
		}
		if ev.Type != "" {
			buf.WriteString("event: ")
			buf.WriteString(ev.Type)
			buf.WriteByte('\n')
		}
		if len(ev.Data) > 0 {
			for i, line := range bytes.Split(ev.Data, []byte("\n")) {
				if i > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString("data: ")
				buf.Write(line)
			}
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')

		ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel2()
		ev2, err := readEvent(ctx2, bufio.NewReader(bytes.NewReader(buf.Bytes())))
		if err != nil {
			t.Fatalf("round-trip parse failed: %v\nencoded:\n%s", err, buf.Bytes())
		}
		if ev2.ID != ev.ID || ev2.Type != ev.Type || !bytes.Equal(ev2.Data, ev.Data) {
			t.Fatalf("round-trip mismatch\nfirst: %+v\nsecond: %+v", ev, ev2)
		}
	})
}
