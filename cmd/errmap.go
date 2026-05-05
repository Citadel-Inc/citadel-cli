package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// FriendlyError translates infrastructure-flavored errors (raw HTTP status
// codes, dial-tcp DNS failures, deadline-exceeded contexts) into actionable
// CLI-user-facing messages. Pass-through for errors that already carry a
// friendly message.
//
// Returns the input unchanged when no mapping applies, so callers can wrap
// every CLI verb's terminal error in one call without losing any detail
// the verb already added.
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}

	// Network reachability — DNS or refused connections.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return errors.New("cannot reach Citadel server: hostname lookup failed — check your internet connection or override the server with --server <url> (or the CITADEL_SERVER env var)")
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return errors.New("cannot reach Citadel server: connection failed — is the server URL reachable from this host? Override with --server / CITADEL_SERVER")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out — server took too long to respond; retry, or check https://status.src.land")
	}

	// Server-side errors mapped to actionable text.
	var he *apiclient.HTTPError
	if errors.As(err, &he) {
		switch {
		case he.StatusCode == http.StatusUnauthorized:
			return errors.New("authentication failed: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN")
		case he.StatusCode == http.StatusForbidden:
			return errors.New("forbidden: the server rejected your token for this resource — check namespace ownership or token scope")
		case he.StatusCode == http.StatusServiceUnavailable:
			return errors.New("citadel server is temporarily unavailable; retry in a few seconds, or check https://status.src.land")
		case he.StatusCode == http.StatusBadGateway, he.StatusCode == http.StatusGatewayTimeout:
			return errors.New("upstream server unreachable; retry in a few seconds, or check https://status.src.land")
		case he.StatusCode == http.StatusTooManyRequests:
			return errors.New("rate limit exceeded — slow down or wait a few minutes before retrying")
		case he.StatusCode >= 500:
			return fmt.Errorf("citadel server error (HTTP %d); retry, and if the problem persists check https://status.src.land", he.StatusCode)
		}
	}

	// Catch the io.EOF / "unexpected EOF" stream-cut cases the apiclient
	// wraps as a generic request-failed error.
	if strings.Contains(err.Error(), "EOF") && !strings.Contains(err.Error(), "decode") {
		return errors.New("connection cut by the server; retry, and if the problem persists check https://status.src.land")
	}

	return err
}
