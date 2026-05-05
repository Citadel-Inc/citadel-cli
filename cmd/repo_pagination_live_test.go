package cmd

import (
	"os"
	"testing"
)

// Live pagination walk (250+ repos) is expensive and needs a dedicated test
// namespace on a real Citadel instance. Enable explicitly:
//
//	CITADEL_TEST_PAGINATION_LIVE=1 CITADEL_TEST_OAUTH_JWT=… go test ./cmd -run TestLiveRepoListPaginationAll -count=1
func TestLiveRepoListPaginationAll(t *testing.T) {
	if os.Getenv("CITADEL_TEST_PAGINATION_LIVE") != "1" {
		t.Skip("set CITADEL_TEST_PAGINATION_LIVE=1 (and CITADEL_TEST_OAUTH_JWT) for live pagination against a populated test namespace")
	}
	if os.Getenv("CITADEL_TEST_OAUTH_JWT") == "" {
		t.Skip("CITADEL_TEST_OAUTH_JWT required for authenticated live calls")
	}
	t.Skip("automated 250-repo population + count assertion not yet wired; run manual smoke per cli-pagination spec C2")
}
