package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// TestLiveOAuthLogin_fullBrowser_optIn drives `citadel-cli auth login` against a
// real Citadel deployment using a real Chromium browser via Playwright.
//
// Required env:
//   - CITADEL_TEST_OAUTH_FULL=1
//   - one of:
//   - CITADEL_TEST_OAUTH_STORAGE_STATE=/abs/path/to/playwright-storage-state.json
//   - CITADEL_TEST_OAUTH_REFRESH_TOKEN=<citadel refresh token for client_id=citadel-cli>
//
// Optional env:
//   - CITADEL_SERVER=https://mcp.src.land   (defaults to prod)
//
// The storage-state file must already contain a signed-in src.land session. The
// refresh-token path mints a fresh JWT via /api/oauth/token, bridges it into the
// OAuth cookie jar, and auto-approves the consent request through the live API.
func TestLiveOAuthLogin_fullBrowser_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_OAUTH_FULL")) != "1" {
		t.Skip("set CITADEL_TEST_OAUTH_FULL=1 for full live OAuth browser integration")
	}
	storageState := strings.TrimSpace(os.Getenv("CITADEL_TEST_OAUTH_STORAGE_STATE"))
	refreshToken := strings.TrimSpace(os.Getenv("CITADEL_TEST_OAUTH_REFRESH_TOKEN"))
	if storageState == "" && refreshToken == "" {
		t.Skip("set CITADEL_TEST_OAUTH_STORAGE_STATE or CITADEL_TEST_OAUTH_REFRESH_TOKEN for full live OAuth browser integration")
	}
	if storageState != "" {
		if _, err := os.Stat(storageState); err != nil {
			t.Skipf("storage-state file unavailable: %v", err)
		}
	}

	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("Playwright unavailable: %v (install browsers with `go run github.com/playwright-community/playwright-go/cmd/playwright install chromium`)", err)
	}
	t.Cleanup(func() { _ = pw.Stop() })

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("launch chromium: %v", err)
	}
	t.Cleanup(func() { _ = browser.Close() })

	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}

	origLaunch := launchBrowser
	t.Cleanup(func() { launchBrowser = origLaunch })
	launchBrowser = func(target string) {
		t.Helper()
		opts := playwright.BrowserNewContextOptions{}
		if storageState != "" {
			opts.StorageStatePath = playwright.String(storageState)
		}
		ctx, err := browser.NewContext(opts)
		if err != nil {
			t.Fatalf("new browser context: %v", err)
		}
		defer func() { _ = ctx.Close() }()

		page, err := ctx.NewPage()
		if err != nil {
			t.Fatalf("new page: %v", err)
		}
		if storageState != "" {
			if _, err := page.Goto(target, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateLoad,
				Timeout:   playwright.Float(60000),
			}); err != nil {
				t.Fatalf("goto auth url: %v", err)
			}
			_ = clickFirstVisible(page,
				`button:has-text("Authorize")`,
				`button:has-text("Allow")`,
				`button:has-text("Continue")`,
			)
			if err := waitForAuthSuccess(page); err != nil {
				t.Fatalf("wait for callback success page: %v (current url: %s)", err, page.URL())
			}
			return
		}

		jwt, err := mintLiveJWTFromRefresh(ctx.Request(), base, refreshToken)
		if err != nil {
			t.Fatalf("mint jwt from refresh token: %v", err)
		}
		if err := completeLiveOAuthWithJWT(ctx, page, base, target, jwt); err != nil {
			t.Fatalf("complete live oauth flow: %v", err)
		}
	}

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	root := NewRootCmd()
	root.SetArgs([]string{"auth", "login"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatalf("auth login: %v", err)
	}

	cfg, err := clicfg.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if strings.TrimSpace(cfg.AgentID) == "" || strings.TrimSpace(cfg.AgentName) == "" {
		t.Fatalf("expected bound agent metadata after login, got %+v", cfg)
	}
	if cfg.AccessToken == "" || strings.Count(cfg.AccessToken, ".") == 2 {
		t.Fatalf("expected opaque agent token after login, got %q", cfg.AccessToken)
	}

	// Production-smoke equivalent: credential survives the login command and the
	// next ordinary verb succeeds against the same server.
	root2 := NewRootCmd()
	root2.SetArgs([]string{"agent", "list"})
	root2.SetOut(io.Discard)
	root2.SetErr(io.Discard)
	root2.SilenceErrors = true
	root2.SilenceUsage = true
	if err := root2.Execute(); err != nil {
		t.Fatalf("agent list after login: %v", err)
	}
}

