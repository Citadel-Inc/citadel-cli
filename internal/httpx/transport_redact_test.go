package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTraceTransportRedactsTokenResponseBody(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = old })

	const access = "super-secret-access-token"
	const refresh = "super-secret-refresh-token"
	body := `{"access_token":"` + access + `","refresh_token":"` + refresh + `","token_type":"Bearer"}`
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:        "200 OK",
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        http.Header{"Content-Type": []string{"application/json"}},
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       req,
		}, nil
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.com/oauth/token", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&TraceTransport{Base: base, DebugHTTP: true}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	dumped, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	out := string(dumped)
	if strings.Contains(out, access) || strings.Contains(out, refresh) {
		t.Fatalf("debug dump leaked token values:\n%s", out)
	}
	if !strings.Contains(out, "access_token") || !strings.Contains(out, "refresh_token") {
		t.Fatalf("debug dump dropped token fields:\n%s", out)
	}
	defer func() { _ = resp.Body.Close() }()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("response body not restored: got %q", got)
	}
}

// A download_url is a presigned link: the signature in the query string is the
// credential, so the --verbose one-liner must not print it verbatim.
func TestVerboseTrace_RedactsPresignedQuery(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &http.Client{Transport: Stack(nil, Options{Verbose: true})}
	reqURL := srv.URL + "/obj/x.bin?X-Amz-Credential=AKIASECRETKEY%2F20260101&X-Amz-Signature=deadbeefSECRETSIG"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	os.Stderr = old
	_ = w.Close()
	dumped, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	out := string(dumped)

	for _, secret := range []string{"AKIASECRETKEY", "deadbeefSECRETSIG"} {
		if strings.Contains(out, secret) {
			t.Fatalf("verbose trace leaked presigned credential %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, "/obj/x.bin") || !strings.Contains(out, "200") {
		t.Fatalf("verbose trace lost the request line: %s", out)
	}
}

func TestRedactJSONTokensHidesClientSecrets(t *testing.T) {
	in := `{"client_id":"c","client_secret":"s3cret","nested":{"cleartext_secret":"hook-secret"}}`
	out := string(redactJSONTokens([]byte(in)))
	for _, leak := range []string{"s3cret", "hook-secret"} {
		if strings.Contains(out, leak) {
			t.Fatalf("%q survived redaction: %s", leak, out)
		}
	}
}

func TestRedactJSONTokensHidesWorkloadProofs(t *testing.T) {
	in := `{"org":"o","oidc_token":"oidc-jwt","aws":{"region":"us-east-1","headers":{"Authorization":"AWS4-HMAC-SHA256 Signature=sig","X-Amz-Security-Token":"aws-session"}}}`
	out := string(redactJSONTokens([]byte(in)))
	for _, leak := range []string{"oidc-jwt", "Signature=sig", "aws-session"} {
		if strings.Contains(out, leak) {
			t.Fatalf("%q survived redaction: %s", leak, out)
		}
	}
}
