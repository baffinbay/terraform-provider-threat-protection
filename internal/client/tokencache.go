package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
)

const (
	cacheVersion  = 1
	cacheDirMode  = 0o700
	cacheFileMode = 0o600
)

type cachedToken struct {
	Version     int       `json:"version"`
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CachePath computes the on-disk cache path for a given OIDC endpoint + client ID.
// Honors $BAFFINBAY_TOKEN_CACHE when set. Otherwise derives from os.UserConfigDir.
func CachePath(oidcURL, clientID string) (string, error) {
	if p := os.Getenv("BAFFINBAY_TOKEN_CACHE"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	sum := sha256.Sum256([]byte(oidcURL + "\x00" + clientID))
	name := "token-cache-" + hex.EncodeToString(sum[:8]) + ".json"
	return filepath.Join(base, "baffinbay", name), nil
}

// loadCache reads the cache file at path. Missing or corrupt files are treated
// as cache misses (returns nil, nil). A file present with wider-than-0600
// permissions logs a warning but still loads.
func loadCache(ctx context.Context, path string) (*oauth2.Token, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		tflog.Warn(ctx, "baffinbay token cache has overly permissive mode; will be rewritten at 0600 on next refresh",
			map[string]interface{}{"path": path, "mode": fmt.Sprintf("%#o", mode)})
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c cachedToken
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, nil
	}
	if c.Version != cacheVersion || c.AccessToken == "" {
		return nil, nil
	}
	tt := c.TokenType
	if tt == "" {
		tt = "Bearer"
	}
	return &oauth2.Token{
		AccessToken: c.AccessToken,
		TokenType:   tt,
		Expiry:      c.ExpiresAt,
	}, nil
}

// saveCache atomically writes tok to path with mode 0600. The containing
// directory is created at 0700 if missing.
func saveCache(path string, tok *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, cacheDirMode); err != nil {
		return err
	}
	payload := cachedToken{
		Version:     cacheVersion,
		AccessToken: tok.AccessToken,
		TokenType:   tok.TokenType,
		ExpiresAt:   tok.Expiry.UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".token-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(cacheFileMode); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// withLock acquires an exclusive flock on a sibling .lock file next to path,
// invokes fn, and releases. The lock file is created at 0600 in a 0700 dir.
func withLock(path string, fn func() error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, cacheDirMode); err != nil {
		return err
	}
	lockPath := path + ".lock"
	lock := flock.New(lockPath)
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("acquire token cache lock: %w", err)
	}
	defer func() { _ = lock.Unlock() }()
	_ = os.Chmod(lockPath, cacheFileMode)
	return fn()
}
