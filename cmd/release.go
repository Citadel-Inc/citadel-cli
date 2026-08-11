package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// ReleaseCmd is the root verb for per-repo release management.
var ReleaseCmd = &cobra.Command{
	Use:     "release",
	GroupID: "repo",
	Short:   "Manage per-repo releases (tag, name, body, draft/prerelease flags)",
	Long: `Release verbs target the repo under a Citadel namespace path via -R <ns/repo>.

Examples:
  citadel-cli release list -R acme/demo
  citadel-cli release latest -R acme/demo
  citadel-cli release view -R acme/demo v1.0.0
  citadel-cli release create -R acme/demo --tag v1.0.0 --name "v1.0.0" --body "Initial GA"
  citadel-cli release edit -R acme/demo v1.0.0 --prerelease=false
  citadel-cli release delete -R acme/demo v1.0.0
  citadel-cli release asset list -R acme/demo v1.0.0`,
}

var releaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List releases for a repo",
	RunE:  runReleaseList,
}

var releaseLatestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Show the most recent non-draft release",
	RunE:  runReleaseLatest,
}

var releaseViewCmd = &cobra.Command{
	Use:   "view <tag>",
	Short: "Show one release by tag name",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseView,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a release for an existing git tag",
	RunE:  runReleaseCreate,
}

var releaseEditCmd = &cobra.Command{
	Use:   "edit <tag>",
	Short: "Edit a release (name, body, draft, prerelease)",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseEdit,
}

var releaseDeleteCmd = &cobra.Command{
	Use:   "delete <tag>",
	Short: "Delete a release (the underlying git tag stays)",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseDelete,
}

var releaseAssetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage release assets",
}

var releaseAssetListCmd = &cobra.Command{
	Use:   "list <tag>",
	Short: "List assets attached to a release",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseAssetList,
}

var releaseAssetUploadCmd = &cobra.Command{
	Use:   "upload <tag> <file>",
	Short: "Upload a file as a release asset",
	Args:  cobra.ExactArgs(2),
	RunE:  runReleaseAssetUpload,
}

var releaseAssetDownloadCmd = &cobra.Command{
	Use:   "download <tag> <asset_id>",
	Short: "Download a release asset",
	Args:  cobra.ExactArgs(2),
	RunE:  runReleaseAssetDownload,
}

var releaseAssetDeleteCmd = &cobra.Command{
	Use:               "delete <tag> <asset_id>",
	Short:             "Delete a release asset",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeReleaseAssetIDs,
	RunE:              runReleaseAssetDelete,
}

type releaseRow struct {
	ID           string     `json:"id"`
	NamespaceID  string     `json:"namespace_id"`
	RepoID       string     `json:"repo_id"`
	TagName      string     `json:"tag_name"`
	Name         string     `json:"name"`
	BodyMarkdown string     `json:"body_markdown"`
	Draft        bool       `json:"draft"`
	Prerelease   bool       `json:"prerelease"`
	AuthorID     string     `json:"author_id"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type releaseAssetRow struct {
	ID          string    `json:"id"`
	ReleaseID   string    `json:"release_id"`
	RepoID      string    `json:"repo_id"`
	UploaderID  string    `json:"uploader_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type releaseCreateReq struct {
	TagName      string `json:"tag_name"`
	Name         string `json:"name"`
	BodyMarkdown string `json:"body_markdown,omitempty"`
	Draft        bool   `json:"draft,omitempty"`
	Prerelease   bool   `json:"prerelease,omitempty"`
}

type releaseUpdateReq struct {
	Name         string `json:"name,omitempty"`
	BodyMarkdown string `json:"body_markdown,omitempty"`
	Draft        *bool  `json:"draft,omitempty"`
	Prerelease   *bool  `json:"prerelease,omitempty"`
}

func releaseBasePath(nsPath string) string {
	return "/namespaces/" + url.PathEscape(nsPath) + "/releases"
}

func releaseAssetPath(nsPath, tag string) string {
	return releaseBasePath(nsPath) + "/" + url.PathEscape(tag) + "/assets"
}

func runReleaseList(cmd *cobra.Command, _ []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	q := url.Values{}
	if includeDrafts, _ := cmd.Flags().GetBool("include-drafts"); includeDrafts {
		q.Set("include_drafts", "true")
	}
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if cursor, _ := cmd.Flags().GetString("cursor"); strings.TrimSpace(cursor) != "" {
		q.Set("cursor", strings.TrimSpace(cursor))
	}
	path := releaseBasePath(nsPath)
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var payload struct {
		Releases []releaseRow `json:"releases"`
		Cursor   string       `json:"cursor"`
	}
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return err
	}
	if payload.Releases == nil {
		payload.Releases = []releaseRow{}
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	case "ndjson":
		return emitNDJSONLines(cmd, payload.Releases)
	default:
		if len(payload.Releases) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No releases for %s.\n", nsPath)
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "TAG\tNAME\tSTATE\tPUBLISHED\tCREATED")
		for _, r := range payload.Releases {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				r.TagName,
				r.Name,
				releaseState(r),
				optionalTime(r.PublishedAt),
				formatRFC3339UTC(r.CreatedAt),
			)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		if strings.TrimSpace(payload.Cursor) != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(use --cursor %s for more)\n", payload.Cursor)
		}
		return nil
	}
}

