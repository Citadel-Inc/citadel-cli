package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func liveOAuthBaseURL() string {
	u := strings.TrimSpace(os.Getenv("CITADEL_SERVER"))
	u = strings.TrimSuffix(u, "/")
	if u == "" {
		return "https://mcp.src.land"
	}
	return u
}

func liveOAuthJWT(t *testing.T) string {
	t.Helper()
	tok := strings.TrimSpace(os.Getenv("CITADEL_TEST_OAUTH_JWT"))
	if tok == "" {
		t.Skip("CITADEL_TEST_OAUTH_JWT unset — live OAuth client integration is opt-in (specs/HUMAN_BLOCKERS.md §71)")
	}
	return tok
}

// TestLiveOAuthClients_create_list_show_rotate_revoke exercises the same HTTP
// surface as citadel-cli oauth clients against a real server when
// CITADEL_TEST_OAUTH_JWT is set (optional CITADEL_SERVER). CI skips.
//
// rotate-secret accepts 412 mfa_required (aal1 token) or 200 when the token
// is already step-up satisfied.
func TestLiveOAuthClients_create_list_show_rotate_revoke(t *testing.T) {
	token := liveOAuthJWT(t)
	base := liveOAuthBaseURL()
	hc := &http.Client{Timeout: 45 * time.Second}

	name := fmt.Sprintf("cli-oauth-live-%d", time.Now().UnixNano())
	payload := map[string]any{
		"name":          name,
		"redirect_uris": []string{"https://localhost/cli-oauth-live/callback"},
		"is_public":     false,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, base+"/api/oauth/clients", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		t.Skipf("403 — JWT needs oauth:manage on target namespace: %s", strings.TrimSpace(string(rb)))
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/oauth/clients: %d %s", resp.StatusCode, strings.TrimSpace(string(rb)))
	}

	var created struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(rb, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.ID == "" || created.ClientSecret == "" {
		t.Fatalf("expected id + client_secret, got %+v", created)
	}

	t.Cleanup(func() {
		reqDel, _ := http.NewRequest(http.MethodDelete, base+"/api/oauth/clients/"+created.ID, nil)
		reqDel.Header.Set("Authorization", "Bearer "+token)
		respDel, err := hc.Do(reqDel)
		if err == nil && respDel != nil {
			_ = respDel.Body.Close()
		}
	})

	// List (personal scope)
	reqList, _ := http.NewRequest(http.MethodGet, base+"/api/oauth/clients", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	respList, err := hc.Do(reqList)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = respList.Body.Close() }()
	listBody, _ := io.ReadAll(respList.Body)
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/oauth/clients: %d %s", respList.StatusCode, strings.TrimSpace(string(listBody)))
	}
	if !bytes.Contains(listBody, []byte(created.ClientID)) {
		t.Fatalf("list response missing client_id %q", created.ClientID)
	}

	// Show
	reqGet, _ := http.NewRequest(http.MethodGet, base+"/api/oauth/clients/"+created.ID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)
	respGet, err := hc.Do(reqGet)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = respGet.Body.Close() }()
	getBody, _ := io.ReadAll(respGet.Body)
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/oauth/clients/{id}: %d %s", respGet.StatusCode, strings.TrimSpace(string(getBody)))
	}
	var row map[string]any
	if err := json.Unmarshal(getBody, &row); err != nil {
		t.Fatalf("decode show: %v", err)
	}
	if row["client_id"] != created.ClientID {
		t.Fatalf("show client_id mismatch: %v vs %q", row["client_id"], created.ClientID)
	}

	// rotate-secret — 412 (mfa_required) or 200 (recent aal2 / marker)
	reqRot, _ := http.NewRequest(http.MethodPost, base+"/api/oauth/clients/"+created.ID+"/rotate-secret", nil)
	reqRot.Header.Set("Authorization", "Bearer "+token)
	respRot, err := hc.Do(reqRot)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = respRot.Body.Close() }()
	rotBody, _ := io.ReadAll(respRot.Body)
	switch respRot.StatusCode {
	case http.StatusPreconditionRequired:
		if !strings.Contains(string(rotBody), "mfa_required") {
			t.Fatalf("412 without mfa_required: %s", strings.TrimSpace(string(rotBody)))
		}
	case http.StatusOK:
		var rot map[string]any
		if err := json.Unmarshal(rotBody, &rot); err != nil {
			t.Fatalf("decode rotate: %v", err)
		}
		sec, _ := rot["client_secret"].(string)
		if sec == "" {
			t.Fatal("rotate 200 but no client_secret in body")
		}
	default:
		t.Fatalf("POST rotate-secret: %d %s", respRot.StatusCode, strings.TrimSpace(string(rotBody)))
	}

	// Revoke
	reqDel, _ := http.NewRequest(http.MethodDelete, base+"/api/oauth/clients/"+created.ID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	respDel, err := hc.Do(reqDel)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = respDel.Body.Close() }()
	if respDel.StatusCode != http.StatusNoContent {
		delBody, _ := io.ReadAll(respDel.Body)
		t.Fatalf("DELETE /api/oauth/clients/{id}: %d %s", respDel.StatusCode, strings.TrimSpace(string(delBody)))
	}
}
