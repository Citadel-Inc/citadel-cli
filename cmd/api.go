package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// APICmd exposes an authenticated escape-hatch mirroring `gh api`.
var APICmd = &cobra.Command{
	Use:   "api <path>",
	Short: "Make an authenticated HTTP request to the Citadel API",
	Long: `Make an authenticated HTTP request to the Citadel REST API and print the response as JSON.

Mirrors 'gh api'. The path must begin with '/'. Fields are supplied with -f key=value and
are JSON-encoded as strings. Use --method / -X to set the HTTP verb (default: GET).

Examples:
  citadel-cli api /namespaces/acme/demo/issues
  citadel-cli api -X POST /namespaces/acme/demo/issues/1/comments -f body_markdown="LGTM"
  citadel-cli api -X PATCH /namespaces/acme/demo/issues/42 -f state=closed
  citadel-cli api -X DELETE /namespaces/acme/demo/issues/42`,
	Args: cobra.ExactArgs(1),
	RunE: runAPI,
}

func runAPI(cmd *cobra.Command, args []string) error {
	method, _ := cmd.Flags().GetString("method")
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}

	path := args[0]
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fields, _ := cmd.Flags().GetStringArray("field")
	var body map[string]any
	if len(fields) > 0 {
		body = make(map[string]any, len(fields))
		for _, f := range fields {
			idx := strings.IndexByte(f, '=')
			if idx < 0 {
				return fmt.Errorf("invalid field %q: must be key=value", f)
			}
			body[f[:idx]] = f[idx+1:]
		}
	}

	c, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	var out json.RawMessage
	switch method {
	case http.MethodGet:
		err = c.Get(cmd.Context(), path, &out)
	case http.MethodPost:
		err = c.Post(cmd.Context(), path, body, &out)
	case http.MethodPut:
		err = c.Put(cmd.Context(), path, body, &out)
	case http.MethodPatch:
		err = c.Patch(cmd.Context(), path, body, &out)
	case http.MethodDelete:
		err = c.Delete(cmd.Context(), path)
		if err == nil {
			return nil
		}
	default:
		return fmt.Errorf("unsupported method %q; use GET, POST, PUT, PATCH, or DELETE", method)
	}
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return nil
	}
	var pretty bytes.Buffer
	if jsonErr := json.Indent(&pretty, out, "", "  "); jsonErr == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), pretty.String())
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
	}
	return nil
}

func init() {
	APICmd.Flags().StringP("method", "X", "", "HTTP method: GET, POST, PUT, PATCH, DELETE (default GET)")
	APICmd.Flags().StringArrayP("field", "f", nil, "Request field as key=value (may be repeated)")
}
