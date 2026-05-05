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
