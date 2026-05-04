package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

// WaitlistCmd is the parent for `citadel-cli waitlist {grant,revoke,list}`.
//
// Per spec go-waitlist-email-allowlist Phase C2, these subcommands share
// the same backend endpoints as the operator UI surface and require the
// signed-in CLI user to hold operator:waitlist:* grants on the system namespace
// (same backend as the operator UI). Callers without grants see the same 404
// response that the UI gets.
var WaitlistCmd = &cobra.Command{
	Use:   "waitlist",
	Short: "Manage the per-email waitlist allowlist (admin-only)",
	Long: `Operator commands for the per-email override on the waitlist gate.

The domain-based gate stays in code; this surface is the explicit-grant
path so trusted users on personal-domain mail (e.g. Gmail) can be granted
access without a redeploy.

Allowlist does NOT confer operator RBAC — operator atoms are granted only on the
system namespace via SQL or citadel-cli grants tooling.`,
}

var waitlistGrantCmd = &cobra.Command{
	Use:   "grant <email>",
	Short: "Grant a single email access through the waitlist gate",
	Args:  cobra.ExactArgs(1),
	RunE:  runWaitlistGrant,
}

var waitlistRevokeCmd = &cobra.Command{
	Use:   "revoke <email>",
	Short: "Soft-revoke an active waitlist grant",
	Args:  cobra.ExactArgs(1),
	RunE:  runWaitlistRevoke,
}

var waitlistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active waitlist allowlist entries",
	RunE:  runWaitlistList,
}

func init() {
	waitlistGrantCmd.Flags().String("note", "", "Optional operator note (e.g. 'issue #1')")
	WaitlistCmd.AddCommand(waitlistGrantCmd)
	WaitlistCmd.AddCommand(waitlistRevokeCmd)
	WaitlistCmd.AddCommand(waitlistListCmd)
}

type waitlistEntry struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	GrantedBy uuid.UUID  `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Note      string     `json:"note,omitempty"`
}

func runWaitlistGrant(cmd *cobra.Command, args []string) error {
	cfg, serverURL, err := waitlistAuthCfg(cmd)
	if err != nil {
		return err
	}
	email := args[0]
	note, _ := cmd.Flags().GetString("note")

	body, _ := json.Marshal(map[string]string{"email": email, "note": note})
	req, _ := http.NewRequest("POST", serverURL+"/operator/waitlist/allowlist", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusCreated:
		var e waitlistEntry
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		fmt.Printf("Granted: %s (id=%s, granted_at=%s)\n", e.Email, e.ID.String()[:8], e.GrantedAt.Format(time.RFC3339))
		return nil
	case http.StatusConflict:
		return fmt.Errorf("already active: %s already has an active waitlist grant", email)
	case http.StatusNotFound:
		return fmt.Errorf("admin required: this account does not have operator access")
	default:
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(b))
	}
}

func runWaitlistRevoke(cmd *cobra.Command, args []string) error {
	cfg, serverURL, err := waitlistAuthCfg(cmd)
	if err != nil {
		return err
	}
	email := args[0]
	revokeURL := serverURL + "/operator/waitlist/allowlist/" + url.PathEscape(email)
	req, _ := http.NewRequest("DELETE", revokeURL, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Printf("Revoked: %s\n", email)
		return nil
	case http.StatusNotFound:
		// Endpoint mounted but row missing OR admin check failed. Disambiguate
		// by reading the body (handler emits {"error":"not_found"} on missing
		// row; non-admin gets the bare 404 page).
		b, _ := io.ReadAll(resp.Body)
		if bytes.Contains(b, []byte("not_found")) {
			return fmt.Errorf("no allowlist row for %s", email)
		}
		return fmt.Errorf("admin required: this account does not have operator access")
	default:
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(b))
	}
}

func runWaitlistList(cmd *cobra.Command, args []string) error {
	cfg, serverURL, err := waitlistAuthCfg(cmd)
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("GET", serverURL+"/operator/waitlist/allowlist", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("admin required: this account does not have operator access")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(b))
	}

	var body struct {
		Entries []waitlistEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if len(body.Entries) == 0 {
		fmt.Println("No active waitlist allowlist entries.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprint(w, "EMAIL\tGRANTED\tNOTE\n"); err != nil {
		return err
	}
	for _, e := range body.Entries {
		note := e.Note
		if len(note) > 40 {
			note = note[:37] + "..."
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", e.Email, e.GrantedAt.Format("2006-01-02 15:04Z"), note); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func waitlistAuthCfg(cmd *cobra.Command) (clicfg.Config, string, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return clicfg.Config{}, "", fmt.Errorf("load config: %w", err)
	}
	if cfg.AccessToken == "" {
		return clicfg.Config{}, "", fmt.Errorf("not authenticated; run 'citadel-cli auth login' first")
	}
	flagServer, _ := cmd.Flags().GetString("server")
	return cfg, cfg.ResolveServerURL(flagServer), nil
}
