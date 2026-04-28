package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "cache.json")
	expiry := time.Now().Add(24 * time.Hour).Round(time.Second).UTC()

	tok := &oauth2.Token{
		AccessToken: "abc123",
		TokenType:   "Bearer",
		Expiry:      expiry,
	}
	if err := saveCache(path, tok); err != nil {
		t.Fatalf("saveCache: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	if mode := fi.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected cache file mode 0600, got %#o", mode)
	}

	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if mode := di.Mode().Perm(); mode != 0o700 {
		t.Errorf("expected cache dir mode 0700, got %#o", mode)
	}

	got, err := loadCache(context.Background(), path)
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil token")
	}
	if got.AccessToken != tok.AccessToken {
		t.Errorf("AccessToken mismatch: got %q want %q", got.AccessToken, tok.AccessToken)
	}
	if got.TokenType != tok.TokenType {
		t.Errorf("TokenType mismatch: got %q want %q", got.TokenType, tok.TokenType)
	}
	if !got.Expiry.Equal(expiry) {
		t.Errorf("Expiry mismatch: got %v want %v", got.Expiry, expiry)
	}
}

func TestLoadCache_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")
	got, err := loadCache(context.Background(), path)
	if err != nil {
		t.Fatalf("loadCache on missing file: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil token, got %+v", got)
	}
}

func TestLoadCache_CorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(path, []byte("not json {{"), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	got, err := loadCache(context.Background(), path)
	if err != nil {
		t.Fatalf("loadCache corrupt: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil token on corrupt JSON, got %+v", got)
	}
}

func TestLoadCache_PermissiveMode_StillLoads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perm.json")
	tok := &oauth2.Token{AccessToken: "t", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	if err := saveCache(path, tok); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	got, err := loadCache(context.Background(), path)
	if err != nil {
		t.Fatalf("loadCache with permissive mode: %v", err)
	}
	if got == nil || got.AccessToken != "t" {
		t.Errorf("expected token to still load; got %+v", got)
	}
}

func TestCachePath_EnvOverride(t *testing.T) {
	t.Setenv("BAFFINBAY_TOKEN_CACHE", "/tmp/override.json")
	got, err := CachePath("https://example.com", "client")
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if got != "/tmp/override.json" {
		t.Errorf("expected override path, got %q", got)
	}
}

func TestCachePath_DefaultUsesUserConfigDir(t *testing.T) {
	t.Setenv("BAFFINBAY_TOKEN_CACHE", "")
	p1, err := CachePath("https://example.com/a", "client-a")
	if err != nil {
		t.Fatalf("CachePath a: %v", err)
	}
	p2, err := CachePath("https://example.com/b", "client-a")
	if err != nil {
		t.Fatalf("CachePath b: %v", err)
	}
	if p1 == p2 {
		t.Errorf("expected different paths for different OIDC URLs, both %q", p1)
	}
	if filepath.Base(filepath.Dir(p1)) != "baffinbay" {
		t.Errorf("expected baffinbay dir, got %q", p1)
	}
}

func TestCachePath_RejectsRelativeOverride(t *testing.T) {
	t.Setenv("BAFFINBAY_TOKEN_CACHE", "relative/cache.json")
	if _, err := CachePath("https://example.com", "client"); err == nil {
		t.Fatal("expected error for relative override path")
	}
}

func TestCachePath_WhitespaceOverrideFallsThrough(t *testing.T) {
	t.Setenv("BAFFINBAY_TOKEN_CACHE", "   ")
	got, err := CachePath("https://example.com", "client")
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if filepath.Base(filepath.Dir(got)) != "baffinbay" {
		t.Errorf("expected fallback to default baffinbay dir, got %q", got)
	}
}
