package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

/*
Limiter is a token bucket per key (an IP address, an account): each key may make burst
attempts at once, then one more every interval. Keys idle long enough to be full again are
forgotten, so the map doesn't grow forever
*/
type Limiter struct {
	burst    float64
	interval time.Duration
	now      func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
	swept   time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

/*
Function:	NewLimiter
Purpose:	a Limiter allowing burst attempts per key, refilling one every interval
*/
func NewLimiter(burst int, interval time.Duration) *Limiter {
	return &Limiter{burst: float64(burst), interval: interval, buckets: map[string]*bucket{}}
}

/*
Function:	Allow
Purpose:	use one attempt for key, false when it has none left
*/
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	l.sweepLocked(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, seen: now}
		l.buckets[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.seen).Seconds()/l.interval.Seconds())
	b.seen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

/*
Function:	Reset
Purpose:	give key its full allowance back, after a successful login
*/
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// forgets keys that have refilled completely, at most once a minute; l.mu must be held
func (l *Limiter) sweepLocked(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now
	full := time.Duration(l.burst) * l.interval
	for key, b := range l.buckets {
		if now.Sub(b.seen) > full {
			delete(l.buckets, key)
		}
	}
}

func (l *Limiter) clock() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

var (
	// requests to the sign in, sign up and session endpoints per IP: 10 at once, then one every 10s
	ipLimiter = NewLimiter(10, 10*time.Second)
	// session refreshes and sign outs per IP: every open tab refreshes, so 30 at once, then
	// one every 2s
	sessionLimiter = NewLimiter(30, 2*time.Second)
	// wrong passwords per account, from any IP: 5 at once, then one a minute
	accountLimiter = NewLimiter(5, time.Minute)
)

/*
Function:	ClientIP
Purpose:	the address of the person making the request. Caddy sets X-Real-IP to the client's
address (taken from Cloudflare's Cf-Connecting-Ip for tunnel traffic) and overwrites anything
the client sent; the backend isn't reachable except through Caddy. X-Forwarded-For isn't used
because its first entry is whatever the client chose to send
*/
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

/*
Function:	RateLimit
Purpose:	middleware limiting requests per client IP, for endpoints that check passwords,
invite codes or session tokens
*/
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ipLimiter.Allow(ClientIP(r)) {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/*
Function:	RateLimitSessions
Purpose:	middleware limiting session refreshes and sign outs per client IP
*/
func RateLimitSessions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sessionLimiter.Allow(ClientIP(r)) {
			w.Header().Set("Retry-After", "2")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/*
Function:	allowPasswordAttempt
Purpose:	whether an account may have its password checked now. Limits guessing one account's
password from many IPs, which the per-IP limit alone doesn't stop
*/
func allowPasswordAttempt(account string) bool {
	return accountLimiter.Allow(account)
}
