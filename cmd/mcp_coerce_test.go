package cmd

import (
	"errors"
	"os"
	"reflect"
	"runtime"
	"testing"
)

func TestCoerceArg(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		// Strings.
		{"hello", "hello"},
		{"", ""},
		{"damon", "damon"},

		// Booleans.
		{"true", true},
		{"false", false},
		{"True", "True"}, // canonical form only
		{"yes", "yes"},

		// Integers.
		{"5", int64(5)},
		{"0", int64(0)},
		{"-7", int64(-7)},
		{"100", int64(100)},

		// Leading-zero IDs / zip codes stay as strings (don't clobber).
		{"07823", "07823"},
		{"00", "00"},

		// Floats.
		{"1.5", 1.5},
		{"-3.14", -3.14},

		// Float-shaped that should NOT parse.
		{".5", ".5"},
		{"5.", "5."},
		{"1.2.3", "1.2.3"},

		// CSV → array, with each element coerced.
		{"a,b,c", []any{"a", "b", "c"}},
		{"1,2,3", []any{int64(1), int64(2), int64(3)}},
		{"1,foo,3", []any{int64(1), "foo", int64(3)}},
		{"true,false", []any{true, false}},

		// Single-element trailing comma still produces array.
		{"x,", []any{"x", ""}},
	}
	for _, tc := range cases {
		got := coerceArg(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("coerceArg(%q) = %#v want %#v", tc.in, got, tc.want)
		}
	}
}

func TestPickToken(t *testing.T) {
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("flag", "jwt"); got != "flag" {
		t.Errorf("flag wins: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "envtok")
	if got := pickToken("", "jwt"); got != "envtok" {
		t.Errorf("env beats jwt: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("", "jwt"); got != "jwt" {
		t.Errorf("jwt fallback: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("", ""); got != "" {
		t.Errorf("nothing: got %q", got)
	}
}

func TestParseArgPairs(t *testing.T) {
	got, err := parseArgPairs([]string{"a=1", "b=hello,world"}, []string{"c=42"})
	if err != nil {
		t.Fatalf("parseArgPairs: %v", err)
	}
	if got["a"] != int64(1) {
		t.Errorf("a coerce: got %v", got["a"])
	}
	if got["c"] != "42" {
		t.Errorf("--arg-string forces string: got %v", got["c"])
	}
	arr, ok := got["b"].([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("b CSV: got %#v", got["b"])
	}

	if _, err := parseArgPairs([]string{"missingequals"}, nil); err == nil {
		t.Error("expected error on bare --arg key")
	}
	if _, err := parseArgPairs(nil, []string{"badpair"}); err == nil {
		t.Error("expected error on bare --arg-string key")
	}
}

func TestSurfaceErr_Passthrough(t *testing.T) {
	in := errors.New("plain")
	if got := surfaceErr(in); got != in {
		t.Errorf("non-auth errors must pass through unchanged: %v", got)
	}
}

func TestClipboardCommand_LinuxNoTools(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only path")
	}
	// Pin PATH so wl-copy / xclip are unreachable.
	t.Setenv("PATH", t.TempDir())
	if _, err := clipboardCommand(); err == nil {
		t.Error("expected error when neither wl-copy nor xclip is on PATH")
	}
}

func TestCopySecretToClipboard_LinuxFakeTool(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only path")
	}
	dir := t.TempDir()
	// Drop in a stub `wl-copy` that just consumes stdin and exits 0.
	stub := dir + "/wl-copy"
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := copySecretToClipboard("secret"); err != nil {
		t.Fatalf("copySecretToClipboard with stub: %v", err)
	}
}

func TestCopySecretToClipboard_ToolFails(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only path")
	}
	dir := t.TempDir()
	stub := dir + "/wl-copy"
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := copySecretToClipboard("secret"); err == nil {
		t.Fatal("expected error from failing clipboard tool")
	}
}

func TestFormatCaller(t *testing.T) {
	if got := formatCaller(impactNode{Name: "foo", Path: "a.go"}); got != "foo  in a.go" {
		t.Errorf("named+path: %q", got)
	}
	if got := formatCaller(impactNode{ID: "bar"}); got != "bar" {
		t.Errorf("id-only fallback: %q", got)
	}
}

func TestSplitOwnerRepo(t *testing.T) {
	owner, repo := splitOwnerRepo("alice/repo")
	if owner != "alice" || repo != "repo" {
		t.Errorf("alice/repo split: %q,%q", owner, repo)
	}
	owner, repo = splitOwnerRepo("alice")
	if owner != "alice" || repo != "" {
		t.Errorf("owner-only split: %q,%q", owner, repo)
	}
}

func TestUpgradeUnauthorized(t *testing.T) {
	if got := upgradeUnauthorized(errors.New("plain")); got.Error() != "plain" {
		t.Errorf("non-401 must pass through: %v", got)
	}
}

func TestResolveMCPURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://api.src.land", "https://mcp.src.land/mcp"},
		{"https://api.src.land/", "https://mcp.src.land/mcp"},
		{"https://mcp.src.land", "https://mcp.src.land/mcp"},
		{"http://localhost:8080", "http://localhost:8080/mcp"},
		{"http://localhost:8080/", "http://localhost:8080/mcp"},
	}
	for _, tc := range cases {
		got := resolveMCPURL(tc.in)
		if got != tc.want {
			t.Errorf("resolveMCPURL(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}
