package httpx

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
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

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T) testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test ca"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return testCA{cert, key}
}

// serve starts a TLS server whose leaf the CA signs for the given names and validity.
func (ca testCA) serve(t *testing.T, dns []string, notBefore, notAfter time.Time) *httptest.Server {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "leaf"},
		DNSNames: dns, IPAddresses: nil, NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if len(dns) == 0 {
		tmpl.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") }))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

// With a root kept for the host, a wrong name, an expired leaf and a chain from
// another root are all still refused, on a fresh connection each time.
func TestServerVerifier_RefusesBadServersWithKeptRoot(t *testing.T) {
	ca, other := newTestCA(t), newTestCA(t)
	system := x509.NewCertPool()
	system.AddCert(ca.cert)
	dir := t.TempDir()
	now := time.Now()

	get := func(srv *httptest.Server) error {
		t.Helper()
		tr := &http.Transport{ForceAttemptHTTP2: true, DialTLSContext: serverVerifier{dir: dir, systemRoots: system}.dialTLS((&net.Dialer{}).DialContext)}
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

	good := ca.serve(t, nil, now.Add(-time.Hour), now.Add(time.Hour))
	if err := get(good); err != nil {
		t.Fatalf("good server: %v", err)
	}
	if _, err := os.Stat(serverVerifier{dir: dir}.path("127.0.0.1")); err != nil {
		t.Fatalf("root not kept: %v", err)
	}
	for name, srv := range map[string]*httptest.Server{
		"wrong hostname":  ca.serve(t, []string{"other.example"}, now.Add(-time.Hour), now.Add(time.Hour)),
		"expired leaf":    ca.serve(t, nil, now.Add(-2*time.Hour), now.Add(-time.Hour)),
		"untrusted chain": other.serve(t, nil, now.Add(-time.Hour), now.Add(time.Hour)),
	} {
		if err := get(srv); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := get(good); err != nil {
		t.Fatalf("good server after the refusals: %v", err)
	}
}
