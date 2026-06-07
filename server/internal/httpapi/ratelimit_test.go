package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiter_AllowBurstThenDeny(t *testing.T) {
	l := newIPRateLimiter(60, 2) // 1 req/sec, burst 2
	at := time.Now()
	if !l.allow("1.1.1.1", at) || !l.allow("1.1.1.1", at) {
		t.Fatal("first two requests (within burst) should pass")
	}
	if l.allow("1.1.1.1", at) {
		t.Fatal("third immediate request should be rate-limited")
	}
	// A distinct IP has an independent bucket.
	if !l.allow("2.2.2.2", at) {
		t.Fatal("a different IP must not be limited by another IP's usage")
	}
}

func TestIPRateLimiter_Disabled(t *testing.T) {
	l := newIPRateLimiter(0, 0) // disabled
	at := time.Now()
	for i := 0; i < 100; i++ {
		if !l.allow("1.1.1.1", at) {
			t.Fatal("a disabled limiter must allow every request")
		}
	}
}

func TestIPRateLimiter_Middleware429(t *testing.T) {
	l := newIPRateLimiter(60, 1) // burst 1
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if got := call().Code; got != http.StatusOK {
		t.Fatalf("first call status = %d, want 200", got)
	}
	rec := call()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second call status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header on 429")
	}
}

func TestIPRateLimiter_CleanupEvictsIdle(t *testing.T) {
	l := newIPRateLimiter(60, 1)
	at := time.Now()
	l.allow("3.3.3.3", at)
	if len(l.buckets) != 1 {
		t.Fatalf("expected 1 bucket after a request, got %d", len(l.buckets))
	}
	l.cleanup(at.Add(time.Hour), 15*time.Minute) // everything is now idle
	if len(l.buckets) != 0 {
		t.Fatalf("expected the idle bucket to be evicted, got %d", len(l.buckets))
	}
}
