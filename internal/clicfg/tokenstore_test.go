package clicfg

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Totative/go-keyring"
)

type fakeBackend struct {
	mu     sync.Mutex
	items  map[string]string
	setErr error
	getErr error
	hang   chan struct{}
}

func (f *fakeBackend) Get(name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.items[name]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (f *fakeBackend) Set(name, value string) error {
	if f.hang != nil {
		<-f.hang
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.items[name] = value
	return nil
}

func (f *fakeBackend) Delete(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[name]; !ok {
		return keyring.ErrNotFound
	}
	delete(f.items, name)
	return nil
}

func useBackend(t *testing.T) (*fakeBackend, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(accessTokenEnv, "")
	keyringStuck.Store(false)
	t.Cleanup(func() { keyringStuck.Store(false) })
	f := &fakeBackend{items: map[string]string{}}
	old := newTokenBackend
	newTokenBackend = func(time.Duration) (tokenBackend, error) { return f, nil }
	t.Cleanup(func() { newTokenBackend = old })
	path, err := configPath()
	if err != nil {
		t.Fatal(err)
	}
	return f, path
}

func fileText(t *testing.T, path string) string {
	t.Helper()
	b, err := safefs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTokensGoToKeyringAndLeaveTheFile(t *testing.T) {
	f, path := useBackend(t)
	if err := (Config{AccessToken: "acc", RefreshToken: "ref", UserUUID: "u"}).Save(); err != nil {
		t.Fatal(err)
	}
	text := fileText(t, path)
	if strings.Contains(text, "'acc'") || strings.Contains(text, "'ref'") {
		t.Fatalf("token in file:\n%s", text)
	}
	if f.items[accessItem(path)] != "acc" || f.items[refreshItem(path)] != "ref" {
		t.Fatalf("items = %v", f.items)
	}
	cfg, err := Load()
	if err != nil || cfg.AccessToken != "acc" || cfg.RefreshToken != "ref" || cfg.UserUUID != "u" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
	if desc, plain, err := TokenStorage(); err != nil || plain || desc != "OS keyring" {
		t.Fatalf("TokenStorage = %q %v %v", desc, plain, err)
	}
}

func TestExistingFileTokensKeepWorkingThenMoveOnSave(t *testing.T) {
	f, path := useBackend(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := "access_token = 'old'\nrefresh_token = 'oldref'\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	f.getErr = errors.New("must not be read")
	cfg, err := Load()
	if err != nil || cfg.AccessToken != "old" || cfg.RefreshToken != "oldref" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
	if _, plain, _ := TokenStorage(); !plain {
		t.Fatal("legacy file not reported as plaintext")
	}
	f.getErr = nil
	if err := Update(func(c *Config) error { c.AccessToken = "new"; return nil }); err != nil {
		t.Fatal(err)
	}
	if text := fileText(t, path); strings.Contains(text, "oldref") || strings.Contains(text, "'new'") {
		t.Fatalf("token left in file:\n%s", text)
	}
	cfg, err = Load()
	if err != nil || cfg.AccessToken != "new" || cfg.RefreshToken != "oldref" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
}

func TestFallbacksKeepTokensInPrivateFileAndSayWhy(t *testing.T) {
	cases := map[string]error{
		"no OS keyring": keyring.ErrNoProvider,
		"size limit":    keyring.ErrTooBig,
		"locked":        keyring.ErrLocked,
		"refused":       errors.New("boom"),
	}
	for want, setErr := range cases {
		t.Run(want, func(t *testing.T) {
			f, path := useBackend(t)
			f.setErr = setErr
			if err := (Config{AccessToken: "acc", RefreshToken: "ref"}).Save(); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("stat = %v, %v", info, err)
			}
			cfg, err := Load()
			if err != nil || cfg.AccessToken != "acc" || cfg.RefreshToken != "ref" {
				t.Fatalf("Load = %+v, %v", cfg, err)
			}
			desc, plain, err := TokenStorage()
			if err != nil || !plain || !strings.Contains(desc, want) {
				t.Fatalf("TokenStorage = %q %v %v", desc, plain, err)
			}
		})
	}
}

func TestLockedKeyringFailsLoadWithoutClearingAnything(t *testing.T) {
	f, path := useBackend(t)
	if err := (Config{AccessToken: "acc", RefreshToken: "ref"}).Save(); err != nil {
		t.Fatal(err)
	}
	f.getErr = keyring.ErrPending
	if _, err := Load(); !errors.Is(err, ErrKeyringLocked) {
		t.Fatalf("Load err = %v", err)
	}
	if err := Update(func(c *Config) error { c.AccessToken = ""; return nil }); !errors.Is(err, ErrKeyringLocked) {
		t.Fatalf("Update err = %v", err)
	}
	f.getErr = nil
	if f.items[refreshItem(path)] != "ref" {
		t.Fatal("refresh token lost")
	}
}

func TestSaveNeverWaitsOnAHungKeyring(t *testing.T) {
	f, _ := useBackend(t)
	f.hang = make(chan struct{})
	defer close(f.hang)
	start := time.Now()
	if err := (Config{AccessToken: "acc"}).Save(); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("Save took %v", d)
	}
	if cfg, err := readBack(); err != nil || cfg.TokenStoreReason != reasonLocked || cfg.AccessToken != "acc" {
		t.Fatalf("file = %+v, %v", cfg, err)
	}
}

func readBack() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	err = readFile(path, &cfg)
	return cfg, err
}

func TestEnvTokenWinsSkipsKeyringAndCannotReplaceLogin(t *testing.T) {
	f, path := useBackend(t)
	if err := (Config{AccessToken: "acc", RefreshToken: "ref"}).Save(); err != nil {
		t.Fatal(err)
	}
	t.Setenv(accessTokenEnv, "env")
	f.getErr = keyring.ErrPending
	cfg, err := Load()
	if err != nil || cfg.AccessToken != "env" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if f.items[accessItem(path)] != "acc" || f.items[refreshItem(path)] != "ref" {
		t.Fatalf("login replaced: %v", f.items)
	}
	if strings.Contains(fileText(t, path), "env") {
		t.Fatal("env token written to file")
	}
}

func TestLogoutDeletesKeyringItems(t *testing.T) {
	f, _ := useBackend(t)
	if err := (Config{AccessToken: "acc", RefreshToken: "ref"}).Save(); err != nil {
		t.Fatal(err)
	}
	if err := (Config{}).Save(); err != nil {
		t.Fatal(err)
	}
	if len(f.items) != 0 {
		t.Fatalf("items left: %v", f.items)
	}
}

func TestOnlyTheFirstLoadSpendsTheKeyringTimeout(t *testing.T) {
	f, _ := useBackend(t)
	if err := (Config{AccessToken: "acc"}).Save(); err != nil {
		t.Fatal(err)
	}
	f.getErr = keyring.ErrPending
	calls := 0
	old := newTokenBackend
	newTokenBackend = func(d time.Duration) (tokenBackend, error) { calls++; return old(d) }
	t.Cleanup(func() { newTokenBackend = old })
	for range 3 {
		if _, err := Load(); !errors.Is(err, ErrKeyringLocked) {
			t.Fatalf("err = %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("backend opened %d times, want 1", calls)
	}
}
