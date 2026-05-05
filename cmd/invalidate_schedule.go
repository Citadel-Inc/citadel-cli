package cmd

import (
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
	"github.com/Rethunk-Tech/citadel-cli/internal/completion"
)

// scheduleCompletionInvalidate drops cached completion entries for the
// resolved server (best-effort, never blocks).
func scheduleCompletionInvalidate(serverFlag string, keys ...string) {
	cfg, err := clicfg.Load()
	if err != nil {
		return
	}
	srv := strings.TrimRight(cfg.ResolveServerURL(serverFlag), "/")
	completion.RemoveAsync(srv, keys...)
}
