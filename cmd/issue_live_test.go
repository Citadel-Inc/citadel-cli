package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLiveIssues_roundTrip_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_ISSUES_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_ISSUES_LIVE=1 for live issue integration")
	}
	ns := strings.TrimSpace(os.Getenv("CITADEL_TEST_ISSUES_NS"))
	if ns == "" {
		t.Skip("CITADEL_TEST_ISSUES_NS unset — provide a writable namespace path")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live issue API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	title := "cli-issues-live-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	var createOut bytes.Buffer
	create := NewRootCmd()
	create.SetArgs([]string{"issue", "create", "-R", ns, "--title", title, "--body", "live body", "--output", "json"})
	create.SetOut(&createOut)
	create.SetErr(io.Discard)
	create.SilenceErrors = true
	create.SilenceUsage = true
	if err := create.Execute(); err != nil {
		t.Fatal(err)
	}
	var created issueRow
	if err := json.Unmarshal(createOut.Bytes(), &created); err != nil {
		t.Fatalf("decode created issue: %v", err)
	}
	if created.Number < 1 {
		t.Fatalf("created issue number=%d", created.Number)
	}
	num := strconv.FormatInt(created.Number, 10)

	comment := NewRootCmd()
	comment.SetArgs([]string{"issue", "comment", "-R", ns, num, "--body", "live comment"})
	comment.SetOut(io.Discard)
	comment.SetErr(io.Discard)
	comment.SilenceErrors = true
	comment.SilenceUsage = true
	if err := comment.Execute(); err != nil {
		t.Fatal(err)
	}

	closeIssue := NewRootCmd()
	closeIssue.SetArgs([]string{"issue", "close", "-R", ns, num})
	closeIssue.SetOut(io.Discard)
	closeIssue.SetErr(io.Discard)
	closeIssue.SilenceErrors = true
	closeIssue.SilenceUsage = true
	if err := closeIssue.Execute(); err != nil {
		t.Fatal(err)
	}

	var refsOut bytes.Buffer
	closeRefs := NewRootCmd()
	closeRefs.SetArgs([]string{"issue", "close-refs", "-R", ns, num, "--output", "json"})
	closeRefs.SetOut(&refsOut)
	closeRefs.SetErr(io.Discard)
	closeRefs.SilenceErrors = true
	closeRefs.SilenceUsage = true
	if err := closeRefs.Execute(); err != nil {
		t.Fatal(err)
	}
	var refs []issueCloseRef
	if err := json.Unmarshal(refsOut.Bytes(), &refs); err != nil {
		t.Fatalf("decode close refs: %v", err)
	}
}
