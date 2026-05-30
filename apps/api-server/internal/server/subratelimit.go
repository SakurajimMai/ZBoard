package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

// Per-endpoint rate limit budgets. These are tuned for legitimate clients —
// real users hit each route at most a few times per minute, so the burst can
// stay tight while still leaving lots of headroom for normal usage.
//
//   - subscription: 60/min/IP — Clash/sing-box pollers refresh every 30 min or
//     more, the dashboard refreshes maybe a few times per visit. 60× real load.
//   - email-code:   8/hour/IP — the user-level cooldown is 120s (≈30/h max for a
//     single email); 8 lets a household share an IP, blocks mass enumeration.
//   - login:        20/min/IP — bcrypt cost 10 already throttles a single host
//     to a handful of attempts per second; 20/min keeps brute-force impractical.
const (
	subRateLimitWindow   = time.Minute
	subRateLimitBurst    = 60
	emailRateLimitWindow = time.Hour
	emailRateLimitBurst  = 8
	loginRateLimitWindow = time.Minute
	loginRateLimitBurst  = 20
	codeRateLimitWindow  = time.Minute
	codeRateLimitBurst   = 20
)

// fixedWindowLimiter is a per-key fixed-window counter. Held in memory because
// the data is cheap to lose (worst case an attacker gets one extra window's
// worth of requests after a restart) and we don't want a Redis dependency for
// these routes.
type fixedWindowLimiter struct {
	window time.Duration
	burst  int
	mu     sync.Mutex
	cells  map[string]*rateCell
}

type rateCell struct {
	count    int
	resetsAt time.Time
}

func newFixedWindowLimiter(window time.Duration, burst int) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		window: window,
		burst:  burst,
		cells:  make(map[string]*rateCell),
	}
}

// allow returns true when `key` is below the burst threshold for the current
// window. It also opportunistically expires stale entries so the map doesn't
// grow forever — every Nth call sweeps the whole map.
func (l *fixedWindowLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cells) > 10_000 {
		for k, w := range l.cells {
			if now.After(w.resetsAt) {
				delete(l.cells, k)
			}
		}
	}

	w, ok := l.cells[key]
	if !ok || now.After(w.resetsAt) {
		l.cells[key] = &rateCell{count: 1, resetsAt: now.Add(l.window)}
		return true
	}
	if w.count >= l.burst {
		return false
	}
	w.count++
	return true
}

// rateLimit returns a gin middleware that rejects requests over the configured
// per-IP window with 429. The same limiter pointer can be reused across mounts
// to share a budget; pass distinct limiters to keep budgets independent.
func rateLimit(l *fixedWindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP(), time.Now()) {
			httpx.Fail(c, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "请求过于频繁"))
			return
		}
		c.Next()
	}
}

// Backwards-compatible aliases. Older call sites used the subscription-specific
// names; the new ones are generic so the same machinery can guard email and
// login routes too.
type subRateLimiter = fixedWindowLimiter

func newSubRateLimiter() *fixedWindowLimiter {
	return newFixedWindowLimiter(subRateLimitWindow, subRateLimitBurst)
}

func subscriptionRateLimit(l *fixedWindowLimiter) gin.HandlerFunc { return rateLimit(l) }
