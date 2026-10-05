package httpx

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ClientIP restituisce l'indirizzo del client. L'header X-Real-IP, impostato
// dal reverse proxy, viene letto solo se la connessione arriva da una delle
// reti in trusted: altrimenti un client che raggiunge il backend direttamente
// potrebbe scrivere l'indirizzo che vuole. Restituisce un indirizzo non valido
// se RemoteAddr non è interpretabile.
func ClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	addr := peer.Addr().Unmap()

	isTrusted := slices.ContainsFunc(trusted, func(p netip.Prefix) bool {
		return p.Contains(addr)
	})
	if !isTrusted {
		return addr
	}

	forwarded, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-IP")))
	if err != nil {
		return addr
	}

	return forwarded.Unmap()
}
