package cmd

import (
	"cmp"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication (login, logout, status)",
	Long:  `Commands for managing authentication with the Citadel server.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth/PKCE flow (experimental)",
	Long: `Starts an OAuth/PKCE authentication flow with Supabase Auth, opens a
browser to the authorization endpoint, and stores the resulting access
and refresh tokens in ~/.config/citadel/config.toml (mode 0600).

EXPERIMENTAL: the OAuth client_id wiring is not yet productised. For
practical use today, prefer 'citadel-cli auth set-token' (paste a JWT
minted via the web app or an external SSO bridge) or set the
CITADEL_ACCESS_TOKEN environment variable.`,
	RunE: runLogin,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long: `Prints whether a session is active, the bound user UUID,
the access-token expiry, and the configured server URL.`,
	RunE: runStatus,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the local authentication session",
	Long:  `Removes the local config file, clearing the stored tokens and session state.`,
	RunE:  runLogout,
}

var setTokenCmd = &cobra.Command{
	Use:   "set-token",
	Short: "Persist a Supabase JWT directly (operator / CI path)",
	Long: `Reads a JWT from --token, the CITADEL_ACCESS_TOKEN env var, or stdin
(in that order), parses sub/exp claims, and writes the token + derived user
UUID + expiry to ~/.config/citadel/config.toml.

Use this when:
  - you already have a Supabase access token from elsewhere (web app dev tools,
    a CI mint job, an external SSO bridge), and
  - the interactive 'auth login' OAuth/PKCE flow is unavailable or undesired
    (headless host, container, runLogin's PKCE client_id is still being
    productised — see specs/HUMAN_BLOCKERS.md).

Examples:
  citadel-cli auth set-token --token "$JWT"
  echo "$JWT" | citadel-cli auth set-token
  CITADEL_ACCESS_TOKEN="$JWT" citadel-cli auth set-token`,
	RunE: runSetToken,
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)
	supabaseURL := resolveSupabaseURL()

	// Start a loopback HTTP server for the OAuth callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Generate PKCE challenge
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return fmt.Errorf("generate PKCE: %w", err)
	}

	authURL := buildAuthorizeURL(supabaseURL, redirectURI, challenge)

	fmt.Printf("Opening browser to authenticate...\n")
	openBrowser(authURL)

	// Wait for callback
	var code string
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			c := r.URL.Query().Get("code")
			if c == "" {
				errChan <- errors.New("missing code parameter")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("missing code"))
				return
			}
			codeChan <- c
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Authentication successful! You can close this window."))
		})

		// The outer defer closes the listener; that is what unblocks Serve.
		// No follow-up Shutdown() is needed — Serve already returned.
		server := &http.Server{Handler: mux}
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("serve: %w", err)
		}
	}()

	select {
	case code = <-codeChan:
		// Exchange code for tokens
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Minute):
		return errors.New("login timeout")
	}

	tokenResp, err := exchangePKCECode(supabaseURL, code, verifier)
	if err != nil {
		return err
	}

	claims, err := claimsFromJWT(tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("parse access token: %w", err)
	}
	userUUID := userUUIDFromClaims(claims)
	expiresAt := expiryFromClaims(claims, time.Now().Add(time.Hour))

	// Save config
	cfg.ServerURL = serverURL
	cfg.AccessToken = tokenResp.AccessToken
	cfg.RefreshToken = tokenResp.RefreshToken
	cfg.ExpiresAt = expiresAt
	cfg.UserUUID = userUUID

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Authentication successful! User UUID: %s\n", userUUID)
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.AccessToken == "" {
		fmt.Println("Not authenticated. Run 'citadel-cli auth login' to authenticate.")
		return nil
	}

	// Decode JWT to get expiry (signature verified server-side; client only inspects claims).
	claims, perr := claimsFromJWT(cfg.AccessToken)
	if perr != nil {
		fmt.Printf("Warning: could not parse access token: %v\n", perr)
	}
	exp := expiryFromClaims(claims, time.Time{})

	remaining := time.Until(exp)
	if remaining < 0 {
		fmt.Println("Session: EXPIRED")
		fmt.Printf("User UUID: %s\n", cfg.UserUUID)
		fmt.Printf("Server: %s\n", cfg.ServerURL)
		return nil
	}

	fmt.Println("Session: ACTIVE")
	fmt.Printf("User UUID: %s\n", cfg.UserUUID)
	fmt.Printf("Expires in: %v\n", remaining.Round(time.Second))
	fmt.Printf("Server: %s\n", cfg.ServerURL)
	return nil
}

func runSetToken(cmd *cobra.Command, _ []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tok, _ := cmd.Flags().GetString("token")
	tok = cmp.Or(tok, os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		// Fall back to stdin, trimmed.
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		tok = strings.TrimSpace(string(buf))
	}
	if tok == "" {
		return errors.New("no token provided: pass --token, set CITADEL_ACCESS_TOKEN, or pipe a JWT on stdin")
	}

	claims, err := claimsFromJWT(tok)
	if err != nil {
		return fmt.Errorf("parse token: %w", err)
	}
	cfg.ServerURL = cfg.ResolveServerURL(serverFlag(cmd))
	cfg.AccessToken = tok
	cfg.RefreshToken = ""
	cfg.UserUUID = userUUIDFromClaims(claims)
	cfg.ExpiresAt = expiryFromClaims(claims, time.Now().Add(time.Hour))

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("Stored token for user %s (expires %s).\n", cfg.UserUUID, cfg.ExpiresAt.Format(time.RFC3339))
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Truncate config, preserving only server URL
	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	cfg.ExpiresAt = time.Time{}
	cfg.UserUUID = ""

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("Logged out successfully.")
	return nil
}

// resolveSupabaseURL returns the SUPABASE_URL env override or the canonical
// Citadel Supabase project URL. Split out so tests can verify the env-vs-default
// precedence without spinning up runLogin.
func resolveSupabaseURL() string {
	return cmp.Or(os.Getenv("SUPABASE_URL"), "https://ucnlqqhgqhenzthzkdpi.supabase.co")
}

// buildAuthorizeURL constructs the GitHub-provider PKCE authorize URL
// served by Supabase Auth.
func buildAuthorizeURL(supabaseURL, redirectURI, challenge string) string {
	return fmt.Sprintf(
		"%s/auth/v1/authorize?provider=github&client_id=&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&response_type=code",
		supabaseURL,
		url.QueryEscape(redirectURI),
		challenge,
	)
}

// pkceTokenResponse is the JSON body returned by Supabase /auth/v1/token.
type pkceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// exchangePKCECode swaps an auth code + verifier for an access/refresh token
// pair. Split out so tests can drive the request with an httptest server.
func exchangePKCECode(supabaseURL, code, verifier string) (pkceTokenResponse, error) {
	tokenURL := supabaseURL + "/auth/v1/token"
	form := url.Values{
		"grant_type":    {"pkce"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {""},
	}
	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return pkceTokenResponse{}, fmt.Errorf("token exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return pkceTokenResponse{}, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var out pkceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return pkceTokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	return out, nil
}

// claimsFromJWT performs an unverified parse of the bearer JWT and returns
// its MapClaims. Signature verification is the server's job; the CLI only
// reads claims locally for UX (display expiry, user UUID).
func claimsFromJWT(tok string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(tok, claims); err != nil {
		return claims, err
	}
	return claims, nil
}

// userUUIDFromClaims returns the `sub` claim as a string, or "" if missing
// or malformed.
func userUUIDFromClaims(claims jwt.MapClaims) string {
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

// expiryFromClaims returns the `exp` claim as a time.Time, or fallback if
// the claim is missing or malformed.
func expiryFromClaims(claims jwt.MapClaims, fallback time.Time) time.Time {
	if expF, ok := claims["exp"].(float64); ok {
		return time.Unix(int64(expF), 0)
	}
	return fallback
}

// generatePKCE generates a PKCE verifier and challenge.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)

	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

// openBrowser opens the URL in the default browser.
func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", u)
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		fmt.Printf("Please open this URL in your browser: %s\n", u)
		return
	}
	_ = cmd.Start()
}

func init() {
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(statusCmd)
	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(setTokenCmd)
	setTokenCmd.Flags().String("token", "", "JWT to persist (overrides CITADEL_ACCESS_TOKEN and stdin)")
}
