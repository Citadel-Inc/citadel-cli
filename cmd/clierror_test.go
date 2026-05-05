package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLIError_Error_nilReceiver(t *testing.T) {
	var e *CLIError
	if got := e.Error(); got != "" {
		t.Fatalf("nil receiver Error() = %q", got)
	}
}

func TestDeepestCLIError_fmtWrap(t *testing.T) {
	inner := &CLIError{Kind: KindAuthRequired, Message: "inner"}
	wrapped := fmt.Errorf("outer: %w", inner)
	got, ok := DeepestCLIError(wrapped)
	if !ok || got != inner {
		t.Fatalf("DeepestCLIError = %v ok=%v", got, ok)
	}
}

func TestDeepestCLIError_none(t *testing.T) {
	if _, ok := DeepestCLIError(errors.New("plain")); ok {
		t.Fatal("expected no *CLIError")
	}
}

func TestResolveCLIExit_plainErrorInternal(t *testing.T) {
	in := errors.New("verb failed: no classification")
	f := FriendlyError(in)
	ce, code := ResolveCLIExit(in, f)
	if code != 1 || ce.Kind != KindInternal {
		t.Fatalf("code=%d kind=%s", code, ce.Kind)
	}
	if !strings.Contains(ce.Message, "verb failed") {
		t.Fatalf("message = %q", ce.Message)
	}
}

func TestWantsMachineErrorEnvelope(t *testing.T) {
	if WantsMachineErrorEnvelope(nil) {
		t.Fatal("nil command must be false")
	}
	tests := []struct {
		args string
		want bool
	}{
		{"--output=json", true},
		{"--output YAML", true},
		{"--output Ndjson", true},
		{"--output=table", false},
		{"", false},
	}
	for _, tc := range tests {
		c := &cobra.Command{Use: "x"}
		c.Flags().String("output", "", "")
		var args []string
		if tc.args != "" {
			args = strings.Fields(tc.args)
		}
		if err := c.ParseFlags(args); err != nil {
			t.Fatalf("ParseFlags %q: %v", tc.args, err)
		}
		if got := WantsMachineErrorEnvelope(c); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.args, got, tc.want)
		}
	}
}

func TestWriteErrorEnvelope_ndjsonOneLine(t *testing.T) {
	ce := &CLIError{Kind: KindForbidden, Message: "no", HTTPStatus: 403}
	var buf bytes.Buffer
	if err := WriteErrorEnvelope(&buf, "ndjson", ce); err != nil {
		t.Fatal(err)
	}
	line := buf.String()
	if !strings.HasSuffix(line, "\n") || strings.Count(line, "\n") != 1 {
		t.Fatalf("want single trailing newline, got %q", line)
	}
	var env errorWireEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error == nil || env.Error.Kind != KindForbidden {
		t.Fatalf("decode: %+v", env)
	}
}

func TestWriteErrorEnvelope_yamlContainsErrorKey(t *testing.T) {
	ce := &CLIError{Kind: KindNotFound, Message: "gone", HTTPStatus: 404}
	var buf bytes.Buffer
	if err := WriteErrorEnvelope(&buf, "yaml", ce); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "kind:") || !strings.Contains(buf.String(), "not_found") {
		t.Fatalf("yaml = %q", buf.String())
	}
}

func TestWriteErrorEnvelope_unknownFormatIsJSON(t *testing.T) {
	ce := &CLIError{Kind: KindInternal, Message: "x"}
	var buf bytes.Buffer
	if err := WriteErrorEnvelope(&buf, "weird", ce); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"error"`) {
		t.Fatalf("expected JSON, got %q", buf.String())
	}
}

func TestKindToExitCode_remainingKinds(t *testing.T) {
	extra := map[CLIErrorKind]int{
		KindForbidden:         3,
		KindMFARequired:       3,
		KindDryRun:            2,
		KindServerError:       7,
		KindServerUnavailable: 7,
		KindTimeout:           7,
	}
	for k, want := range extra {
		if got := KindToExitCode(k); got != want {
			t.Errorf("%s: got %d want %d", k, got, want)
		}
	}
}
