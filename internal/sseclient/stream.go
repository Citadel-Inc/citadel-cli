// Package sseclient reads Citadel list watch streams (Server-Sent Events).
package sseclient

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/httpx"
)

// IdleTimeout is how long we wait for the next byte row before treating the
// stream as stalled (matches cli-watch plan: heartbeat every 15s, drop after 30s idle).
const IdleTimeout = 30 * time.Second

var (
	// ErrIdle indicates no SSE traffic within IdleTimeout.
	ErrIdle = errors.New("sse: idle timeout waiting for event data")
)

// Event is one logical SSE dispatch (after blank-line framing).
type Event struct {
	ID   string
	Type string
	Data []byte
}

// Stream follows an SSE GET with unbounded reconnect and Last-Event-ID resume.
type Stream struct {
	ctx  context.Context
	api  *apiclient.Client
	path string

	br       *bufio.Reader
	body     io.Closer
	lastID   string
	fail streak // consecutive reconnect backoff steps
}

type streak struct {
	n int
}

func (s *streak) bump() (d time.Duration) {
	d = httpx.Backoff(s.n)
	if s.n < 1024 {
		s.n++
	}
	return d
}

func (s *streak) reset() { s.n = 0 }

// Open prepares a stream handle; call Next in a loop until context cancel.
func Open(ctx context.Context, api *apiclient.Client, path string) *Stream {
	return &Stream{ctx: ctx, api: api, path: path}
}

// Close releases the active HTTP response body, if any.
func (s *Stream) Close() error {
	if s.body != nil {
		err := s.body.Close()
		s.body = nil
		s.br = nil
		return err
	}
	return nil
}

func (s *Stream) closeBody() {
	if s.body != nil {
		_ = s.body.Close()
		s.body = nil
		s.br = nil
	}
}

func terminalHTTP(err error) bool {
	var he *apiclient.HTTPError
	if !errors.As(err, &he) {
		return false
	}
	switch he.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone:
		return true
	default:
		return false
	}
}

// Next blocks until one SSE event is decoded, the context is cancelled, or a
// terminal server error is returned. Transient disconnects reconnect internally.
func (s *Stream) Next() (Event, error) {
	for {
		if err := s.ctx.Err(); err != nil {
			return Event{}, err
		}

		if s.br == nil {
			resp, err := s.api.GetEventStream(s.ctx, s.path, s.lastID)
			if err != nil {
				if terminalHTTP(err) {
					return Event{}, err
				}
				select {
				case <-s.ctx.Done():
					return Event{}, s.ctx.Err()
				case <-time.After(s.fail.bump()):
				}
				continue
			}
			s.fail.reset()
			s.body = resp.Body
			s.br = bufio.NewReader(resp.Body)
		}

		ev, err := readEvent(s.ctx, s.br)
		if err != nil {
			s.closeBody()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return Event{}, err
			}
			if terminalHTTP(err) {
				return Event{}, err
			}
			if errors.Is(err, io.EOF) || errors.Is(err, ErrIdle) {
				select {
				case <-s.ctx.Done():
					return Event{}, s.ctx.Err()
				case <-time.After(s.fail.bump()):
				}
				continue
			}
			select {
			case <-s.ctx.Done():
				return Event{}, s.ctx.Err()
			case <-time.After(s.fail.bump()):
			}
			continue
		}

		s.fail.reset()
		if ev.ID != "" {
			s.lastID = ev.ID
		}
		typ := ev.Type
		if typ == "" {
			typ = "message"
		}
		ev.Type = typ
		if typ == "error" {
			return Event{}, fmt.Errorf("sse stream error: %s", strings.TrimSpace(string(ev.Data)))
		}
		return ev, nil
	}
}

func readLineCtx(ctx context.Context, br *bufio.Reader, idle time.Duration) ([]byte, error) {
	type res struct {
		b   []byte
		err error
	}
	ch := make(chan res, 1)
	go func() {
		b, err := br.ReadBytes('\n')
		ch <- res{b, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(idle):
		return nil, ErrIdle
	case r := <-ch:
		return r.b, r.err
	}
}

func readEvent(ctx context.Context, br *bufio.Reader) (Event, error) {
	var ev Event
	var databuf bytes.Buffer
	for {
		line, err := readLineCtx(ctx, br, IdleTimeout)
		if err != nil {
			return Event{}, err
		}
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		if len(line) == 0 {
			if ev.Type != "" || databuf.Len() > 0 {
				ev.Data = append([]byte(nil), databuf.Bytes()...)
				databuf.Reset()
				return ev, nil
			}
			continue
		}
		if line[0] == ':' {
			continue
		}
		switch {
		case bytes.HasPrefix(line, []byte("id:")):
			ev.ID = strings.TrimSpace(string(line[len("id:"):]))
		case bytes.HasPrefix(line, []byte("event:")):
			ev.Type = strings.TrimSpace(string(line[len("event:"):]))
		case bytes.HasPrefix(line, []byte("data:")):
			payload := line[len("data:"):]
			if len(payload) > 0 && payload[0] == ' ' {
				payload = payload[1:]
			}
			if databuf.Len() > 0 {
				_ = databuf.WriteByte('\n')
			}
			_, _ = databuf.Write(payload)
		default:
			// Ignore unknown field names (retry:, etc.).
		}
	}
}
