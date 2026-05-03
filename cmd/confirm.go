package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// confirmSlug prompts the user to type slug to confirm a destructive action.
// Skipped when yes is true (--yes flag).
func confirmSlug(yes bool, action, slug string) error {
	if yes {
		return nil
	}
	fmt.Fprintf(os.Stderr, "This action is irreversible. To confirm %s, type '%s': ", action, slug)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != slug {
		return fmt.Errorf("confirmation mismatch — operation aborted")
	}
	return nil
}
