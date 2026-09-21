//go:build !windows

package clicfg

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockExclusive takes an exclusive advisory lock on f, returning the unlock
// func. Windows has no flock(2), so the lock is per-platform; see
// flock_windows.go.
func lockExclusive(f *os.File) (func(), error) {
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return nil, err
	}
	return func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }, nil
}
