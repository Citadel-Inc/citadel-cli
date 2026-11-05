package httpx

import (
	"context"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestServerVerifier(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") }))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}
	trusted := x509.NewCertPool()
	trusted.AddCert(srv.Certificate())
	empty := x509.NewCertPool()
	dir := t.TempDir()
	dialer := &net.Dialer{}

	get := func(roots *x509.CertPool) error {
		t.Helper()
		tr := &http.Transport{ForceAttemptHTTP2: true, DialTLSContext: serverVerifier{dir: dir, systemRoots: roots}.dialTLS(dialer.DialContext)}
		defer tr.CloseIdleConnections()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := tr.RoundTrip(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	if err := get(empty); err == nil {
		t.Fatal("a server nothing vouches for was accepted")
	}
	if err := get(trusted); err != nil {
		t.Fatalf("trusted by the system store: %v", err)
	}
	v := serverVerifier{dir: dir}
	if fi, err := os.Stat(v.path(host)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("kept root: %v, %v", fi, err)
	}
	if err := get(empty); err != nil {
		t.Fatalf("the kept root should answer without the system store: %v", err)
	}

	// The kept root vouches for this host only.
	if err := v.verify("elsewhere.example", []*x509.Certificate{srv.Certificate()}); err == nil {
		t.Fatal("certificate accepted for another host")
	}

	// Past its time, the kept root no longer answers: trust goes back to the system store.
	old := time.Now().Add(-2 * rootCacheTTL)
	if err := os.Chtimes(v.path(host), old, old); err != nil {
		t.Fatal(err)
	}
	if err := get(empty); err == nil {
		t.Fatal("an expired kept root was trusted")
	}

	// A damaged file is the same as none.
	if err := os.WriteFile(v.path(host), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := get(trusted); err != nil {
		t.Fatalf("after damage: %v", err)
	}
}
