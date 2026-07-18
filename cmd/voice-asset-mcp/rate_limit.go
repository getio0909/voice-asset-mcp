package main

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const maxRateLimitClients = 8192

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

type rateLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	now         func() time.Time
	buckets     map[string]rateLimitBucket
	lastCleanup time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit: limit, window: window, now: time.Now,
		buckets: make(map[string]rateLimitBucket),
	}
}

func (limiter *rateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, remaining, resetAt := limiter.allow(rateLimitClientKey(r.RemoteAddr))
		w.Header().Set("RateLimit-Limit", strconv.Itoa(limiter.limit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		if !allowed {
			retryDuration := time.Until(resetAt)
			retryAfter := max(1, int((retryDuration+time.Second-1)/time.Second))
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (limiter *rateLimiter) allow(key string) (bool, int, time.Time) {
	now := limiter.now().UTC()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if limiter.lastCleanup.IsZero() || now.Sub(limiter.lastCleanup) >= limiter.window {
		for client, bucket := range limiter.buckets {
			if now.Sub(bucket.windowStart) >= 2*limiter.window {
				delete(limiter.buckets, client)
			}
		}
		limiter.lastCleanup = now
	}
	if len(limiter.buckets) >= maxRateLimitClients {
		key = "__overflow__"
	}
	bucket := limiter.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= limiter.window {
		bucket = rateLimitBucket{windowStart: now}
	}
	resetAt := bucket.windowStart.Add(limiter.window)
	if bucket.count >= limiter.limit {
		return false, 0, resetAt
	}
	bucket.count++
	limiter.buckets[key] = bucket
	return true, limiter.limit - bucket.count, resetAt
}

func rateLimitClientKey(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}
	if len(remoteAddr) > 200 {
		return remoteAddr[:200]
	}
	return remoteAddr
}
