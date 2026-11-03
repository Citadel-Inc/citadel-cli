package cmd

import "time"

// SetLabelCreateWait shortens the rate-limit pause for tests and returns a restore func.
func SetLabelCreateWait(d time.Duration) func() {
	old := labelCreateWait
	labelCreateWait = d
	return func() { labelCreateWait = old }
}
