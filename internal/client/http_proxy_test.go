package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testHTTPProxyID = "3fa85f64-5717-4562-b3fc-2c963f66afa6"

func TestHTTPProxyCachePolicyMarshalsToWireContract(t *testing.T) {
	req := HTTPProxyRequest{Data: HTTPProxyRequestData{
		Type:       "httpProxy",
		Attributes: HTTPProxyAttributes{Name: "p", CachePolicy: &HTTPProxyCachePolicy{Strategy: "BACKEND_RESPONSE_HEADERS"}},
	}}
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if want := `"cachePolicy":{"strategy":"BACKEND_RESPONSE_HEADERS"}`; !strings.Contains(string(payload), want) {
		t.Errorf("payload missing %s: %s", want, payload)
	}
}

func TestHTTPProxyCachePolicyOmittedWhenNil(t *testing.T) {
	payload, err := json.Marshal(HTTPProxyAttributes{Name: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "cachePolicy") {
		t.Errorf("payload unexpectedly contains cachePolicy: %s", payload)
	}
}

func TestHTTPProxyCachePolicyUnmarshalsFromResponse(t *testing.T) {
	body := `{"data":{"id":"` + testHTTPProxyID + `","type":"httpProxy","attributes":{"name":"p","cachePolicy":{"strategy":"DISABLED"}}}}`
	var resp HTTPProxyResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Attributes.CachePolicy == nil || resp.Data.Attributes.CachePolicy.Strategy != "DISABLED" {
		t.Fatalf("cachePolicy not decoded: %+v", resp.Data.Attributes.CachePolicy)
	}
}

func TestPurgeHTTPProxyContentCache(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotIfMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotIfMatch = r.Method, r.URL.Path, r.Header.Get("If-Match")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"data":{"id":"`+testHTTPProxyID+`","type":"httpProxy","attributes":{"name":"p","cachePolicy":{"strategy":"BACKEND_RESPONSE_HEADERS"}},"relationships":{"activeRollout":{"data":{"type":"change","id":"rollout-1"}}}}}`)
	}))
	defer server.Close()

	resp, err := NewClient(server.URL).PurgeHTTPProxyContentCache(context.Background(), testHTTPProxyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if want := "/api/v2/traffic-mgmt/traffic-configs/" + testHTTPProxyID + "/content-cache/purge"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
	if gotBody != "" {
		t.Errorf("expected empty request body, got %q", gotBody)
	}
	if gotIfMatch != "" {
		t.Errorf("expected no If-Match header, got %q", gotIfMatch)
	}
	if got := resp.Data.ActiveRolloutID(); got != "rollout-1" {
		t.Errorf("ActiveRolloutID = %q, want rollout-1", got)
	}
}

func TestPurgeHTTPProxyContentCacheErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"not deployed", http.StatusBadRequest, `{"title":"Invalid state transition","detail":"Content cache can only be purged for a deployed HTTP proxy that is ready for updates."}`, "Content cache can only be purged"},
		{"wrong type", http.StatusBadRequest, `Traffic config id is not an HTTP proxy`, "not an HTTP proxy"},
		{"not found", http.StatusNotFound, `Traffic Config not found`, "not found"},
		{"revision conflict", http.StatusPreconditionFailed, `The revision provided in the If-Match header does not match`, "revision"},
		{"unexpected ok", http.StatusOK, `{}`, "200"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()

			_, err := NewClient(server.URL).PurgeHTTPProxyContentCache(context.Background(), testHTTPProxyID)
			if err == nil {
				t.Fatal("expected error")
			}
			if !IsHTTPStatus(err, tc.status) {
				t.Errorf("expected HTTP status %d error, got %v", tc.status, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}
