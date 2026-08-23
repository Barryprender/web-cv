package site

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fixedClock lets the bucket tests advance time without sleeping.
type fixedClock struct{ t time.Time }

func (c *fixedClock) now() time.Time      { return c.t }
func (c *fixedClock) add(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(burst float64, per time.Duration) (*ipRateLimiter, *fixedClock) {
	clock := &fixedClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := newIPRateLimiter(burst, per)
	l.now = clock.now
	return l, clock
}

func TestRateLimiterBurstThenDeny(t *testing.T) {
	l, _ := newTestLimiter(3, time.Minute)

	for i := 1; i <= 3; i++ {
		if ok, _ := l.allow("1.2.3.4"); !ok {
			t.Fatalf("request %d within burst was denied", i)
		}
	}
	ok, retryAfter := l.allow("1.2.3.4")
	if ok {
		t.Fatal("request past the burst was allowed")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter = %v, want a positive duration", retryAfter)
	}
}

func TestRateLimiterRefills(t *testing.T) {
	l, clock := newTestLimiter(2, time.Minute)

	l.allow("1.2.3.4")
	l.allow("1.2.3.4")
	if ok, _ := l.allow("1.2.3.4"); ok {
		t.Fatal("bucket should be empty")
	}

	// Half a refill period buys nothing...
	clock.add(30 * time.Second)
	if ok, _ := l.allow("1.2.3.4"); ok {
		t.Error("half a refill period should not yield a whole token")
	}
	// ...a full one does.
	clock.add(30 * time.Second)
	if ok, _ := l.allow("1.2.3.4"); !ok {
		t.Error("a full refill period should yield a token")
	}
}

func TestRateLimiterDoesNotExceedBurstWhenIdle(t *testing.T) {
	l, clock := newTestLimiter(3, time.Minute)

	l.allow("1.2.3.4")
	clock.add(24 * time.Hour) // long idle must not bank unlimited tokens

	for i := 1; i <= 3; i++ {
		if ok, _ := l.allow("1.2.3.4"); !ok {
			t.Fatalf("request %d after idle was denied", i)
		}
	}
	if ok, _ := l.allow("1.2.3.4"); ok {
		t.Error("idling banked more than the burst")
	}
}

func TestRateLimiterIsolatesClients(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)

	if ok, _ := l.allow("1.1.1.1"); !ok {
		t.Fatal("first client denied")
	}
	if ok, _ := l.allow("1.1.1.1"); ok {
		t.Fatal("first client not limited")
	}
	if ok, _ := l.allow("2.2.2.2"); !ok {
		t.Error("second client was affected by the first client's budget")
	}
}

func TestRateLimiterEvictsIdleBuckets(t *testing.T) {
	l, clock := newTestLimiter(1, time.Minute)
	l.idle = time.Minute

	for i := 0; i < maxTrackedIPs; i++ {
		l.allow(string(rune(i)))
	}
	if got := len(l.buckets); got != maxTrackedIPs {
		t.Fatalf("tracked %d buckets, want %d", got, maxTrackedIPs)
	}

	// Past the idle window, the next new client triggers a sweep.
	clock.add(2 * time.Minute)
	l.allow("fresh")

	if len(l.buckets) > 1 {
		t.Errorf("sweep left %d buckets, want the single fresh one", len(l.buckets))
	}
}

func TestLimitMiddleware(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)
	h := l.limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contact", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("first request = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contact", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 response is missing Retry-After")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{"host and port", "203.0.113.7:54321", "", "203.0.113.7"},
		{"ipv6", "[2001:db8::1]:443", "", "2001:db8::1"},
		{"no port", "203.0.113.7", "", "203.0.113.7"},
		{"ignores X-Forwarded-For", "203.0.113.7:54321", "9.9.9.9", "203.0.113.7"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/contact", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.forwarded != "" {
				r.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			if got := clientIP(r); got != tc.want {
				t.Errorf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The limiter is shared across every in-flight request, so its accounting has
// to hold under concurrency. Run with -race where a C toolchain is available.
func TestRateLimiterIsSafeUnderConcurrency(t *testing.T) {
	const burst = 50
	const goroutines = 200

	l, _ := newTestLimiter(burst, time.Hour) // no meaningful refill during the test

	var wg sync.WaitGroup
	granted := make(chan struct{}, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.allow("1.2.3.4"); ok {
				granted <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(granted)

	if got := len(granted); got != burst {
		t.Errorf("granted %d tokens concurrently, want exactly %d", got, burst)
	}
}
