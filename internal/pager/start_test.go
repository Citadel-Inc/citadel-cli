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

func TestStart_ExplicitEmptyPagerOnTTYIsNoop(t *testing.T) {
	old := isStdoutInteractive
	isStdoutInteractive = func() bool { return true }
	t.Cleanup(func() { isStdoutInteractive = old })

	t.Setenv("CITADEL_PAGER", "")
	cleanup, err := Start(false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	cleanup()
}

func TestStart_CatPagerOnTTYIsNoop(t *testing.T) {
	old := isStdoutInteractive
	isStdoutInteractive = func() bool { return true }
	t.Cleanup(func() { isStdoutInteractive = old })

	t.Setenv("CITADEL_PAGER", "cat")
	cleanup, err := Start(false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	cleanup()
}

// Minimal pager command: drains stdin and exits — exercises pipe/exec/wait without less.
func TestStart_CustomPagerTrueHarness(t *testing.T) {
	old := isStdoutInteractive
	isStdoutInteractive = func() bool { return true }
	t.Cleanup(func() { isStdoutInteractive = old })

	t.Setenv("CITADEL_PAGER", "true")
	cleanup, err := Start(false)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	cleanup()
}
