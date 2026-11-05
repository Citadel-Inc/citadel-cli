package httpx

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Citadel-Inc/citadel-cli/internal/clicfg"
)

const (
	// rootCacheTTL is how long a removal from the system's trust store can go unnoticed.
	rootCacheTTL     = 24 * time.Hour
	handshakeTimeout = 10 * time.Second
)

// serverVerifier checks a server's chain as the TLS library would, but tries first the one
// root that vouched for this host recently. Each CLI run is a new process, and loading the
// machine's whole certificate store costs more than the request itself. A chain the kept
// root does not vouch for goes to the system store, whose answer is final.
type serverVerifier struct {
	dir string
	// systemRoots replaces the system store in tests.
	systemRoots *x509.CertPool
}

func (v serverVerifier) path(host string) string {
	sum := sha256.Sum256([]byte(host))
	return filepath.Join(v.dir, hex.EncodeToString(sum[:]))
}

func (v serverVerifier) keptRoots(host string) *x509.CertPool {
	path := v.path(host)
	fi, err := os.Stat(path)
	if err != nil || time.Since(fi.ModTime()) > rootCacheTTL {
		return nil
	}
	der, err := os.ReadFile(path) //nolint:gosec // path is under the CLI's own config directory
	if err != nil {
		return nil
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		return nil
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	return pool
}

func (v serverVerifier) verify(host string, certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return errors.New("httpx: server sent no certificate")
	}
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	opts := x509.VerifyOptions{DNSName: host, Intermediates: intermediates, Roots: v.keptRoots(host)}
	if opts.Roots != nil {
		if _, err := certs[0].Verify(opts); err == nil {
			return nil
		}
	}
	opts.Roots = v.systemRoots
	chains, err := certs[0].Verify(opts)
	if err != nil {
		return err
	}
	// Failing to keep the root costs the next run a store load and nothing else.
	_ = v.keep(host, chains[0][len(chains[0])-1])
	return nil
}

func (v serverVerifier) keep(host string, root *x509.Certificate) error {
	if err := os.MkdirAll(v.dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(v.dir, "tmp-")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(root.Raw); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), v.path(host))
}

// dialTLS verifies through the verifier: InsecureSkipVerify turns off only the library's own
// check, which VerifyConnection replaces on every handshake, resumed ones included.
func (v serverVerifier) dialTLS(dial func(ctx context.Context, network, addr string) (net.Conn, error)) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		raw, err := dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		conn := tls.Client(raw, &tls.Config{
			ServerName: host,
			NextProtos: []string{"h2", "http/1.1"},
			//nolint:gosec // the library's own check is replaced by VerifyConnection below, which runs on every handshake
			InsecureSkipVerify: true,
			VerifyConnection:   func(cs tls.ConnectionState) error { return v.verify(host, cs.PeerCertificates) },
		})
		hctx, cancel := context.WithTimeout(ctx, handshakeTimeout)
		defer cancel()
		if err := conn.HandshakeContext(hctx); err != nil {
			_ = raw.Close()
			return nil, err
		}
		return conn, nil
	}
}

// defaultBase is the stock transport with kept roots beside the CLI's config. Connections
// through a proxy skip DialTLSContext and verify against the system store as usual.
func defaultBase() http.RoundTripper {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok || base.DialContext == nil {
		return http.DefaultTransport
	}
	path, err := clicfg.ConfigPath()
	if err != nil {
		return base
	}
	t := base.Clone()
	t.DialTLSContext = serverVerifier{dir: filepath.Join(filepath.Dir(path), "tls-roots")}.dialTLS(base.DialContext)
	return t
}
