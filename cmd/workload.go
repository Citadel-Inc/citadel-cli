package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

// WorkloadCmd is the top-level `citadel-cli workload` command.
var WorkloadCmd = &cobra.Command{
	Use:     "workload",
	GroupID: "auth",
	Short:   "Sign in as a server-side workload, with no developer login",
}

var workloadLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Trade a platform identity for a 15-minute session and print env exports",
	Long: `Proves the identity of the workload to an org's trust policy and prints shell
exports for an agent process: CITADEL_ACCESS_TOKEN (the session token),
CITADEL_WORKLOAD_SESSION_ID, CITADEL_WORKLOAD_EXPIRES_AT and GIT_SSH_COMMAND
(a fresh SSH key registered for the session). Use it as

  eval "$(citadel-cli workload login --org acme --oidc-token-file /var/run/secrets/tokens/citadel)"

--oidc-token-file reads an OIDC JWT whose audience is the server's configured
workload audience (an EKS projected service-account token, a GitHub Actions
OIDC token, any OIDC workload issuer). --aws signs an sts:GetCallerIdentity
request with the ambient AWS credentials (IRSA, an instance role, Bedrock
roles) and sends it for the server to verify.`,
	Args: cobra.NoArgs,
	RunE: runWorkloadLogin,
}

func registerWorkloadCommands() {
	WorkloadCmd.AddCommand(workloadLoginCmd)
	f := workloadLoginCmd.Flags()
	f.String("org", "", "Org slug whose trust policy applies (required)")
	f.String("policy", "", "Policy name, when more than one matches")
	f.String("oidc-token-file", "", "File holding an OIDC JWT for the workload")
	f.Bool("aws", false, "Prove identity with the ambient AWS credentials")
	f.String("aws-region", "", "AWS region of the STS endpoint (default: AWS_REGION)")
	f.String("audience", "", "Server workload audience, required with --aws (default: CITADEL_WORKLOAD_AUDIENCE)")
	f.StringSlice("scope", nil, "Scopes to request, a subset of the policy's (default: all of them)")
	f.String("client", "citadel-cli", "Client name recorded on the session")
	f.String("key-dir", "", "Directory for the session SSH key (default: the user runtime dir)")
}

// stsBody is the only body the server accepts in an AWS proof.
const stsBody = "Action=GetCallerIdentity&Version=2011-06-15"

type awsProof struct {
	Region  string            `json:"region"`
	Headers map[string]string `json:"headers"`
}

// buildAWSProof pre-signs sts:GetCallerIdentity for the regional endpoint with
// the audience header signed in, so the proof only works against this server.
func buildAWSProof(ctx context.Context, creds aws.Credentials, region, audience string, now time.Time) (*awsProof, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sts."+region+".amazonaws.com/", strings.NewReader(stsBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("X-Citadel-Audience", audience)
	if creds.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	}
	sum := sha256.Sum256([]byte(stsBody))
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "sts", region, now); err != nil {
		return nil, fmt.Errorf("sign sts request: %w", err)
	}
	p := &awsProof{Region: region, Headers: map[string]string{}}
	for _, h := range []string{"Authorization", "X-Amz-Date", "X-Amz-Security-Token", "Content-Type", "X-Citadel-Audience"} {
		if v := req.Header.Get(h); v != "" {
			p.Headers[h] = v
		}
	}
	return p, nil
}

func runWorkloadLogin(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(contextForCommand(cmd), 30*time.Second)
	defer cancel()
	flags := cmd.Flags()
	org, _ := flags.GetString("org")
	tokenFile, _ := flags.GetString("oidc-token-file")
	useAWS, _ := flags.GetBool("aws")
	if org == "" {
		return errors.New("--org is required")
	}
	if (tokenFile == "") == !useAWS {
		return errors.New("give exactly one of --oidc-token-file and --aws")
	}
	body := map[string]any{"org": org}
	if v, _ := flags.GetString("policy"); v != "" {
		body["policy"] = v
	}
	if v, _ := flags.GetString("client"); v != "" {
		body["client"] = v
	}
	if v, _ := flags.GetStringSlice("scope"); len(v) > 0 {
		body["scopes"] = v
	}
	if useAWS {
		proof, err := awsProofFromEnv(ctx, flags.Lookup("aws-region").Value.String(), flags.Lookup("audience").Value.String())
		if err != nil {
			return err
		}
		body["aws"] = proof
	} else {
		raw, err := os.ReadFile(filepath.Clean(tokenFile))
		if err != nil {
			return fmt.Errorf("read oidc token: %w", err)
		}
		body["oidc_token"] = strings.TrimSpace(string(raw))
	}

	dir, _ := flags.GetString("key-dir")
	if dir == "" {
		gitDir, err := authGitKeyDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(filepath.Dir(gitDir), "workload")
	}
	pub, err := writeSessionKey(dir)
	if err != nil {
		return fmt.Errorf("create session key: %w", err)
	}
	body["ssh_public_key"] = pub

	var sess struct {
		SessionID string    `json:"session_id"`
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := doPublicJSON(ctx, cmd, http.MethodPost, "/workload/sessions", body, &sess); err != nil {
		return fmt.Errorf("workload login: %w", err)
	}
	sshCmd := "ssh -i " + shellQuote(filepath.Join(dir, "id_ed25519")) + " -o IdentitiesOnly=yes"
	out := cmd.OutOrStdout()
	for _, kv := range [][2]string{
		{"CITADEL_ACCESS_TOKEN", sess.Token},
		{"CITADEL_WORKLOAD_SESSION_ID", sess.SessionID},
		{"CITADEL_WORKLOAD_EXPIRES_AT", sess.ExpiresAt.UTC().Format(time.RFC3339)},
		{"GIT_SSH_COMMAND", sshCmd},
	} {
		if _, err := fmt.Fprintf(out, "export %s=%s\n", kv[0], shellQuote(kv[1])); err != nil {
			return err
		}
	}
	return nil
}

func awsProofFromEnv(ctx context.Context, region, audience string) (*awsProof, error) {
	if audience == "" {
		audience = os.Getenv("CITADEL_WORKLOAD_AUDIENCE")
	}
	if audience == "" {
		return nil, errors.New("--audience (or CITADEL_WORKLOAD_AUDIENCE) is required with --aws")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(firstNonEmpty(region, os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"))))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	if cfg.Region == "" {
		return nil, errors.New("--aws-region (or AWS_REGION) is required with --aws")
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws credentials: %w", err)
	}
	return buildAWSProof(ctx, creds, cfg.Region, audience, time.Now().UTC())
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
