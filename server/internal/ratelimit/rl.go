package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

type Limiter struct {
	mu    sync.Mutex
	tokens map[string][]time.Time
	n     int
	window time.Duration
}

func New(n int, window time.Duration) *Limiter {
	return &Limiter{tokens: map[string][]time.Time{}, n: n, window: window}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	var kept []time.Time
	for _, t := range l.tokens[key] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.n {
		l.tokens[key] = kept
		return false
	}
	kept = append(kept, now)
	l.tokens[key] = kept
	return true
}

func Middleware(l *Limiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(keyFn(r)) {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
