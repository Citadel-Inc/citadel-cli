package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

func TestClient_GetStream_PreservesBinaryBody(t *testing.T) {
	payload := []byte{0x00, 0x01, 0x7f, 0x80, 0xff}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/octet-stream" {
			t.Errorf("Accept = %q", got)
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetStream(context.Background(), "/asset")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("body = %v, want %v", got, payload)
	}
}

func TestClient_GetStream_ReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"missing"}`))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetStream(context.Background(), "/missing")
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusNotFound || httpErr.Body != `{"error":"missing"}` || httpErr.RetryAfter != 9 {
		t.Fatalf("HTTPError = %#v", httpErr)
	}
}

func TestClient_GetStream_RetriesAfterTokenRefresh(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			if got := r.Header.Get("Authorization"); got != "Bearer old" {
				t.Errorf("first Authorization = %q", got)
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer new" {
			t.Errorf("retry Authorization = %q", got)
		}
		_, _ = w.Write([]byte{0x00, 0xff})
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "old"}, Options{
		RetryOn401: func(context.Context) (string, error) { return "new", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetStream(context.Background(), "/asset")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string([]byte{0x00, 0xff}) {
		t.Fatalf("body = %v", got)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestClient_GetStream_AbsoluteSignedURLOmitsAPIHeaders(t *testing.T) {
	signed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want empty for cross-origin URL", got)
		}
		if got := r.Header.Get("Accept"); got != "" {
			t.Errorf("Accept = %q, want empty for signed URL", got)
		}
		_, _ = w.Write([]byte{0x00, 0xff})
	}))
	defer signed.Close()
	api := httptest.NewServer(http.NotFoundHandler())
	defer api.Close()

	c, err := New(clicfg.Config{ServerURL: api.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.GetStream(context.Background(), signed.URL+"/artifact.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string([]byte{0x00, 0xff}) {
		t.Fatalf("body = %v", got)
	}
}

func TestClient_PostMultipart_StreamsFileField(t *testing.T) {
	payload := []byte("release asset")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != "file" || part.FileName() != "artifact.bin" {
			t.Fatalf("part = %q/%q", part.FormName(), part.FileName())
		}
		got, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(payload) {
			t.Fatalf("payload = %q, want %q", got, payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "asset-1"})
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := c.PostMultipart(context.Background(), "/assets", "file", "artifact.bin", strings.NewReader(string(payload)), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "asset-1" {
		t.Fatalf("response ID = %q", out.ID)
	}
}

func TestClient_PostMultipart_ReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte("too large"))
	}))
	defer srv.Close()

	c, err := New(clicfg.Config{ServerURL: srv.URL, AccessToken: "tok"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = c.PostMultipart(context.Background(), "/assets", "file", "artifact.bin", strings.NewReader("payload"), nil)
	if !IsStatus(err, http.StatusRequestEntityTooLarge) {
		t.Fatalf("error = %v", err)
	}
}

func TestNewMultipartUploadBody_UsesMultipartReader(t *testing.T) {
	body, contentType := newMultipartUploadBody("file", "artifact.bin", strings.NewReader("payload"))
	defer func() { _ = body.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", contentType)
	mr, err := multipart.NewReader(req.Body, strings.TrimPrefix(contentType, "multipart/form-data; boundary=")).NextPart()
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(mr)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Fatalf("payload = %q", got)
	}
}
