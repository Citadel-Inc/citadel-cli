package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
)

// GistCmd is the top-level command for standalone gist management.
var GistCmd = &cobra.Command{
	Use:     "gist",
	GroupID: "repo",
	Short:   "Manage standalone gists",
	Long: `Manage standalone gists owned by the authenticated user.

Examples:
  citadel-cli gist list
  citadel-cli gist view <id>
  citadel-cli gist create --title "Snippet" --file main.go='package main'
  citadel-cli gist edit <id> --visibility private
  citadel-cli gist delete <id>`,
}

var gistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List gists",
	RunE:  runGistList,
}

var gistViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "Show one gist",
	Args:  cobra.ExactArgs(1),
	RunE:  runGistView,
}

var gistCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a gist",
	RunE:  runGistCreate,
}

var gistEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a gist",
	Args:  cobra.ExactArgs(1),
	RunE:  runGistEdit,
}

var gistDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a gist",
	Args:  cobra.ExactArgs(1),
	RunE:  runGistDelete,
}

var gistRawCmd = &cobra.Command{
	Use:   "raw <id> <file>",
	Short: "Download one gist file",
	Args:  cobra.ExactArgs(2),
	RunE:  runGistRaw,
}

type gistRow struct {
	ID          string `json:"id"`
	OwnerUserID string `json:"owner_user_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility"`
	HeadSHA     string `json:"head_sha,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func (r gistRow) CSVHeader() []string {
	return []string{"id", "title", "visibility", "created_at", "updated_at"}
}

func (r gistRow) CSVRecord() []string {
	return []string{r.ID, r.Title, r.Visibility, r.CreatedAt, r.UpdatedAt}
}

type gistFile struct {
	Path string `json:"path"`
	Raw  string `json:"raw,omitempty"`
	SHA  string `json:"sha,omitempty"`
}

type gistListPayload struct {
	Gists []gistRow `json:"gists"`
}

type gistViewPayload struct {
	Gist      gistRow    `json:"gist"`
	Files     []gistFile `json:"files"`
	Comments  []any      `json:"comments,omitempty"`
	Reactions any        `json:"reactions,omitempty"`
	Symbols   []any      `json:"symbols,omitempty"`
}

type gistWriteResult struct {
	ID             string `json:"id"`
	CommitSHA      string `json:"commit_sha"`
	SecretWarnings []any  `json:"secret_warnings,omitempty"`
}

type gistCreateRequest struct {
	Title                string            `json:"title"`
	Description          string            `json:"description,omitempty"`
	Visibility           string            `json:"visibility"`
	Files                map[string]string `json:"files"`
	ConfirmSecretPublish bool              `json:"confirm_secret_publish,omitempty"`
	ExpiresIn            string            `json:"expires_in,omitempty"`
}

type gistUpdateRequest struct {
	Title                *string            `json:"title,omitempty"`
	Description          *string            `json:"description,omitempty"`
	Visibility           *string            `json:"visibility,omitempty"`
	Files                map[string]*string `json:"files,omitempty"`
	ParentSHA            string             `json:"parent_sha,omitempty"`
	Message              string             `json:"message,omitempty"`
	ConfirmSecretPublish bool               `json:"confirm_secret_publish,omitempty"`
	ExpiresIn            *string            `json:"expires_in,omitempty"`
}

func gistPath(id string) string {
	return "/gists/" + url.PathEscape(strings.TrimSpace(id))
}

func gistRawPath(id, file string) (string, error) {
	file = strings.Trim(strings.TrimSpace(file), "/")
	if file == "" {
		return "", fmt.Errorf("file path required")
	}
	parts := strings.Split(file, "/")
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid file path %q", file)
		}
		encoded = append(encoded, url.PathEscape(part))
	}
	return gistPath(id) + "/raw/" + strings.Join(encoded, "/"), nil
}

func normalizeGistVisibility(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "public", "unlisted":
		return strings.ToLower(strings.TrimSpace(value)), nil
	case "private", "secret":
		return "private", nil
	default:
		return "", fmt.Errorf("visibility must be public, unlisted, or private (secret)")
	}
}

func parseGistFiles(values []string) (map[string]string, error) {
	files := make(map[string]string, len(values))
	for _, value := range values {
		path, content, ok := strings.Cut(value, "=")
		path = strings.TrimSpace(path)
		if !ok || path == "" {
			return nil, fmt.Errorf("--file must use path=content")
		}
		if strings.HasPrefix(content, "@") {
			data, err := os.ReadFile(strings.TrimPrefix(content, "@"))
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", strings.TrimPrefix(content, "@"), err)
			}
			content = string(data)
		}
		files[path] = content
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one --file path=content is required")
	}
	return files, nil
}

func runGistList(cmd *cobra.Command, _ []string) error {
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateListOutput(output); err != nil {
		return err
	}
	q := url.Values{}
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset, _ := cmd.Flags().GetInt("offset"); offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	path := "/gists"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var payload gistListPayload
	if err := c.Get(cmd.Context(), path, &payload); err != nil {
		return err
	}
	if payload.Gists == nil {
		payload.Gists = []gistRow{}
	}
	if len(payload.Gists) == 0 && output == "csv" {
		return emitCSVHeaderOnly[gistRow](cmd)
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	case "ndjson":
		return emitNDJSONLines(cmd, payload.Gists)
	case "csv":
		csvHeader := false
		return emitCSVRows(cmd, &csvHeader, payload.Gists)
	default:
		if len(payload.Gists) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No gists.")
			return nil
		}
		w := newTabWriter(cmd)
		_, _ = fmt.Fprintln(w, "ID\tTITLE\tVISIBILITY\tUPDATED")
		for _, row := range payload.Gists {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.ID, row.Title, row.Visibility, row.UpdatedAt)
		}
		return w.Flush()
	}
}

func runGistView(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("gist id required")
	}
	var payload gistViewPayload
	if err := c.Get(cmd.Context(), gistPath(id), &payload); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("gist %s not found", id)
		}
		return err
	}
	switch output {
	case "json":
		return emitJSON(cmd, payload)
	case "yaml":
		return emitYAML(cmd, payload)
	default:
		return renderGistView(cmd, payload)
	}
}

func runGistCreate(cmd *cobra.Command, _ []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	title, _ := cmd.Flags().GetString("title")
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("--title required")
	}
	visibility, _ := cmd.Flags().GetString("visibility")
	visibility, err := normalizeGistVisibility(visibility)
	if err != nil {
		return err
	}
	fileSpecs, _ := cmd.Flags().GetStringArray("file")
	files, err := parseGistFiles(fileSpecs)
	if err != nil {
		return err
	}
	description, _ := cmd.Flags().GetString("description")
	confirm, _ := cmd.Flags().GetBool("confirm-secret-publish")
	expires, _ := cmd.Flags().GetString("expires-in")
	req := gistCreateRequest{
		Title:                title,
		Description:          description,
		Visibility:           visibility,
		Files:                files,
		ConfirmSecretPublish: confirm,
		ExpiresIn:            strings.TrimSpace(expires),
	}
	path := "/gists"
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would POST %s title=%s (skipped; --dry-run)\n", path, title)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var result gistWriteResult
	if err := c.Post(cmd.Context(), path, req, &result); err != nil {
		return err
	}
	return renderGistWriteResult(cmd, output, "Created gist", result)
}

func runGistEdit(cmd *cobra.Command, args []string) error {
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	if err := validateGetOutput(output); err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("gist id required")
	}
	req := gistUpdateRequest{}
	changed := false
	if cmd.Flags().Changed("title") {
		value, _ := cmd.Flags().GetString("title")
		req.Title = &value
		changed = true
	}
	if cmd.Flags().Changed("description") {
		value, _ := cmd.Flags().GetString("description")
		req.Description = &value
		changed = true
	}
	if cmd.Flags().Changed("visibility") {
		value, _ := cmd.Flags().GetString("visibility")
		value, err := normalizeGistVisibility(value)
		if err != nil {
			return err
		}
		req.Visibility = &value
		changed = true
	}
	if cmd.Flags().Changed("file") {
		fileSpecs, _ := cmd.Flags().GetStringArray("file")
		files, err := parseGistFiles(fileSpecs)
		if err != nil {
			return err
		}
		req.Files = make(map[string]*string, len(files))
		for path, content := range files {
			value := content
			req.Files[path] = &value
		}
		changed = true
	}
	if cmd.Flags().Changed("parent-sha") {
		req.ParentSHA, _ = cmd.Flags().GetString("parent-sha")
		changed = true
	}
	if cmd.Flags().Changed("message") {
		req.Message, _ = cmd.Flags().GetString("message")
		changed = true
	}
	if cmd.Flags().Changed("expires-in") {
		value, _ := cmd.Flags().GetString("expires-in")
		req.ExpiresIn = &value
		changed = true
	}
	if confirm, _ := cmd.Flags().GetBool("confirm-secret-publish"); confirm {
		req.ConfirmSecretPublish = true
		changed = true
	}
	if !changed {
		return fmt.Errorf("nothing to update: pass --title, --description, --visibility, --file, --parent-sha, --message, or --expires-in")
	}
	path := gistPath(id)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would PUT %s (skipped; --dry-run)\n", path)
		return nil
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	var result gistWriteResult
	if err := c.Put(cmd.Context(), path, req, &result); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("gist %s not found", id)
		}
		return err
	}
	return renderGistWriteResult(cmd, output, "Updated gist", result)
}

func runGistDelete(cmd *cobra.Command, args []string) error {
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("gist id required")
	}
	path := gistPath(id)
	if dryRunFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would DELETE %s (skipped; --dry-run)\n", path)
		return nil
	}
	if !yesFlag(cmd) {
		if err := confirmTypedValue(false, "delete gist", id); err != nil {
			return err
		}
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if err := c.Delete(cmd.Context(), path); err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("gist %s not found", id)
		}
		return err
	}
	output := strings.TrimSpace(strings.ToLower(outputFlag(cmd)))
	switch output {
	case "json":
		return emitJSON(cmd, map[string]any{"deleted": true, "id": id})
	case "yaml":
		return emitYAML(cmd, map[string]any{"deleted": true, "id": id})
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted gist %s.\n", id)
		return nil
	}
}

func runGistRaw(cmd *cobra.Command, args []string) error {
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("gist id required")
	}
	path, err := gistRawPath(id, args[1])
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	resp, err := c.GetStream(cmd.Context(), path)
	if err != nil {
		if apiclient.IsStatus(err, http.StatusNotFound) {
			return fmt.Errorf("gist file %s not found", args[1])
		}
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read gist file: %w", err)
	}
	outputFile, _ := cmd.Flags().GetString("output-file")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", outputFile, err)
		}
		return nil
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func renderGistView(cmd *cobra.Command, payload gistViewPayload) error {
	w := newTabWriter(cmd)
	_, _ = fmt.Fprintf(w, "ID\t%s\n", payload.Gist.ID)
	_, _ = fmt.Fprintf(w, "Title\t%s\n", payload.Gist.Title)
	_, _ = fmt.Fprintf(w, "Visibility\t%s\n", payload.Gist.Visibility)
	if payload.Gist.Description != "" {
		_, _ = fmt.Fprintf(w, "Description\t%s\n", payload.Gist.Description)
	}
	if payload.Gist.CreatedAt != "" {
		_, _ = fmt.Fprintf(w, "Created\t%s\n", payload.Gist.CreatedAt)
	}
	if payload.Gist.UpdatedAt != "" {
		_, _ = fmt.Fprintf(w, "Updated\t%s\n", payload.Gist.UpdatedAt)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if len(payload.Files) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Files:")
		for _, file := range payload.Files {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", file.Path)
		}
	}
	return nil
}

func renderGistWriteResult(cmd *cobra.Command, output, label string, result gistWriteResult) error {
	switch output {
	case "json":
		return emitJSON(cmd, result)
	case "yaml":
		return emitYAML(cmd, result)
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s.\n", label, result.ID)
		if result.CommitSHA != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", result.CommitSHA)
		}
		return nil
	}
}

func init() {
	GistCmd.AddCommand(gistListCmd, gistViewCmd, gistCreateCmd, gistEditCmd, gistDeleteCmd, gistRawCmd)

	addOutputFlag(gistListCmd, gistViewCmd, gistCreateCmd, gistEditCmd, gistDeleteCmd)
	addYesFlag(gistDeleteCmd)
	addDryRunFlag(gistCreateCmd, gistEditCmd, gistDeleteCmd)

	gistListCmd.Flags().Int("limit", 0, "Max gists per page")
	gistListCmd.Flags().Int("offset", 0, "Number of gists to skip")

	gistCreateCmd.Flags().String("title", "", "Gist title (required)")
	gistCreateCmd.Flags().String("description", "", "Gist description")
	gistCreateCmd.Flags().String("visibility", "public", "Visibility: public, unlisted, or private (secret)")
	gistCreateCmd.Flags().StringArray("file", nil, "File entry as path=content; repeat for multiple files")
	gistCreateCmd.Flags().Bool("confirm-secret-publish", false, "Confirm publishing content flagged by the secret scanner")
	gistCreateCmd.Flags().String("expires-in", "", "Expiration: 24h, 7d, or 30d")

	gistEditCmd.Flags().String("title", "", "Update gist title")
	gistEditCmd.Flags().String("description", "", "Update gist description")
	gistEditCmd.Flags().String("visibility", "", "Update visibility: public, unlisted, or private (secret)")
	gistEditCmd.Flags().StringArray("file", nil, "Update a file entry as path=content; repeat for multiple files")
	gistEditCmd.Flags().String("parent-sha", "", "Expected current commit SHA")
	gistEditCmd.Flags().String("message", "", "Commit message")
	gistEditCmd.Flags().Bool("confirm-secret-publish", false, "Confirm publishing content flagged by the secret scanner")
	gistEditCmd.Flags().String("expires-in", "", "Set expiration or use clear to remove it")

	gistRawCmd.Flags().StringP("output-file", "o", "", "Write downloaded bytes to a file instead of stdout")
}
