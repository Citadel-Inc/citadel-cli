package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// newAPIClient loads the on-disk config (with env overrides) and returns an
// apiclient.Client honoring the persistent --server flag override.
func newAPIClient(cmd *cobra.Command) (*apiclient.Client, error) {
	cfg, err := clicfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return apiclient.New(cfg, serverFlag(cmd))
}
