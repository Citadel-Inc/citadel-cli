package completion

import (
	"os"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/internal/apiclient"
)

func TestMain(m *testing.M) {
	apiclient.ProbeLoopback = false
	os.Exit(m.Run())
}
