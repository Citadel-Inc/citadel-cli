package cmd

import "testing"

func TestCompleteWebhookEvents_Prefix(t *testing.T) {
	results, dir := completeWebhookEvents(nil, nil, "issue")
	if dir == 0 {
		t.Fatal("expected non-zero directive")
	}
	for _, r := range results {
		if len(r) < 5 || r[:5] != "issue" {
			t.Errorf("result %q does not start with 'issue'", r)
		}
	}
	if len(results) == 0 {
		t.Fatal("expected at least one completion for 'issue' prefix")
	}
}

func TestCompleteWebhookEvents_Empty(t *testing.T) {
	results, _ := completeWebhookEvents(nil, nil, "")
	if len(results) != len(webhookEventKinds) {
		t.Fatalf("empty prefix: got %d results, want %d", len(results), len(webhookEventKinds))
	}
}

func TestCompleteWebhookEvents_NoMatch(t *testing.T) {
	results, _ := completeWebhookEvents(nil, nil, "zzz")
	if len(results) != 0 {
		t.Fatalf("no-match prefix: got %d results, want 0", len(results))
	}
}
