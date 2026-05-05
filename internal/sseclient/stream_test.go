package sseclient

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

func TestReadEvent_basic(t *testing.T) {
	raw := "event: init\nid: 42\ndata: {\"a\":1}\n\n"
	ev, err := readEvent(context.Background(), bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if ev.ID != "42" || ev.Type != "init" || string(ev.Data) != `{"a":1}` {
		t.Fatalf("got %#v", ev)
	}
}

func TestReadEvent_keepaliveSkipped(t *testing.T) {
	raw := ":keepalive\n\nevent: add\nid: 7\ndata: {}\n\n"
	ev, err := readEvent(context.Background(), bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "add" || ev.ID != "7" {
		t.Fatalf("got %#v", ev)
	}
}

func TestReadEvent_unknownFieldIgnored(t *testing.T) {
	raw := "retry: 5000\nevent: x\ndata: z\n\n"
	ev, err := readEvent(context.Background(), bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "x" || string(ev.Data) != "z" {
		t.Fatalf("got %#v", ev)
	}
}

func TestReadEvent_multiDataLines(t *testing.T) {
	raw := "event: msg\ndata: line1\ndata: line2\n\n"
	ev, err := readEvent(context.Background(), bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if string(ev.Data) != "line1\nline2" {
		t.Fatalf("got %q", ev.Data)
	}
}