func runReleaseLatest(cmd *cobra.Command, _ []string) error {
	return getReleaseAtPath(cmd, "/latest", "")
}

func runReleaseView(cmd *cobra.Command, args []string) error {
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	return getReleaseAtPath(cmd, "/"+url.PathEscape(tag), tag)
}

func getReleaseAtPath(cmd *cobra.Command, suffix, tagForErr string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var row releaseRow
	if err := c.Get(cmd.Context(), releaseBasePath(nsPath)+suffix, &row); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			if tagForErr != "" {
				return fmt.Errorf("release %s not found in %s", tagForErr, nsPath)
			}
			return fmt.Errorf("no published releases for %s", nsPath)
		}
		return err
	}
	return renderReleaseDetail(cmd, nsPath, row, output)
}

func runReleaseCreate(cmd *cobra.Command, _ []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	tag, _ := cmd.Flags().GetString("tag")
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return fmt.Errorf("--tag required")
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	if name == "" {
		name = tag
	}
	body, _ := cmd.Flags().GetString("body")
	draft, _ := cmd.Flags().GetBool("draft")
	prerelease, _ := cmd.Flags().GetBool("prerelease")

	req := releaseCreateReq{
		TagName:      tag,
		Name:         name,
		BodyMarkdown: body,
		Draft:        draft,
		Prerelease:   prerelease,
	}
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would POST %s tag=%s (skipped; --dry-run)\n", releaseBasePath(nsPath), tag)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var created releaseRow
	if err := c.Post(cmd.Context(), releaseBasePath(nsPath), req, &created); err != nil {
		if apiclient.IsStatus(err, http.StatusConflict) {
			return fmt.Errorf("release %s already exists in %s", tag, nsPath)
		}
		if apiclient.IsStatus(err, http.StatusUnprocessableEntity) {
			return fmt.Errorf("tag %s does not exist on the remote — push it first", tag)
		}
		return err
	}
	return renderReleaseDetail(cmd, nsPath, created, output)
}

func runReleaseEdit(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag required")
	}

	req := releaseUpdateReq{}
	hasUpdate := false
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		req.Name = name
		hasUpdate = true
	}
	if cmd.Flags().Changed("body") {
		body, _ := cmd.Flags().GetString("body")
		req.BodyMarkdown = body
		hasUpdate = true
	}
	if cmd.Flags().Changed("draft") {
		v, _ := cmd.Flags().GetBool("draft")
		req.Draft = &v
		hasUpdate = true
	}
	if cmd.Flags().Changed("prerelease") {
		v, _ := cmd.Flags().GetBool("prerelease")
		req.Prerelease = &v
		hasUpdate = true
	}
	if !hasUpdate {
		return fmt.Errorf("nothing to update: pass --name, --body, --draft, or --prerelease")
	}

	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	path := releaseBasePath(nsPath) + "/" + url.PathEscape(tag)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would PATCH %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var updated releaseRow
	if err := c.Patch(cmd.Context(), path, req, &updated); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("release %s not found in %s", tag, nsPath)
		}
		return err
	}
	return renderReleaseDetail(cmd, nsPath, updated, output)
}

func runReleaseDelete(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateMutationOutput(output, "delete"); err != nil {
		return err
	}
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	path := releaseBasePath(nsPath) + "/" + url.PathEscape(tag)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if !yesFlag(cmd) {
		if err := confirmTypedValue(false, "delete release", tag); err != nil {
			return err
		}
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("release %s not found in %s", tag, nsPath)
		}
		return err
	}
	if strings.EqualFold(output, "json") {
		return emitJSON(cmd, map[string]any{
			"status":         "ok",
			"namespace_path": nsPath,
			"tag":            tag,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted release %s from %s.\n", tag, nsPath)
	return nil
}

func runReleaseAssetList(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag required")
	}

	var payload struct {
		Assets []releaseAssetRow `json:"assets"`
	}
	if err := c.Get(cmd.Context(), releaseAssetPath(nsPath, tag), &payload); err != nil {
		return err
	}
	if payload.Assets == nil {
		payload.Assets = []releaseAssetRow{}
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	case "ndjson":
		return emitNDJSONLines(cmd, payload.Assets)
	default:
		if len(payload.Assets) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No assets for %s in %s.\n", tag, nsPath)
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ASSET ID\tFILENAME\tTYPE\tSIZE\tCREATED")
		for _, asset := range payload.Assets {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				asset.ID,
				asset.Filename,
				asset.ContentType,
				asset.SizeBytes,
				formatRFC3339UTC(asset.CreatedAt),
			)
		}
		return w.Flush()
	}
}

