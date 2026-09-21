//go:build windows

package clicfg

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockExclusive takes an exclusive lock on f, returning the unlock func.
// LockFileEx is the Windows equivalent of flock(2); see flock_unix.go.
func lockExclusive(f *os.File) (func(), error) {
	h := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol); err != nil {
		return nil, err
	}
	return func() { _ = windows.UnlockFileEx(h, 0, 1, 0, new(windows.Overlapped)) }, nil
}
