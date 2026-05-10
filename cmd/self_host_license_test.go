package cmd_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// signedLicenseBytes builds a license JSON signed with the supplied Ed25519
// private key. Used to round-trip the validator against canonical input.
func signedLicenseBytes(t *testing.T, priv ed25519.PrivateKey, fields map[string]any) []byte {
	t.Helper()
	delete(fields, "signature")
	payload, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, payload)
	fields["signature"] = base64.URLEncoding.EncodeToString(sig)
	out, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func newSignedLicenseFile(t *testing.T, fields map[string]any) (path, pubKeyB64 string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data := signedLicenseBytes(t, priv, fields)
	path = filepath.Join(t.TempDir(), "license.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, base64.StdEncoding.EncodeToString(pub)
}

func TestSelfHostLicenseValidate_Happy(t *testing.T) {
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	issued := time.Now().UTC().Format(time.RFC3339)
	path, pubKey := newSignedLicenseFile(t, map[string]any{
		"license_id":     "lic-1",
		"instance_id":    "inst-1",
		"customer_name":  "Acme",
		"customer_email": "ops@acme.io",
		"tier":           "premium",
		"seat_count":     10,
		"issued_at":      issued,
		"expires_at":     expires,
	})
	var out strings.Builder
	rc := rootForOut(cmd.SelfHostCmd, &out, "license", "validate", "--file", path, "--pubkey", pubKey)
	if err := rc.Execute(); err != nil {
		t.Fatalf("validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "verified") {
		t.Fatalf("expected SIGNATURE verified, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "lic-1") {
		t.Fatalf("expected license id in output: %s", out.String())
	}
}

func TestSelfHostLicenseValidate_BadSignature(t *testing.T) {
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	path, pubKey := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-1",
		"instance_id": "inst-1",
		"expires_at":  expires,
	})
	// Use a different pubkey (random) — verification must fail.
	otherPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherB64 := base64.StdEncoding.EncodeToString(otherPub)
	_ = pubKey

	err = rootFor(cmd.SelfHostCmd, "license", "validate", "--file", path, "--pubkey", otherB64).Execute()
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("want signature error, got %v", err)
	}
}

func TestSelfHostLicenseValidate_SkipSignature(t *testing.T) {
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	path, _ := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-1",
		"instance_id": "inst-1",
		"expires_at":  expires,
	})
	var out strings.Builder
	rc := rootForOut(cmd.SelfHostCmd, &out, "license", "validate", "--file", path, "--skip-signature")
	if err := rc.Execute(); err != nil {
		t.Fatalf("validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Fatalf("expected SIGNATURE skipped, got: %s", out.String())
	}
}

func TestSelfHostLicenseValidate_RequiresPubkey(t *testing.T) {
	t.Setenv(citadelLicensePubkeyEnvForTest(), "")
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	path, _ := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-1",
		"instance_id": "inst-1",
		"expires_at":  expires,
	})
	err := rootFor(cmd.SelfHostCmd, "license", "validate", "--file", path).Execute()
	if err == nil || !strings.Contains(err.Error(), "public key not configured") {
		t.Fatalf("want pubkey-required error, got %v", err)
	}
}

func TestSelfHostLicenseValidate_MissingField(t *testing.T) {
	bad := map[string]any{"license_id": "lic-1"} // missing instance_id, expires_at, signature
	raw, _ := json.Marshal(bad)
	path := filepath.Join(t.TempDir(), "license.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	err := rootFor(cmd.SelfHostCmd, "license", "validate", "--file", path, "--skip-signature").Execute()
	if err == nil || !strings.Contains(err.Error(), "missing required fields") {
		t.Fatalf("want missing-fields error, got %v", err)
	}
}

func TestSelfHostLicenseValidate_ExpiredPastGrace(t *testing.T) {
	// expired 30 days ago; grace is 7 days → must error.
	past := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
	path, pub := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-1",
		"instance_id": "inst-1",
		"expires_at":  past,
	})
	err := rootFor(cmd.SelfHostCmd, "license", "validate", "--file", path, "--pubkey", pub).Execute()
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("want expired error, got %v", err)
	}
}

func TestSelfHostLicenseValidate_GraceWindow(t *testing.T) {
	// expired 3 days ago; still inside the 7-day grace window.
	past := time.Now().UTC().AddDate(0, 0, -3).Format(time.RFC3339)
	path, pub := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-1",
		"instance_id": "inst-1",
		"expires_at":  past,
	})
	var out strings.Builder
	rc := rootForOut(cmd.SelfHostCmd, &out, "license", "validate", "--file", path, "--pubkey", pub)
	if err := rc.Execute(); err != nil {
		t.Fatalf("validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "GRACE") {
		t.Fatalf("expected GRACE state, got: %s", out.String())
	}
}

func TestSelfHostLicenseValidate_Stdin(t *testing.T) {
	expires := time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	data := signedLicenseBytes(t, priv, map[string]any{
		"license_id":  "lic-stdin",
		"instance_id": "inst-1",
		"expires_at":  expires,
	})

	// rootFor wires SetIn at the root level; cobra's InOrStdin() walks up
	// the parent chain so the leaf validate command sees this reader.
	rc := rootForOut(cmd.SelfHostCmd, new(bytes.Buffer), "license", "validate", "--pubkey", base64.StdEncoding.EncodeToString(pub), "--file", "-")
	rc.SetIn(bytes.NewReader(data))
	if err := rc.Execute(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestSelfHostLicenseValidate_JSONOutput(t *testing.T) {
	expires := time.Now().UTC().AddDate(0, 6, 0).Format(time.RFC3339)
	path, pub := newSignedLicenseFile(t, map[string]any{
		"license_id":  "lic-json",
		"instance_id": "inst-1",
		"expires_at":  expires,
		"tier":        "standard",
		"features":    []string{"audit-retention", "scim"},
	})
	var out strings.Builder
	rc := rootForOut(cmd.SelfHostCmd, &out, "license", "validate", "--file", path, "--pubkey", pub, "--output", "json")
	if err := rc.Execute(); err != nil {
		t.Fatalf("validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"signature_ok": true`) {
		t.Fatalf("expected signature_ok true in JSON, got: %s", out.String())
	}
	if !strings.Contains(out.String(), `"state": "OK"`) && !strings.Contains(out.String(), `"state": "WARNING"`) {
		t.Fatalf("expected state field, got: %s", out.String())
	}
}

// citadelLicensePubkeyEnvForTest is the same env name the cmd package
// references; duplicated here as a constant so tests don't import unexported
// identifiers.
func citadelLicensePubkeyEnvForTest() string {
	return "CITADEL_LICENSE_PUBKEY"
}
