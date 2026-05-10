package cmd

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	citadelLicensePubkeyEnv = "CITADEL_LICENSE_PUBKEY"
	licenseGracePeriod      = 7 * 24 * time.Hour
)

type licensePayload struct {
	LicenseID     string    `json:"license_id"`
	InstanceID    string    `json:"instance_id"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	SeatCount     int       `json:"seat_count"`
	Tier          string    `json:"tier"`
	Features      []string  `json:"features"`
	AllowedIPs    []string  `json:"allowed_ips"`
	Signature     string    `json:"signature"`
}

var selfHostLicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Inspect and validate a Citadel license file",
}

var selfHostLicenseValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a license file (signature, expiry, fields)",
	Long: `Parses a Citadel license JSON file and reports its fields and expiry state.

Signature verification runs when CITADEL_LICENSE_PUBKEY env (or --pubkey)
provides the Ed25519 public key (base64-standard or base64url). Pass
--skip-signature to only check schema + expiry when no key is configured.

Examples:
  citadel-cli self-host license validate --file ./license.json
  CITADEL_LICENSE_PUBKEY=$KEY citadel-cli self-host license validate --file ./license.json
  citadel-cli self-host license validate --file ./license.json --skip-signature
  cat license.json | citadel-cli self-host license validate --skip-signature`,
	RunE: runSelfHostLicenseValidate,
}

func runSelfHostLicenseValidate(cmd *cobra.Command, _ []string) error {
	path, _ := cmd.Flags().GetString("file")
	skipSig, _ := cmd.Flags().GetBool("skip-signature")
	pubB64, _ := cmd.Flags().GetString("pubkey")
	if strings.TrimSpace(pubB64) == "" {
		pubB64 = strings.TrimSpace(os.Getenv(citadelLicensePubkeyEnv))
	}

	raw, err := readLicenseInput(cmd, path)
	if err != nil {
		return err
	}

	var lic licensePayload
	if err := json.Unmarshal(raw, &lic); err != nil {
		return fmt.Errorf("parse license JSON: %w", err)
	}

	var missing []string
	if lic.LicenseID == "" {
		missing = append(missing, "license_id")
	}
	if lic.InstanceID == "" {
		missing = append(missing, "instance_id")
	}
	if lic.ExpiresAt.IsZero() {
		missing = append(missing, "expires_at")
	}
	if lic.Signature == "" {
		missing = append(missing, "signature")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	sigStatus := ""
	sigOK := false
	switch {
	case skipSig:
		sigStatus = "skipped (--skip-signature)"
	case pubB64 == "":
		return fmt.Errorf("license public key not configured: set %s env, pass --pubkey, or use --skip-signature", citadelLicensePubkeyEnv)
	default:
		if err := verifyLicenseSignature(raw, lic.Signature, pubB64); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
		sigOK = true
		sigStatus = "verified"
	}

	now := time.Now().UTC()
	daysLeft := int(lic.ExpiresAt.Sub(now).Hours() / 24)
	var warnings []string
	var state string
	switch {
	case daysLeft > 30:
		state = "OK"
	case daysLeft >= 0:
		state = "WARNING"
		warnings = append(warnings, fmt.Sprintf("expires in %d day(s)", daysLeft+1))
	case now.Before(lic.ExpiresAt.Add(licenseGracePeriod)):
		state = "GRACE"
		graceLeft := int(lic.ExpiresAt.Add(licenseGracePeriod).Sub(now).Hours() / 24)
		warnings = append(warnings, fmt.Sprintf("expired %d day(s) ago; %d day(s) grace remaining", -daysLeft, graceLeft))
	default:
		return fmt.Errorf("license expired %d day(s) ago; grace period elapsed", -daysLeft)
	}

	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}

	if output == "json" {
		return emitJSON(cmd, map[string]any{
			"license_id":       lic.LicenseID,
			"instance_id":      lic.InstanceID,
			"customer_name":    lic.CustomerName,
			"customer_email":   lic.CustomerEmail,
			"tier":             lic.Tier,
			"seat_count":       lic.SeatCount,
			"features":         lic.Features,
			"allowed_ips":      lic.AllowedIPs,
			"issued_at":        lic.IssuedAt,
			"expires_at":       lic.ExpiresAt,
			"days_left":        daysLeft,
			"state":            state,
			"signature_ok":     sigOK,
			"signature_status": sigStatus,
			"warnings":         warnings,
		})
	}
	if output == "yaml" {
		return emitYAML(cmd, map[string]any{
			"license_id":       lic.LicenseID,
			"instance_id":      lic.InstanceID,
			"customer_name":    lic.CustomerName,
			"customer_email":   lic.CustomerEmail,
			"tier":             lic.Tier,
			"seat_count":       lic.SeatCount,
			"features":         lic.Features,
			"allowed_ips":      lic.AllowedIPs,
			"issued_at":        lic.IssuedAt,
			"expires_at":       lic.ExpiresAt,
			"days_left":        daysLeft,
			"state":            state,
			"signature_ok":     sigOK,
			"signature_status": sigStatus,
			"warnings":         warnings,
		})
	}

	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "LICENSE\t%s\n", lic.LicenseID)
	_, _ = fmt.Fprintf(w, "INSTANCE\t%s\n", lic.InstanceID)
	if lic.CustomerName != "" || lic.CustomerEmail != "" {
		_, _ = fmt.Fprintf(w, "CUSTOMER\t%s <%s>\n", lic.CustomerName, lic.CustomerEmail)
	}
	if lic.Tier != "" {
		_, _ = fmt.Fprintf(w, "TIER\t%s\n", lic.Tier)
	}
	seatStr := "unlimited"
	if lic.SeatCount > 0 {
		seatStr = fmt.Sprintf("%d", lic.SeatCount)
	}
	_, _ = fmt.Fprintf(w, "SEATS\t%s\n", seatStr)
	if !lic.IssuedAt.IsZero() {
		_, _ = fmt.Fprintf(w, "ISSUED\t%s\n", lic.IssuedAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintf(w, "EXPIRES\t%s (in %d day(s))\n", lic.ExpiresAt.Format(time.RFC3339), daysLeft)
	_, _ = fmt.Fprintf(w, "STATE\t%s\n", state)
	_, _ = fmt.Fprintf(w, "SIGNATURE\t%s\n", sigStatus)
	if len(lic.Features) > 0 {
		_, _ = fmt.Fprintf(w, "FEATURES\t%s\n", strings.Join(lic.Features, ", "))
	}
	if len(lic.AllowedIPs) > 0 {
		_, _ = fmt.Fprintf(w, "ALLOWED IPS\t%s\n", strings.Join(lic.AllowedIPs, ", "))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	for _, warn := range warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "WARN: %s\n", warn)
	}
	return nil
}

func readLicenseInput(cmd *cobra.Command, path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return b, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read license file: %w", err)
	}
	return b, nil
}

// verifyLicenseSignature mirrors the citadel-side validator: parse the JSON,
// remove the "signature" field, re-marshal (sorted keys via encoding/json),
// and verify the Ed25519 signature against the supplied public key.
func verifyLicenseSignature(raw []byte, sigB64, pubB64 string) error {
	sig, err := decodeBase64Loose(sigB64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("signature must be %d bytes, got %d", ed25519.SignatureSize, len(sig))
	}
	pubKey, err := decodeBase64Loose(pubB64)
	if err != nil {
		return fmt.Errorf("decode pubkey: %w", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("pubkey must be %d bytes, got %d", ed25519.PublicKeySize, len(pubKey))
	}

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("re-parse for signing: %w", err)
	}
	delete(fields, "signature")
	payload, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("re-serialize for signing: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pubKey), payload, sig) {
		return fmt.Errorf("signature does not match")
	}
	return nil
}

// decodeBase64Loose accepts both standard and URL base64, with or without padding.
func decodeBase64Loose(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

func init() {
	SelfHostCmd.AddCommand(selfHostLicenseCmd)
	selfHostLicenseCmd.AddCommand(selfHostLicenseValidateCmd)

	selfHostLicenseValidateCmd.Flags().String("file", "", "Path to license JSON file (default: stdin)")
	selfHostLicenseValidateCmd.Flags().Bool("skip-signature", false, "Skip signature verification (schema + expiry only)")
	selfHostLicenseValidateCmd.Flags().String("pubkey", "", "Ed25519 public key (base64); falls back to CITADEL_LICENSE_PUBKEY env")
	addOutputFlag(selfHostLicenseValidateCmd)
}
