package pager

import "testing"

// Operator passes --no-pager: Start must not swap stdout or spawn a process.
func TestStart_DisabledIsNoop(t *testing.T) {
	cleanup, err := Start(true)
	if err != nil {
		t.Fatalf("Start(disabled): %v", err)
	}
	cleanup()
}

// Non-interactive stdout (typical under go test): never start a pager.
func TestStart_NonTTYIsNoop(t *testing.T) {
	cleanup, err := Start(false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	cleanup()
}
