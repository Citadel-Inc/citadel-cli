package cmd

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ── domain types ─────────────────────────────────────────────────────────────

type insightsCounts struct {
	OpenIssues      int `json:"open_issues"`
	OpenMilestones  int `json:"open_milestones"`
	Branches        int `json:"branches"`
	Tags            int `json:"tags"`
	Contributors30d int `json:"contributors_30d"`
}

type insightsRelease struct {
	Name        string `json:"name"`
	SHA         string `json:"sha"`
	TaggedAt    string `json:"tagged_at"`
	IsAnnotated bool   `json:"is_annotated"`
	Annotation  string `json:"annotation"`
}

type insightsContributor struct {
	Email                string `json:"email"`
	Author               string `json:"author"`
	Count                int    `json:"count"`
	Slug                 string `json:"slug"`
	DisplayName          string `json:"display_name"`
	GravatarHash         string `json:"gravatar_hash"`
	GravatarDisabledBool bool   `json:"gravatar_disabled"`
}

type insightsLicense struct {
	SPDX string `json:"spdx"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type insightsResponse struct {
	Topics             []string              `json:"topics"`
	Counts             insightsCounts        `json:"counts"`
	StarCount          int                   `json:"star_count"`
	PinCount           int                   `json:"pin_count"`
	Releases           []insightsRelease     `json:"releases"`
	Activity           []int                 `json:"activity"`
	RecentContributors []insightsContributor `json:"recent_contributors"`
	Languages          map[string]int64      `json:"languages"`
	License            *insightsLicense      `json:"license"`
}

// ── command ───────────────────────────────────────────────────────────────────

var repoInsightsCmd = &cobra.Command{
	Use:   "insights [<namespace>/<repo>]",
	Short: "Show aggregate insights for a repository",
	Long: `Display a summary of repository health metrics including topics, issue counts,
languages, recent contributors, latest releases, commit activity, and license.`,
	Example: `  citadel-cli repo insights acme/myrepo
  citadel-cli repo insights acme/myrepo --output json`,
	RunE: runRepoInsights,
}

// ── handler ───────────────────────────────────────────────────────────────────

func runRepoInsights(cmd *cobra.Command, args []string) error {
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

	var resp insightsResponse
	if err := client.Get(cmd.Context(),
		fmt.Sprintf("/api/namespaces/%s/repos/%s/insights", ns, slug),
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

	renderInsights(cmd, resp)
	return nil
}

func renderInsights(cmd *cobra.Command, r insightsResponse) {
	out := cmd.OutOrStdout()

	// Topics
	if len(r.Topics) > 0 {
		_, _ = fmt.Fprintf(out, "Topics:  %s\n", strings.Join(r.Topics, ", "))
	}

	// Stars / pins
	_, _ = fmt.Fprintf(out, "Stars:   %d    Pins: %d\n", r.StarCount, r.PinCount)

	// Counts table
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Counts")
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "  Open issues\t%d\n", r.Counts.OpenIssues)
	_, _ = fmt.Fprintf(w, "  Open milestones\t%d\n", r.Counts.OpenMilestones)
	_, _ = fmt.Fprintf(w, "  Branches\t%d\n", r.Counts.Branches)
	_, _ = fmt.Fprintf(w, "  Tags\t%d\n", r.Counts.Tags)
	_, _ = fmt.Fprintf(w, "  Contributors (30d)\t%d\n", r.Counts.Contributors30d)
	_ = w.Flush()

	// License
	_, _ = fmt.Fprintln(out)
	if r.License != nil {
		_, _ = fmt.Fprintf(out, "License: %s\n", r.License.Name)
	} else {
		_, _ = fmt.Fprintln(out, "License: (none)")
	}

	// Languages (top 5 bar)
	if len(r.Languages) > 0 {
		_, _ = fmt.Fprintln(out)
		renderLanguages(cmd, r.Languages)
	}

	// Latest release
	if len(r.Releases) > 0 {
		rel := r.Releases[0]
		_, _ = fmt.Fprintln(out)
		rtype := "lightweight"
		if rel.IsAnnotated {
			rtype = "annotated"
		}
		_, _ = fmt.Fprintf(out, "Latest release: %s (%s, %s)\n", rel.Name, rtype, rel.TaggedAt[:10])
	}

	// Recent contributors
	if len(r.RecentContributors) > 0 {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Recent contributors (30d)")
		cw := newTabWriter(cmd)
		_, _ = fmt.Fprintln(cw, "  NAME\tCOMMITS")
		for _, c := range r.RecentContributors {
			name := c.DisplayName
			if name == "" {
				name = c.Author
			}
			_, _ = fmt.Fprintf(cw, "  %s\t%d\n", name, c.Count)
		}
		_ = cw.Flush()
	}

	// Activity sparkline (last 52 weeks, oldest → newest)
	if len(r.Activity) > 0 {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintf(out, "Activity (52w): %s\n", sparkline(r.Activity))
	}
}

// sparkline converts a slice of non-negative ints into a string of
// UTF-8 block characters (▁▂▃▄▅▆▇█).
func sparkline(vals []int) string {
	if len(vals) == 0 {
		return ""
	}
	bars := []rune("▁▂▃▄▅▆▇█")
	max := 0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		out := make([]rune, len(vals))
		for i := range out {
			out[i] = ' '
		}
		return string(out)
	}
	out := make([]rune, len(vals))
	for i, v := range vals {
		idx := int(float64(v) / float64(max) * float64(len(bars)-1))
		if v == 0 {
			out[i] = ' '
		} else {
			out[i] = bars[idx]
		}
	}
	return string(out)
}

func renderLanguages(cmd *cobra.Command, langs map[string]int64) {
	out := cmd.OutOrStdout()
	type langEntry struct {
		name  string
		bytes int64
	}
	entries := make([]langEntry, 0, len(langs))
	var total int64
	for name, b := range langs {
		entries = append(entries, langEntry{name, b})
		total += b
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].bytes > entries[j].bytes
	})
	if len(entries) > 5 {
		entries = entries[:5]
	}
	_, _ = fmt.Fprintln(out, "Languages")
	for _, e := range entries {
		pct := float64(e.bytes) / float64(total) * 100
		_, _ = fmt.Fprintf(out, "  %-20s %5.1f%%\n", e.name, pct)
	}
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	addOutputFlag(repoInsightsCmd)
	addRepoFlag(repoInsightsCmd)
}
