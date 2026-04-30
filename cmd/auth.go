package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/internal/clicfg"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication (login, logout, status)",
	Long:  `Commands for managing authentication with the Citadel server.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the server via OAuth/PKCE flow",
	Long: `Starts an OAuth/PKCE authentication flow with Supabase Auth.
Opens a browser to the authorization endpoint and stores the resulting
access and refresh tokens in ~/.config/citadel/config.toml (mode 0600).`,
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
	RunE: runLogout,
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := clicfg.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	flagServer, _ := cmd.Flags().GetString("server")
	serverURL := cfg.ResolveServerURL(flagServer)

	// Extract Supabase URL from server URL (assumed to be https://api.src.land or similar)
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		// Try to infer from Citadel server (this is a simplification;
		// production would need a better discovery mechanism).
		supabaseURL = "https://ucnlqqhgqhenzthzkdpi.supabase.co"
	}

	// Start a loopback HTTP server for the OAuth callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Generate PKCE challenge
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return fmt.Errorf("generate PKCE: %w", err)
	}

	// OAuth authorize URL
	authURL := fmt.Sprintf(
		"%s/auth/v1/authorize?provider=github&client_id=&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&response_type=code",
		supabaseURL,
		url.QueryEscape(redirectURI),
		challenge,
	)

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
				errChan <- fmt.Errorf("missing code parameter")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("missing code"))
				return
			}
			codeChan <- c
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Authentication successful! You can close this window."))
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		server := &http.Server{Handler: mux}
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("serve: %w", err)
		}
		server.Shutdown(ctx)
	}()

	select {
	case code = <-codeChan:
		// Exchange code for tokens
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("login timeout")
	}

	// Exchange auth code for tokens
	tokenURL := fmt.Sprintf("%s/auth/v1/token", supabaseURL)
	tokenReq := url.Values{
		"grant_type":    {"pkce"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {""}, // Will be obtained from Supabase or skipped
	}

	resp, err := http.PostForm(tokenURL, tokenReq)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	// Extract user UUID from access token (unverified parse)
	var claims jwt.MapClaims
	jwt.ParseWithClaims(tokenResp.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return nil, nil // Don't verify signature here
	})

	userUUID := ""
	if sub, ok := claims["sub"]; ok {
		userUUID = sub.(string)
	}

	// Compute expiry from exp claim
	expiresAt := time.Now().Add(1 * time.Hour) // Default to 1 hour
	if exp, ok := claims["exp"]; ok {
		if expF, ok := exp.(float64); ok {
			expiresAt = time.Unix(int64(expF), 0)
		}
	}

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
		fmt.Println("Not authenticated. Run 'citadel auth login' to authenticate.")
		return nil
	}

	// Decode JWT to get expiry (no signature verify needed)
	var claims jwt.MapClaims
	jwt.ParseWithClaims(cfg.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return nil, nil
	})

	var exp time.Time
	if expFloat, ok := claims["exp"]; ok {
		if ef, ok := expFloat.(float64); ok {
			exp = time.Unix(int64(ef), 0)
		}
	}

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
}
