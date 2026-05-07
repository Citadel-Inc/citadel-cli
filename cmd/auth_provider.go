package cmd

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

var authProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "List and manage OAuth providers",
	Long: `Inspect enabled OAuth providers and manage linked provider identities.

Examples:
  citadel-cli auth provider list
  citadel-cli auth provider link github
  citadel-cli auth provider unlink github --yes`,
}

var authProviderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List enabled OAuth providers",
	RunE:  runAuthProviderList,
}

var authProviderLinkCmd = &cobra.Command{
	Use:               "link <provider>",
	Short:             "Start linking a provider to your account",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAuthProviderIDs,
	RunE:              runAuthProviderLink,
}

var authProviderUnlinkCmd = &cobra.Command{
	Use:               "unlink <provider>",
	Short:             "Unlink a provider from your account",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAuthProviderIDs,
	RunE:              runAuthProviderUnlink,
}

type authProviderRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (authProviderRow) CSVHeader() []string {
	return []string{"id", "label"}
}

func (r authProviderRow) CSVRecord() []string {
	return []string{r.ID, r.Label}
}

type authProviderLinkResponse struct {
	Provider    string `json:"provider"`
	RedirectURL string `json:"redirect_url"`
}

func normalizeAuthProviderID(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	if provider == "" {
		return "", fmt.Errorf("provider required")
	}
	return provider, nil
}

func fetchAuthProviders(ctx context.Context, cmd *cobra.Command) ([]authProviderRow, error) {
	var payload struct {
		Providers []authProviderRow `json:"providers"`
	}
	if err := doPublicJSON(cmd, http.MethodGet, "/auth/providers", nil, &payload); err != nil {
		return nil, err
	}
	if payload.Providers == nil {
		return []authProviderRow{}, nil
	}
	return payload.Providers, nil
}

func completeAuthProviderIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	rows, err := fetchAuthProviders(cmd.Context(), cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if id := strings.TrimSpace(row.ID); id != "" {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	return out, cobra.ShellCompDirectiveNoFileComp
}

func runAuthProviderList(cmd *cobra.Command, _ []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	rows, err := fetchAuthProviders(cmd.Context(), cmd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		switch output {
		case "json":
			return emitJSON(cmd, []authProviderRow{})
		case "yaml":
			return emitYAML(cmd, []authProviderRow{})
		case "ndjson":
			return nil
		case "csv":
			return emitCSVHeaderOnly[authProviderRow](cmd)
		default:
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No auth providers configured.")
			return nil
		}
	}
	switch output {
	case "json":
		return emitJSON(cmd, rows)
	case "yaml":
		return emitYAML(cmd, rows)
	case "ndjson":
		return emitNDJSONLines(cmd, rows)
	case "csv":
		header := false
		return emitCSVRows(cmd, &header, rows)
	default:
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tLABEL")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", row.ID, row.Label)
		}
		return w.Flush()
	}
}

func runAuthProviderLink(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	provider, err := normalizeAuthProviderID(args[0])
	if err != nil {
		return err
	}
	var resp authProviderLinkResponse
	if err := c.Post(cmd.Context(), "/auth/link-provider", map[string]string{"provider": provider}, &resp); err != nil {
		return err
	}
	if jsonFlag(cmd) {
		return emitJSON(cmd, resp)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Starting browser link flow for provider %s.\n", resp.Provider)
	launchBrowser(resp.RedirectURL)
	return nil
}

func runAuthProviderUnlink(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	provider, err := normalizeAuthProviderID(args[0])
	if err != nil {
		return err
	}
	if err := confirmTypedValue(yesFlag(cmd), "unlink provider", provider); err != nil {
		return err
	}
	var resp map[string]any
	if err := c.Post(cmd.Context(), "/auth/unlink-provider", map[string]string{"provider": provider}, &resp); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return err
		}
		return err
	}
	if jsonFlag(cmd) {
		if resp == nil {
			resp = map[string]any{}
		}
		if _, ok := resp["provider"]; !ok {
			resp["provider"] = provider
		}
		return emitJSON(cmd, resp)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Unlinked provider %s.\n", provider)
	return nil
}

func init() {
	authProviderCmd.AddCommand(authProviderListCmd)
	authProviderCmd.AddCommand(authProviderLinkCmd)
	authProviderCmd.AddCommand(authProviderUnlinkCmd)

	addOutputFlag(authProviderListCmd)
	addJSONFlag(authProviderLinkCmd, authProviderUnlinkCmd)
	addYesFlag(authProviderUnlinkCmd)
}
