package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── domain types ─────────────────────────────────────────────────────────────

type topicsResponse struct {
	Topics []string `json:"topics"`
}

type popularTopic struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}

// ── command tree ─────────────────────────────────────────────────────────────

var repoTopicCmd = &cobra.Command{
	Use:   "topic",
	Short: "Manage repository topics",
}

var repoTopicListCmd = &cobra.Command{
	Use:   "list [<namespace>/<repo>]",
	Short: "List the topics attached to a repository",
	Example: `  citadel-cli repo topic list acme/myrepo
  citadel-cli repo topic list acme/myrepo --output json`,
	RunE: runRepoTopicList,
}

var repoTopicSetCmd = &cobra.Command{
	Use:   "set [<namespace>/<repo>] <topic>...",
	Short: "Replace the full topic set for a repository",
	Long: `Replace all topics on a repository with the provided list.
The operation is a full replace — existing topics not in the new list are removed.
Pass no topics to clear all topics from the repository.`,
	Example: `  # Set topics
  citadel-cli repo topic set acme/myrepo go cli devtools

  # Clear all topics
  citadel-cli repo topic set acme/myrepo

  # Output result as JSON
  citadel-cli repo topic set acme/myrepo go cli --output json`,
	RunE: runRepoTopicSet,
}

var repoTopicPopularCmd = &cobra.Command{
	Use:   "popular",
	Short: "List the most popular topics across all repositories",
	Example: `  citadel-cli repo topic popular
  citadel-cli repo topic popular --limit 20
  citadel-cli repo topic popular --output json`,
	RunE: runRepoTopicPopular,
}

// ── handlers ─────────────────────────────────────────────────────────────────

func runRepoTopicList(cmd *cobra.Command, args []string) error {
	posArg := ""
	if len(args) > 0 {
		posArg = args[0]
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, posArg)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	if err := validateGetOutput(output); err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var resp topicsResponse
	if err := client.Get(cmd.Context(),
		fmt.Sprintf("/api/namespaces/%s/repos/%s/topics", ns, slug),
		&resp,
	); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("repository not found: %s/%s", ns, slug)
		}
		if apiclient.IsStatus(err, http.StatusUnauthorized) {
			return fmt.Errorf("authentication required — run: citadel-cli auth login")
		}
		return err
	}

	if output != "" && output != "table" {
		return emitJSON(cmd, resp)
	}

	if len(resp.Topics) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(no topics)")
		return nil
	}
	for _, t := range resp.Topics {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), t)
	}
	return nil
}

func runRepoTopicSet(cmd *cobra.Command, args []string) error {
	ns, slug, topics, err := resolveTopicSetArgs(cmd, args)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	if err := validateGetOutput(output); err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	body := map[string][]string{"topics": topics}
	var resp topicsResponse
	if err := client.Put(cmd.Context(),
		fmt.Sprintf("/api/namespaces/%s/repos/%s/topics", ns, slug),
		body, &resp,
	); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("repository not found: %s/%s", ns, slug)
		}
		if apiclient.IsStatus(err, http.StatusUnauthorized) {
			return fmt.Errorf("authentication required — run: citadel-cli auth login")
		}
		if apiclient.IsStatus(err, http.StatusBadRequest) {
			return fmt.Errorf("invalid topic(s): check format (lowercase alphanumeric + hyphen, ≤50 chars each)")
		}
		return err
	}

	if output != "" && output != "table" {
		return emitJSON(cmd, resp)
	}

	if len(resp.Topics) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Topics cleared.")
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Topics set: %s\n", strings.Join(resp.Topics, ", "))
	return nil
}

// resolveTopicSetArgs extracts ns, slug, and the topic list from args.
// The first positional arg may be a repo slug (ns/repo). Remaining args
// are topic names.
func resolveTopicSetArgs(cmd *cobra.Command, args []string) (ns, slug string, topics []string, err error) {
	posArg := ""
	rest := args
	if len(args) > 0 && strings.Contains(args[0], "/") {
		posArg = args[0]
		rest = args[1:]
	}
	ns, slug, err = resolveRepoFromPosOrFlag(cmd, posArg)
	if err != nil {
		return
	}
	topics = rest
	if topics == nil {
		topics = []string{}
	}
	return
}

func runRepoTopicPopular(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	if err := validateGetOutput(output); err != nil {
		return err
	}
	limit, _ := cmd.Flags().GetInt("limit")

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	q := url.Values{}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	apiPath := "/api/topics/popular"
	if len(q) > 0 {
		apiPath += "?" + q.Encode()
	}

	// Popular returns a JSON array, not a wrapped object.
	var raw json.RawMessage
	if err := client.Get(cmd.Context(), apiPath, &raw); err != nil {
		if apiclient.IsStatus(err, http.StatusUnauthorized) {
			return fmt.Errorf("authentication required — run: citadel-cli auth login")
		}
		return err
	}

	if output != "" && output != "table" {
		_, err := cmd.OutOrStdout().Write(raw)
		return err
	}

	var rows []popularTopic
	if err := json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("unexpected response from server: %w", err)
	}

	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "(no topics found)")
		return nil
	}

	tw := newTabWriter(cmd)
	_, _ = fmt.Fprintln(tw, "TOPIC\tCOUNT")
	for _, r := range rows {
		_, _ = fmt.Fprintf(tw, "%s\t%d\n", r.Topic, r.Count)
	}
	return tw.Flush()
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	repoTopicCmd.AddCommand(repoTopicListCmd)
	repoTopicCmd.AddCommand(repoTopicSetCmd)
	repoTopicCmd.AddCommand(repoTopicPopularCmd)

	addOutputFlag(repoTopicListCmd, repoTopicSetCmd, repoTopicPopularCmd)
	addRepoFlag(repoTopicListCmd, repoTopicSetCmd)

	repoTopicPopularCmd.Flags().Int("limit", 0, "Maximum number of popular topics to return (default: 50)")
}