func runReleaseAssetUpload(cmd *cobra.Command, args []string) error {
	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	filePath := strings.TrimSpace(args[1])
	if filePath == "" {
		return fmt.Errorf("file required")
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	path := releaseAssetPath(nsPath, tag)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would POST %s file=%s (skipped; --dry-run)\n", path, filePath)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open asset file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var asset releaseAssetRow
	if err := c.PostMultipart(cmd.Context(), path, "file", filepath.Base(filePath), file, &asset); err != nil {
		return releaseAssetUploadError(err)
	}
	return renderReleaseAssetDetail(cmd, nsPath, tag, asset, output, "Uploaded")
}

func runReleaseAssetDownload(cmd *cobra.Command, args []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	tag := strings.TrimSpace(args[0])
	assetID := strings.TrimSpace(args[1])
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	if assetID == "" {
		return fmt.Errorf("asset ID required")
	}
	outputFile, _ := cmd.Flags().GetString("output-file")
	var asset releaseAssetRow
	if err := c.Get(cmd.Context(), releaseAssetPath(nsPath, tag)+"/"+url.PathEscape(assetID), &asset); err != nil {
		return err
	}
	downloadURL := strings.TrimSpace(asset.DownloadURL)
	if downloadURL == "" {
		return fmt.Errorf("release asset %s has no download URL; the object store may be unconfigured", assetID)
	}
	resp, err := c.GetStream(cmd.Context(), downloadURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var dst io.Writer = cmd.OutOrStdout()
	var file *os.File
	if outputFile != "" && outputFile != "-" {
		file, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		dst = file
	}

	if file == nil && downloadOutputIsTTY(dst) {
		prefix := make([]byte, 512)
		n, readErr := io.ReadFull(resp.Body, prefix)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read release asset: %w", readErr)
		}
		if downloadLooksBinary(resp.Header.Get("Content-Type"), prefix[:n]) {
			return fmt.Errorf("refusing to write binary release asset to a terminal; redirect stdout or pass --output-file")
		}
		_, err = io.Copy(dst, io.MultiReader(bytes.NewReader(prefix[:n]), resp.Body))
	} else {
		_, err = io.Copy(dst, resp.Body)
	}
	if err != nil {
		return fmt.Errorf("write release asset: %w", err)
	}
	if file != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Downloaded %s to %s.\n", assetID, outputFile)
	}
	return nil
}

