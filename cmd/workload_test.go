package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

func TestWorkloadLoginOIDCPrintsExports(t *testing.T) {
	var got map[string]any
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /workload/sessions": func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			if r.Header.Get("Authorization") != "" {
				t.Error("workload login must not send a bearer token")
			}
			writeJSON(t, w, 201, map[string]any{"session_id": "s-1", "token": "tok-abc", "expires_at": time.Now().Add(15 * time.Minute)})
		},
	}))
	dir := t.TempDir()
	tokFile := filepath.Join(dir, "jwt")
	if err := os.WriteFile(tokFile, []byte("header.payload.sig\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	keyDir := filepath.Join(dir, "keys")
	if err := rootForOut(cmd.WorkloadCmd, &out, "login", "--org", "acme", "--oidc-token-file", tokFile, "--key-dir", keyDir, "--scope", "code:read").Execute(); err != nil {
		t.Fatal(err)
	}
	pub, _ := got["ssh_public_key"].(string)
	if got["org"] != "acme" || got["oidc_token"] != "header.payload.sig" || !strings.HasPrefix(pub, "ssh-ed25519 ") {
		t.Fatalf("request body: %v", got)
	}
	for _, want := range []string{"export CITADEL_ACCESS_TOKEN='tok-abc'", "export CITADEL_WORKLOAD_SESSION_ID='s-1'", "export GIT_SSH_COMMAND='ssh -i '\\''" + keyDir} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(keyDir, "id_ed25519")); err != nil {
		t.Errorf("session key not written: %v", err)
	}
}

func TestWorkloadLoginNeedsExactlyOneProof(t *testing.T) {
	withServer(t, route(t, nil))
	for _, args := range [][]string{
		{"login", "--org", "acme"},
		{"login", "--org", "acme", "--aws", "--oidc-token-file", "x"},
		{"login", "--oidc-token-file", "x"},
	} {
		if err := rootForOut(cmd.WorkloadCmd, &bytes.Buffer{}, args...).Execute(); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}

func TestBuildAWSProofSignsAudienceAndHost(t *testing.T) {
	p, err := cmd.BuildAWSProofForTest(t.Context(), aws.Credentials{AccessKeyID: "AKIATEST", SecretAccessKey: "s", SessionToken: "tok"},
		"eu-west-1", "citadel-aud", time.Date(2026, 11, 3, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	authz := p.Headers["Authorization"]
	signed := regexp.MustCompile(`SignedHeaders=([^,]+)`).FindStringSubmatch(authz)
	if signed == nil || !strings.Contains(";"+signed[1]+";", ";x-citadel-audience;") || !strings.Contains(";"+signed[1]+";", ";host;") {
		t.Fatalf("audience and host must be signed: %s", authz)
	}
	if !strings.Contains(authz, "/20261103/eu-west-1/sts/aws4_request") || p.Region != "eu-west-1" || p.Headers["X-Amz-Security-Token"] != "tok" {
		t.Fatalf("proof: %+v", p)
	}
}
