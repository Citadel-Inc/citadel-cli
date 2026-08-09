package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// APICmd exposes an authenticated escape-hatch mirroring `gh api`.
var APICmd = &cobra.Command{
	Use:     "api <path>",
	GroupID: "ops",
	Short:   "Make an authenticated HTTP request to the Citadel API",
	Long: `Make an authenticated HTTP request to the Citadel REST API and print the response as JSON.

Mirrors 'gh api'. The path must begin with '/'. Fields are supplied with -f key=value and
are JSON-encoded as strings, or use --input to read a raw JSON body from a file or stdin.
Use --method / -X to set the HTTP verb (default: GET).

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

	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return fmt.Errorf("unsupported method %q; use GET, POST, PUT, PATCH, or DELETE", method)
	}

	fields, _ := cmd.Flags().GetStringArray("field")
	input, _ := cmd.Flags().GetString("input")
	inputSet := cmd.Flags().Changed("input")
	if inputSet && len(fields) > 0 {
		return fmt.Errorf("--input and --field are mutually exclusive")
	}

	var body any
	if inputSet {
		switch method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
		default:
			return fmt.Errorf("--input is only supported with POST, PUT, or PATCH")
		}

		var inputJSON []byte
		var err error
		if input == "-" {
			inputJSON, err = io.ReadAll(cmd.InOrStdin())
		} else {
			inputJSON, err = os.ReadFile(input)
		}
		if err != nil {
			return fmt.Errorf("read API input %q: %w", input, err)
		}
		var inputValue any
		if err := json.Unmarshal(inputJSON, &inputValue); err != nil {
			return fmt.Errorf("--input: invalid JSON: %w", err)
		}
		body = json.RawMessage(inputJSON)
	}

	if len(fields) > 0 {
		fieldsBody := make(map[string]any, len(fields))
		for _, f := range fields {
			idx := strings.IndexByte(f, '=')
			if idx < 0 {
				return fmt.Errorf("invalid field %q: must be key=value", f)
			}
			fieldsBody[f[:idx]] = f[idx+1:]
		}
		body = fieldsBody
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
	APICmd.Flags().String("input", "", "Read raw JSON request body from file or stdin with '-' (POST, PUT, PATCH only)")
}