func clickFirstVisible(page playwright.Page, selectors ...string) error {
	for _, selector := range selectors {
		loc := page.Locator(selector)
		count, err := loc.Count()
		if err != nil || count == 0 {
			continue
		}
		if err := loc.First().Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(5000),
		}); err == nil {
			return nil
		}
	}
	return nil
}

func waitForAuthSuccess(page playwright.Page) error {
	return page.Locator(`text=Authentication successful!`).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(60000),
	})
}

func mintLiveJWTFromRefresh(req playwright.APIRequestContext, base, refreshToken string) (string, error) {
	res, err := req.Post(base+"/api/oauth/token", playwright.APIRequestContextPostOptions{
		Form: map[string]any{
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
			"client_id":     oauthClientID,
		},
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Dispose() }()
	if res.Status() != 200 {
		body, _ := res.Text()
		return "", fmt.Errorf("status %d: %s", res.Status(), strings.TrimSpace(body))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := res.JSON(&out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", fmt.Errorf("missing access_token in refresh response")
	}
	return out.AccessToken, nil
}

func completeLiveOAuthWithJWT(ctx playwright.BrowserContext, page playwright.Page, base, target, jwt string) error {
	req := ctx.Request()
	handoff, err := req.Post(base+"/api/oauth/_handoff", playwright.APIRequestContextPostOptions{
		Headers: map[string]string{"Content-Type": "application/json"},
		Data:    map[string]string{"access_token": jwt},
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		return err
	}
	defer func() { _ = handoff.Dispose() }()
	if handoff.Status() != 200 {
		body, _ := handoff.Text()
		return fmt.Errorf("handoff status %d: %s", handoff.Status(), strings.TrimSpace(body))
	}

	authz, err := req.Get(target, playwright.APIRequestContextGetOptions{
		MaxRedirects: playwright.Int(0),
		Timeout:      playwright.Float(60000),
	})
	if err != nil {
		return err
	}
	defer func() { _ = authz.Dispose() }()
	if authz.Status() != 302 {
		body, _ := authz.Text()
		return fmt.Errorf("authorize status %d: %s", authz.Status(), strings.TrimSpace(body))
	}
	loc := strings.TrimSpace(authz.Headers()["location"])
	if loc == "" {
		return fmt.Errorf("authorize redirect missing location")
	}
	u, err := url.Parse(loc)
	if err != nil {
		return err
	}
	reqID := strings.TrimSpace(u.Query().Get("citadel_req"))
	if reqID == "" {
		return fmt.Errorf("authorize redirect missing citadel_req")
	}

	ctxRes, err := req.Get(base+"/api/oauth/consent-context/"+reqID, playwright.APIRequestContextGetOptions{
		Headers: map[string]string{"Authorization": "Bearer " + jwt},
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		return err
	}
	defer func() { _ = ctxRes.Dispose() }()
	if ctxRes.Status() != 200 {
		body, _ := ctxRes.Text()
		return fmt.Errorf("consent context status %d: %s", ctxRes.Status(), strings.TrimSpace(body))
	}

	approve, err := req.Post(base+"/api/oauth/consent/approve", playwright.APIRequestContextPostOptions{
		Headers: map[string]string{"Authorization": "Bearer " + jwt, "Content-Type": "application/json"},
		Data:    map[string]string{"request_id": reqID},
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		return err
	}
	defer func() { _ = approve.Dispose() }()
	if approve.Status() != 200 {
		body, _ := approve.Text()
		return fmt.Errorf("consent approve status %d: %s", approve.Status(), strings.TrimSpace(body))
	}
	var out struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := approve.JSON(&out); err != nil {
		return err
	}
	if strings.TrimSpace(out.RedirectURL) == "" {
		return fmt.Errorf("consent approve missing redirect_url")
	}

	if _, err := page.Goto(out.RedirectURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(60000),
	}); err != nil {
		return err
	}
	return waitForAuthSuccess(page)
}
