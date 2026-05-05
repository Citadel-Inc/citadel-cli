package sseclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

func TestOpen_Close_noBody(t *testing.T) {
	ctx := context.Background()
	c, err := apiclient.New(clicfg.Config{ServerURL: "http://unused.test", AccessToken: "x"}, apiclient.Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := Open(ctx, c, "/z")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStream_Next_contextAlreadyCanceled(t *testing.T) {
	c, err := apiclient.New(clicfg.Config{ServerURL: "http://unused.test", AccessToken: "x"}, apiclient.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := Open(ctx, c, "/p")
	_, err = s.Next()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestStream_Next_singleEventThenCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Last-Event-ID") != "" {
			t.Errorf("unexpected Last-Event-ID on first connect: %q", r.Header.Get("Last-Event-ID"))
		}
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "event: add\nid: 7\ndata: {\"x\":1}\n\n")
		fl.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, err := apiclient.New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, apiclient.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := Open(ctx, c, "/stream")
	defer func() { _ = s.Close() }()

	ev, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "add" || ev.ID != "7" || string(ev.Data) != `{"x":1}` {
		t.Fatalf("ev = %#v", ev)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err = s.Next()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want canceled, got %v", err)
	}
}

func TestStream_Next_reconnectEOFPreservesLastEventID(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := n.Add(1)
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		switch seq {
		case 1:
			if r.Header.Get("Last-Event-ID") != "" {
				t.Errorf("first connect Last-Event-ID = %q", r.Header.Get("Last-Event-ID"))
			}
			_, _ = fmt.Fprintf(w, "event: init\nid: 100\ndata: one\n\n")
			fl.Flush()
			return // close connection → client EOF
		case 2:
			if got := r.Header.Get("Last-Event-ID"); got != "100" {
				t.Errorf("reconnect Last-Event-ID = %q want 100", got)
			}
			_, _ = fmt.Fprintf(w, "event: add\nid: 101\ndata: two\n\n")
			fl.Flush()
			<-r.Context().Done()
		default:
			http.Error(w, "unexpected extra connect", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	c, _ := apiclient.New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, apiclient.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := Open(ctx, c, "/s")
	defer func() { _ = s.Close() }()

	ev1, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev1.ID != "100" || string(ev1.Data) != "one" {
		t.Fatalf("ev1 = %#v", ev1)
	}
	ev2, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev2.ID != "101" || ev2.Type != "add" {
		t.Fatalf("ev2 = %#v", ev2)
	}
	cancel()
}

func TestStream_Next_HTTP401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	c, _ := apiclient.New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, apiclient.Options{})
	s := Open(context.Background(), c, "/x")
	defer func() { _ = s.Close() }()

	_, err := s.Next()
	if err == nil {
		t.Fatal("expected error")
	}
	var he *apiclient.HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got %v", err)
	}
}

func TestStream_Next_SSE_errorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "event: error\ndata: boom\n\n")
		fl.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, _ := apiclient.New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, apiclient.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := Open(ctx, c, "/s")
	defer func() { _ = s.Close() }()

	_, err := s.Next()
	if err == nil || !strings.Contains(err.Error(), "sse stream error") {
		t.Fatalf("got %v", err)
	}
}

func TestStream_Next_defaultMessageEventType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: hello\n\n")
		fl.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, _ := apiclient.New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, apiclient.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := Open(ctx, c, "/s")
	defer func() { _ = s.Close() }()

	ev, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "message" || string(ev.Data) != "hello" {
		t.Fatalf("ev = %#v", ev)
	}
	cancel()
}
