package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

// SSHKeyCmd is the top-level `citadel-cli ssh-key` command.
var SSHKeyCmd = &cobra.Command{
	Use:     "ssh-key",
	Aliases: []string{"ssh-keys"},
	Short:   "Manage SSH public keys for your account",
	Long: `List, add, and delete SSH **public** keys registered for Git authentication.

Only public key material is ever sent to the server. Private key files
(OPENSSH PRIVATE KEY / RSA PRIVATE KEY blocks) are rejected.`,
}

var sshKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSH public keys on your account",
	RunE:  runSSHKeyList,
}

var sshKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register an SSH public key",
	Long: `Registers a public key with your account.

Provide the key via --public-key, --key-file (preferred), or pipe the key on
stdin when stdin is not a TTY. Use --key-file - to read from stdin even on a
TTY.

Optional --label is stored with the key.`,
	RunE: runSSHKeyAdd,
}

var sshKeyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an SSH public key by id",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSHKeyDelete,
}

type sshKeyRow struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	PublicKey   string    `json:"public_key"`
	Label       *string   `json:"label"`
	CreatedAt   time.Time `json:"created_at"`
}

func runSSHKeyList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	var payload struct {
		Keys []sshKeyRow `json:"keys"`
	}
	if err := c.Get(cmd.Context(), "/account/ssh-keys", &payload); err != nil {
		return err
	}
	rows := payload.Keys
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		if len(rows) == 0 {
			return emitCSVHeaderOnly[sshKeyRow](cmd)
		}
		var hdr bool
		return emitCSVRows(cmd, &hdr, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	default:
		if len(rows) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No SSH public keys registered.")
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tFINGERPRINT\tLABEL\tCREATED")
		for _, r := range rows {
			short := r.ID
			if len(short) > 8 {
				short = short[:8]
			}
			lbl := ""
			if r.Label != nil {
				lbl = *r.Label
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", short, r.Fingerprint, lbl, r.CreatedAt.Format(time.RFC3339))
		}
		return w.Flush()
	}
}

func runSSHKeyAdd(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	pubFlag, _ := cmd.Flags().GetString("public-key")
	keyFile, _ := cmd.Flags().GetString("key-file")
	label, _ := cmd.Flags().GetString("label")
	label = strings.TrimSpace(label)

	material, source, err := resolveSSHPublicKeyMaterial(pubFlag, keyFile)
	if err != nil {
		return err
	}
	if err := validateSSHPublicKeyMaterial(material, source); err != nil {
		return err
	}

	body := map[string]string{
		"public_key": strings.TrimSpace(material),
	}
	if label != "" {
		body["label"] = label
	}

	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	switch output {
	case "", "json":
	default:
		return fmt.Errorf("--output for add supports json or default human summary only; got %q", output)
	}

	var created sshKeyRow
	if err := c.Post(cmd.Context(), "/account/ssh-keys", body, &created); err != nil {
		return err
	}
	if output == "json" {
		return emitJSON(cmd, created)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Registered SSH key %s (%s).\n", created.ID, created.Fingerprint)
	return nil
}

func resolveSSHPublicKeyMaterial(publicKeyFlag, keyFile string) (material string, source string, err error) {
	publicKeyFlag = strings.TrimSpace(publicKeyFlag)
	keyFile = strings.TrimSpace(keyFile)

	switch {
	case publicKeyFlag != "" && keyFile != "":
		return "", "", fmt.Errorf("use either --public-key or --key-file, not both")
	case publicKeyFlag != "":
		return publicKeyFlag, "--public-key", nil
	case keyFile != "":
		if keyFile == "-" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", "", fmt.Errorf("read stdin: %w", err)
			}
			return string(b), "stdin", nil
		}
		b, err := os.ReadFile(keyFile)
		if err != nil {
			return "", "", fmt.Errorf("read --key-file: %w", err)
		}
		return string(b), keyFile, nil
	default:
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return "", "", fmt.Errorf("no key source: pass --public-key, --key-file path, --key-file -, or pipe a public key on stdin")
		}
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), "stdin", nil
	}
}

func validateSSHPublicKeyMaterial(raw string, source string) error {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fmt.Errorf("empty public key from %s", source)
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "private key") || strings.Contains(lower, "openssh private key") {
		return fmt.Errorf("refusing %s: input looks like a private key — pass a .pub file or the single-line public key only", source)
	}
	first := strings.TrimSpace(strings.Split(s, "\n")[0])
	if strings.HasPrefix(first, "ssh-ed25519") || strings.HasPrefix(first, "ssh-rsa") || strings.HasPrefix(first, "ecdsa-sha2-") {
		return nil
	}
	return fmt.Errorf("public key from %s must start with ssh-ed25519, ssh-rsa, or ecdsa-sha2-*", source)
}

func runSSHKeyDelete(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	path := "/account/ssh-keys/" + url.PathEscape(id)
	if err := c.Delete(cmd.Context(), path); err != nil {
		var he *apiclient.HTTPError
		if errors.As(err, &he) && (he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusForbidden) {
			return fmt.Errorf("SSH key %s not found for this account", id)
		}
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted SSH key %s.\n", id)
	return nil
}

func completeSSHKeyIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	vals, err := completion.Lookup(cmd.Context(), serverFlag(cmd), completion.KeySSHKeys, completion.FetchSSHKeyIDs)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return vals, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	SSHKeyCmd.AddCommand(sshKeyListCmd)
	SSHKeyCmd.AddCommand(sshKeyAddCmd)
	SSHKeyCmd.AddCommand(sshKeyDeleteCmd)

	addOutputFlag(sshKeyListCmd, sshKeyAddCmd)
	sshKeyAddCmd.Flags().String("public-key", "", "Public key string (single line)")
	sshKeyAddCmd.Flags().String("key-file", "", "Path to a .pub file; use - for stdin")
	sshKeyAddCmd.Flags().String("label", "", "Optional human-readable label stored with the key")

	sshKeyDeleteCmd.ValidArgsFunction = completeSSHKeyIDs
	sshKeyAddCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeySSHKeys)
	}
	sshKeyDeleteCmd.PostRun = func(cmd *cobra.Command, _ []string) {
		scheduleCompletionInvalidate(serverFlag(cmd), completion.KeySSHKeys)
	}
}
