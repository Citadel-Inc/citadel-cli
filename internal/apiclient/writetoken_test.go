package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

func TestWriteTokenAuthorisesWritesOnly(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	for _, tc := range []struct {
		name  string
		write func(context.Context) (string, error)
		want  string
	}{
		{"jwt", func(context.Context) (string, error) { return "jwt", nil }, "POST Bearer jwt"},
		{"falls back", func(context.Context) (string, error) { return "", errors.New("no") }, "POST Bearer agent"},
		{"unset", nil, "POST Bearer agent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got = nil
			c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "agent"}, Options{WriteToken: tc.write})
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Get(t.Context(), "/x", nil); err != nil {
				t.Fatal(err)
			}
			if err := c.Post(t.Context(), "/x", nil, nil); err != nil {
				t.Fatal(err)
			}
			if got[0] != "GET Bearer agent" || got[1] != tc.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}
