package cmd

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
)

const humanOnlyBody = `{"error":"forbidden","reason":"human_only"}`

func humanOnly403() *apiclient.HTTPError {
	return &apiclient.HTTPError{StatusCode: http.StatusForbidden, Body: humanOnlyBody}
}

// A scope cannot open a human-only route, so the message must not send anyone
// looking for one, and the commands that word their own 403 must not hide it.
func TestHumanOnlyRefusalSaysSo(t *testing.T) {
	for name, err := range map[string]error{
		"generic":       humanOnly403(),
		"agent create":  decorateAgentCreateError(humanOnly403(), "a"),
		"deploy tokens": decorateDeployTokenError(humanOnly403(), "acme/r", "create"),
		"webhooks":      decorateWebhookError(humanOnly403(), "acme/r", "create"),
	} {
		var cli *CLIError
		if !errors.As(FriendlyError(err), &cli) || !strings.Contains(cli.Message, "web app") || strings.Contains(cli.Message, "scope —") {
			t.Errorf("%s: %v", name, FriendlyError(err))
		}
	}
	plain := httpErrorToCLI(&apiclient.HTTPError{StatusCode: http.StatusForbidden, Body: `{"error":"forbidden","required_scope":"admin"}`})
	if strings.Contains(plain.Message, "web app") {
		t.Errorf("a missing scope is not human-only: %s", plain.Message)
	}
}