func runReleaseAssetDelete(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateMutationOutput(output, "delete"); err != nil {
		return err
	}
	tag := strings.TrimSpace(args[0])
	assetID := strings.TrimSpace(args[1])
	if tag == "" {
		return fmt.Errorf("tag required")
	}
	if assetID == "" {
		return fmt.Errorf("asset ID required")
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return err
	}
	path := releaseAssetPath(nsPath, tag) + "/" + url.PathEscape(assetID)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if err := confirmTypedValue(yesFlag(cmd), "delete release asset", assetID); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		return err
	}
	if output == "json" {
		return emitJSON(cmd, map[string]any{
			"status":         "ok",
			"namespace_path": nsPath,
			"tag":            tag,
			"asset_id":       assetID,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted release asset %s from %s.\n", assetID, nsPath)
	return nil
}

func completeReleaseAssetIDs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	nsPath, err := resolveIssueNamespacePath(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var payload struct {
		Assets []releaseAssetRow `json:"assets"`
	}
	if err := c.Get(cmd.Context(), releaseAssetPath(nsPath, args[0]), &payload); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(payload.Assets))
	for _, asset := range payload.Assets {
		out = append(out, asset.ID+"\t"+asset.Filename)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func releaseAssetUploadError(err error) error {
	if apiclient.IsStatus(err, http.StatusRequestEntityTooLarge) {
		return fmt.Errorf("release asset exceeds the server size limit")
	}
	if apiclient.IsStatus(err, http.StatusServiceUnavailable) {
		var httpErr *apiclient.HTTPError
		if errors.As(err, &httpErr) && strings.Contains(httpErr.Body, "asset_store_unavailable") {
			return fmt.Errorf("release asset storage is unavailable; configure the object store before uploading")
		}
	}
	return err
}

func renderReleaseAssetDetail(cmd *cobra.Command, nsPath, tag string, asset releaseAssetRow, output, action string) error {
	switch output {
	case "json":
		return emitJSON(cmd, asset)
	case "yaml":
		return emitYAML(cmd, asset)
	default:
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintf(w, "ASSET ID\t%s\n", asset.ID)
		_, _ = fmt.Fprintf(w, "NAMESPACE\t%s\n", nsPath)
		_, _ = fmt.Fprintf(w, "TAG\t%s\n", tag)
		_, _ = fmt.Fprintf(w, "FILENAME\t%s\n", asset.Filename)
		_, _ = fmt.Fprintf(w, "TYPE\t%s\n", asset.ContentType)
		_, _ = fmt.Fprintf(w, "SIZE\t%d\n", asset.SizeBytes)
		_, _ = fmt.Fprintf(w, "ACTION\t%s\n", action)
		return w.Flush()
	}
}

func releaseState(r releaseRow) string {
	switch {
	case r.Draft:
		return "draft"
	case r.Prerelease:
		return "prerelease"
	default:
		return "published"
	}
}

func renderReleaseDetail(cmd *cobra.Command, nsPath string, row releaseRow, output string) error {
	switch output {
	case "json":
		return emitJSON(cmd, row)
	case "yaml":
		return emitYAML(cmd, row)
	default:
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintf(w, "TAG\t%s\n", row.TagName)
		_, _ = fmt.Fprintf(w, "NAMESPACE\t%s\n", nsPath)
		_, _ = fmt.Fprintf(w, "NAME\t%s\n", row.Name)
		_, _ = fmt.Fprintf(w, "STATE\t%s\n", releaseState(row))
		_, _ = fmt.Fprintf(w, "PUBLISHED\t%s\n", optionalTime(row.PublishedAt))
		_, _ = fmt.Fprintf(w, "CREATED\t%s\n", formatRFC3339UTC(row.CreatedAt))
		_, _ = fmt.Fprintf(w, "UPDATED\t%s\n", formatRFC3339UTC(row.UpdatedAt))
		if err := w.Flush(); err != nil {
			return err
		}
		if body := strings.TrimSpace(row.BodyMarkdown); body != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), body)
		}
		return nil
	}
}

func init() {
	ReleaseCmd.AddCommand(
		releaseListCmd,
		releaseLatestCmd,
		releaseViewCmd,
		releaseCreateCmd,
		releaseEditCmd,
		releaseDeleteCmd,
		releaseAssetCmd,
	)
	releaseAssetCmd.AddCommand(
		releaseAssetListCmd,
		releaseAssetUploadCmd,
		releaseAssetDownloadCmd,
		releaseAssetDeleteCmd,
	)

	addIssuePathFlag(releaseListCmd, releaseLatestCmd, releaseViewCmd, releaseCreateCmd, releaseEditCmd, releaseDeleteCmd)
	addOutputFlag(releaseListCmd, releaseLatestCmd, releaseViewCmd, releaseCreateCmd, releaseEditCmd, releaseDeleteCmd)
	addYesFlag(releaseDeleteCmd)
	addDryRunFlag(releaseCreateCmd, releaseEditCmd, releaseDeleteCmd)
	addIssuePathFlag(releaseAssetListCmd, releaseAssetUploadCmd, releaseAssetDownloadCmd, releaseAssetDeleteCmd)
	addOutputFlag(releaseAssetListCmd, releaseAssetUploadCmd, releaseAssetDeleteCmd)
	addYesFlag(releaseAssetDeleteCmd)
	addDryRunFlag(releaseAssetUploadCmd, releaseAssetDeleteCmd)
	releaseAssetDownloadCmd.Flags().StringP("output-file", "o", "", "Write the asset to this file instead of stdout")

	releaseListCmd.Flags().Bool("include-drafts", false, "Include draft releases in the listing")
	releaseListCmd.Flags().Int("limit", 0, "Max releases per page (server default = 20)")
	releaseListCmd.Flags().String("cursor", "", "Pagination cursor from a prior --output json call")

	releaseCreateCmd.Flags().String("tag", "", "Existing git tag name (must already be pushed)")
	releaseCreateCmd.Flags().String("name", "", "Release display name (defaults to tag)")
	releaseCreateCmd.Flags().String("body", "", "Release notes (markdown)")
	releaseCreateCmd.Flags().Bool("draft", false, "Mark release as draft (hidden until published)")
	releaseCreateCmd.Flags().Bool("prerelease", false, "Mark release as a prerelease")
	_ = releaseCreateCmd.MarkFlagRequired("tag")

	releaseEditCmd.Flags().String("name", "", "Update display name")
	releaseEditCmd.Flags().String("body", "", "Update release notes (markdown)")
	releaseEditCmd.Flags().Bool("draft", false, "Update draft flag (use --draft=false to publish)")
	releaseEditCmd.Flags().Bool("prerelease", false, "Update prerelease flag")
}
