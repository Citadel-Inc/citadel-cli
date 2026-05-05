// Package pager pipes stdout through an external pager (less by default)
// when the CLI is talking to an interactive terminal. Mirrors gh / git
// behavior: long list verbs page automatically, scripts are unaffected.
package pager

import (
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// Overridable for tests: whether stdout is an interactive terminal.
var isStdoutInteractive = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Resolve returns the pager command line in precedence order:
// CITADEL_PAGER > GIT_PAGER > PAGER > "less -FRX". An explicit empty value
// at any tier disables paging (matches git's PAGER="" convention).
//
// Returned string is shell-tokenized later via /bin/sh -c so users can
// embed flags ("less -R", "moar -mousable") naturally.
func Resolve() string {
	for _, k := range []string{"CITADEL_PAGER", "GIT_PAGER", "PAGER"} {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
	}
	return "less -FRX"
}

// Start swaps os.Stdout for a pipe to the resolved pager when the original
// stdout is an interactive TTY and the user has not opted out (--no-pager,
// or any PAGER tier set to ""). Returns a cleanup that closes the pipe and
// waits for the pager to exit.
//
// On any failure path the cleanup is a no-op and os.Stdout is unchanged,
// so callers can defer cleanup() unconditionally.
func Start(disabled bool) (cleanup func(), err error) {
	noop := func() {}
	if disabled {
		return noop, nil
	}
	if !isStdoutInteractive() {
		return noop, nil
	}
	cmdline := strings.TrimSpace(Resolve())
	if cmdline == "" || cmdline == "cat" {
		return noop, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return noop, err
	}

	cmd := exec.Command("/bin/sh", "-c", cmdline)
	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// LESS=FRX gives a sensible default for users who set PAGER=less without
	// flags. Mirrors git's PAGER environment.
	cmd.Env = append(os.Environ(), "LESS=FRX", "LV=-c")
	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return noop, err
	}
	_ = pr.Close()

	originalStdout := os.Stdout
	os.Stdout = pw

	return func() {
		_ = pw.Close()
		_ = cmd.Wait()
		os.Stdout = originalStdout
	}, nil
}

