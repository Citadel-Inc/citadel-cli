package apiclient

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ProbeLoopback = false
	os.Exit(m.Run())
}
