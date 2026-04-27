package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// TestConcurrentMint simulates matrix CI jobs sharing a token cache path.
// With flock + double-checked load, we expect exactly one OIDC mint across N
// concurrent callers.
func TestConcurrentMint(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oidcResponse{
			AccessToken: "minted-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "cache.json")
	const workers = 10

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ts, err := BuildTokenSource(context.Background(), TokenSourceConfig{
				OIDCURL:      srv.URL,
				ClientID:     "cid",
				ClientSecret: "csecret",
				CachePath:    cachePath,
			})
			if err != nil {
				errs <- err
				return
			}
			if _, err := ts.Token(); err != nil {
				errs <- err
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("worker error: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 OIDC mint across %d concurrent callers, got %d", workers, got)
	}
}
