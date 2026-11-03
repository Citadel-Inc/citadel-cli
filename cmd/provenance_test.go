package cmd_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

// Produced by the server's provenance.Sign over a fixed seed, so this test
// fails if the CLI's canonicalisation drifts from the signer's.
const (
	goldenEnvelope = `{"report":{"big":12345678901234567,"gaps":["1 commit(s) have no receive record <&>"],"kind":"citadel.provenance.report","repo":{"path":"acme/app"},"scope":{"pr":7,"type":"pull_request"},"version":1},"signature":{"alg":"ed25519","key_id":"56475aa75463474c","value":"aW+7FwTUkhRCArAJtYDtnmetgoPRq4f4PTXRBm9Y50G4IWClShpDimPy2OTgbEZar2tVtQOJTyvy5HHh7ZJ3BA=="}}`
	goldenPub      = "A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg="
)

func runProvenance(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd.NewRootCmd()
	var out strings.Builder
	root := rootForOut(cmd.ProvenanceCmd, &out, args...)
	err := root.Execute()
	return out.String(), err
}

func TestProvenanceVerify(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "report.json")
	if err := os.WriteFile(good, []byte(goldenEnvelope), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runProvenance(t, "verify", good, "--key", goldenPub)
	if err != nil || !strings.Contains(out, "signature OK") || !strings.Contains(out, "gap: 1 commit(s)") {
		t.Fatalf("verify: %v %q", err, out)
	}

	// The published key-set shape, searched by key id, also verifies.
	set := filepath.Join(dir, "keys.json")
	if err := os.WriteFile(set, []byte(`{"keys":[{"key_id":"other","public_key":"`+goldenPub+`"},{"key_id":"56475aa75463474c","public_key":"`+goldenPub+`"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runProvenance(t, "verify", good, "--key", set); err != nil {
		t.Fatalf("key set: %v", err)
	}

	// One flipped byte in the signed report fails, as does a different key.
	tampered := strings.Replace(goldenEnvelope, `"pr":7`, `"pr":8`, 1)
	bad := filepath.Join(dir, "tampered.json")
	if err := os.WriteFile(bad, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runProvenance(t, "verify", bad, "--key", goldenPub); err == nil || !strings.Contains(err.Error(), "INVALID") {
		t.Fatalf("tampered report verified: %v", err)
	}
	other := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if _, err := runProvenance(t, "verify", good, "--key", other); err == nil {
		t.Fatal("verified under the wrong key")
	}

	// Reformatting does not break the signature.
	pretty := filepath.Join(dir, "pretty.json")
	if err := os.WriteFile(pretty, []byte(strings.ReplaceAll(goldenEnvelope, `","`, "\",\n  \"")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runProvenance(t, "verify", pretty, "--key", goldenPub); err != nil {
		t.Fatalf("pretty-printed: %v", err)
	}
}

func TestHookCommitTrailers_InstallAndRemove(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("CITADEL_GIT_HOSTS", "")
	srv := withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sessions"):
			writeJSON(t, w, 201, map[string]any{"session_id": "sess-9", "agent_id": "a1", "agent_name": "bot", "token": "tok", "expires_at": time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
	_ = srv
	repo := gitRepoWithOrigin(t, "git@src.land:acme/app.git")
	hook := filepath.Join(repo, ".git", "hooks", "prepare-commit-msg")
	in := `{"session_id":"sess-9","cwd":"` + repo + `"}`

	runHook(t, in, "session-start")
	script, err := readFile(hook)
	if err != nil || !strings.Contains(string(script), "citadel-cli provenance hook") {
		t.Fatalf("hook not installed: %v", err)
	}

	msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msg, []byte("work\n\nbody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "citadel", "sessions", "sess-9")
	runHook(t, "", "commit-trailers", "--state", stateDir, msg)
	got, _ := readFile(msg)
	if !strings.Contains(string(got), "Agent-Session: sess-9") || !strings.Contains(string(got), "Agent: bot") {
		t.Fatalf("trailers missing:\n%s", got)
	}

	runHook(t, in, "session-end")
	if _, err := os.Stat(hook); err == nil {
		t.Fatal("hook survived session end")
	}

	// A developer's own hook is never replaced or removed.
	mine := "#!/bin/sh\necho mine\n"
	if err := os.WriteFile(hook, []byte(mine), 0o755); err != nil { //nolint:gosec // test hook must be executable
		t.Fatal(err)
	}
	runHook(t, in, "session-start")
	if b, _ := readFile(hook); string(b) != mine {
		t.Fatalf("developer hook overwritten: %q", b)
	}
	runHook(t, in, "session-end")
	if b, _ := readFile(hook); string(b) != mine {
		t.Fatalf("developer hook removed: %q", b)
	}
}

// A verifier must judge the same report a reader sees. Go matches JSON keys
// case-insensitively and keeps the last duplicate, so without strict decoding a
// signed report can sit beside a forged one that other tools read instead.
func TestProvenanceVerifyRefusesAmbiguousEnvelopes(t *testing.T) {
	report := goldenEnvelope[len(`{"report":`):strings.Index(goldenEnvelope, `,"signature":`)]
	sig := goldenEnvelope[strings.Index(goldenEnvelope, `"signature":`)+len(`"signature":`) : len(goldenEnvelope)-1]
	forged := strings.Replace(report, `acme/app`, `evil/app`, 1)
	for name, env := range map[string]string{
		"forged report before a signed REPORT": `{"report":` + forged + `,"REPORT":` + report + `,"signature":` + sig + `}`,
		"duplicate report key":                 `{"report":` + forged + `,"report":` + report + `,"signature":` + sig + `}`,
		"escaped duplicate report key":         `{"report":` + forged + `,"report":` + report + `,"signature":` + sig + `}`,
		"unknown envelope key":                 `{"report":` + report + `,"signature":` + sig + `,"note":"x"}`,
		"cased signature key":                  `{"report":` + report + `,"SIGNATURE":` + sig + `}`,
		"duplicate key inside the report":      `{"report":{"repo":{"path":"evil/app"},` + report[1:] + `,"signature":` + sig + `}`,
		"duplicate key inside the signature":   `{"report":` + report + `,"signature":{"alg":"none",` + sig[1:] + `}`,
	} {
		file := filepath.Join(t.TempDir(), "report.json")
		if err := os.WriteFile(file, []byte(env), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := runProvenance(t, "verify", file, "--key", goldenPub); err == nil {
			t.Errorf("%s verified: %q", name, out)
		}
	}
}
