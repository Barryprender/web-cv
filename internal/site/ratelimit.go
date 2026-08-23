package site

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// maxTrackedIPs caps the bucket map, so that traffic from many distinct
// sources cannot turn the limiter itself into a memory exhaustion vector.
const maxTrackedIPs = 4096

// ipRateLimiter is a per-client token bucket sized for a contact form: a small
// burst for someone retrying a failed send, then a slow drip. It exists to stop
// one client flooding the endpoint — and, once delivery is wired up, the inbox
// behind it.
//
// State is per process. That is the right size for a single small instance;
// a multi-instance deployment needs a shared store instead.
type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket

	burst  float64       // tokens available from cold
	refill float64       // tokens regained per second
	idle   time.Duration // drop a bucket untouched for this long
	now    func() time.Time

	// deny renders the 429. Plain text by default; a caller that speaks a
	// richer format (JSON, a redirect) replaces it.
	deny func(http.ResponseWriter, *http.Request, time.Duration)
}

type tokenBucket struct {
	tokens float64
	seen   time.Time
}

// newIPRateLimiter allows burst requests up front, then one more every per.
func newIPRateLimiter(burst float64, per time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: make(map[string]*tokenBucket),
		burst:   burst,
		refill:  1 / per.Seconds(),
		idle:    10 * time.Minute,
		now:     time.Now,
		deny: func(w http.ResponseWriter, _ *http.Request, _ time.Duration) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		},
	}
}

// allow reports whether key may spend a token now, and if not, how long it
// should wait before trying again.
func (l *ipRateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= maxTrackedIPs {
			l.sweepLocked(now)
		}
		b = &tokenBucket{tokens: l.burst, seen: now}
		l.buckets[key] = b
	}

	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.seen).Seconds()*l.refill)
	b.seen = now

	if b.tokens < 1 {
		wait := time.Duration(math.Ceil((1-b.tokens)/l.refill)) * time.Second
		return false, wait
	}
	b.tokens--
	return true, 0
}

// sweepLocked drops buckets nothing has touched recently. Callers hold l.mu.
func (l *ipRateLimiter) sweepLocked(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.seen) > l.idle {
			delete(l.buckets, key)
		}
	}
}

// limit wraps h so each client IP spends from its own budget.
func (l *ipRateLimiter) limit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := l.allow(clientIP(r))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			l.deny(w, r, retryAfter)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// clientIP keys the limiter off the peer address.
//
// Deliberately not read from X-Forwarded-For or similar: those are attacker
// controlled unless a proxy is guaranteed to overwrite them, and trusting one
// here would let a single client mint unlimited buckets and bypass the limit
// entirely. Deploying behind a proxy means reading that proxy's own client-IP
// header here, after confirming it strips any inbound copy.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
