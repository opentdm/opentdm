package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ipRateLimiter applies a per-client-IP token-bucket limit. It is in-process
// (no Redis/external store): adequate for a single replica, or behind a proxy
// that pins clients to a replica. Behind a load balancer across replicas the
// effective limit is per-replica — front it with a shared limiter for strict
// global limits. Idle buckets are swept by startCleanup.
//
// A zero/negative rate disables limiting entirely (Middleware is a pass-through),
// so operators who terminate rate limiting at a reverse proxy can turn it off.
type ipRateLimiter struct {
	disabled bool
	rate     rate.Limit
	burst    int

	mu      sync.Mutex
	buckets map[string]*ipBucket
}

type ipBucket struct {
	lim  *rate.Limiter
	seen time.Time
}

// newIPRateLimiter allows rpm requests per minute per IP, tolerating short
// bursts of burst. rpm <= 0 disables limiting.
func newIPRateLimiter(rpm, burst int) *ipRateLimiter {
	l := &ipRateLimiter{buckets: make(map[string]*ipBucket)}
	if rpm <= 0 {
		l.disabled = true
		return l
	}
	if burst < 1 {
		burst = 1
	}
	l.rate = rate.Limit(float64(rpm) / 60.0)
	l.burst = burst
	return l
}

// allow consumes one token for ip, reporting whether the request is within
// budget.
func (l *ipRateLimiter) allow(ip string, atSeen time.Time) bool {
	if l.disabled {
		return true
	}
	l.mu.Lock()
	b, ok := l.buckets[ip]
	if !ok {
		b = &ipBucket{lim: rate.NewLimiter(l.rate, l.burst)}
		l.buckets[ip] = b
	}
	b.seen = atSeen
	lim := b.lim
	l.mu.Unlock()
	return lim.Allow()
}

// Middleware rejects requests exceeding the per-IP budget with 429 problem+json.
func (l *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r), time.Now()) {
			w.Header().Set("Retry-After", "60")
			WriteProblem(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests",
				"Too many requests from your address; slow down and retry shortly.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cleanup evicts buckets not seen within maxIdle.
func (l *ipRateLimiter) cleanup(now time.Time, maxIdle time.Duration) {
	cutoff := now.Add(-maxIdle)
	l.mu.Lock()
	for ip, b := range l.buckets {
		if b.seen.Before(cutoff) {
			delete(l.buckets, ip)
		}
	}
	l.mu.Unlock()
}

// startCleanup sweeps idle buckets every interval until ctx is cancelled. It is
// a no-op when limiting is disabled.
func (l *ipRateLimiter) startCleanup(ctx context.Context, interval, maxIdle time.Duration) {
	if l.disabled {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				l.cleanup(time.Now(), maxIdle)
			}
		}
	}()
}
