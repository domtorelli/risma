package httpx

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("172.18.0.0/16")}

	tests := []struct {
		name       string
		remoteAddr string
		realIP     string
		want       string
	}{
		{"direct client", "203.0.113.7:5000", "", "203.0.113.7"},
		{"untrusted client cannot spoof", "203.0.113.7:5000", "198.51.100.1", "203.0.113.7"},
		{"trusted proxy", "172.18.0.5:5000", "198.51.100.1", "198.51.100.1"},
		{"trusted proxy without header", "172.18.0.5:5000", "", "172.18.0.5"},
		{"trusted proxy with invalid header", "172.18.0.5:5000", "not-an-ip", "172.18.0.5"},
		{"ipv6 client", "[2001:db8::1]:5000", "", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.realIP != "" {
				r.Header.Set("X-Real-IP", tt.realIP)
			}

			if got := ClientIP(r, trusted).String(); got != tt.want {
				t.Errorf("ClientIP = %s, want %s", got, tt.want)
			}
		})
	}
}
