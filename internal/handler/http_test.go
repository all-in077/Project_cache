package handler

import (
	"context"
	"encoding/json"
	"github.com/all-in077/Project_cache/internal/cache"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := cache.NewLRUcache(10, time.Minute, ctx)
	r := chi.NewRouter()
	NewHandler(c).RegisterRoutes(r)
	return r
}

func do(t *testing.T, h http.Handler, method, url, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSetThenGet(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, http.MethodPost, "/set", `{"key":"a","value":"hello"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("set: expected 200, got %d", rec.Code)
	}

	rec = do(t, h, http.MethodGet, "/get?key=a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data != "hello" {
		t.Fatalf("expected hello, got %q", resp.Data)
	}
}

func TestGetNotFound(t *testing.T) {
	h := newTestServer(t)
	if rec := do(t, h, http.MethodGet, "/get?key=nope", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetWithoutKey(t *testing.T) {
	h := newTestServer(t)
	if rec := do(t, h, http.MethodGet, "/get", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSetBadRequests(t *testing.T) {
	h := newTestServer(t)

	cases := map[string]string{
		"broken json": `{not json`,
		"no key":      `{"value":"x"}`,
		"no value":    `{"key":"a"}`,
		"empty body":  ``,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := do(t, h, http.MethodPost, "/set", body); rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestSetBodyTooLarge(t *testing.T) {
	h := newTestServer(t)
	big := `{"key":"a","value":"` + strings.Repeat("x", maxBodyBytes+1) + `"}`
	if rec := do(t, h, http.MethodPost, "/set", big); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDelete(t *testing.T) {
	h := newTestServer(t)
	do(t, h, http.MethodPost, "/set", `{"key":"a","value":"1"}`)

	if rec := do(t, h, http.MethodDelete, "/delete?key=a", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/delete?key=a", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/delete", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestStatsAndHealth(t *testing.T) {
	h := newTestServer(t)

	if rec := do(t, h, http.MethodGet, "/healthz", ""); rec.Code != http.StatusOK {
		t.Fatalf("healthz: expected 200, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/stats", ""); rec.Code != http.StatusOK {
		t.Fatalf("stats: expected 200, got %d", rec.Code)
	}
}
