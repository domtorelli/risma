package httpx

import (
	"context"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

// RateLimiter limita le richieste per indirizzo IP con un token bucket:
// ogni IP ha fino a capacity gettoni, ogni richiesta ne consuma uno e i
// gettoni si ricaricano in modo continuo (capacity ogni period). Lo stato è
// in memoria: vale per una sola istanza del backend e si azzera al riavvio.
type RateLimiter struct {
	capacity float64
	rate     float64 // gettoni ricaricati al secondo
	trusted  []netip.Prefix
	now      func() time.Time

	mu      sync.Mutex
	buckets map[netip.Addr]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter permette a ogni IP capacity richieste ogni period, anche
// tutte di seguito. trusted sono i proxy di cui si accetta X-Real-IP
// (vedi ClientIP).
func NewRateLimiter(capacity int, period time.Duration, trusted []netip.Prefix) *RateLimiter {
	return &RateLimiter{
		capacity: float64(capacity),
		rate:     float64(capacity) / period.Seconds(),
		trusted:  trusted,
		now:      time.Now,
		buckets:  map[netip.Addr]*bucket{},
	}
}

// refill aggiorna i gettoni di b al momento now.
func (l *RateLimiter) refill(b *bucket, now time.Time) {
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(l.capacity, b.tokens+elapsed*l.rate)
	b.last = now
}

// allow consuma un gettone dell'IP. Se non ce ne sono restituisce false e
// il tempo da attendere prima del prossimo.
func (l *RateLimiter) allow(ip netip.Addr) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[ip] = b
	}

	l.refill(b, now)

	if b.tokens < 1 {
		wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
		return false, wait
	}

	b.tokens--

	return true, 0
}

// Limit risponde 429 con l'header Retry-After quando l'IP del client ha
// esaurito le richieste.
func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.allow(ClientIP(r, l.trusted))
		if !ok {
			seconds := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			Error(w, http.StatusTooManyRequests, "troppe richieste: riprova tra "+strconv.Itoa(seconds)+" secondi")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanup elimina i secchi tornati pieni: per quegli IP la situazione è la
// stessa di chi non ha mai fatto richieste, quindi tenerli è solo memoria.
func (l *RateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	for ip, b := range l.buckets {
		l.refill(b, now)
		if b.tokens >= l.capacity {
			delete(l.buckets, ip)
		}
	}
}

// Run esegue la pulizia ogni interval finché ctx non viene cancellato
// (all'arresto del server). Va avviato in una goroutine.
func (l *RateLimiter) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.cleanup()
		}
	}
}
