// Package completion implements disk-backed caching and API-backed lookup for
// shell completion candidates (repo slugs, namespace slugs, etc.).
package completion

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const cacheTTL = 60 * time.Second

// now returns the current time; tests may override.
var now = time.Now

type envelope struct {
	FetchedAt time.Time `json:"fetched_at"`
	Server    string    `json:"server"`
	Resource  string    `json:"resource"`
	Values    []string  `json:"values"`
}

func diskCacheDisabled() bool {
	return strings.TrimSpace(os.Getenv("CITADEL_NO_COMPLETION_CACHE")) == "1"
}

func cacheBaseDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "citadel-cli", "completion"), nil
}

func safeHostDir(resolvedServer string) string {
	resolvedServer = strings.TrimSpace(resolvedServer)
	if resolvedServer == "" {
		return "default"
	}
	// Strip scheme for a stable directory name.
	s := resolvedServer
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	s = strings.TrimSuffix(s, "/")
	repl := strings.NewReplacer(":", "_", "/", "_", "\\", "_", "?", "_", "*", "_")
	s = repl.Replace(s)
	if s == "." || s == ".." {
		return "_" + s
	}
	return s
}

func safeResourceFileStem(resourceKey string) string {
	var b strings.Builder
	for _, r := range resourceKey {
		switch r {
		case ':', '/', '\\', '?', '*', '"', '<', '>', '|':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cacheFilePath(resolvedServer, resourceKey string) (string, error) {
	dir, err := cacheBaseDir()
	if err != nil {
		return "", err
	}
	host := safeHostDir(resolvedServer)
	stem := safeResourceFileStem(resourceKey)
	return filepath.Join(dir, host, stem+".json"), nil
}

func withCacheRoot(resolvedServer, resourceKey string, create bool, fn func(*os.Root, string) error) error {
	dir, err := cacheBaseDir()
	if err != nil {
		return err
	}
	if create {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	relative := filepath.Join(safeHostDir(resolvedServer), safeResourceFileStem(resourceKey)+".json")
	fnErr := fn(root, relative)
	closeErr := root.Close()
	return errors.Join(fnErr, closeErr)
}

var (
	memMu sync.Mutex
	mem   = map[string]envelope{}
)

func memKey(resolvedServer, resourceKey string) string {
	return safeHostDir(resolvedServer) + "\x00" + resourceKey
}

func readCache(resolvedServer, resourceKey string) ([]string, bool) {
	key := memKey(resolvedServer, resourceKey)
	memMu.Lock()
	ent, ok := mem[key]
	memMu.Unlock()
	if ok && now().Sub(ent.FetchedAt) < cacheTTL && ent.Server == resolvedServer && ent.Resource == resourceKey {
		return append([]string(nil), ent.Values...), true
	}
	if diskCacheDisabled() {
		return nil, false
	}
	var data []byte
	err := withCacheRoot(resolvedServer, resourceKey, false, func(root *os.Root, relative string) error {
		var err error
		data, err = root.ReadFile(relative)
		return err
	})
	if err != nil {
		return nil, false
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, false
	}
	if env.Server != resolvedServer || env.Resource != resourceKey {
		return nil, false
	}
	if now().Sub(env.FetchedAt) >= cacheTTL {
		return nil, false
	}
	return append([]string(nil), env.Values...), true
}

func writeCache(resolvedServer, resourceKey string, values []string) {
	env := envelope{
		FetchedAt: now(),
		Server:    resolvedServer,
		Resource:  resourceKey,
		Values:    append([]string(nil), values...),
	}
	key := memKey(resolvedServer, resourceKey)
	memMu.Lock()
	mem[key] = env
	memMu.Unlock()
	if diskCacheDisabled() {
		return
	}
	if err := withCacheRoot(resolvedServer, resourceKey, true, func(root *os.Root, relative string) error {
		if err := root.MkdirAll(filepath.Dir(relative), 0o700); err != nil {
			return err
		}
		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		tmp := relative + ".tmp"
		if err := root.WriteFile(tmp, data, 0o600); err != nil {
			return err
		}
		if err := root.Rename(tmp, relative); err != nil {
			if removeErr := root.Remove(tmp); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return errors.Join(err, removeErr)
			}
			return err
		}
		return nil
	}); err != nil {
		return
	}
}

// Remove deletes the on-disk cache entry (if any) and the in-memory copy
// for resolvedServer + resourceKey.
func Remove(resolvedServer, resourceKey string) {
	memMu.Lock()
	delete(mem, memKey(resolvedServer, resourceKey))
	memMu.Unlock()
	if diskCacheDisabled() {
		return
	}
	if err := withCacheRoot(resolvedServer, resourceKey, false, func(root *os.Root, relative string) error {
		return root.Remove(relative)
	}); err != nil {
		return
	}
}

// RemoveAsync invokes Remove in a background goroutine.
func RemoveAsync(resolvedServer string, resourceKeys ...string) {
	if len(resourceKeys) == 0 {
		return
	}
	go func(srv string, keys []string) {
		for _, k := range keys {
			Remove(srv, k)
		}
	}(resolvedServer, append([]string(nil), resourceKeys...))
}
