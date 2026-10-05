package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

// fakeClock è un orologio che avanza solo quando lo dice il test.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(capacity int, period time.Duration) (*RateLimiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := NewRateLimiter(capacity, period, nil)
	l.now = clock.now
	return l, clock
}

func TestRateLimiterBurstAndRefill(t *testing.T) {
	l, clock := newTestLimiter(3, time.Minute) // 3 richieste al minuto: 1 ogni 20 s
	ip := netip.MustParseAddr("203.0.113.7")

	for i := range 3 {
		if ok, _ := l.allow(ip); !ok {
			t.Fatalf("request %d rejected within burst", i+1)
		}
	}

	ok, wait := l.allow(ip)
	if ok || wait != 20*time.Second {
		t.Fatalf("4th request: ok = %v, wait = %v, want rejected with 20s", ok, wait)
	}

	clock.advance(20 * time.Second)
	if ok, _ := l.allow(ip); !ok {
		t.Fatal("request rejected after refill")
	}

	if ok, _ := l.allow(ip); ok {
		t.Fatal("only one token should have been refilled")
	}
}

func TestRateLimiterSeparatesIPs(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)

	if ok, _ := l.allow(netip.MustParseAddr("203.0.113.7")); !ok {
		t.Fatal("first IP rejected")
	}

	if ok, _ := l.allow(netip.MustParseAddr("203.0.113.8")); !ok {
		t.Fatal("second IP limited by the first one")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	l, clock := newTestLimiter(2, time.Minute)
	l.allow(netip.MustParseAddr("203.0.113.7"))

	l.cleanup()
	if len(l.buckets) != 1 {
		t.Fatalf("buckets = %d, want 1 (not yet refilled)", len(l.buckets))
	}

	clock.advance(time.Minute)
	l.cleanup()
	if len(l.buckets) != 0 {
		t.Fatalf("buckets = %d, want 0 after full refill", len(l.buckets))
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	l, _ := newTestLimiter(1, time.Minute)
	h := l.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		r.RemoteAddr = "203.0.113.7:5000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	if w := request(); w.Code != http.StatusNoContent {
		t.Fatalf("first request: status = %d", w.Code)
	}

	w := request()
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "60" {
		t.Fatalf("second request: status = %d, Retry-After = %q", w.Code, w.Header().Get("Retry-After"))
	}
}
