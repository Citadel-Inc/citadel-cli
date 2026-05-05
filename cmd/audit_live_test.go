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

// TestLiveAudit_agentCreateRoundTrip hits a real Citadel when both
// CITADEL_TEST_AUDIT_LIVE=1 and CITADEL_TEST_OAUTH_JWT are set. Creates a
// throwaway agent, then lists audit events for agent.created.
func TestLiveAudit_agentCreateRoundTrip(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_AUDIT_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_AUDIT_LIVE=1 for live audit integration")
	}
	token := liveOAuthJWT(t)
	base := liveOAuthBaseURL()
	hc := &http.Client{Timeout: 45 * time.Second}

	name := fmt.Sprintf("cli-audit-live-%d", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]any{"name": name})
	req, err := http.NewRequest(http.MethodPost, base+"/api/agents", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/agents: %d %s", resp.StatusCode, strings.TrimSpace(string(rb)))
	}

	q := base + "/api/audit/events?since=2m&kind=agent.created&limit=50"
	req2, err := http.NewRequest(http.MethodGet, q, nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := hc.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()
	rb2, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/audit/events: %d %s", resp2.StatusCode, strings.TrimSpace(string(rb2)))
	}
	var payload struct {
		Events []struct {
			Kind string `json:"kind"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rb2, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, e := range payload.Events {
		if e.Kind == "agent.created" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected agent.created in audit list, got %d events", len(payload.Events))
	}
}
