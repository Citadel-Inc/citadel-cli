package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type fakeGitInvocation struct {
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

func TestHelperProcessGit(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_GIT") != "1" {
		return
	}
	logPath := os.Getenv("CITADEL_GIT_TEST_LOG")
	args := os.Args
	for i := range args {
		if args[i] == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	env := map[string]string{}
	for _, key := range []string{"GIT_ASKPASS", "GIT_TERMINAL_PROMPT", "SSH_ASKPASS"} {
		env[key] = os.Getenv(key)
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		os.Exit(3)
	}
	_ = json.NewEncoder(f).Encode(fakeGitInvocation{Args: args, Env: env})
	_ = f.Close()

	switch strings.Join(args, " ") {
	case "git remote get-url origin":
		_, _ = io.WriteString(os.Stdout, "https://src.land/myorg/r1.git\n")
	case "git symbolic-ref --quiet --short HEAD":
		_, _ = io.WriteString(os.Stdout, "feature\n")
	}
	os.Exit(0)
}

func TestRepoClone_InvokesGitCloneWithHTTPSRepo(t *testing.T) {
	logPath, cleanup := patchGitExec(t)
	defer cleanup()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "https://mcp.src.land")
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	root := repoTestRoot(t)
	var out, errBuf strings.Builder
	root.SetOut(&out)
	root.SetErr(&errBuf)
	setOutRecursiveTest(root, &out, &errBuf)
	root.SetArgs([]string{"repo", "clone", "myorg/r1"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "r1\n") {
		t.Fatalf("stdout=%q", got)
	}
	invocations := readGitLog(t, logPath)
	if len(invocations) != 1 {
		t.Fatalf("want 1 invocation, got %d", len(invocations))
	}
	gotArgs := strings.Join(invocations[0].Args, " ")
	if gotArgs != "git clone https://src.land/myorg/r1.git" {
		t.Fatalf("args=%q", gotArgs)
	}
	if invocations[0].Env["GIT_TERMINAL_PROMPT"] != "0" {
		t.Fatalf("env=%v", invocations[0].Env)
	}
	if invocations[0].Env["GIT_ASKPASS"] == "" {
		t.Fatalf("missing GIT_ASKPASS in env=%v", invocations[0].Env)
	}
}

func TestRepoPush_CreateFlagCreatesMissingRepoThenPushesOrigin(t *testing.T) {
	logPath, cleanup := patchGitExec(t)
	defer cleanup()

	var createdBranch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /namespaces/myorg/r1":
			w.WriteHeader(http.StatusNotFound)
		case "POST /namespaces/myorg/repos":
			var body struct {
				Slug          string  `json:"slug"`
				DefaultBranch *string `json:"default_branch"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.DefaultBranch != nil {
				createdBranch = *body.DefaultBranch
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"parent_slug": "myorg",
				"slug":        "r1",
				"visibility":  "private",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	root := repoTestRoot(t)
	var out, errBuf strings.Builder
	root.SetOut(&out)
	root.SetErr(&errBuf)
	setOutRecursiveTest(root, &out, &errBuf)
	root.SetArgs([]string{"repo", "push", "--create"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if createdBranch != "feature" {
		t.Fatalf("default_branch=%q want feature", createdBranch)
	}
	invocations := readGitLog(t, logPath)
	if len(invocations) != 3 {
		t.Fatalf("want 3 invocations (remote,get branch,push), got %d", len(invocations))
	}
	if got := strings.Join(invocations[2].Args, " "); got != "git push origin" {
		t.Fatalf("push args=%q", got)
	}
}

func TestRepoPush_MissingRepoWithoutCreateFailsNonInteractive(t *testing.T) {
	logPath, cleanup := patchGitExec(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /namespaces/myorg/r1":
			w.WriteHeader(http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	oldTTY := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	defer func() { stdinIsTerminal = oldTTY }()

	root := repoTestRoot(t)
	root.SetArgs([]string{"repo", "push"})
	err := root.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "--create") {
		t.Fatalf("err=%v", err)
	}
	if invocations := readGitLog(t, logPath); len(invocations) != 1 {
		t.Fatalf("want only remote lookup before failure, got %d", len(invocations))
	}
}

func TestRepoPush_ExplicitRepoSetsUpstreamBranch(t *testing.T) {
	logPath, cleanup := patchGitExec(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /namespaces/myorg/r1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"parent_slug": "myorg",
				"slug":        "r1",
				"visibility":  "private",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	root := repoTestRoot(t)
	root.SetArgs([]string{"repo", "push", "myorg/r1"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	invocations := readGitLog(t, logPath)
	if len(invocations) != 2 {
		t.Fatalf("want 2 git invocations (branch,push), got %d", len(invocations))
	}
	want := "git push --set-upstream " + srv.URL + "/myorg/r1.git feature"
	if got := strings.Join(invocations[1].Args, " "); got != want {
		t.Fatalf("push args=%q", got)
	}
}

func TestRepoPull_ExplicitRepoUsesCurrentBranch(t *testing.T) {
	logPath, cleanup := patchGitExec(t)
	defer cleanup()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "https://mcp.src.land")
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	root := repoTestRoot(t)
	root.SetArgs([]string{"repo", "pull", "myorg/r1"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	invocations := readGitLog(t, logPath)
	if len(invocations) != 2 {
		t.Fatalf("want 2 git invocations (branch,pull), got %d", len(invocations))
	}
	if got := strings.Join(invocations[1].Args, " "); got != "git pull https://src.land/myorg/r1.git feature" {
		t.Fatalf("pull args=%q", got)
	}
}

func TestRepoClone_GitMissingFriendlyError(t *testing.T) {
	oldLookPath := execLookPath
	execLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	defer func() { execLookPath = oldLookPath }()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	root := repoTestRoot(t)
	root.SetArgs([]string{"repo", "clone", "myorg/r1"})
	err := root.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "git is not available on PATH") {
		t.Fatalf("err=%v", err)
	}
}

func TestConfirmCreateRepo_InteractiveYes(t *testing.T) {
	oldTTY := stdinIsTerminal
	stdinIsTerminal = func() bool { return true }
	defer func() { stdinIsTerminal = oldTTY }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Errorf("close read pipe: %v", err)
		}
	}()
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()
	if _, err := io.WriteString(w, "y\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	if err := confirmCreateRepo(false, "myorg/r1"); err != nil {
		t.Fatalf("confirmCreateRepo: %v", err)
	}
}

func patchGitExec(t *testing.T) (string, func()) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "git.log")
	oldExec := execCommandContext
	oldLookPath := execLookPath
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmdArgs := append([]string{"-test.run=TestHelperProcessGit", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS_GIT=1",
			"CITADEL_GIT_TEST_LOG="+logPath,
		)
		return cmd
	}
	execLookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	return logPath, func() {
		execCommandContext = oldExec
		execLookPath = oldLookPath
	}
}

func readGitLog(t *testing.T, logPath string) []fakeGitInvocation {
	t.Helper()
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close git log: %v", err)
		}
	}()
	var out []fakeGitInvocation
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var row fakeGitInvocation
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		out = append(out, row)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func repoTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	root := NewRootCmd()
	resetFlagsRecursiveTest(root)
	t.Cleanup(func() {
		resetCommandStateRecursiveTest(root)
	})
	return root
}

func resetFlagsRecursiveTest(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, child := range c.Commands() {
		resetFlagsRecursiveTest(child)
	}
}

func resetCommandStateRecursiveTest(c *cobra.Command) {
	c.SetArgs(nil)
	c.SetOut(nil)
	c.SetErr(nil)
	resetFlagsRecursiveTest(c)
	for _, child := range c.Commands() {
		resetCommandStateRecursiveTest(child)
	}
}

func setOutRecursiveTest(c *cobra.Command, out, err io.Writer) {
	c.SetOut(out)
	c.SetErr(err)
	for _, child := range c.Commands() {
		setOutRecursiveTest(child, out, err)
	}
}
