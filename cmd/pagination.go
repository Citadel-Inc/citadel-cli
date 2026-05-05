package cmd

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/pagination"
	"github.com/spf13/cobra"
)

func addPaginationFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().Int("limit", pagination.DefaultLimit, "Maximum rows per page (1–200; server default 50)")
		c.Flags().String("cursor", "", "Opaque pagination cursor from a prior list response")
		c.Flags().Bool("all", false, "Fetch every page until exhausted (serial; use with --output ndjson to stream)")
	}
}

func readPagination(cmd *cobra.Command) (limit int, cursor string, all bool, err error) {
	all, _ = cmd.Flags().GetBool("all")
	cursor, _ = cmd.Flags().GetString("cursor")
	limit, _ = cmd.Flags().GetInt("limit")
	if limit == 0 {
		limit = pagination.DefaultLimit
	}
	if limit < 1 || limit > pagination.MaxLimit {
		return 0, "", false, fmt.Errorf("--limit must be between 1 and %d", pagination.MaxLimit)
	}
	if all {
		cursor = ""
	}
	return limit, strings.TrimSpace(cursor), all, nil
}

func validateDescCursor(cur string) error {
	if cur == "" {
		return nil
	}
	_, err := pagination.DecodeDesc(cur)
	return err
}

func validateMemberCursor(cur string) error {
	if cur == "" {
		return nil
	}
	_, err := pagination.DecodeMemberAsc(cur)
	return err
}

func validateAuditCursor(cur string) error {
	if cur == "" {
		return nil
	}
	_, err := pagination.DecodeAuditDesc(cur)
	return err
}
