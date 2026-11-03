package cmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ProvenanceCmd is the top-level `citadel-cli provenance` command.
var ProvenanceCmd = &cobra.Command{
	Use:     "provenance",
	GroupID: "ops",
	Short:   "Fetch and verify signed change-provenance reports",
	Long: `A provenance report states who and what changed a repository's code and who
approved it, signed by the server with an Ed25519 key. Server-observed facts
(pusher, session) are kept apart from claims (commit author, trailers).

Fetching needs repo read plus the audit:read grant, or org admin.`,
}

var provenanceReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Fetch a signed report for a pull request, a commit range or a date range",
	Long: `Writes the signed report envelope to stdout (or --out). Pick exactly one scope:
--pr N, --to SHA [--from SHA], or --since T --until T (RFC3339).`,
	Args:         cobra.NoArgs,
	RunE:         runProvenanceReport,
	SilenceUsage: true,
}

var provenanceVerifyCmd = &cobra.Command{
	Use:   "verify FILE",
	Short: "Verify a report offline against a pinned or supplied public key",
	Long: `Checks the Ed25519 signature over the report's canonical JSON. No network call is
made. The key comes from --key (a base64 public key, or a file holding either one
base64 key or the JSON served at /.well-known/citadel-provenance-keys), else from
the pinned file ` + "`<user config dir>/citadel/provenance-keys.json`" + `. A key set is
searched by the key_id the report names.

Exit status is non-zero when the signature does not verify.`,
	Args:         cobra.ExactArgs(1),
	RunE:         runProvenanceVerify,
	SilenceUsage: true,
}

func registerProvenanceCommands() {
	ProvenanceCmd.AddCommand(provenanceReportCmd, provenanceVerifyCmd)
	f := provenanceReportCmd.Flags()
	f.StringP("repo", "R", "", "Repository path (org/repo)")
	f.Int("pr", 0, "Pull request number")
	f.String("from", "", "Commit range start (exclusive) SHA")
	f.String("to", "", "Commit range end SHA")
	f.String("since", "", "Date range start (RFC3339)")
	f.String("until", "", "Date range end (RFC3339)")
	f.String("out", "", "Write the envelope to this file instead of stdout")
	_ = provenanceReportCmd.MarkFlagRequired("repo")
	provenanceVerifyCmd.Flags().String("key", "", "Public key: base64 string or file path (default: pinned key file)")
}

func runProvenanceReport(cmd *cobra.Command, _ []string) error {
	repo, _ := cmd.Flags().GetString("repo")
	q := url.Values{"repo": {repo}}
	if n, _ := cmd.Flags().GetInt("pr"); n > 0 {
		q.Set("pr", strconv.Itoa(n))
	}
	for _, k := range []string{"from", "to", "since", "until"} {
		if v, _ := cmd.Flags().GetString(k); v != "" {
			q.Set(k, v)
		}
	}
	if !q.Has("pr") && !q.Has("to") && (!q.Has("since") || !q.Has("until")) {
		return errors.New("pick a scope: --pr, --to [--from], or --since with --until")
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var raw json.RawMessage
	if err := c.Get(cmd.Context(), "/provenance/report?"+q.Encode(), &raw); err != nil {
		return err
	}
	if out, _ := cmd.Flags().GetString("out"); out != "" {
		return os.WriteFile(out, append(raw, '\n'), 0o644) //nolint:gosec // a report is not secret; the caller picks the path
	}
	_, err = fmt.Fprintln(commandOut(cmd), string(raw))
	return err
}

type provEnvelope struct {
	Report    json.RawMessage `json:"report"`
	Signature struct {
		Alg   string `json:"alg"`
		KeyID string `json:"key_id"`
		Value string `json:"value"`
	} `json:"signature"`
}

// provCanonical re-encodes JSON with sorted keys and no whitespace, numbers as
// written, matching what the server signs.
func provCanonical(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

type provKey struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
}

// loadProvKeys reads a key spec: a base64 key, or a file holding one or the
// published key set.
func loadProvKeys(spec string) ([]ed25519.PublicKey, []string, error) {
	data := []byte(strings.TrimSpace(spec))
	if st, err := os.Stat(spec); err == nil && !st.IsDir() {
		b, err := os.ReadFile(spec) //nolint:gosec // the operator names this file
		if err != nil {
			return nil, nil, err
		}
		data = bytes.TrimSpace(b)
	}
	var set struct {
		Keys []provKey `json:"keys"`
	}
	var entries []provKey
	if json.Unmarshal(data, &set) == nil && len(set.Keys) > 0 {
		entries = set.Keys
	} else {
		entries = []provKey{{PublicKey: string(data)}}
	}
	var keys []ed25519.PublicKey
	var ids []string
	for _, e := range entries {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(e.PublicKey))
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, nil, errors.New("public key must be a base64 32-byte Ed25519 key")
		}
		id := e.KeyID
		if id == "" {
			h := sha256.Sum256(raw)
			id = hex.EncodeToString(h[:8])
		}
		keys = append(keys, raw)
		ids = append(ids, id)
	}
	return keys, ids, nil
}

func pinnedProvKeyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "citadel", "provenance-keys.json"), nil
}

func runProvenanceVerify(cmd *cobra.Command, args []string) error {
	spec, _ := cmd.Flags().GetString("key")
	if spec == "" {
		p, err := pinnedProvKeyPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("no --key given and no pinned key at %s; save the output of /.well-known/citadel-provenance-keys there", p)
		}
		spec = p
	}
	keys, ids, err := loadProvKeys(spec)
	if err != nil {
		return err
	}
	file, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var env provEnvelope
	if err := json.Unmarshal(file, &env); err != nil {
		return fmt.Errorf("not a provenance envelope: %w", err)
	}
	if env.Signature.Alg != "ed25519" {
		return fmt.Errorf("unsupported signature algorithm %q", env.Signature.Alg)
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature.Value)
	if err != nil {
		return errors.New("signature is not valid base64")
	}
	canon, err := provCanonical(env.Report)
	if err != nil {
		return fmt.Errorf("report is not valid JSON: %w", err)
	}
	var key ed25519.PublicKey
	for i, id := range ids {
		if id == env.Signature.KeyID || len(keys) == 1 {
			key = keys[i]
			break
		}
	}
	if key == nil {
		return fmt.Errorf("no supplied key has id %s", env.Signature.KeyID)
	}
	if !ed25519.Verify(key, canon, sig) {
		return fmt.Errorf("signature INVALID: the report was altered or was not signed by key %s", env.Signature.KeyID)
	}
	var summary provSummary
	_ = json.Unmarshal(canon, &summary)
	w := commandOut(cmd)
	_, _ = fmt.Fprintf(w, "signature OK  key=%s  repo=%s  scope=%s\n", env.Signature.KeyID, summary.Repo.Path, summary.Scope.Type)
	for _, g := range summary.Gaps {
		_, _ = fmt.Fprintf(w, "gap: %s\n", g)
	}
	return nil
}

type provSummary struct {
	Repo  provRepo  `json:"repo"`
	Scope provScope `json:"scope"`
	Gaps  []string  `json:"gaps"`
}

type provRepo struct {
	Path string `json:"path"`
}

type provScope struct {
	Type string `json:"type"`
}
