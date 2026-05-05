package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

func TestNew_RequiresToken(t *testing.T) {
	if _, err := New(clicfg.Config{}, ""); err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected not-authenticated error, got %v", err)
	}
}

func TestClient_GetDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("missing/incorrect bearer header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"alice"}`))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ Name string }
	if err := c.Get(context.Background(), "/whoami", &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "alice" {
		t.Fatalf("got %q", out.Name)
	}
}

func TestClient_PostSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type")
		}
		var body struct{ X int }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.X != 42 {
			t.Errorf("got %d", body.X)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	var out struct{ ID string }
	if err := c.Post(context.Background(), "/things", map[string]int{"x": 42}, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "abc" {
		t.Fatalf("got %q", out.ID)
	}
}

func TestClient_DeleteAcceptsNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	if err := c.Delete(context.Background(), "/things/1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_NonSuccessReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden body"))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	err := c.Get(context.Background(), "/x", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.StatusCode != http.StatusForbidden {
		t.Errorf("got status %d", he.StatusCode)
	}
	if he.Body != "forbidden body" {
		t.Errorf("got body %q", he.Body)
	}
}

func TestHTTPError_ErrorAndDecodeBody(t *testing.T) {
	he := &HTTPError{StatusCode: 409, Body: `{"error":"has_repos","detail":"x"}`}
	if got := he.Error(); !strings.Contains(got, "409") || !strings.Contains(got, "has_repos") {
		t.Fatalf("Error() = %q", got)
	}
	var body struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if err := he.DecodeBody(&body); err != nil {
		t.Fatalf("DecodeBody: %v", err)
	}
	if body.Error != "has_repos" || body.Detail != "x" {
		t.Fatalf("decoded %+v", body)
	}
	// nil target and empty body each short-circuit.
	if err := he.DecodeBody(nil); err != nil {
		t.Fatalf("DecodeBody(nil): %v", err)
	}
	if err := (&HTTPError{Body: ""}).DecodeBody(&body); err != nil {
		t.Fatalf("DecodeBody empty: %v", err)
	}
}

func TestIsStatus(t *testing.T) {
	err := &HTTPError{StatusCode: 404}
	if !IsStatus(err, 404) {
		t.Fatal("expected match on direct *HTTPError")
	}
	wrapped := errors.Join(errors.New("ctx"), err)
	if !IsStatus(wrapped, 404) {
		t.Fatal("expected match through errors.Join")
	}
	if IsStatus(err, 500) {
		t.Fatal("did not expect match for different code")
	}
	if IsStatus(errors.New("plain"), 404) {
		t.Fatal("plain errors must not match")
	}
	if IsStatus(nil, 404) {
		t.Fatal("nil must not match")
	}
}

func TestClient_ServerAndToken(t *testing.T) {
	c, err := New(clicfg.Config{ServerURL: "https://x.test/", AccessToken: "tok"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Server(); got != "https://x.test" {
		t.Fatalf("Server() = %q", got)
	}
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() = %q", got)
	}
}

func TestClient_PutSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body struct{ Y int }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Y != 7 {
			t.Errorf("got %d", body.Y)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	var out struct{ OK bool }
	if err := c.Put(context.Background(), "/things/1", map[string]int{"y": 7}, &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Fatalf("got %+v", out)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Get(ctx, "/x", nil); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled, got %v", err)
	}
}
