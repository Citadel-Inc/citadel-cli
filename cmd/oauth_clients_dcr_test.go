package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestOAuthClientsDcrFilter(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/oauth/clients" {
			t.Errorf("request = %s %s, want GET /oauth/clients", r.Method, r.URL.Path)
		}
		if _, ok := r.URL.Query()["dcr"]; ok {
			t.Error("request unexpectedly included a dcr query parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if got := r.URL.Query().Get("cursor"); got != "" {
				t.Errorf("first cursor = %q, want empty", got)
			}
			_, _ = fmt.Fprint(w, `{"clients":[{"id":"c1","client_id":"static-1","name":"Static","dcr":false},{"id":"c2","client_id":"dcr-1","name":"DCR one","dcr":true}],"next_cursor":"page-2"}`)
		case 2:
			if got := r.URL.Query().Get("cursor"); got != "page-2" {
				t.Errorf("second cursor = %q, want page-2", got)
			}
			_, _ = fmt.Fprint(w, `{"clients":[{"id":"c3","client_id":"static-2","name":"Static two","dcr":false},{"id":"c4","client_id":"dcr-2","name":"DCR two","dcr":true}],"next_cursor":""}`)
		default:
			t.Errorf("unexpected request %d", requests)
			http.Error(w, "too many requests", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	var out bytes.Buffer
	if err := oauthClientsDcrListCommand(&out, "--dcr", "--all", "--output", "ndjson").Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("output lines = %d, want 2: %q", len(lines), out.String())
	}
	var got []oauthClient
	for _, line := range lines {
		var client oauthClient
		if err := json.Unmarshal([]byte(line), &client); err != nil {
			t.Fatalf("decode output line %q: %v", line, err)
		}
		got = append(got, client)
	}
	if got[0].ClientID != "dcr-1" || got[1].ClientID != "dcr-2" {
		t.Errorf("client IDs = %q, %q; want dcr-1, dcr-2", got[0].ClientID, got[1].ClientID)
	}
	for _, client := range got {
		if !client.Dcr {
			t.Errorf("client %q was not marked DCR", client.ClientID)
		}
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2", requests)
	}
}

func TestOAuthClientsDcrFilterFindsLaterPage(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/oauth/clients" {
			t.Errorf("request = %s %s, want GET /oauth/clients", r.Method, r.URL.Path)
		}
		if _, ok := r.URL.Query()["dcr"]; ok {
			t.Error("request unexpectedly included a dcr query parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			_, _ = fmt.Fprint(w, `{"clients":[{"id":"c1","client_id":"static-1","name":"Static","dcr":false}],"next_cursor":"page-2"}`)
		case 2:
			_, _ = fmt.Fprint(w, `{"clients":[{"id":"c2","client_id":"dcr-1","name":"DCR one","dcr":true}],"next_cursor":""}`)
		default:
			t.Errorf("unexpected request %d", requests)
			http.Error(w, "too many requests", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	var out bytes.Buffer
	if err := oauthClientsDcrListCommand(&out, "--dcr", "--all", "--output", "ndjson").Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("output lines = %d, want 1: %q", len(lines), out.String())
	}
	var got oauthClient
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("decode output line %q: %v", lines[0], err)
	}
	if got.ClientID != "dcr-1" || !got.Dcr {
		t.Errorf("client = %#v, want DCR client dcr-1", got)
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2", requests)
	}
}

func TestOAuthClientsDcrRejectsWatch(t *testing.T) {
	err := oauthClientsDcrListCommand(&bytes.Buffer{}, "--dcr", "--watch").Execute()
	if err == nil {
		t.Fatal("expected --watch and --dcr to be rejected")
	}
	if !strings.Contains(err.Error(), "--watch cannot be combined with --dcr") {
		t.Fatalf("error = %q, want DCR/watch conflict", err)
	}
}

func oauthClientsDcrListCommand(out *bytes.Buffer, args ...string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "list",
		RunE: runOAuthClientsList,
	}
	cmd.SetOut(out)
	addOutputFlag(cmd)
	addPaginationFlags(cmd)
	addWatchFlag(cmd)
	cmd.Flags().String("org", "", "")
	cmd.Flags().Bool("dcr", false, "")
	cmd.Flags().String("server", "", "")
	cmd.SetArgs(args)
	return cmd
}
