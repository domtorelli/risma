package auth

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/matteotorelli/risma/backend/internal/httpx"
)

// Protezione CSRF con token double-submit: al login il backend imposta un
// cookie con un token casuale, leggibile da JavaScript; per ogni richiesta
// che modifica dati il frontend rimanda lo stesso valore nell'header
// X-CSRF-Token. Un sito esterno può far inviare i cookie al browser, ma non
// può leggerli, quindi non può scrivere l'header giusto.
//
// Il prefisso __Host- obbliga il browser ad accettare il cookie solo se è
// Secure, senza Domain e con Path=/: un sottodominio non può sovrascriverlo
// con un valore scelto dall'attaccante.
const (
	csrfCookieName = "__Host-csrf_token"
	CSRFHeader     = "X-CSRF-Token"
)

// csrfCookie crea il cookie del token CSRF. Non è HttpOnly perché il
// frontend deve leggerlo per copiarlo nell'header.
func csrfCookie(value string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

// setCSRFCookie imposta il token CSRF per una sessione che scade a
// expiresAt. Con rotate un token nuovo sostituisce quello esistente (al
// login, così un token precedente non sopravvive al cambio di utente);
// altrimenti si conserva quello già presente, così le altre schede aperte
// continuano a funzionare dopo un rinnovo della sessione.
func setCSRFCookie(w http.ResponseWriter, r *http.Request, expiresAt time.Time, rotate bool) {
	token := newRandomToken()

	if !rotate {
		if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
			token = cookie.Value
		}
	}

	http.SetCookie(w, csrfCookie(token, time.Until(expiresAt)))
}

// isSafeMethod indica i metodi che non modificano dati e non richiedono
// il token CSRF.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// RequireCSRF rifiuta le richieste che modificano dati senza un token CSRF
// valido: il cookie e l'header devono essere presenti e uguali. Il
// confronto in tempo costante non rivela, dalla durata, quanti caratteri
// iniziali coincidono. Va usato sulle rotte autenticate, dopo RequireAuth.
func (m *Middleware) RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		header := r.Header.Get(CSRFHeader)

		if err != nil || cookie.Value == "" || header == "" ||
			subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			httpx.Error(w, http.StatusForbidden, "richiesta non valida: token CSRF mancante o errato, ricarica la pagina")
			return
		}

		next.ServeHTTP(w, r)
	})
}
