package clicfg

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/term"

	"github.com/Totative/go-keyring"
)

const (
	storeKeyring = "keyring"
	storeFile    = "file"

	reasonNoProvider = "no_provider"
	reasonTooBig     = "too_big"
	reasonLocked     = "locked"
	reasonError      = "keyring_error"

	// Hooks and git credential helpers run without a terminal under short
	// timeouts, so they must give up on an unlock prompt almost at once.
	nonInteractiveTimeout = 500 * time.Millisecond
	interactiveTimeout    = 30 * time.Second
)

// ErrKeyringLocked means the tokens are in the OS keyring and it did not answer.
// The tokens are intact; nothing should be cleared because of this error.
var ErrKeyringLocked = errors.New("the OS keyring is locked: unlock it, or set CITADEL_ACCESS_TOKEN for this command")

var keyringStuck atomic.Bool

type tokenBackend interface {
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
}

// newTokenBackend is replaced by this package's tests. Other packages' tests
// save configs freely and must not reach the developer's real keyring.
var newTokenBackend = func(timeout time.Duration) (tokenBackend, error) {
	if testing.Testing() {
		return nil, keyring.ErrNoProvider
	}
	return keyring.New(keyring.Options{
		Service:     "citadel-cli",
		SealInfo:    "citadel-cli tokens v1",
		Folder:      "citadel-cli",
		ReadTimeout: timeout,
	})
}

func keyringTimeout() time.Duration {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return interactiveTimeout
	}
	return nonInteractiveTimeout
}

// Names carry the config path so separate XDG_CONFIG_HOME profiles stay apart.
func accessItem(path string) string  { return "access_token:" + path }
func refreshItem(path string) string { return "refresh_token:" + path }

func loadKeyringTokens(path string, cfg *Config) error {
	// Several Loads run per command; only the first may spend the timeout.
	if keyringStuck.Load() {
		return fmt.Errorf("%w (%w)", ErrKeyringLocked, keyring.ErrPending)
	}
	timeout := keyringTimeout()
	backend, err := newTokenBackend(timeout)
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}
	if cfg.AccessToken, err = getItem(backend, accessItem(path)); err == nil {
		cfg.RefreshToken, err = getItem(backend, refreshItem(path))
	}
	if errors.Is(err, keyring.ErrPending) {
		keyringStuck.Store(true)
	}
	return err
}

func getItem(backend tokenBackend, name string) (string, error) {
	v, err := backend.Get(name)
	switch {
	case err == nil:
		return v, nil
	case errors.Is(err, keyring.ErrNotFound):
		return "", nil
	case errors.Is(err, keyring.ErrPending), errors.Is(err, keyring.ErrLocked):
		return "", fmt.Errorf("%w (%w)", ErrKeyringLocked, err)
	default:
		return "", fmt.Errorf("read keyring: %w", err)
	}
}

// bounded runs a keyring write, which the library does not bound: a locked
// keyring would hold it on the unlock prompt indefinitely.
func bounded(timeout time.Duration, write func() error) error {
	done := make(chan error, 1)
	go func() { done <- write() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return keyring.ErrPending
	}
}

// storeTokens tries the keyring and reports where the tokens belong. A locked
// keyring also falls back to the file: a rotated refresh token that is not
// written anywhere logs the user out, which is worse than a 0600 file.
func storeTokens(path, access, refresh string) (store, reason string) {
	if keyringStuck.Load() {
		return storeFile, reasonLocked
	}
	timeout := keyringTimeout()
	backend, err := newTokenBackend(timeout)
	if err != nil {
		return storeFile, fallbackReason(err)
	}
	err = bounded(timeout, func() error {
		if err := setItem(backend, accessItem(path), access); err != nil {
			return err
		}
		return setItem(backend, refreshItem(path), refresh)
	})
	if err != nil {
		if errors.Is(err, keyring.ErrPending) {
			keyringStuck.Store(true)
		}
		return storeFile, fallbackReason(err)
	}
	return storeKeyring, ""
}

func setItem(backend tokenBackend, name, value string) error {
	if value == "" {
		err := backend.Delete(name)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	return backend.Set(name, value)
}

func fallbackReason(err error) string {
	switch {
	case errors.Is(err, keyring.ErrNoProvider):
		return reasonNoProvider
	case errors.Is(err, keyring.ErrTooBig):
		return reasonTooBig
	case errors.Is(err, keyring.ErrPending), errors.Is(err, keyring.ErrLocked):
		return reasonLocked
	default:
		return reasonError
	}
}

func deleteKeyringTokens(path string) {
	timeout := keyringTimeout()
	backend, err := newTokenBackend(timeout)
	if err != nil {
		return
	}
	_ = bounded(timeout, func() error {
		return errors.Join(setItem(backend, accessItem(path), ""), setItem(backend, refreshItem(path), ""))
	})
}

// readStoreMarker reads the file alone, so it never touches the keyring.
func readStoreMarker(path string) (store, reason string) {
	var cfg Config
	if err := readFile(path, &cfg); err != nil {
		return "", ""
	}
	return cfg.TokenStore, cfg.TokenStoreReason
}

// TokenStorage describes where the tokens live, for `doctor`. plaintext is true
// when they are in the config file. It never touches the keyring.
func TokenStorage() (description string, plaintext bool, err error) {
	path, err := configPath()
	if err != nil {
		return "", false, err
	}
	var cfg Config
	if err := readFile(path, &cfg); err != nil {
		return "", false, err
	}
	switch {
	case cfg.TokenStore == storeKeyring:
		return "OS keyring", false, nil
	case cfg.AccessToken == "" && cfg.RefreshToken == "":
		return "no stored tokens", false, nil
	case cfg.TokenStoreReason == reasonNoProvider:
		return "0600 config file: no OS keyring is available here", true, nil
	case cfg.TokenStoreReason == reasonTooBig:
		return "0600 config file: the token is over the OS keyring's size limit", true, nil
	case cfg.TokenStoreReason == reasonLocked:
		return "0600 config file: the OS keyring was locked at the last save", true, nil
	case cfg.TokenStoreReason == reasonError:
		return "0600 config file: the OS keyring refused the last save", true, nil
	default:
		return "0600 config file: moves to the OS keyring at the next login or refresh", true, nil
	}
}
