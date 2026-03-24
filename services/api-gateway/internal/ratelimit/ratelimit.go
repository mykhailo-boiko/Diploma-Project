package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/haradrim/chainorchestra/internal/pkg/httpresponse"
)

type entry struct {
	count    int
	resetAt  time.Time
}

type Limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	limit   int
	window  time.Duration
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		entries: make(map[string]*entry),
		limit:   limit,
		window:  window,
	}

	go l.cleanup()

	return l
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !l.allow(ip) {
			httpresponse.Err(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests, try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[ip]
	if !ok || now.After(e.resetAt) {
		l.entries[ip] = &entry{count: 1, resetAt: now.Add(l.window)}
		return true
	}

	e.count++
	return e.count <= l.limit
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, e := range l.entries {
			if now.After(e.resetAt) {
				delete(l.entries, ip)
			}
		}
		l.mu.Unlock()
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := range len(xff) {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
